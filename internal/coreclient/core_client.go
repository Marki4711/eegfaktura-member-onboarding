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
	"net"
	"net/http"
	"strings"
	"time"
)

// CoreClient creates a productive participant in the eegFaktura core.
// Implementations must be safe for concurrent use.
type CoreClient interface {
	CreateParticipant(ctx context.Context, payload any, bearerToken, tenant string) (participantID string, err error)
	// ListTariffs fetches the active tariff catalogue for the given tenant.
	// Used by PROJ-27 to populate the tariff selection dropdowns at import time.
	ListTariffs(ctx context.Context, bearerToken, tenant string) ([]CoreTariff, error)
	// UpdateParticipantField applies a partial update on a participant
	// (PUT /participant/v2/{id} with {"path": path, "value": value}). Used by
	// PROJ-27 to set the member-level tariffId after participant creation,
	// because the core's POST /participant ignores the tariffId field on
	// insert (goqu:"skipinsert" on EegParticipantBase.TariffId).
	UpdateParticipantField(ctx context.Context, bearerToken, tenant, participantID, path string, value any) error
	// ListParticipants returns the participantNumber field of every existing
	// participant for the tenant — used to compute "next free member number"
	// at import time and to detect duplicates when the admin overrides the
	// suggested number in the import dialog.
	ListParticipants(ctx context.Context, bearerToken, tenant string) ([]CoreParticipantSummary, error)
}

// CoreParticipantSummary holds just the fields we need to derive / validate
// member numbers. participantNumber is VARCHAR in the core schema, so we
// keep it as a pointer-string and ignore non-numeric values when computing
// the next free number.
type CoreParticipantSummary struct {
	ID                string  `json:"id"`
	ParticipantNumber *string `json:"participantNumber,omitempty"`
}

// CoreTariff is the subset of fields PROJ-27 needs from the eegFaktura core's
// GET /eeg/tariff response. The full core model has additional pricing fields
// (participantFee, baseFee, freeKWh, ...) that we don't need for selection.
type CoreTariff struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"` // EEG | VZP | EZP | AKONTO
	Name          string  `json:"name"`
	CentPerKWh    float64 `json:"centPerKWh"`
	Discount      float64 `json:"discount"`
	UseVat        bool    `json:"useVat"`
	VatInPercent  float64 `json:"vatInPercent"`
	InactiveSince *string `json:"inactiveSince,omitempty"`
}

