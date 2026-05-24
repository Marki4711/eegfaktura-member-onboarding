package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSinkHandler(captured **KeycloakClaims) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

func TestTestHeaderAuthMiddleware_RejectsWithoutHeaders(t *testing.T) {
	var captured *KeycloakClaims
	h := TestHeaderAuthMiddleware()(newSinkHandler(&captured))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if captured != nil {
		t.Fatalf("handler should not be reached when headers absent")
	}
}

func TestTestHeaderAuthMiddleware_AcceptsTenantHeader(t *testing.T) {
	var captured *KeycloakClaims
	h := TestHeaderAuthMiddleware()(newSinkHandler(&captured))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	req.Header.Set("X-Test-Tenant", "RC1, RC2")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if captured == nil {
		t.Fatal("claims should be injected")
	}
	if len(captured.Tenant) != 2 || captured.Tenant[0] != "RC1" || captured.Tenant[1] != "RC2" {
		t.Fatalf("unexpected tenant: %v", captured.Tenant)
	}
	if captured.IsSuperuser() {
		t.Fatal("non-superuser header should not yield superuser")
	}
}

func TestTestHeaderAuthMiddleware_AcceptsSuperuserHeader(t *testing.T) {
	var captured *KeycloakClaims
	h := TestHeaderAuthMiddleware()(newSinkHandler(&captured))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	req.Header.Set("X-Test-Superuser", "true")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if captured == nil || !captured.IsSuperuser() {
		t.Fatalf("expected superuser claims, got %+v", captured)
	}
}

func TestTestHeaderAuthMiddleware_CustomSubject(t *testing.T) {
	var captured *KeycloakClaims
	h := TestHeaderAuthMiddleware()(newSinkHandler(&captured))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	req.Header.Set("X-Test-Superuser", "true")
	req.Header.Set("X-Test-Subject", "alice@example.com")
	h.ServeHTTP(rr, req)
	if captured == nil || captured.Subject != "alice@example.com" {
		t.Fatalf("expected subject alice@example.com, got %q", captured.Subject)
	}
}
