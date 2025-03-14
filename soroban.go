package soroban

import (
	"context"
	"time"
)

// Service interface
type Service interface{}

// Soroban interface
type Soroban interface {
	ID() string
	Register(ctx context.Context, name string, service Service) error
	Start(ctx context.Context, hostname string, port int, statsLabel string, statusLabel string) error
	StartWithTor(ctx context.Context, hostname string, port int, seed string, statsLabel string, statusLabel string) error
	Stop(ctx context.Context)
	WaitForStart(ctx context.Context)
}

type NameValue map[string]string

type StatusInfo struct {
	Clients      NameValue `json:"clients,omitempty"`
	Cluster      NameValue `json:"cluster,omitempty"`
	Commandstats NameValue `json:"commandstats,omitempty"`
	CPU          NameValue `json:"cpu,omitempty"`
	Keyspace     NameValue `json:"keyspace,omitempty"`
	Memory       NameValue `json:"memory,omitempty"`
	Persistence  NameValue `json:"persistence,omitempty"`
	Replication  NameValue `json:"replication,omitempty"`
	Server       NameValue `json:"server,omitempty"`
	Stats        NameValue `json:"stats,omitempty"`
	Raw          string    `json:"_raw,omitempty"`
}

// Directory interface
type Directory interface {
	// Status returs internal informations
	Status() (StatusInfo, error)

	// TimeToLive return duration from mode.
	TimeToLive(mode string) time.Duration

	// List return all known values for this key.
	List(key string) ([]string, error)

	// Add value in key.
	// TimeToLive must be greter or equals to 1 second.
	// Multiple values can be store with the same key.
	// TTL is the same for all values.
	Add(key, value string, TTL time.Duration) error

	// Remove value from key.
	Remove(key, value string) error
}
