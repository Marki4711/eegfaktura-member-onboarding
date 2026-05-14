package application

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
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

// CompareEmailConfirmationHash performs a constant-time comparison between a
// candidate hash and a stored hash. Use this when matching a freshly hashed
// request token against the value loaded from the DB.
func CompareEmailConfirmationHash(candidate, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(stored)) == 1
}

// BuildEmailConfirmationURL composes the link that goes into the outgoing
// e-mail. The token plaintext is appended as the last path segment.
func BuildEmailConfirmationURL(baseURL, plaintext string) string {
	base := strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/confirm-email/%s", base, plaintext)
}
