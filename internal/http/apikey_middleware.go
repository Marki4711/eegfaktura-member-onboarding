package http

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

const (
	dailyQuotaLimit  = 200
	burstWindowSecs  = 60
	burstRateLimit   = 10
)

type contextKeyType string

const externalRCNumberKey contextKeyType = "external_rc_number"

// ExternalRCNumberFromContext retrieves the RC number set by APIKeyMiddleware.
func ExternalRCNumberFromContext(ctx context.Context) string {
	v, _ := ctx.Value(externalRCNumberKey).(string)
	return v
}

// rateBucket tracks in-memory burst rate limiting for a single API key.
type rateBucket struct {
	mu        sync.Mutex
	timestamps []time.Time
}

// allow returns true when the request is within the burst limit.
func (b *rateBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-burstWindowSecs * time.Second)

	// Remove timestamps outside the sliding window.
	kept := b.timestamps[:0]
	for _, t := range b.timestamps {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	b.timestamps = kept

	if len(b.timestamps) >= burstRateLimit {
		return false
	}
	b.timestamps = append(b.timestamps, now)
	return true
}

var (
	rateBuckets   = make(map[string]*rateBucket)
	rateBucketsMu sync.Mutex
)

func getBucket(keyHash string) *rateBucket {
	rateBucketsMu.Lock()
	defer rateBucketsMu.Unlock()
	b, ok := rateBuckets[keyHash]
	if !ok {
		b = &rateBucket{}
		rateBuckets[keyHash] = b
	}
	return b
}

// hashAPIKey returns the lowercase SHA-256 hex digest of the given key.
func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", sum)
}

// APIKeyMiddleware authenticates requests via `Authorization: Bearer <api-key>`.
// On success it stores the EEG RC number in the request context.
func APIKeyMiddleware(repo *application.ExternalAPIKeyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := extractBearerToken(r)
			if rawKey == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"code":    "unauthorized",
					"message": "API-Key fehlt.",
				})
				return
			}

			keyHash := hashAPIKey(rawKey)

			apiKey, err := repo.GetByKeyHash(keyHash)
			if err != nil {
				if errors.Is(err, shared.ErrNotFound) {
					writeJSON(w, http.StatusUnauthorized, map[string]string{
						"code":    "unauthorized",
						"message": "Ungültiger oder widerrufener API-Key.",
					})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"code":    "internal_error",
					"message": "Interner Fehler bei der Authentifizierung.",
				})
				return
			}

			// Burst rate limit (in-memory, per pod).
			if !getBucket(keyHash).allow() {
				w.Header().Set("Retry-After", "60")
				writeJSON(w, http.StatusTooManyRequests, map[string]string{
					"code":    "rate_limit_exceeded",
					"message": "Zu viele Anfragen. Bitte in 60 Sekunden erneut versuchen.",
				})
				return
			}

			// Daily quota (DB-backed, pod-safe).
			newCount, err := repo.IncrementDailyCount(apiKey.ID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"code":    "internal_error",
					"message": "Interner Fehler.",
				})
				return
			}
			if newCount > dailyQuotaLimit {
				w.Header().Set("Retry-After", secondsUntilMidnightUTC())
				writeJSON(w, http.StatusTooManyRequests, map[string]string{
					"code":    "quota_exceeded",
					"message": "Tageskontingent erschöpft. Einreichungen werden ab Mitternacht UTC wieder akzeptiert.",
				})
				return
			}

			ctx := context.WithValue(r.Context(), externalRCNumberKey, apiKey.RCNumber)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func secondsUntilMidnightUTC() string {
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return fmt.Sprintf("%d", int(midnight.Sub(now).Seconds()))
}
