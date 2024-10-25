package p2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"sync"

	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	rand2 "math/rand/v2"
	"os"
	"strings"
	"time"

	soroban "code.samourai.io/wallet/samourai-soroban"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/multiformats/go-multiaddr"

	log "github.com/sirupsen/logrus"
)

// P2P for distributed soroban
type P2P struct {
	OnMessage chan Message
	ChildID   int
	topic     *pubsub.Topic
	host      host.Host
	dht       *dht.IpfsDHT
}

func (p *P2P) Valid() bool {
	return p.topic != nil
}

func (p *P2P) Start(ctx context.Context, optionsP2P soroban.P2PInfo, optionsGossip soroban.GossipInfo, ready chan struct{}) error {
	p2pSeed := optionsP2P.Seed
	listenPort := optionsP2P.ListenPort
	lowWater := optionsP2P.LowWater
	highWater := optionsP2P.HighWater
	bootstrap := optionsP2P.Bootstrap
	room := optionsP2P.Room
	isDHTServerMode := optionsP2P.DHTServerMode

	d := optionsGossip.D
	dhi := optionsGossip.Dhi
	dlo := optionsGossip.Dlo
	dout := optionsGossip.Dout
	dlazy := optionsGossip.Dlazy
	dscore := optionsGossip.Dscore
	prunePeers := optionsGossip.PrunePeers
	limit := optionsGossip.Limit

	ctx = network.WithDialPeerTimeout(ctx, 3*time.Minute)
	defer func() {
		ready <- struct{}{}
	}()

	mgr, err := connmgr.NewConnManager(lowWater, highWater)
	if err != nil {
		return err
	}

	// Generates a p2p seed automatically if none has been provided
	if len(p2pSeed) == 0 {
		_, pri, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Fatal(err)
		}
		p2pSeed = hex.EncodeToString(pri.Seed())
	}

	var opts []libp2p.Option
	p2pOpts, err := initTorP2P(ctx, p2pSeed, mgr, listenPort)
	if err != nil {
		return err
	}
	opts = append(opts, p2pOpts...)

	// create the swarm
	swarm.BackoffBase = 30 * time.Second
	p.host, err = libp2p.New(opts...)
	if err != nil {
		return err
	}

	isBoostrapNode := false

	localAddresses := []string{}
	for _, a := range p.host.Addrs() {
		localAddresses = append(localAddresses, a.String())
		log.Printf("Peer address: %s/p2p/%s", a.String(), p.host.ID().String())
	}

	bootstrapAddresses := []multiaddr.Multiaddr{}
	bootstrapAddr := strings.Split(bootstrap, ",")
	for _, b := range bootstrapAddr {
		a, err := multiaddr.NewMultiaddr(b)
		if err != nil {
			return err
		}
		bootstrapAddresses = append(bootstrapAddresses, a)
		log.Debugf("Bootstrap address: %s", a.String())
		for _, l := range localAddresses {
			if strings.HasPrefix(a.String(), l) {
				isBoostrapNode = true
				break
			}
		}
	}

	mode := dht.ModeClient
	if isBoostrapNode {
		mode = dht.ModeServer
	} else if isDHTServerMode {
		mode = dht.ModeAuto
	}

	log.Debugf("isBootstrap: %t", isBoostrapNode)
	log.Debugf("DHT mode: %s", mode)

	// Connect node to a few peers persisted on disk
	if !isBoostrapNode && optionsP2P.PeerstoreFile != "-" {
		err = p.ConnectToPersistedPeers(ctx, optionsP2P)
		if err != nil {
			return err
		}
	}

	// Initialize and bootstrap the DHT
	p.dht, err = NewDHT(ctx, p.host, mode, bootstrapAddresses...)
	if err != nil {
		return err
	}

	// Initialize the routing discovery for the pubsub protocol
	routingDiscovery := drouting.NewRoutingDiscovery(p.dht)
	discOpts := []discovery.Option{discovery.Limit(limit), discovery.TTL(30 * time.Second)}

	// Initialize the gossipsub protocol
	params := pubsub.DefaultGossipSubParams()
	params.D = d
	params.Dlo = dlo
	params.Dhi = dhi
	params.Dout = dout
	params.Dscore = dscore
	params.Dlazy = dlazy
	params.GossipFactor = 0.25
	params.PrunePeers = prunePeers

	gossipSub, err := pubsub.NewGossipSub(
		ctx,
		p.host,
		pubsub.WithGossipSubParams(params),
		pubsub.WithDiscovery(routingDiscovery, pubsub.WithDiscoveryOpts(discOpts...)),
	)
	if err != nil {
		return err
	}

	topic, err := gossipSub.Join(room)
	if err != nil {
		return err
	}

	p.topic = topic

	// subscribe to topic
	subscriber, err := topic.Subscribe()
	if err != nil {
		return err
	}

	go p.subscribe(ctx, subscriber)

	// Start persisting the peerstore
	if optionsP2P.PeerstoreFile != "-" {
		go StartPeerstorePersistence(ctx, optionsP2P, p)
	}

	return nil
}

