package coreclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// twoCallServer routes /cash/api/billingConfigs/tenant/{rc} and
// /cash/api/billingConfigs/{id}/logoImage to two handler hooks so each
// test can shape both responses independently.
type twoCallServer struct {
	billingConfig http.HandlerFunc
	logoImage     http.HandlerFunc
}

func (s *twoCallServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/cash/api/billingConfigs/tenant/"):
		s.billingConfig(w, r)
	case strings.HasSuffix(r.URL.Path, "/logoImage"):
		s.logoImage(w, r)
	default:
		http.NotFound(w, r)
	}
}

func TestFetchEEGLogo_Success(t *testing.T) {
	logoPNG := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 1, 2, 3, 4}
	srv := httptest.NewServer(&twoCallServer{
		billingConfig: func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/TE100200") {
				t.Errorf("billingConfig path = %q, want suffix /TE100200", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer tok" {
				t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
			}
			if r.Header.Get("tenant") != "TE100200" {
				t.Errorf("tenant header = %q", r.Header.Get("tenant"))
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bc-123","headerImageFileDataId":"fd-xyz"}`)
		},
		logoImage: func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/bc-123/") {
				t.Errorf("logoImage path = %q, want to contain /bc-123/", r.URL.Path)
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(logoPNG)
		},
	})
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	bytes, mime, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("mime = %q, want image/png", mime)
	}
	if len(bytes) != len(logoPNG) {
		t.Errorf("got %d bytes, want %d", len(bytes), len(logoPNG))
	}
}

func TestFetchEEGLogo_NoHeaderImage(t *testing.T) {
	srv := httptest.NewServer(&twoCallServer{
		billingConfig: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bc-123","headerImageFileDataId":null}`)
		},
		logoImage: func(w http.ResponseWriter, r *http.Request) {
			t.Error("logoImage should not be called when headerImageFileDataId is null")
		},
	})
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	_, _, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
	if !errors.Is(err, ErrLogoNotFound) {
		t.Errorf("got %v, want ErrLogoNotFound", err)
	}
}

func TestFetchEEGLogo_BillingConfigNotFound(t *testing.T) {
	srv := httptest.NewServer(&twoCallServer{
		billingConfig: func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		},
	})
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	_, _, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
	if !errors.Is(err, ErrLogoNotFound) {
		t.Errorf("got %v, want ErrLogoNotFound", err)
	}
}

func TestFetchEEGLogo_LogoNotFound(t *testing.T) {
	srv := httptest.NewServer(&twoCallServer{
		billingConfig: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bc-123","headerImageFileDataId":"fd-xyz"}`)
		},
		logoImage: func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		},
	})
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	_, _, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
	if !errors.Is(err, ErrLogoNotFound) {
		t.Errorf("got %v, want ErrLogoNotFound", err)
	}
}

func TestFetchEEGLogo_TooLarge(t *testing.T) {
	oversized := make([]byte, EEGLogoMaxBytes+10)
	srv := httptest.NewServer(&twoCallServer{
		billingConfig: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bc-123","headerImageFileDataId":"fd-xyz"}`)
		},
		logoImage: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Write(oversized)
		},
	})
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	_, _, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
	if !errors.Is(err, ErrLogoTooLarge) {
		t.Errorf("got %v, want ErrLogoTooLarge", err)
	}
}

func TestFetchEEGLogo_ExactlyAtCap(t *testing.T) {
	atCap := make([]byte, EEGLogoMaxBytes)
	for i := range atCap {
		atCap[i] = 0xff
	}
	srv := httptest.NewServer(&twoCallServer{
		billingConfig: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"bc-123","headerImageFileDataId":"fd-xyz"}`)
		},
		logoImage: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.Write(atCap)
		},
	})
	defer srv.Close()

	c := NewHTTPCoreClient(srv.URL, 5*time.Second)
	bytes, _, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
	if err != nil {
		t.Fatalf("at-cap should succeed, got %v", err)
	}
	if len(bytes) != EEGLogoMaxBytes {
		t.Errorf("got %d bytes, want exactly %d", len(bytes), EEGLogoMaxBytes)
	}
}

func TestFetchEEGLogo_UnsupportedMIME(t *testing.T) {
	cases := []string{"image/svg+xml", "image/webp", "application/octet-stream", ""}
	for _, ct := range cases {
		t.Run(ct, func(t *testing.T) {
			srv := httptest.NewServer(&twoCallServer{
				billingConfig: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"id":"bc-123","headerImageFileDataId":"fd-xyz"}`)
				},
				logoImage: func(w http.ResponseWriter, r *http.Request) {
					if ct != "" {
						w.Header().Set("Content-Type", ct)
					}
					w.Write([]byte("payload"))
				},
			})
			defer srv.Close()

			c := NewHTTPCoreClient(srv.URL, 5*time.Second)
			_, _, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
			if !errors.Is(err, ErrLogoUnsupportedMIME) {
				t.Errorf("ct=%q: got %v, want ErrLogoUnsupportedMIME", ct, err)
			}
		})
	}
}

func TestFetchEEGLogo_AcceptsJPEGAndGIF(t *testing.T) {
	for _, ct := range []string{"image/jpeg", "image/gif", "IMAGE/PNG", "image/jpeg; charset=binary"} {
		t.Run(ct, func(t *testing.T) {
			srv := httptest.NewServer(&twoCallServer{
				billingConfig: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"id":"bc-123","headerImageFileDataId":"fd-xyz"}`)
				},
				logoImage: func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", ct)
					w.Write([]byte{1, 2, 3})
				},
			})
			defer srv.Close()

			c := NewHTTPCoreClient(srv.URL, 5*time.Second)
			_, mime, err := c.FetchEEGLogo(context.Background(), "tok", "TE100200")
			if err != nil {
				t.Fatalf("got %v", err)
			}
			if !allowedLogoMIMEs[mime] {
				t.Errorf("normalised mime %q not in whitelist", mime)
			}
		})
	}
}

func TestFetchEEGLogo_NotConfigured(t *testing.T) {
	c := NewHTTPCoreClient("", 5*time.Second)
	_, _, err := c.FetchEEGLogo(context.Background(), "tok", "RC")
	if !errors.Is(err, ErrCoreNotConfigured) {
		t.Errorf("got %v, want ErrCoreNotConfigured", err)
	}
}

func TestFetchEEGLogo_RequiresTokenAndTenant(t *testing.T) {
	c := NewHTTPCoreClient("http://example.invalid", 5*time.Second)
	if _, _, err := c.FetchEEGLogo(context.Background(), "", "RC"); !errors.Is(err, ErrBearerTokenRequired) {
		t.Errorf("missing token: got %v", err)
	}
	if _, _, err := c.FetchEEGLogo(context.Background(), "tok", ""); !errors.Is(err, ErrTenantRequired) {
		t.Errorf("missing tenant: got %v", err)
	}
}
