package http

import (
	"net/http/httptest"
	"testing"
)

// TestCoreBearerToken_DirectMode verifies that the default mode forwards the
// admin's session token from the Authorization header, ignoring any
// X-Core-Authorization header even if present.
func TestCoreBearerToken_DirectMode(t *testing.T) {
	h := &AdminHandler{coreAuthMode: "direct"}

	req := httptest.NewRequest("GET", "/whatever", nil)
	req.Header.Set("Authorization", "Bearer session-tok")
	req.Header.Set("X-Core-Authorization", "Bearer core-tok-should-be-ignored")

	got := h.coreBearerToken(req)
	if got != "session-tok" {
		t.Errorf("direct mode: want session-tok, got %q", got)
	}
}

// TestCoreBearerToken_ExchangeMode verifies that the exchange mode prefers
// the X-Core-Authorization header. Without it, it falls back to the session
// token so GraphQL-Calls (PROJ-32 sync) continue to work even when the
// frontend has not yet acquired a Faktura-side token.
func TestCoreBearerToken_ExchangeMode(t *testing.T) {
	h := &AdminHandler{coreAuthMode: "exchange"}

	t.Run("uses X-Core-Authorization when set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/whatever", nil)
		req.Header.Set("Authorization", "Bearer session-tok")
		req.Header.Set("X-Core-Authorization", "Bearer core-tok")

		got := h.coreBearerToken(req)
		if got != "core-tok" {
			t.Errorf("exchange mode with X-Core-Authorization: want core-tok, got %q", got)
		}
	})

	t.Run("falls back to Authorization when X-Core-Authorization absent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/whatever", nil)
		req.Header.Set("Authorization", "Bearer session-tok")

		got := h.coreBearerToken(req)
		if got != "session-tok" {
			t.Errorf("exchange mode without X-Core-Authorization: want session-tok fallback, got %q", got)
		}
	})

	t.Run("ignores malformed X-Core-Authorization", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/whatever", nil)
		req.Header.Set("Authorization", "Bearer session-tok")
		req.Header.Set("X-Core-Authorization", "not-bearer-prefixed")

		got := h.coreBearerToken(req)
		if got != "session-tok" {
			t.Errorf("exchange mode with malformed header: want session-tok fallback, got %q", got)
		}
	})
}

// TestSetCoreAuthMode verifies that the setter normalises unknown values to
// the safe default. Otherwise a typo in helm values would silently leave the
// backend in an undefined state.
func TestSetCoreAuthMode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"direct", "direct"},
		{"exchange", "exchange"},
		{"", "direct"},
		{"DIRECT", "direct"}, // case-sensitive — uppercase falls back
		{"foobar", "direct"},
	}
	for _, c := range cases {
		h := &AdminHandler{}
		h.SetCoreAuthMode(c.in)
		if h.coreAuthMode != c.want {
			t.Errorf("SetCoreAuthMode(%q): want %q, got %q", c.in, c.want, h.coreAuthMode)
		}
	}
}
