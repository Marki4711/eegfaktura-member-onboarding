// Package coreclient wraps every HTTP call from the onboarding to the
// eegFaktura core service: the participant import (PROJ-4), tariff
// lookup (PROJ-27), and the EEG master-data sync (PROJ-32).
//
// # Auth model — user-context bearer forwarding
//
// All methods take a `bearerToken` parameter and forward it verbatim as
// `Authorization: Bearer <jwt>` to the core, together with a `tenant`
// header set to the EEG's RC number. The token is the **logged-in
// admin's Keycloak JWT**, threaded through from the HTTP handler. There
// is NO service account, NO client_credentials grant, NO cached token.
//
// This is a deliberate decision (locked 2026-05-14): the admin is
// already authenticated for the Onboarding UI, their `Tenants` JWT
// claim already enumerates the RCs they may operate on, and the core
// records the actual human as the actor in its audit trail. Reasons to
// reconsider would be background jobs that need to call the core
// without an admin click — none exist today.
//
// The core API contract (endpoints, headers, payload mapping, gotchas)
// is captured in features/PROJ-4-core-import.md and the
// "eegFaktura Core API contract" memory.
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
//
// Status (PROJ-46 Stage D) is the participant's lifecycle state in the core:
// `NEW` (just created), `PENDING` (waiting for confirmation), or `ACTIVE`
// (live in the EEG). Only `ACTIVE` triggers an auto-transition to
// `activated` in the onboarding's activation-check, mode `participant_active`.
//
// Meters (PROJ-53) carries the per-meter EDA process state. Empty if the
// core deserialization didn't include the array (older core versions) or
// the participant has no meters yet. Used by activation-check mode
// `any_meter_registration_started`.
type CoreParticipantSummary struct {
	ID                string               `json:"id"`
	ParticipantNumber *string              `json:"participantNumber,omitempty"`
	Status            string               `json:"status,omitempty"`
	Meters            []CoreMeterSummary   `json:"meters,omitempty"`
}

