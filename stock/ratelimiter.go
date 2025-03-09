package stock

import (
    "sync"
    "time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
    tokensPerMinute int
    tokens          int
    lastRefill      time.Time
    mu              sync.Mutex
}

// NewRateLimiter creates a new rate limiter with specified requests per minute
func NewRateLimiter(tokensPerMinute int) *RateLimiter {
    return &RateLimiter{
        tokensPerMinute: tokensPerMinute,
        tokens:          tokensPerMinute, // Start with full tokens
        lastRefill:      time.Now(),
    }
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    // Refill tokens based on elapsed time
    now := time.Now()
    elapsed := now.Sub(rl.lastRefill)
    
    // Calculate how many tokens to add based on elapsed time
    newTokens := int(float64(elapsed.Seconds()) * float64(rl.tokensPerMinute) / 60.0)
    if newTokens > 0 {
        rl.tokens += newTokens
        if rl.tokens > rl.tokensPerMinute {
            rl.tokens = rl.tokensPerMinute
        }
        rl.lastRefill = now
    }
    
    // If no tokens available, sleep until next token
    if rl.tokens <= 0 {
        rl.mu.Unlock() // Release lock while waiting
        sleepTime := time.Second * 60 / time.Duration(rl.tokensPerMinute)
        time.Sleep(sleepTime)
        rl.mu.Lock()
        rl.tokens = 1 // Guaranteed to have 1 token after sleeping
    }
    
    // Consume token
    rl.tokens--
}