package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// emailConfirmationTokenBytes is the entropy budget for the URL-token. 32 bytes
// produces a 43-char base64url string and 256-bit search space — practically
// unguessable.
const emailConfirmationTokenBytes = 32

// GenerateEmailConfirmationToken returns a pair (plaintext, sha256-hex) where
// plaintext goes into the e-mail URL and the hash is what we persist on the
// application row. The plaintext is **never** persisted, so a database dump
// does not leak active confirmation tokens.
func GenerateEmailConfirmationToken() (string, string, error) {
	buf := make([]byte, emailConfirmationTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("rand: %w", err)
	}
	plaintext := base64.RawURLEncoding.EncodeToString(buf)
	return plaintext, hashEmailConfirmationToken(plaintext), nil
}

// hashEmailConfirmationToken produces the canonical SHA-256 hex digest used
// for DB persistence and lookup.
func hashEmailConfirmationToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// HashEmailConfirmationToken is the public lookup-side hashing helper. Use
// this when a token arrives from the client and we want to find the matching
// application row.
func HashEmailConfirmationToken(plaintext string) string {
	return hashEmailConfirmationToken(plaintext)
}

// Note: the matching path queries Postgres by hash directly (`WHERE
// email_confirmation_token_hash = $1`) — a timing oracle there would
// require probing 256-bit hash prefixes, which is computationally
// infeasible. So we deliberately don't use `subtle.ConstantTimeCompare`
// at the application layer.

// BuildEmailConfirmationURL composes the link that goes into the outgoing
// e-mail. The token is placed in the URL **fragment** (`#…`) rather than
// the path so it never reaches the server's access logs, reverse-proxy
// access logs, or CDN logs. The client-side page reads
// `window.location.hash` and POSTs the token to the backend.
func BuildEmailConfirmationURL(baseURL, plaintext string) string {
	base := strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/confirm-email#%s", base, plaintext)
}
