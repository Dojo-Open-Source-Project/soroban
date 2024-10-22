package p2p

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	soroban "code.samourai.io/wallet/samourai-soroban"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
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
	topic     *pubsub.Topic
	OnMessage chan Message
}

func (p *P2P) Valid() bool {
	return p.topic != nil
}

func (p *P2P) Start(ctx context.Context, optionsP2P soroban.P2PInfo, optionsGossip soroban.GossipInfo, ready chan struct{}) error {
	p2pSeed := optionsP2P.Seed
	hostname := optionsP2P.Hostname
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

	var opts []libp2p.Option
	if len(p2pSeed) > 0 {
		p2pOpts, err := initTorP2P(ctx, p2pSeed, mgr, listenPort)
		if err != nil {
			return err
		}
		opts = append(opts, p2pOpts...)
	}

	// fallback to clearnet
	if len(opts) == 0 {
		log.Info("P2P Start clearnet")
		opts = append(opts, []libp2p.Option{
			libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", hostname, listenPort)),
			libp2p.DefaultTransports,
			libp2p.DefaultMuxers,
			libp2p.NoSecurity,
			libp2p.RandomIdentity,
			libp2p.DefaultPeerstore,
			libp2p.DefaultMultiaddrResolver,
			libp2p.ConnectionManager(mgr),
		}...)
	}

	// create the swarm
	swarm.BackoffBase = 30 * time.Second
	host, err := libp2p.New(opts...)
	if err != nil {
		return err
	}

	isBoostrapNode := false

	localAddresses := []string{}
	for _, a := range host.Addrs() {
		localAddresses = append(localAddresses, a.String())
		log.Printf("Peer address: %s/p2p/%s", a.String(), host.ID().String())
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

	dht, err := NewDHT(ctx, host, mode, bootstrapAddresses...)
	if err != nil {
		return err
	}

	// Initialize the routing discovery for the pubsub protocol
	routingDiscovery := drouting.NewRoutingDiscovery(dht)
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
		host,
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

	go p.subscribe(ctx, subscriber, host.ID())

	return nil
}

// start subsriber to topic
func (p *P2P) subscribe(ctx context.Context, subscriber *pubsub.Subscription, hostID peer.ID) {
	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			log.Printf("failed to get next message")
			<-time.After(time.Second)
			continue
		}

		// only consider messages delivered by other peers
		if msg.ReceivedFrom == hostID {
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
