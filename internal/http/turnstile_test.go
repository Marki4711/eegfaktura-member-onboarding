package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyTurnstileToken_SkipsWhenNoSecretKey(t *testing.T) {
	code, err := verifyTurnstileToken("", "any-token")
	if err != nil || code != "" {
		t.Errorf("expected no error when secret key is empty, got code=%q err=%v", code, err)
	}
}

func TestVerifyTurnstileToken_MissingTokenWithSecretKey(t *testing.T) {
	code, err := verifyTurnstileToken("secret", "")
	if err == nil {
		t.Fatal("expected error for missing token with secret key configured")
	}
	if code != "turnstile_missing" {
		t.Errorf("expected code=turnstile_missing, got %q", code)
	}
}

func TestVerifyTurnstileToken_SuccessResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}))
	defer srv.Close()

	// Temporarily override the URL for testing
	orig := turnstileSiteverifyURL
	turnstileSiteverifyURL = srv.URL
	defer func() { turnstileSiteverifyURL = orig }()

	code, err := verifyTurnstileToken("test-secret", "valid-token")
	if err != nil || code != "" {
		t.Errorf("expected success, got code=%q err=%v", code, err)
	}
}

func TestVerifyTurnstileToken_FailureResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     false,
			"error-codes": []string{"invalid-input-response"},
		})
	}))
	defer srv.Close()

	orig := turnstileSiteverifyURL
	turnstileSiteverifyURL = srv.URL
	defer func() { turnstileSiteverifyURL = orig }()

	code, err := verifyTurnstileToken("test-secret", "invalid-token")
	if err == nil {
		t.Fatal("expected error for failed verification")
	}
	if code != "turnstile_failed" {
		t.Errorf("expected code=turnstile_failed, got %q", code)
	}
}

func TestVerifyTurnstileToken_NetworkError(t *testing.T) {
	orig := turnstileSiteverifyURL
	turnstileSiteverifyURL = "http://127.0.0.1:0" // unreachable
	defer func() { turnstileSiteverifyURL = orig }()

	code, err := verifyTurnstileToken("test-secret", "any-token")
	if err == nil {
		t.Fatal("expected error on network failure")
	}
	if code != "turnstile_failed" {
		t.Errorf("expected code=turnstile_failed, got %q", code)
	}
}
