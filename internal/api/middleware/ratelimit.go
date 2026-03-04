package middleware

import (
	"net/http"
	"sync"
	"time"
)

// tokenBucket is a simple per-key in-memory token bucket rate limiter.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	maxToken float64
	refillPS float64 // tokens added per second
	lastTime time.Time
}

func newTokenBucket(capacity float64, refillPerSecond float64) *tokenBucket {
	return &tokenBucket{
		tokens:   capacity,
		maxToken: capacity,
		refillPS: refillPerSecond,
		lastTime: time.Now(),
	}
}

// allow returns true and consumes one token if a token is available.
func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.lastTime = now

	b.tokens += elapsed * b.refillPS
	if b.tokens > b.maxToken {
		b.tokens = b.maxToken
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// bucketStore manages per-project token buckets.
type bucketStore struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	capacity float64
	refillPS float64
}

func newBucketStore(capacity float64, refillPerSecond float64) *bucketStore {
	return &bucketStore{
		buckets:  make(map[string]*tokenBucket),
		capacity: capacity,
		refillPS: refillPerSecond,
	}
}

func (s *bucketStore) get(key string) *tokenBucket {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[key]
	if !ok {
		b = newTokenBucket(s.capacity, s.refillPS)
		s.buckets[key] = b
	}
	return b
}

// pre-allocated stores for each rate limit tier.
var (
	generationLimiter = newBucketStore(10, 10.0/3600)   // 10/hour
	webhookLimiter    = newBucketStore(100, 100.0/60)    // 100/minute
	defaultLimiter    = newBucketStore(1000, 1000.0/60)  // 1000/minute
)

func rateLimitMiddleware(store *bucketStore, keyFn func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			b := store.get(key)
			if !b.allow() {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func projectKey(r *http.Request) string {
	pid := ProjectIDFromCtx(r.Context())
	if pid.String() != "00000000-0000-0000-0000-000000000000" {
		return pid.String()
	}
	// Fallback to remote address when project context is absent.
	return r.RemoteAddr
}

// GenerationRateLimit enforces 10 requests/hour per project for generation endpoints.
func GenerationRateLimit() func(http.Handler) http.Handler {
	return rateLimitMiddleware(generationLimiter, projectKey)
}

// WebhookRateLimit enforces 100 requests/minute per project for webhook endpoints.
func WebhookRateLimit() func(http.Handler) http.Handler {
	return rateLimitMiddleware(webhookLimiter, projectKey)
}

// DefaultRateLimit enforces 1000 requests/minute per project for all other endpoints.
func DefaultRateLimit() func(http.Handler) http.Handler {
	return rateLimitMiddleware(defaultLimiter, projectKey)
}
