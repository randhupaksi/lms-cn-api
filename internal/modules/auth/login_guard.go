package auth

import (
	"crypto/sha256"
	"strings"
	"sync"
	"time"
)

type loginAttempt struct {
	count    int
	resetsAt time.Time
}

// LoginGuard limits repeated credential attempts without retaining a raw
// school identifier in process memory. A successful login clears its bucket.
type LoginGuard struct {
	mu       sync.Mutex
	attempts map[[sha256.Size]byte]loginAttempt
	limit    int
	window   time.Duration
	now      func() time.Time
}

func NewLoginGuard(limit int, window time.Duration) *LoginGuard {
	return &LoginGuard{
		attempts: make(map[[sha256.Size]byte]loginAttempt),
		limit:    limit,
		window:   window,
		now:      time.Now,
	}
}

func (g *LoginGuard) Consume(identifier string) bool {
	key := loginIdentifierKey(identifier)
	now := g.now().UTC()

	g.mu.Lock()
	defer g.mu.Unlock()

	attempt := g.attempts[key]
	if attempt.resetsAt.IsZero() || !now.Before(attempt.resetsAt) {
		attempt = loginAttempt{resetsAt: now.Add(g.window)}
	}
	if attempt.count >= g.limit {
		return false
	}
	attempt.count++
	g.attempts[key] = attempt

	if len(g.attempts) > 10000 {
		for attemptKey, candidate := range g.attempts {
			if !now.Before(candidate.resetsAt) {
				delete(g.attempts, attemptKey)
			}
		}
	}
	return true
}

func (g *LoginGuard) Reset(identifier string) {
	g.mu.Lock()
	delete(g.attempts, loginIdentifierKey(identifier))
	g.mu.Unlock()
}

func loginIdentifierKey(identifier string) [sha256.Size]byte {
	normalized := strings.ToLower(strings.TrimSpace(identifier))
	return sha256.Sum256([]byte(normalized))
}
