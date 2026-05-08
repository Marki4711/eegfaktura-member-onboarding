// Package coreclient wraps the HTTP call to the eegFaktura core service used
// by the onboarding import endpoint (PROJ-4). The core API contract is taken
// from github.com/eegfaktura/eegfaktura-backend (POST /participant, JWT auth,
// tenant HTTP header). See features/PROJ-4-core-import.md for details.
package coreclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CoreClient creates a productive participant in the eegFaktura core.
// Implementations must be safe for concurrent use.
type CoreClient interface {
	CreateParticipant(ctx context.Context, payload any, bearerToken, tenant string) (participantID string, err error)
}

// ErrCoreNotConfigured is returned when no CORE_BASE_URL has been configured.
var ErrCoreNotConfigured = errors.New("core service not configured")

// ErrCoreTimeout is returned when the HTTP call to the core times out.
var ErrCoreTimeout = errors.New("core service timeout")

// CoreHTTPError captures a non-2xx response from the core. The body is
// truncated to keep error messages bounded.
type CoreHTTPError struct {
	StatusCode int
	Body       string
}

func (e *CoreHTTPError) Error() string {
	body := e.Body
	if body == "" {
		body = "<empty>"
	}
	return fmt.Sprintf("core returned HTTP %d: %s", e.StatusCode, body)
}

// CoreParseError is returned when the core's response could not be decoded.
type CoreParseError struct {
	Detail string
}

func (e *CoreParseError) Error() string {
	return fmt.Sprintf("could not parse core response: %s", e.Detail)
}

// HTTPCoreClient is the production implementation that talks to the core
// service over HTTP/JSON.
type HTTPCoreClient struct {
	baseURL string
	http    *http.Client
}

// NewHTTPCoreClient builds a client targeting baseURL. When baseURL is empty,
// the returned client always returns ErrCoreNotConfigured — this lets the
// server boot in environments where the import feature is disabled.
func NewHTTPCoreClient(baseURL string, timeout time.Duration) *HTTPCoreClient {
	return &HTTPCoreClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// CreateParticipant POSTs the participant payload to {baseURL}/participant
// using the caller's Keycloak Bearer token and a tenant HTTP header set to
// the EEG's RC number. On success returns the participant ID extracted from
// the response body field "id".
func (c *HTTPCoreClient) CreateParticipant(ctx context.Context, payload any, bearerToken, tenant string) (string, error) {
	if c.baseURL == "" {
		return "", ErrCoreNotConfigured
	}
	if bearerToken == "" {
		return "", errors.New("bearer token required for core call")
	}
	if tenant == "" {
		return "", errors.New("tenant required for core call")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/participant", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("tenant", tenant)

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", ErrCoreTimeout
		}
		return "", fmt.Errorf("core request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", &CoreParseError{Detail: err.Error()}
	}
	if parsed.ID == "" {
		return "", &CoreParseError{Detail: "response missing id field"}
	}
	return parsed.ID, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
