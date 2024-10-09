package soroban

import (
	"os"

	"gopkg.in/yaml.v2"
)

var (
	DefaultOptions = Options{
		LogLevel: "info",
		LogFile:  "-",
		Soroban: SorobanInfo{
			Config:        "",
			Confidential:  "",
			Domain:        "samourai",
			DirectoryType: "default",
			WithTor:       false,
			Seed:          "",
			Hostname:      "localhost",
			Port:          4242,
			Announce:      "soroban.announce.nodes",
		},
		P2P: P2PInfo{
			Seed:          "",
			Bootstrap:     "",
			Hostname:      "",
			ListenPort:    1042,
			LowWater:      16, // = 2*Gossip.Dlo
			HighWater:     40, // = 2*Gossip.Dhi
			Room:          "samourai-p2p",
			DHTServerMode: false,
		},
		Gossip: GossipInfo{
			D:          10, // = ceil(exp(ln(NB_P2P_NODES)/AVG_NB_HOPS))
			Dlo:        8,  // = 0.8*Gossip.D
			Dhi:        20, // = 2*Gossip.D
			Dout:       5,  // = min(Gossip.Dlo, Gossip.D/2)
			Dscore:     7,  // = ceil(2*Gossip.D/3)
			Dlazy:      10, // = Gossip.D
			PrunePeers: 40, // = 2*Gossip.Dhi
			Limit:      40, // = 2*Gossip.Dhi
		},
		IPC: IPCInfo{
			Subject:           "ipc.server",
			ChildID:           0,
			ChildProcessCount: 0,
			NatsHost:          "localhost",
			NatsPort:          4322,
		},
	}
)

type Options struct {
	LogLevel string
	LogFile  string
	Soroban  SorobanInfo
	P2P      P2PInfo
	IPC      IPCInfo
	Gossip   GossipInfo
}

func (p *Options) Load(config string) {
	if len(config) == 0 {
		return
	}
	if data, err := os.ReadFile(config); err == nil {
		var o Options
		if err := o.parse(data); err == nil {
			p.Merge(o)
		}
	}
}

func (p *Options) parse(data []byte) error {
	return yaml.Unmarshal(data, p)
}

func (p *Options) Merge(o Options) {
	if len(o.LogLevel) > 0 {
		p.LogLevel = o.LogLevel
	}

	if len(o.LogFile) > 0 {
		p.LogFile = o.LogFile
	}

	p.Soroban.Merge(o.Soroban)
	p.P2P.Merge(o.P2P)
	p.Gossip.Merge(o.Gossip)
	p.IPC.Merge(o.IPC)
}

type SorobanInfo struct {
	Config        string
	Confidential  string
	Domain        string
	DirectoryType string
	WithTor       bool
	Seed          string
	Hostname      string
	Port          int
	Announce      string
	IPv4          bool
}

func (p *SorobanInfo) Merge(s SorobanInfo) {
	if len(s.Config) > 0 {
		p.Config = s.Config
	}
	if len(s.Confidential) > 0 {
		p.Confidential = s.Confidential
	}
	if len(s.Domain) > 0 {
		p.Domain = s.Domain
	}
	if len(s.DirectoryType) > 0 {
		p.DirectoryType = s.DirectoryType
	}
	if s.WithTor {
		p.WithTor = s.WithTor
	}
	if len(s.Seed) > 0 {
		p.Seed = s.Seed
	}
	if len(s.Hostname) > 0 {
		p.Hostname = s.Hostname
	}
	if s.Port > 0 {
		p.Port = s.Port
	}
	if len(s.Announce) > 0 {
		p.Announce = s.Announce
	}
	if s.IPv4 {
		p.IPv4 = s.IPv4
	}
}

type P2PInfo struct {
	Seed          string
	Bootstrap     string
	Hostname      string
	ListenPort    int
	LowWater      int
	HighWater     int
	Room          string
	DHTServerMode bool
}

func (p *P2PInfo) Merge(i P2PInfo) {
	if len(i.Seed) > 0 {
		p.Seed = i.Seed
	}
	if len(i.Bootstrap) > 0 {
		p.Bootstrap = i.Bootstrap
	}
	if len(i.Hostname) > 0 {
		p.Hostname = i.Hostname
	}
	if i.ListenPort > 0 {
		p.ListenPort = i.ListenPort
	}
	if i.LowWater > 0 {
		p.LowWater = i.LowWater
	}
	if i.HighWater > 0 {
		p.HighWater = i.HighWater
	}
	if len(i.Room) > 0 {
		p.Room = i.Room
	}
	if i.DHTServerMode {
		p.DHTServerMode = i.DHTServerMode
	}
}

type GossipInfo struct {
	D          int
	Dlo        int
	Dhi        int
	Dout       int
	Dscore     int
	Dlazy      int
	PrunePeers int
	Limit      int
}

func (p *GossipInfo) Merge(i GossipInfo) {
	if i.D > 0 {
		p.D = i.D
	}
	if i.Dlo > 0 {
		p.Dlo = i.Dlo
	}
	if i.Dhi > 0 {
		p.Dhi = i.Dhi
	}
	if i.Dout > 0 {
		p.Dout = i.Dout
	}
	if i.Dscore > 0 {
		p.Dscore = i.Dscore
	}
	if i.Dlazy > 0 {
		p.Dlazy = i.Dlazy
	}
	if i.PrunePeers > 0 {
		p.PrunePeers = i.PrunePeers
	}
	if i.Limit > 0 {
		p.Limit = i.Limit
	}
}

type IPCInfo struct {
	Subject           string
	ChildID           int
	ChildProcessCount int
	NatsHost          string
	NatsPort          int
}

func (p *IPCInfo) Merge(i IPCInfo) {
	if len(i.Subject) > 0 {
		p.Subject = i.Subject
	}
	if i.ChildID > 0 {
		p.ChildID = i.ChildID
	}
	if i.ChildProcessCount > 0 {
		p.ChildProcessCount = i.ChildProcessCount
	}
	if len(i.NatsHost) > 0 {
		p.NatsHost = i.NatsHost
	}
	if i.NatsPort > 0 {
		p.NatsPort = i.NatsPort
	}
}
