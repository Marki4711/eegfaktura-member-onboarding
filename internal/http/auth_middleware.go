package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const keycloakClaimsKey contextKey = "keycloak_claims"

// TenantClaim handles the Keycloak tenant attribute which is stored as a
// JSON-array string e.g. `["RC101665","RC101294"]` and emitted by the
// non-multivalued mapper as a plain string claim in the JWT.
type TenantClaim []string

func (t *TenantClaim) UnmarshalJSON(data []byte) error {
	// Happy path: already a proper JSON array.
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*t = arr
		return nil
	}
	// Fallback: the value is a JSON string that itself encodes a JSON array.
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(str), &arr); err != nil {
		// Not a JSON array — treat the whole string as a single tenant.
		*t = []string{str}
		return nil
	}
	*t = arr
	return nil
}

// KeycloakClaims holds the JWT claims we care about from Keycloak.
type KeycloakClaims struct {
	jwt.RegisteredClaims
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	// Tenant contains the RC numbers the user is allowed to manage.
	// Uses TenantClaim to handle both proper arrays and JSON-array strings.
	Tenant TenantClaim `json:"tenant"`
}

// IsSuperuser returns true when the token carries the superuser realm role.
func (c *KeycloakClaims) IsSuperuser() bool {
	for _, r := range c.RealmAccess.Roles {
		if r == "superuser" {
			return true
		}
	}
	return false
}

// IsTenantAdmin returns true when the token carries at least one tenant RC number.
func (c *KeycloakClaims) IsTenantAdmin() bool {
	return len(c.Tenant) > 0
}

// ClaimsFromContext retrieves KeycloakClaims from the request context.
// Returns nil when no claims are present (auth middleware disabled or unauthenticated).
func ClaimsFromContext(ctx context.Context) *KeycloakClaims {
	c, _ := ctx.Value(keycloakClaimsKey).(*KeycloakClaims)
	return c
}

// KeycloakAuthMiddleware validates the Bearer JWT on every request.
// When jwksURL is empty (dev mode), the middleware is a no-op and all requests pass through.
// Otherwise:
//   - missing/invalid token → 401
//   - valid token but no superuser role and no tenant → 403
//   - valid + authorized → claims stored in context
func KeycloakAuthMiddleware(jwksURL, issuer string) func(http.Handler) http.Handler {
	if jwksURL == "" {
		return func(next http.Handler) http.Handler { return next }
	}

	k, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		panic("failed to initialize Keycloak JWKS: " + err.Error())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"code":    "unauthorized",
					"message": "Authentifizierung erforderlich.",
				})
				return
			}

			claims := &KeycloakClaims{}
			opts := []jwt.ParserOption{jwt.WithExpirationRequired()}
			if issuer != "" {
				opts = append(opts, jwt.WithIssuer(issuer))
			}

			_, err := jwt.ParseWithClaims(tokenStr, claims, k.Keyfunc, opts...)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"code":    "unauthorized",
					"message": "Ungültiger oder abgelaufener Token.",
				})
				return
			}

			if !claims.IsSuperuser() && !claims.IsTenantAdmin() {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"code":    "forbidden",
					"message": "Kein Zugriff auf den Admin-Bereich.",
				})
				return
			}

			ctx := context.WithValue(r.Context(), keycloakClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TestHeaderAuthMiddleware liest Test-Claims aus Request-Headern statt aus
// einem signierten JWT. Ausschließlich für E2E-Tests in CI gedacht — ist
// `cmd/server/main.go` so verdrahtet, dass diese Middleware verweigert
// startet, wenn `ENVIRONMENT=production`.
//
// Headers:
//
//	X-Test-Tenant      — Komma-Liste von RC-Numbers (z. B. "RC123456,RC456")
//	X-Test-Superuser   — "true" → Superuser-Rolle injiziert
//	X-Test-Subject     — optionaler Subject-Claim (sonst "e2e-test-user")
//
// Verhalten:
//
//   - Beide Header fehlen → 401 (Tests können auth-required asserten)
//   - X-Test-Superuser=true → Claims mit Realm-Role "superuser"
//   - X-Test-Tenant gesetzt → Claims mit Tenant-RC-Numbers
//   - Sonst → 403 (kein Tenant, kein Superuser)
//
// Sicherheits-Kontext: die Header sind triviale Forgery-Möglichkeiten und
// dürfen NIE auf einem öffentlich erreichbaren Endpoint stehen. Die
// `ENVIRONMENT=production`-Sperre in main.go ist der einzige Schutz.
func TestHeaderAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantHeader := r.Header.Get("X-Test-Tenant")
			superuser := r.Header.Get("X-Test-Superuser") == "true"
			subject := r.Header.Get("X-Test-Subject")
			if subject == "" {
				subject = "e2e-test-user"
			}
			if tenantHeader == "" && !superuser {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"code":    "unauthorized",
					"message": "Authentifizierung erforderlich.",
				})
				return
			}
			claims := &KeycloakClaims{}
			claims.Subject = subject
			if superuser {
				claims.RealmAccess.Roles = []string{"superuser"}
			}
			if tenantHeader != "" {
				for _, rc := range strings.Split(tenantHeader, ",") {
					rc = strings.TrimSpace(rc)
					if rc != "" {
						claims.Tenant = append(claims.Tenant, rc)
					}
				}
			}
			if !claims.IsSuperuser() && !claims.IsTenantAdmin() {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"code":    "forbidden",
					"message": "Kein Zugriff auf den Admin-Bereich.",
				})
				return
			}
			ctx := context.WithValue(r.Context(), keycloakClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}
