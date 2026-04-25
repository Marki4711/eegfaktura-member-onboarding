package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// publicIPRateLimiter holds per-IP sliding-window buckets for the public API.
// Keys are client IP addresses; entries are evicted lazily on next access.
var (
	publicIPBuckets   = make(map[string]*ipBucket)
	publicIPBucketsMu sync.Mutex
)

type ipBucket struct {
	mu         sync.Mutex
	timestamps []time.Time
}

func (b *ipBucket) allow(limit int, window time.Duration) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-window)
	kept := b.timestamps[:0]
	for _, t := range b.timestamps {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	b.timestamps = kept
	if len(b.timestamps) >= limit {
		return false
	}
	b.timestamps = append(b.timestamps, now)
	return true
}

func getIPBucket(ip string) *ipBucket {
	publicIPBucketsMu.Lock()
	defer publicIPBucketsMu.Unlock()
	b, ok := publicIPBuckets[ip]
	if !ok {
		b = &ipBucket{}
		publicIPBuckets[ip] = b
	}
	return b
}

// realIP extracts the client IP, honouring X-Real-IP and X-Forwarded-For set by nginx.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.SplitN(fwd, ",", 2)[0]
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// PublicSubmitRateLimitMiddleware limits POST requests to the public applications
// endpoint to 10 per 10 minutes per IP to mitigate automated bulk submissions.
func PublicSubmitRateLimitMiddleware(next http.Handler) http.Handler {
	const limit = 10
	const window = 10 * time.Minute
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			ip := realIP(r)
			if !getIPBucket(ip).allow(limit, window) {
				w.Header().Set("Retry-After", "600")
				writeJSON(w, http.StatusTooManyRequests, map[string]string{
					"code":    "rate_limit_exceeded",
					"message": "Zu viele Einreichungen. Bitte in 10 Minuten erneut versuchen.",
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware sets defensive HTTP headers on every response.
// HSTS is set for 2 years; the API never serves HTML so CSP is restrictive.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		next.ServeHTTP(w, r)
	})
}

// StartIPBucketCleanup starts a background goroutine that evicts idle IP buckets
// every 10 minutes to prevent unbounded memory growth. Stops when ctx is cancelled.
func StartIPBucketCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				publicIPBucketsMu.Lock()
				for ip, b := range publicIPBuckets {
					b.mu.Lock()
					empty := len(b.timestamps) == 0
					b.mu.Unlock()
					if empty {
						delete(publicIPBuckets, ip)
					}
				}
				publicIPBucketsMu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func SlogRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
			"remote_addr", r.RemoteAddr,
		)
	})
}
