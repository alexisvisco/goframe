package coretypes

import "time"

type (
	CacheEntry struct {
		Key       string     `json:"key"`
		Value     []byte     `json:"value"`
		ExpiresAt *time.Time `json:"expires_at"`
	}

	CacheEvent struct {
		Type  string      `json:"type"`
		Key   string      `json:"key"`
		Value interface{} `json:"value,omitempty"`
	}

	CacheOptions struct {
		TTL time.Duration
	}

	CacheOption func(*CacheOptions)
)

func WithTTL(ttl time.Duration) CacheOption {
	return func(c *CacheOptions) { c.TTL = ttl }
}