// CoreMeterSummary holds the per-meter fields PROJ-53's activation-check
// needs. processState is the EDA state at the network operator:
//   - INVALID  — not yet registered, or rejected
//   - PENDING  — Netzbetreiber bestätigt Empfang (ANTWORT_ECON, Code 99)
//   - APPROVED — Netzbetreiber stimmt zu (ZUSTIMMUNG_ECON)
//   - ACTIVE   — Online-Registrierung abgeschlossen (ABSCHLUSS_ECON)
//   - INACTIVE — deaktiviert
//
// "Anmeldung gestartet" = processState in {PENDING, APPROVED, ACTIVE}.
type CoreMeterSummary struct {
	MeteringPoint string `json:"meteringPoint,omitempty"`
	Status        string `json:"status,omitempty"`
	ProcessState  string `json:"processState,omitempty"`
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
//
// baseURL — hostname of the eegFaktura core (e.g. `https://eegfaktura.at`).
// Per-call path prefixes are hardcoded in each method, because the deployed
// reverse-proxy multiplexes several services behind one hostname under
// different prefixes:
//   - `/api/...`      — REST (PROJ-4 participant import, tariffs) + GraphQL (PROJ-32 master-data sync)
//   - `/cash/api/...` — eegfaktura-billing (PROJ-32 Phase 2 logo embed)
//
// Set CORE_BASE_URL to the hostname only — do not append `/api`.
type HTTPCoreClient struct {
	baseURL string
	http    *http.Client
}

// NewHTTPCoreClient builds a client. Empty baseURL disables every method
// (each returns ErrCoreNotConfigured), which is useful for local dev where
// the core service isn't reachable.
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/participant", bytes.NewReader(body))
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/eeg/tariff", nil)
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

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}
	if isHTMLResponse(resp.Header.Get("Content-Type"), respBody) {
		return nil, &CoreParseError{Detail: "core returned HTML instead of JSON for /eeg/tariff"}
	}
	// Diagnose-Hilfe analog ListParticipants — siehe Kommentar dort.
	if len(respBody) == 0 || readErr != nil {
		detail := fmt.Sprintf(
			"decode /eeg/tariff: empty body — status=%d proto=%q content-type=%q content-length=%q read-bytes=%d read-err=%v server=%q www-authenticate=%q location=%q response-headers=[%s]",
			resp.StatusCode,
			resp.Proto,
			resp.Header.Get("Content-Type"),
			resp.Header.Get("Content-Length"),
			len(respBody),
			readErr,
			resp.Header.Get("Server"),
			resp.Header.Get("Www-Authenticate"),
			resp.Header.Get("Location"),
			summariseHeaders(resp.Header),
		)
		return nil, &CoreParseError{Detail: detail}
	}

	var tariffs []CoreTariff
	if err := json.Unmarshal(respBody, &tariffs); err != nil {
		return nil, &CoreParseError{Detail: fmt.Sprintf(
			"decode /eeg/tariff: %v — status=%d content-type=%q body-prefix=%q",
			err,
			resp.StatusCode,
			resp.Header.Get("Content-Type"),
			truncate(string(respBody), 200),
		)}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/participant/v2/"+participantID, bytes.NewReader(body))
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/participant", nil)
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

	// 4 MiB cap. The full EegParticipant payload is ~2 KB per row, so this
	// holds ~2000 participants safely. The cap exists to bound memory, not
	// as a per-tenant limit — silent truncation at 1 MiB would break the
	// JSON decode for any larger EEG. If we ever push past 2000/tenant, the
	// right fix is a thinner core endpoint (id + participantNumber only).
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}
	if isHTMLResponse(resp.Header.Get("Content-Type"), respBody) {
		return nil, &CoreParseError{Detail: "core returned HTML instead of JSON for /participant"}
	}
	// Diagnose-Hilfe: bei leerem oder verkürztem Body Kontext mitliefern,
	// damit wir zwischen "Core liefert 200 + leer", "Read-Fehler" und
	// "JSON-Schema-Drift" unterscheiden können (siehe auch ListTariffs).
	if len(respBody) == 0 || readErr != nil {
		detail := fmt.Sprintf(
			"decode /participant: empty body — status=%d proto=%q content-type=%q content-length=%q read-bytes=%d read-err=%v server=%q www-authenticate=%q location=%q response-headers=[%s]",
			resp.StatusCode,
			resp.Proto,
			resp.Header.Get("Content-Type"),
			resp.Header.Get("Content-Length"),
			len(respBody),
			readErr,
			resp.Header.Get("Server"),
			resp.Header.Get("Www-Authenticate"),
			resp.Header.Get("Location"),
			summariseHeaders(resp.Header),
		)
		return nil, &CoreParseError{Detail: detail}
	}

	var participants []CoreParticipantSummary
	if err := json.Unmarshal(respBody, &participants); err != nil {
		return nil, &CoreParseError{Detail: fmt.Sprintf(
			"decode /participant: %v — status=%d content-type=%q body-prefix=%q",
			err,
			resp.StatusCode,
			resp.Header.Get("Content-Type"),
			truncate(string(respBody), 200),
		)}
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

// summariseHeaders renders a compact one-line representation of a response
// header set for diagnostic logging. Used when the core returns an unexpected
// response (empty body, wrong content-type) so we can tell whether the
// response actually came from the core or from a proxy/auth-gate in front of
// it. Sensitive headers (Set-Cookie, Authorization) are masked.
func summariseHeaders(h http.Header) string {
	if len(h) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(h))
	for k, vs := range h {
		val := strings.Join(vs, ",")
		lower := strings.ToLower(k)
		if lower == "set-cookie" || lower == "authorization" {
			val = "<redacted>"
		}
		if len(val) > 200 {
			val = val[:200] + "…"
		}
		parts = append(parts, k+"="+val)
	}
	return strings.Join(parts, "; ")
}
