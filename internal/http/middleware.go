package http

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/your-org/eegfaktura-member-onboarding/internal/metrics"
)

// publicIPRateLimiter holds per-IP sliding-window buckets for the public API.
// Two independent bucket maps so the relatively expensive submit endpoint can
// not exhaust the quota of the cheap confirm-email endpoint (and vice versa).
// Keys are client IP addresses; entries are evicted lazily on next access.
var (
	publicSubmitBuckets   = make(map[string]*ipBucket)
	publicSubmitBucketsMu sync.Mutex

	publicConfirmBuckets   = make(map[string]*ipBucket)
	publicConfirmBucketsMu sync.Mutex
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

func getIPBucket(buckets map[string]*ipBucket, mu *sync.Mutex, ip string) *ipBucket {
	mu.Lock()
	defer mu.Unlock()
	b, ok := buckets[ip]
	if !ok {
		b = &ipBucket{}
		buckets[ip] = b
	}
	return b
}

// realIP extracts the client IP. X-Real-IP and X-Forwarded-For are only
// honoured when the immediate peer (r.RemoteAddr) is itself a configured
// trusted proxy — otherwise an attacker reaching the pod directly could
// spoof their source IP by sending the header themselves and bypass the
// per-IP rate limit.
//
// Fall back to r.RemoteAddr (port stripped) when no trusted proxy chain
// applies.
func realIP(r *http.Request) string {
	peer := remoteAddrIP(r)
	if isTrustedProxy(peer) {
		if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
			return ip
		}
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			// First entry = original client; trim and return.
			return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
		}
	}
	return peer
}

// rateLimitMiddleware returns a per-IP sliding-window limiter for POST requests
// against `buckets`. Non-POST requests pass through untouched. The Retry-After
// header is set to `window` rounded up to whole seconds.
func rateLimitMiddleware(
	buckets map[string]*ipBucket,
	mu *sync.Mutex,
	limit int,
	window time.Duration,
	message string,
) func(http.Handler) http.Handler {
	retryAfter := int((window + time.Second - 1) / time.Second)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				ip := realIP(r)
				if !getIPBucket(buckets, mu, ip).allow(limit, window) {
					metrics.RateLimitHitsTotal.Inc()
					w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
					writeJSON(w, http.StatusTooManyRequests, map[string]string{
						"code":    "rate_limit_exceeded",
						"message": message,
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PublicSubmitRateLimitMiddleware limits POST requests to the public applications
// endpoint to 10 per 10 minutes per IP to mitigate automated bulk submissions.
var PublicSubmitRateLimitMiddleware = rateLimitMiddleware(
	publicSubmitBuckets, &publicSubmitBucketsMu,
	10, 10*time.Minute,
	"Zu viele Einreichungen. Bitte in 10 Minuten erneut versuchen.",
)

// PublicConfirmEmailRateLimitMiddleware limits POST requests to the e-mail
// confirmation endpoint. The 32-byte token already makes brute-force
// astronomical; this limit only exists as cheap defence-in-depth and is
// deliberately permissive (30/min/IP) so a tester behind shared NAT or a
// user re-opening the link a few times never hits it. Separate bucket from
// /applications so submits don't consume the confirm-email quota.
var PublicConfirmEmailRateLimitMiddleware = rateLimitMiddleware(
	publicConfirmBuckets, &publicConfirmBucketsMu,
	30, 1*time.Minute,
	"Zu viele Bestätigungsversuche. Bitte in einer Minute erneut versuchen.",
)

// MaxBodySize returns a middleware that wraps r.Body with http.MaxBytesReader.
// Decoding a body larger than `max` bytes returns an error from the standard
// json package; handlers already translate decode errors to 400, so the limit
// surfaces as a clean validation error to the client.
//
// Applied per route group (public/external get a tight limit; admin gets a
// larger budget for intro text + admin notes). GET/HEAD requests are passed
// through unchanged — they have no body to limit.
func MaxBodySize(max int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead {
				r.Body = http.MaxBytesReader(w, r.Body, max)
			}
			next.ServeHTTP(w, r)
		})
	}
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
	sweep := func(buckets map[string]*ipBucket, mu *sync.Mutex) {
		mu.Lock()
		defer mu.Unlock()
		for ip, b := range buckets {
			b.mu.Lock()
			empty := len(b.timestamps) == 0
			b.mu.Unlock()
			if empty {
				delete(buckets, ip)
			}
		}
	}
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sweep(publicSubmitBuckets, &publicSubmitBucketsMu)
				sweep(publicConfirmBuckets, &publicConfirmBucketsMu)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// healthProbePaths are noisy probe endpoints hit by Kubernetes every few
// seconds. We skip request-logging for them entirely; the duration
// histogram still gets the data points via the metrics path.
var healthProbePaths = map[string]bool{
	"/health": true,
	"/livez":  true,
	"/readyz": true,
}

func SlogRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthProbePaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		dur := time.Since(start)
		// Prometheus histogram: keep cardinality low — only method and
		// status-class are labelled, never the raw path or status code.
		metrics.HTTPRequestDurationSeconds.
			WithLabelValues(r.Method, metrics.StatusClass(ww.Status())).
			Observe(dur.Seconds())
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", dur.Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
			"remote_addr", realIP(r),
		)
	})
}
