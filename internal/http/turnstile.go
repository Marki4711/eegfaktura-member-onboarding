package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var turnstileSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// verifyTurnstileToken validates a Cloudflare Turnstile token against the siteverify API.
// Returns ("", nil) when secretKey is empty (dev mode — verification skipped).
// Returns ("turnstile_missing", err) when the token is absent but a secret key is configured.
// Returns ("turnstile_failed", err) when Cloudflare rejects the token.
func verifyTurnstileToken(secretKey, token string) (errCode string, err error) {
	if secretKey == "" {
		return "", nil
	}
	if token == "" {
		return "turnstile_missing", fmt.Errorf("turnstile token missing")
	}

	resp, err := http.PostForm(turnstileSiteverifyURL, url.Values{
		"secret":   {secretKey},
		"response": {token},
	})
	if err != nil {
		return "turnstile_failed", fmt.Errorf("turnstile verification request failed: %w", err)
	}
	defer resp.Body.Close()

	var result turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "turnstile_failed", fmt.Errorf("turnstile response decode failed: %w", err)
	}

	if !result.Success {
		return "turnstile_failed", fmt.Errorf("turnstile rejected: %s", strings.Join(result.ErrorCodes, ", "))
	}
	return "", nil
}
