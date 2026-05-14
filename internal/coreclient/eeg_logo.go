package coreclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// EEGLogoMaxBytes caps how many bytes we are willing to pull from the
// eegfaktura-billing service in one logo fetch. PNG logos at 600×600 px
// rarely exceed 150 KB; 256 KB gives headroom for moderately oversized
// exports without letting a misconfigured upload blow up the DB column
// and slow every PDF render.
const EEGLogoMaxBytes = 256 * 1024

// allowedLogoMIMEs is what gofpdf.RegisterImageReader can ingest. Anything
// else is rejected at fetch time so we never write a logo into the DB that
// would later fail a PDF render.
var allowedLogoMIMEs = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
}

// Sentinel errors specific to the logo fetch path. CoreHTTPError / CoreParseError
// / ErrCoreTimeout from core_client.go are also returned where appropriate;
// the three below are returned only by FetchEEGLogo.
var (
	// ErrLogoNotFound is returned when the eegfaktura-billing service reports
	// no logo for the tenant: either no billing config, or the billing config
	// has headerImageFileDataId == null, or the bytes endpoint replies 404.
	// Callers should treat this as a soft signal (logo simply isn't there)
	// rather than a failure.
	ErrLogoNotFound = errors.New("eeg logo not found")

	// ErrLogoTooLarge is returned when the logo bytes exceed EEGLogoMaxBytes.
	// The DB column is NOT written in that case; the caller surfaces the
	// message to the admin so they shrink the upload in eegFaktura.
	ErrLogoTooLarge = errors.New("eeg logo exceeds size limit")

	// ErrLogoUnsupportedMIME is returned when the billing service responds
	// with a Content-Type that gofpdf cannot embed (anything outside PNG /
	// JPEG / GIF). Suggests SVG or HEIC — neither is supported by fpdf.
	ErrLogoUnsupportedMIME = errors.New("eeg logo MIME type not supported")
)

// billingConfigResponse is the subset of GET /cash/api/billingConfigs/tenant/{rc}
// that the onboarding cares about. The full response carries ~20 fields
// (text blocks for invoices/credit notes, numbering prefixes, …) which are
// internal to the billing service.
type billingConfigResponse struct {
	ID                      string  `json:"id"`
	HeaderImageFileDataID   *string `json:"headerImageFileDataId"`
}

// FetchEEGLogo retrieves the EEG logo from the eegfaktura-billing service in
// two HTTP calls:
//
//  1. GET /cash/api/billingConfigs/tenant/{rcNumber} → billing config JSON
//  2. If headerImageFileDataId is non-null:
//     GET /cash/api/billingConfigs/{billingConfigId}/logoImage → image bytes
//
// On success returns the bytes + the MIME type taken from the response
// Content-Type header (always one of the entries in allowedLogoMIMEs).
//
// Errors:
//   - ErrCoreNotConfigured    – CORE_BASE_URL is empty.
//   - ErrBearerTokenRequired  – caller passed an empty bearer.
//   - ErrTenantRequired       – caller passed an empty tenant.
//   - ErrCoreTimeout          – HTTP client timed out on either call.
//   - ErrLogoNotFound         – no logo configured in the core (soft signal).
//   - ErrLogoTooLarge         – logo bytes exceed EEGLogoMaxBytes.
//   - ErrLogoUnsupportedMIME  – Content-Type outside PNG/JPEG/GIF.
//   - CoreHTTPError           – any other non-2xx response.
//   - CoreParseError          – malformed billing-config JSON.
func (c *HTTPCoreClient) FetchEEGLogo(ctx context.Context, bearerToken, tenant string) (logoBytes []byte, mime string, err error) {
	if c.baseURL == "" {
		return nil, "", ErrCoreNotConfigured
	}
	if bearerToken == "" {
		return nil, "", ErrBearerTokenRequired
	}
	if tenant == "" {
		return nil, "", ErrTenantRequired
	}

	cfg, err := c.fetchBillingConfig(ctx, bearerToken, tenant)
	if err != nil {
		return nil, "", err
	}
	if cfg.HeaderImageFileDataID == nil || *cfg.HeaderImageFileDataID == "" {
		return nil, "", ErrLogoNotFound
	}
	if cfg.ID == "" {
		return nil, "", &CoreParseError{Detail: "billing config response missing id"}
	}

	return c.fetchLogoBytes(ctx, bearerToken, tenant, cfg.ID)
}

func (c *HTTPCoreClient) fetchBillingConfig(ctx context.Context, bearerToken, tenant string) (*billingConfigResponse, error) {
	url := c.baseURL + "/cash/api/billingConfigs/tenant/" + tenant
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build billingConfig request: %w", err)
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
		return nil, fmt.Errorf("billingConfig request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No billing config means no logo. Soft-signal.
		return nil, ErrLogoNotFound
	}

	// 64 KiB is plenty — the billing-config JSON is dominated by text
	// blocks (invoice templates), not arbitrary user input.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}
	if isHTMLResponse(resp.Header.Get("Content-Type"), respBody) {
		return nil, &CoreParseError{Detail: "billing service returned HTML instead of JSON for billingConfigs/tenant"}
	}

	var cfg billingConfigResponse
	if err := json.Unmarshal(respBody, &cfg); err != nil {
		return nil, &CoreParseError{Detail: fmt.Sprintf("unmarshal billingConfig: %s", err.Error())}
	}
	return &cfg, nil
}

func (c *HTTPCoreClient) fetchLogoBytes(ctx context.Context, bearerToken, tenant, billingConfigID string) ([]byte, string, error) {
	url := c.baseURL + "/cash/api/billingConfigs/" + billingConfigID + "/logoImage"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build logoImage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("tenant", tenant)

	resp, err := c.http.Do(req)
	if err != nil {
		if isTimeoutErr(err) {
			return nil, "", ErrCoreTimeout
		}
		if errors.Is(err, context.Canceled) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("logoImage request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", ErrLogoNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read a small body slice for diagnostics — the response is supposed
		// to be image bytes, so anything else is an error envelope.
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return nil, "", &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(errBody), 1000)}
	}

	// MIME whitelist before reading bytes — saves work + clearer error.
	mime := normalizeMIME(resp.Header.Get("Content-Type"))
	if !allowedLogoMIMEs[mime] {
		return nil, "", ErrLogoUnsupportedMIME
	}

	// Read EEGLogoMaxBytes + 1 so we can distinguish "exactly at cap" (which
	// we still accept) from "would have been larger" (which we reject).
	body, err := io.ReadAll(io.LimitReader(resp.Body, EEGLogoMaxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read logo bytes: %w", err)
	}
	if len(body) > EEGLogoMaxBytes {
		return nil, "", ErrLogoTooLarge
	}
	if len(body) == 0 {
		return nil, "", ErrLogoNotFound
	}
	return body, mime, nil
}

// normalizeMIME strips parameters ("image/png; charset=binary" → "image/png")
// and lower-cases the type. RFC 6838 says MIME types are case-insensitive in
// the type/subtype part.
func normalizeMIME(contentType string) string {
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}