// Sentinel errors returned by the client. Callers can use errors.Is to detect
// each condition without depending on the message text.
var (
	// ErrCoreNotConfigured is returned when no CORE_BASE_URL has been configured.
	ErrCoreNotConfigured = errors.New("core service not configured")

	// ErrCoreTimeout is returned when the HTTP call to the core times out.
	ErrCoreTimeout = errors.New("core service timeout")

	// ErrBearerTokenRequired is returned when the caller did not pass a bearer
	// token to forward.
	ErrBearerTokenRequired = errors.New("bearer token required for core call")

	// ErrTenantRequired is returned when the caller did not pass a tenant.
	ErrTenantRequired = errors.New("tenant required for core call")
)

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
		return "", ErrBearerTokenRequired
	}
	if tenant == "" {
		return "", ErrTenantRequired
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
		if isTimeoutErr(err) {
			return "", ErrCoreTimeout
		}
		if errors.Is(err, context.Canceled) {
			// Caller cancelled the request (e.g., HTTP client disconnected).
			// Return as-is so the caller can recognise it via errors.Is.
			return "", err
		}
		return "", fmt.Errorf("core request failed: %w", err)
	}
	defer resp.Body.Close()

	// Cap the body read to bound memory usage. Cores normally return small
	// JSON payloads; an unbounded read would let a misbehaving server OOM us.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}

	// Detect non-JSON responses early. The most common misconfiguration is that
	// CORE_BASE_URL points at a frontend / SPA that returns index.html with a
	// 200, or at an auth proxy that returns an HTML login page.
	if isHTMLResponse(resp.Header.Get("Content-Type"), respBody) {
		return "", &CoreParseError{Detail: "core returned HTML instead of JSON — CORE_BASE_URL likely points to a frontend or auth proxy, not the API endpoint"}
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

// ListTariffs implements CoreClient. See interface doc for semantics.
func (c *HTTPCoreClient) ListTariffs(ctx context.Context, bearerToken, tenant string) ([]CoreTariff, error) {
	if c.baseURL == "" {
		return nil, ErrCoreNotConfigured
	}
	if bearerToken == "" {
		return nil, ErrBearerTokenRequired
	}
	if tenant == "" {
		return nil, ErrTenantRequired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/eeg/tariff", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("tenant", tenant)

	resp, err := c.http.Do(req)
	if err != nil {
		if isTimeoutErr(err) {
			return nil, ErrCoreTimeout
		}
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("core request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}
	if isHTMLResponse(resp.Header.Get("Content-Type"), respBody) {
		return nil, &CoreParseError{Detail: "core returned HTML instead of JSON for /eeg/tariff"}
	}

	var tariffs []CoreTariff
	if err := json.Unmarshal(respBody, &tariffs); err != nil {
		return nil, &CoreParseError{Detail: err.Error()}
	}
	return tariffs, nil
}

// UpdateParticipantField implements CoreClient. See interface doc for semantics.
func (c *HTTPCoreClient) UpdateParticipantField(ctx context.Context, bearerToken, tenant, participantID, path string, value any) error {
	if c.baseURL == "" {
		return ErrCoreNotConfigured
	}
	if bearerToken == "" {
		return ErrBearerTokenRequired
	}
	if tenant == "" {
		return ErrTenantRequired
	}
	if participantID == "" {
		return errors.New("participantID required")
	}

	body, err := json.Marshal(map[string]any{"path": path, "value": value})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/participant/v2/"+participantID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("tenant", tenant)

	resp, err := c.http.Do(req)
	if err != nil {
		if isTimeoutErr(err) {
			return ErrCoreTimeout
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
		return fmt.Errorf("core request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}
	return nil
}

// ListParticipants fetches the participant list for the given tenant, used
// by the import dialog to suggest the next free member number and to detect
// duplicates before sending POST /participant.
func (c *HTTPCoreClient) ListParticipants(ctx context.Context, bearerToken, tenant string) ([]CoreParticipantSummary, error) {
	if c.baseURL == "" {
		return nil, ErrCoreNotConfigured
	}
	if bearerToken == "" {
		return nil, ErrBearerTokenRequired
	}
	if tenant == "" {
		return nil, ErrTenantRequired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/participant", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("tenant", tenant)

	resp, err := c.http.Do(req)
	if err != nil {
		if isTimeoutErr(err) {
			return nil, ErrCoreTimeout
		}
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("core request failed: %w", err)
	}
	defer resp.Body.Close()

	// 1 MiB cap — a tenant with 10k participants is well below this.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}
	if isHTMLResponse(resp.Header.Get("Content-Type"), respBody) {
		return nil, &CoreParseError{Detail: "core returned HTML instead of JSON for /participant"}
	}

	var participants []CoreParticipantSummary
	if err := json.Unmarshal(respBody, &participants); err != nil {
		return nil, &CoreParseError{Detail: fmt.Sprintf("decode /participant: %v", err)}
	}
	return participants, nil
}

// isHTMLResponse returns true when the response body is HTML rather than JSON.
// We check both the Content-Type header (when present) and the first
// non-whitespace byte (since some misconfigured servers send no Content-Type).
func isHTMLResponse(contentType string, body []byte) bool {
	if strings.Contains(strings.ToLower(contentType), "text/html") {
		return true
	}
	for _, b := range body {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '<':
			return true
		default:
			return false
		}
	}
	return false
}

// isTimeoutErr returns true for the various ways the standard library
// surfaces a request-level timeout: a wrapped context.DeadlineExceeded
// (modern http.Client), or a net.Error whose Timeout() reports true
// (older runtimes and some transport-level cases).
func isTimeoutErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return false
}

// truncate shortens s to at most maxRunes runes, appending an ellipsis when
// truncated. Slicing by runes (not bytes) avoids cutting a multi-byte UTF-8
// sequence — which can happen when the core embeds Unicode in its error
// bodies.
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}
