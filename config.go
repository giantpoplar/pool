package pool

import "time"

type CacheMethod int

const (
	UndefinedCacheMethod CacheMethod = iota
	FIFO
	FILO
)

var (
	defaultConfig = Config{
		CacheMethod: FIFO,
		InitCap:     0,
		MaxCap:      1,
		IdleTimeout: 3 * time.Second,
		WaitTimeout: 3 * time.Second,
		IOTimeout:   30 * time.Second,
		DialTimeout: 30 * time.Second,
	}
)

// Config defines all to create a pool
type Config struct {
	CacheMethod

	// Connection number pool will create in initializing
	InitCap int

	// Max connection number pool can create
	MaxCap int

	// Pool will close connection when connection is unused after live timeout
	IdleTimeout time.Duration

	//timeout to Get, default to 3
	WaitTimeout time.Duration

	// Read and write timeout
	IOTimeout time.Duration

	// Dial to server timeout
	DialTimeout time.Duration
}

// Merge config with new one and return merged result and whether config is changed
func (c *Config) Merge(new Config) (Config, bool) {
	result := *c

	changed := false
	if new.MaxCap > 0 && new.MaxCap != result.MaxCap {
		changed = true
		result.MaxCap = new.MaxCap
	}
	if new.DialTimeout > 0 && new.DialTimeout != result.DialTimeout {
		changed = true
		result.DialTimeout = new.DialTimeout
	}
	if new.IOTimeout > 0 && new.IOTimeout != result.IOTimeout {
		changed = true
		result.IOTimeout = new.IOTimeout
	}
	if new.WaitTimeout > 0 && new.WaitTimeout != result.WaitTimeout {
		changed = true
		result.WaitTimeout = new.WaitTimeout
	}
	if new.IdleTimeout > 0 && new.IdleTimeout != result.IdleTimeout {
		changed = true
		result.IdleTimeout = new.IdleTimeout
	}
	return result, changed

}
