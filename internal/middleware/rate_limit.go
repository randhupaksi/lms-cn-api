package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	count    int
	resetsAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
	limit   int
	window  time.Duration
	now     func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{buckets: make(map[string]rateBucket), limit: limit, window: window, now: time.Now}
}

func (l *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := l.now().UTC()
		key := c.ClientIP()
		l.mu.Lock()
		bucket := l.buckets[key]
		if bucket.resetsAt.IsZero() || !now.Before(bucket.resetsAt) {
			bucket = rateBucket{resetsAt: now.Add(l.window)}
		}
		bucket.count++
		l.buckets[key] = bucket
		remaining := max(0, l.limit-bucket.count)
		resetSeconds := max(1, int(time.Until(bucket.resetsAt).Seconds()))
		if len(l.buckets) > 10000 {
			for bucketKey, candidate := range l.buckets {
				if !now.Before(candidate.resetsAt) {
					delete(l.buckets, bucketKey)
				}
			}
		}
		l.mu.Unlock()

		c.Header("X-RateLimit-Limit", strconv.Itoa(l.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.Itoa(resetSeconds))
		if bucket.count > l.limit {
			c.Header("Retry-After", strconv.Itoa(resetSeconds))
			response.Error(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Terlalu banyak permintaan. Silakan coba lagi beberapa saat.")
			c.Abort()
			return
		}
		c.Next()
	}
}
