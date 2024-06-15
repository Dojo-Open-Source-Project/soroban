package onion

import (
	"context"
	"encoding/base32"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/cretz/bine/tor"
	"golang.org/x/net/proxy"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	tpt "github.com/libp2p/go-libp2p/core/transport"
	ma "github.com/multiformats/go-multiaddr"
)

type OnionTransport struct {
	sk       crypto.PrivKey
	service  *tor.OnionService
	dialer   proxy.Dialer
	laddr    ma.Multiaddr
	upgrader tpt.Upgrader
}

type OnionTransportC func(upgrader tpt.Upgrader) (tpt.Transport, error)

func NewOnionTransportC(sk crypto.PrivKey, dialer proxy.Dialer, service *tor.OnionService) OnionTransportC {
	return func(upgrader tpt.Upgrader) (tpt.Transport, error) {
		return NewOnionTransport(sk, dialer, service, upgrader)
	}
}

type OnionConn struct {
	net.Conn
	transport tpt.Transport
	laddr     ma.Multiaddr
	raddr     ma.Multiaddr
}

func (o *OnionConn) LocalMultiaddr() ma.Multiaddr {
	return o.laddr
}

func (o *OnionConn) RemoteMultiaddr() ma.Multiaddr {
	return o.raddr
}

func NewOnionTransport(sk crypto.PrivKey, dialer proxy.Dialer, service *tor.OnionService, upgrader tpt.Upgrader) (tpt.Transport, error) {
	return &OnionTransport{
		sk:       sk,
		service:  service,
		dialer:   dialer,
		upgrader: upgrader,
	}, nil
}

func (t *OnionTransport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (tpt.CapableConn, error) {
	onionAddress, err := raddr.ValueForProtocol(ma.P_ONION3)
	if err != nil {
		return nil, err
	}
	onionConn := OnionConn{
		transport: tpt.Transport(t),
		laddr:     t.laddr,
		raddr:     raddr,
	}
	split := strings.Split(onionAddress, ":")
	onionConn.Conn, err = t.dialer.Dial("tcp4", split[0]+".onion:"+split[1])
	if err != nil {
		return nil, err
	}

	return t.upgrader.Upgrade(ctx, t, &onionConn, network.DirOutbound, p, &network.NullScope{})
}

func (t *OnionTransport) CanDial(addr ma.Multiaddr) bool {
	// only dial out on onion addresses
	return isValidOnionMultiAddr(addr)
}

func (t *OnionTransport) Listen(laddr ma.Multiaddr) (tpt.Listener, error) {
	netaddr, err := laddr.ValueForProtocol(ma.P_ONION3)
	if err != nil {
		return nil, err
	}

	// retreive onion service virtport
	addr := strings.Split(netaddr, ":")
	if len(addr) != 2 {
		return nil, fmt.Errorf("failed to parse onion address")
	}

	listener := OnionListener{
		laddr:     laddr,
		upgrader:  t.upgrader,
		transport: t,
	}

	listener.listener = t.service.LocalListener
	t.laddr = laddr

	ul := t.upgrader.UpgradeListener(t, &listener)

	return ul, nil
}

func (t *OnionTransport) Protocols() []int {
	return []int{ma.P_ONION3}
}

func (t *OnionTransport) Proxy() bool {
	return true
}

func isValidOnionMultiAddr(a ma.Multiaddr) bool {
	if len(a.Protocols()) != 1 {
		return false
	}

	// check for correct network type
	if a.Protocols()[0].Name != "onion3" {
		return false
	}

	// split into onion address and port
	addr, err := a.ValueForProtocol(ma.P_ONION3)
	if err != nil {
		return false
	}
	split := strings.Split(addr, ":")
	if len(split) != 2 {
		return false
	}

	_, err = base32.StdEncoding.DecodeString(strings.ToUpper(split[0]))
	if err != nil {
		return false
	}

	// onion port number
	port, err := strconv.Atoi(split[1])
	if err != nil {
		return false
	}
	if port >= 65536 || port < 1024 {
		return false
	}

	return true
}
