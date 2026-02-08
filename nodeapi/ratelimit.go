package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// ipLimiter tracks per-IP request rates using a token bucket.
type ipLimiter struct {
	mu       sync.Mutex
	visitors map[string]*bucket
	rate     float64 // tokens per second
	burst    int     // max tokens
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

func newIPLimiter(rate float64, burst int) *ipLimiter {
	l := &ipLimiter{
		visitors: make(map[string]*bucket),
		rate:     rate,
		burst:    burst,
	}
	go l.cleanup()
	return l
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.visitors[ip]
	if !ok {
		l.visitors[ip] = &bucket{tokens: float64(l.burst) - 1, lastSeen: now}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// cleanup evicts stale entries every 60 seconds.
func (l *ipLimiter) cleanup() {
	for {
		time.Sleep(60 * time.Second)
		l.mu.Lock()
		for ip, b := range l.visitors {
			if time.Since(b.lastSeen) > 5*time.Minute {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

// rateLimitMiddleware wraps a handler with per-IP rate limiting.
func rateLimitMiddleware(limiter *ipLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}
		if !limiter.allow(ip) {
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}
