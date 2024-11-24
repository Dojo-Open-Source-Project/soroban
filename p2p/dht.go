package p2p

import (
	"context"
	"math/rand/v2"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func NewDHT(ctx context.Context, host host.Host, mode dht.ModeOpt, bootstrapPeers ...multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	// Keep only 2 randomly selected boostrap peers
	filteredPeers := bootstrapPeers
	if len(filteredPeers) > 2 {
		rand.Shuffle(len(filteredPeers), func(i, j int) { filteredPeers[i], filteredPeers[j] = filteredPeers[j], filteredPeers[i] })
		filteredPeers = filteredPeers[:2]
	}

	var bootstrapAddr []peer.AddrInfo
	for _, peerAddr := range filteredPeers {
		peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			return nil, err
		}
		bootstrapAddr = append(bootstrapAddr, *peerinfo)
	}

	kdht, err := dht.New(
		ctx,
		host,
		dht.ProtocolPrefix("/soroban"),
		dht.Mode(mode),
		dht.BootstrapPeers(bootstrapAddr...),
		dht.Concurrency(16),
	)
	if err != nil {
		return nil, err
	}

	return kdht, nil
}
