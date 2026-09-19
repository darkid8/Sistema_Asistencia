package http

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// loginRateLimiter throttles repeated failed sign-in attempts for the same
// username/IP pair. It only counts failures: a caller who eventually types
// the right password is never penalized for the attempts that came before.
// State is kept in memory, which is enough for a single instance; a
// multi-instance deployment would need a shared store (e.g. Redis) instead.
type loginRateLimiter struct {
	mu          sync.Mutex
	failure     map[string][]time.Time
	maxAttempts int
	window      time.Duration
}

// newLoginRateLimiter builds a limiter allowing maxAttempts failures per
// window, per key.
func newLoginRateLimiter(maxAttempts int, window time.Duration) *loginRateLimiter {
	return &loginRateLimiter{
		failure:     make(map[string][]time.Time),
		maxAttempts: maxAttempts,
		window:      window,
	}
}

// blocked reports whether key has already reached maxAttempts failures inside
// the current window. It never leaves an entry behind for a key with no
// recent failures -- every successful sign-in calls this first, so leaving
// an empty slice in the map here would grow it by one entry per login ever
// attempted, for the life of the process.
func (l *loginRateLimiter) blocked(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent := l.prune(key, now)
	if len(recent) == 0 {
		delete(l.failure, key)
		return false
	}
	l.failure[key] = recent
	return len(recent) >= l.maxAttempts
}

// recordFailure appends a failed attempt for key.
func (l *loginRateLimiter) recordFailure(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent := l.prune(key, now)
	l.failure[key] = append(recent, now)
}

// reset clears the failure history for key, called after a successful sign in.
func (l *loginRateLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failure, key)
}

// prune drops attempts that fell outside the window. Caller must hold l.mu.
func (l *loginRateLimiter) prune(key string, now time.Time) []time.Time {
	kept := l.failure[key][:0]
	for _, moment := range l.failure[key] {
		if now.Sub(moment) < l.window {
			kept = append(kept, moment)
		}
	}
	return kept
}

// loginRateLimitKey combines the caller IP and the attempted username so a
// single malicious IP cannot lock out a legitimate user's account by
// hammering someone else's login, and a shared IP (e.g. an office NAT) does
// not lock out every technician because one of them mistyped a password.
func loginRateLimitKey(request *http.Request, username string) string {
	return clientIP(request) + "|" + strings.ToLower(strings.TrimSpace(username))
}

// clientIP extracts the caller's address, preferring the first hop recorded
// by a trusted reverse proxy when present.
func clientIP(request *http.Request) string {
	if forwarded := request.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}
	return host
}
