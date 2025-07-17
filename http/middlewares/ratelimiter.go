package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
)

// RateLimiterOptions configures the rate limiter middleware.
type RateLimiterOptions struct {
	Cache contracts.Cache
	Rules []RateRule

	// Global determines if rate limiting applies globally (per IP) or per path (per IP+path).
	// When true, rate limits apply across all paths for an IP.
	// When false, each path has its own rate limit counter.
	// Defaults to true.
	Global bool

	// HeaderLimit is the name of the header containing the request limit.
	// Defaults to "X-RateLimit-Limit".
	HeaderLimit string
	// HeaderRemaining is the name of the header that indicates how many
	// requests are left in the current window. Defaults to
	// "X-RateLimit-Remaining".
	HeaderRemaining string
	// HeaderReset is the name of the header giving the Unix time when the
	// rate limit will reset. Defaults to "X-RateLimit-Reset".
	HeaderReset string
}

// RateRule defines a rate limit rule with a simple API
type RateRule struct {
	Requests int           // Number of requests allowed
	Per      time.Duration // Time window
}

// Common rate limit presets
var (
	Per1Second  = RateRule{Requests: 1, Per: time.Second}
	Per5Second  = RateRule{Requests: 5, Per: time.Second}
	Per10Second = RateRule{Requests: 10, Per: time.Second}

	Per1Minute   = RateRule{Requests: 1, Per: time.Minute}
	Per10Minute  = RateRule{Requests: 10, Per: time.Minute}
	Per60Minute  = RateRule{Requests: 60, Per: time.Minute}
	Per100Minute = RateRule{Requests: 100, Per: time.Minute}

	Per100Hour  = RateRule{Requests: 100, Per: time.Hour}
	Per1000Hour = RateRule{Requests: 1000, Per: time.Hour}
	Per5000Hour = RateRule{Requests: 5000, Per: time.Hour}

	Per10000Day = RateRule{Requests: 10000, Per: 24 * time.Hour}
	Per50000Day = RateRule{Requests: 50000, Per: 24 * time.Hour}
)

// Helper constructors
func RequestsPer(requests int, duration time.Duration) RateRule {
	return RateRule{Requests: requests, Per: duration}
}

func RequestsPerSecond(requests int) RateRule {
	return RateRule{Requests: requests, Per: time.Second}
}

func RequestsPerMinute(requests int) RateRule {
	return RateRule{Requests: requests, Per: time.Minute}
}

func RequestsPerHour(requests int) RateRule {
	return RateRule{Requests: requests, Per: time.Hour}
}

func RequestsPerDay(requests int) RateRule {
	return RateRule{Requests: requests, Per: 24 * time.Hour}
}

// RateLimiter limits the number of requests per IP using contracts.Cache.
func RateLimiter(opts *RateLimiterOptions) func(http.Handler) http.Handler {
	if opts == nil || opts.Cache == nil {
		panic("RateLimiterOptions.Cache is required")
	}
	if len(opts.Rules) == 0 {
		opts.Rules = []RateRule{Per60Minute} // Default: 60 requests per minute
	}

	// Set default headers
	if opts.HeaderLimit == "" {
		opts.HeaderLimit = "X-RateLimit-Limit"
	}
	if opts.HeaderRemaining == "" {
		opts.HeaderRemaining = "X-RateLimit-Remaining"
	}
	if opts.HeaderReset == "" {
		opts.HeaderReset = "X-RateLimit-Reset"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			path := r.URL.Path
			discriminant := fmt.Sprintf("%s.%s", path, ip)
			if opts.Global {
				discriminant = ip
			}
			now := time.Now()
			ctx := r.Context()

			// Check all rules - if any rule is violated, block the request
			for _, rule := range opts.Rules {
				if violated, reset := checkRule(ctx, opts.Cache, discriminant, now, rule); violated {
					w.Header().Set(opts.HeaderLimit, fmt.Sprintf("%d", rule.Requests))
					w.Header().Set(opts.HeaderRemaining, "0")
					w.Header().Set(opts.HeaderReset, fmt.Sprintf("%d", reset.Unix()))
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
			}

			// Increment counters for all rules and find the most restrictive
			var mostRestrictiveRule RateRule
			var minRemaining int = int(^uint(0) >> 1) // max int
			var nextReset time.Time

			for _, rule := range opts.Rules {
				remaining, reset := incrementRule(ctx, opts.Cache, discriminant, now, rule)
				if remaining < minRemaining {
					minRemaining = remaining
					mostRestrictiveRule = rule
					nextReset = reset
				}
			}

			w.Header().Set(opts.HeaderLimit, fmt.Sprintf("%d", mostRestrictiveRule.Requests))
			w.Header().Set(opts.HeaderRemaining, fmt.Sprintf("%d", minRemaining))
			w.Header().Set(opts.HeaderReset, fmt.Sprintf("%d", nextReset.Unix()))

			next.ServeHTTP(w, r)
		})
	}
}

func checkRule(ctx context.Context, cache contracts.Cache, discriminant string, now time.Time, rule RateRule) (bool, time.Time) {
	windowStart := now.Add(-rule.Per)
	key := fmt.Sprintf("ratelimit:%s:%s", discriminant, rule.Per.String())

	// Get current request timestamps
	var timestamps []int64
	_ = cache.Get(ctx, key, &timestamps)

	// Filter out old timestamps
	var validTimestamps []int64
	for _, ts := range timestamps {
		if time.Unix(ts, 0).After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if limit would be exceeded
	if len(validTimestamps) >= rule.Requests {
		// Reset time is when the oldest request expires
		oldestRequest := time.Unix(validTimestamps[0], 0)
		reset := oldestRequest.Add(rule.Per)
		return true, reset
	}

	return false, time.Time{}
}

func incrementRule(ctx context.Context, cache contracts.Cache, discriminant string, now time.Time, rule RateRule) (int, time.Time) {
	windowStart := now.Add(-rule.Per)
	key := fmt.Sprintf("ratelimit:%s:%s", discriminant, rule.Per.String())

	// Get current request timestamps
	var timestamps []int64
	_ = cache.Get(ctx, key, &timestamps)

	// Filter out old timestamps and add current request
	var validTimestamps []int64
	for _, ts := range timestamps {
		if time.Unix(ts, 0).After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Add current request
	validTimestamps = append(validTimestamps, now.Unix())

	// Store updated timestamps
	_ = cache.Put(ctx, key, validTimestamps, coretypes.WithTTL(rule.Per))

	// Calculate remaining requests
	remaining := rule.Requests - len(validTimestamps)
	if remaining < 0 {
		remaining = 0
	}

	// Calculate reset time (when oldest request expires)
	var reset time.Time
	if len(validTimestamps) > 0 {
		oldestRequest := time.Unix(validTimestamps[0], 0)
		reset = oldestRequest.Add(rule.Per)
	} else {
		reset = now.Add(rule.Per)
	}

	return remaining, reset
}
