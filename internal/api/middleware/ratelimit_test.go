package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestMemoryStore_AllowAndDeny(t *testing.T) {
	s := NewMemoryStore()

	// Should allow up to capacity.
	for i := 0; i < 5; i++ {
		allowed, _ := s.Allow("test-key", 5, 1.0)
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// Next request should be denied.
	allowed, retryAfter := s.Allow("test-key", 5, 1.0)
	if allowed {
		t.Fatal("request after capacity should be denied")
	}
	if retryAfter < 1 {
		t.Fatalf("retryAfter should be >= 1, got %d", retryAfter)
	}
}

func TestMemoryStore_DifferentKeys(t *testing.T) {
	s := NewMemoryStore()

	// Exhaust key A.
	for i := 0; i < 3; i++ {
		s.Allow("key-a", 3, 0.1)
	}
	allowed, _ := s.Allow("key-a", 3, 0.1)
	if allowed {
		t.Fatal("key-a should be exhausted")
	}

	// Key B should still be allowed.
	allowed, _ = s.Allow("key-b", 3, 0.1)
	if !allowed {
		t.Fatal("key-b should be allowed independently")
	}
}

func TestRateLimitMiddleware_Returns429(t *testing.T) {
	cfg := LimitConfig{
		Capacity:     1,
		RefillPerSec: 0.001, // very slow refill
		KeyFunc:      func(r *http.Request) string { return "test" },
	}

	handler := rateLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should pass.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rec.Code)
	}

	// Second request should be rate limited.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rec.Code)
	}

	retryAfter := rec.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Fatal("expected Retry-After header")
	}
	secs, err := strconv.Atoi(retryAfter)
	if err != nil || secs < 1 {
		t.Fatalf("Retry-After should be a positive integer, got %q", retryAfter)
	}
}

func TestRateLimitMiddleware_OrgKey(t *testing.T) {
	// Without org context, should fall back to IP key.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	key := orgKey(req)
	if key != "ip:1.2.3.4:5678" {
		t.Fatalf("expected IP-based key, got %q", key)
	}
}

func TestRateLimitMiddleware_IPKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	key := ipKey(req)
	if key != "ip:10.0.0.1:9999" {
		t.Fatalf("expected ip:10.0.0.1:9999, got %q", key)
	}
}
