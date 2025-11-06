/*
 * Simple token bucket limiter with observable token count, compatible with our Limiter interface.
 */

package ratelimit

import (
	"context"
	"sync"
	"time"
)

// tokenLimiter implements a basic token-bucket rate limiter.
// It is intentionally minimal but thread-safe and provides Tokens() for metrics.
type tokenLimiter struct {
	mu         sync.Mutex
	rate       float64   // tokens per second
	burst      int       // bucket capacity
	tokens     float64   // current tokens
	lastRefill time.Time // last refill time
}

// NewTokenLimiter creates a new token-bucket limiter.
func NewTokenLimiter(r float64, b int) *tokenLimiter {
	if r < 0 {
		r = 0
	}
	if b < 0 {
		b = 0
	}
	return &tokenLimiter{
		rate:       r,
		burst:      b,
		tokens:     float64(b),
		lastRefill: time.Now(),
	}
}

// refill adds tokens based on elapsed time.
func (t *tokenLimiter) refill(now time.Time) {
	if t.rate <= 0 {
		// No refill if rate is zero or negative
		return
	}
	elapsed := now.Sub(t.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	t.tokens += elapsed * t.rate
	if t.tokens > float64(t.burst) {
		t.tokens = float64(t.burst)
	}
	t.lastRefill = now
}

// Allow attempts to take 1 token.
func (t *tokenLimiter) Allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.refill(now)
	if t.tokens >= 1 {
		t.tokens -= 1
		return true
	}
	return false
}

// AllowN attempts to take n tokens.
func (t *tokenLimiter) AllowN(n int) bool {
	if n <= 0 {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.refill(now)
	need := float64(n)
	if t.tokens >= need {
		t.tokens -= need
		return true
	}
	return false
}

// Wait blocks until one token is available or the context is canceled.
func (t *tokenLimiter) Wait(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		t.mu.Lock()
		now := time.Now()
		t.refill(now)
		if t.tokens >= 1 {
			t.tokens -= 1
			t.mu.Unlock()
			return nil
		}

		// Compute time to wait for next token
		var sleep time.Duration
		if t.rate <= 0 {
			// If no refill rate, effectively infinite wait; respect context only
			sleep = time.Second
		} else {
			deficit := 1 - t.tokens
			// Time to generate enough tokens
			secs := deficit / t.rate
			if secs < 0.0 {
				secs = 0.0
			}
			sleep = time.Duration(secs * 1e9) // seconds to duration
			if sleep <= 0 {
				sleep = time.Millisecond
			}
		}
		t.mu.Unlock()

		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
			// loop and try again
		}
	}
}

// Tokens returns the current available tokens (refilled to now).
func (t *tokenLimiter) Tokens() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.refill(time.Now())
	return t.tokens
}

// Limit returns the configured rate (tokens per second).
func (t *tokenLimiter) Limit() float64 { return t.rate }

// Burst returns the bucket capacity.
func (t *tokenLimiter) Burst() int { return t.burst }
