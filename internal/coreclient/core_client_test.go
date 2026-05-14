package coreclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateParticipant_Success(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotTenant, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotTenant = r.Header.Get("tenant")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"abc-123","firstname":"Anna"}`))
	}))
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	id, err := c.CreateParticipant(context.Background(), map[string]string{"firstname": "Anna"}, "tok-xyz", "RC101665")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "abc-123" {
		t.Errorf("got id %q, want abc-123", id)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/participant" {
		t.Errorf("path = %q, want /api/participant", gotPath)
	}
	if gotAuth != "Bearer tok-xyz" {
		t.Errorf("auth header = %q, want Bearer tok-xyz", gotAuth)
	}
	if gotTenant != "RC101665" {
		t.Errorf("tenant header = %q, want RC101665", gotTenant)
	}
	if !strings.Contains(gotBody, `"firstname":"Anna"`) {
		t.Errorf("body did not contain payload, got %q", gotBody)
	}
}

func TestCreateParticipant_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("validation failed: missing field xyz"))
	}))
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	_, err := c.CreateParticipant(context.Background(), struct{}{}, "tok", "RC101665")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var httpErr *CoreHTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *CoreHTTPError, got %T (%v)", err, err)
	}
	if httpErr.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", httpErr.StatusCode)
	}
	if !strings.Contains(httpErr.Body, "missing field xyz") {
		t.Errorf("body not preserved: %q", httpErr.Body)
	}
}

func TestCreateParticipant_MissingID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"firstname":"Anna"}`))
	}))
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	_, err := c.CreateParticipant(context.Background(), struct{}{}, "tok", "RC101665")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	var parseErr *CoreParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected *CoreParseError, got %T", err)
	}
}

func TestCreateParticipant_HTMLResponse(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
	}{
		{"explicit html content-type", "text/html; charset=utf-8", `{"id":"x"}`},
		{"sniff html body without content-type", "", "<!DOCTYPE html><html><body>login</body></html>"},
		{"sniff html with leading whitespace", "", "  \n<html></html>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.contentType != "" {
					w.Header().Set("Content-Type", tc.contentType)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			c := NewHTTPCoreClient(srv.URL, 5*time.Second)
			_, err := c.CreateParticipant(context.Background(), struct{}{}, "tok", "RC")
			var parseErr *CoreParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("expected *CoreParseError, got %T (%v)", err, err)
			}
			if !strings.Contains(parseErr.Detail, "HTML instead of JSON") {
				t.Errorf("expected HTML hint in error, got %q", parseErr.Detail)
			}
		})
	}
}

func TestCreateParticipant_NotConfigured(t *testing.T) {
	c := NewHTTPCoreClient("", 5*time.Second)
	_, err := c.CreateParticipant(context.Background(), struct{}{}, "tok", "RC")
	if !errors.Is(err, ErrCoreNotConfigured) {
		t.Errorf("got %v, want ErrCoreNotConfigured", err)
	}
}

func TestCreateParticipant_RequiresTokenAndTenant(t *testing.T) {
	c := NewHTTPCoreClient("http://example.invalid", 5*time.Second)
	if _, err := c.CreateParticipant(context.Background(), struct{}{}, "", "RC"); err == nil {
		t.Error("expected error for missing token")
	}
	if _, err := c.CreateParticipant(context.Background(), struct{}{}, "tok", ""); err == nil {
		t.Error("expected error for missing tenant")
	}
}