// start subsriber to topic
func (p *P2P) subscribe(ctx context.Context, subscriber *pubsub.Subscription) {
	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			log.Printf("failed to get next message")
			<-time.After(time.Second)
			continue
		}

		// only consider messages delivered by other peers
		if msg.ReceivedFrom == p.host.ID() {
			continue
		}

		message, err := MessageFromBytes(msg.Data)
		if err != nil {
			log.Debug("Skip unkown message")
			continue
		}

		p.OnMessage <- message
	}
}

// Publish to topic
func (p *P2P) Publish(ctx context.Context, msg string) error {
	if len(msg) == 0 {
		return errors.New("failed to publish empty message")
	}
	if p.topic == nil {
		return nil
	}
	p.topic.Publish(ctx, []byte(msg))
	return nil
}

// Publish to topic
func (p *P2P) PublishJson(ctx context.Context, context string, payload interface{}) error {
	message, err := NewMessage(context, payload)
	if err != nil {
		return err
	}

	data, err := message.ToBytes()
	if err != nil {
		return err
	}

	return p.Publish(ctx, string(data))
}

// Persist the Peerstore
func StartPeerstorePersistence(ctx context.Context, optionsP2P soroban.P2PInfo, p *P2P) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Exiting Announce Loop")
			return
		case <-ticker.C:
			p.PersistPeerstore(ctx, optionsP2P)
		}
	}
}

func (p *P2P) PersistPeerstore(ctx context.Context, optionsP2P soroban.P2PInfo) error {
	var peersAddrs []peer.AddrInfo
	var i int
	for _, peerId := range p.host.Network().Peerstore().PeersWithAddrs() {
		if p.host.ID() == peerId {
			continue
		}
		peerInfo := p.host.Network().Peerstore().PeerInfo(peerId)
		if len(peerInfo.Addrs) > 0 {
			peersAddrs = append(peersAddrs, peerInfo)
			i++
			if i > optionsP2P.HighWater {
				break
			}
		}
	}

	if i > 0 {
		bytes, _ := json.Marshal(peersAddrs)
		filepath := fmt.Sprintf(optionsP2P.PeerstoreFile+".c%d.json", p.ChildID)
		file, err := os.Create(filepath)
		if err != nil {
			return err
		}
		defer file.Close()
		file.Write(bytes)
		file.Write([]byte("\n"))
	}

	return nil
}

func (p *P2P) ConnectToPersistedPeers(ctx context.Context, optionsP2P soroban.P2PInfo) error {
	filepath := fmt.Sprintf(optionsP2P.PeerstoreFile+".c%d.json", p.ChildID)
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil
	}

	var peersAddrs []peer.AddrInfo
	err = json.Unmarshal(bytes, &peersAddrs)
	if err != nil {
		return err
	}

	maxNbSelectedPeers := optionsP2P.LowWater / 2
	if len(peersAddrs) > maxNbSelectedPeers {
		rand2.Shuffle(len(peersAddrs), func(i, j int) { peersAddrs[i], peersAddrs[j] = peersAddrs[j], peersAddrs[i] })
		peersAddrs = peersAddrs[:maxNbSelectedPeers+1]
	}

	if len(peersAddrs) > 0 {
		var wg sync.WaitGroup
		for _, peerinfo := range peersAddrs {
			if p.host.ID() != peerinfo.ID {
				wg.Add(1)
				go func(peerinfo peer.AddrInfo) {
					log.Debugf("Boostrapping attempt with %v", peerinfo)
					defer wg.Done()
					if err := p.host.Connect(ctx, peerinfo); err != nil {
						log.WithError(err).Warnf("Bootstrap warning: %v", peerinfo)
					}
				}(peerinfo)
			}
		}
		wg.Wait()
	}

	return nil
}
