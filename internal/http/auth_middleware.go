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

// KeycloakClaims holds the JWT claims we care about from Keycloak.
type KeycloakClaims struct {
	jwt.RegisteredClaims
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	// tenant is a multivalued user attribute mapped via Client Scope Mapper.
	// It contains the RC numbers the user is allowed to manage.
	Tenant []string `json:"tenant"`
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
