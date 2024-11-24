package internal

import (
	soroban "soroban"
	"soroban/internal/memory"
)

type DirectoryType string

const (
	DirectoryTypeMemory DirectoryType = "directory-memory"
)

func DefaultDirectory(domain string) soroban.Directory {
	return NewDirectory(domain, DirectoryTypeMemory)
}

func NewDirectory(domain string, DirectoryType DirectoryType) soroban.Directory {
	switch DirectoryType {
	case DirectoryTypeMemory:
		return memory.NewWithDomain(domain, memory.DefaultCacheCapacity, memory.DefaultCacheTTL)
	default:
		return memory.NewWithDomain(domain, memory.DefaultCacheCapacity, memory.DefaultCacheTTL)
	}
}
