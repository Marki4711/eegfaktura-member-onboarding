package coreclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// EEGMasterData is the subset of fields PROJ-32 syncs from the eegFaktura
// core's GraphQL `query { eeg { … } }` response. The core's full Eeg type
// exposes many more fields (description, businessNr, area, legal, …) which
// the onboarding does not need; we deliberately keep this DTO small so
// schema drift in the core only impacts code that actually consumes a
// field.
//
// All fields use pointers so the DTO can express the difference between
// "core sent null" and "field not present in response". The resolver then
// decides whether a null is acceptable (e.g. CreditorID often is) or
// should leave the local DB value alone.
type EEGMasterData struct {
	ID          string                  `json:"id"`
	Name        *string                 `json:"name"`
	RCNumber    *string                 `json:"rcNumber"`
	Address     *EEGMasterDataAddress   `json:"address"`
	Contact     *EEGMasterDataContact   `json:"contact"`
	AccountInfo *EEGMasterDataAccount   `json:"accountInfo"`
}

type EEGMasterDataAddress struct {
	Street       *string `json:"street"`
	StreetNumber *string `json:"streetNumber"`
	Zip          *string `json:"zip"`
	City         *string `json:"city"`
}

type EEGMasterDataContact struct {
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type EEGMasterDataAccount struct {
	IBAN       *string `json:"iban"`
	Owner      *string `json:"owner"`
	BankName   *string `json:"bankName"`
	CreditorID *string `json:"creditorId"`
	BIC        *string `json:"bic"`
	SEPA       *bool   `json:"sepa"`
}

// graphqlRequest is the JSON envelope all GraphQL servers accept on POST.
type graphqlRequest struct {
	Query string `json:"query"`
}

// graphqlEEGResponse is what we expect back from `query { eeg { … } }`.
// GraphQL responses always carry a `data` object on success; failures
// carry an `errors` array with a message and (optionally) a path.
type graphqlEEGResponse struct {
	Data *struct {
		EEG *EEGMasterData `json:"eeg"`
	} `json:"data,omitempty"`
	Errors []struct {
		Message string   `json:"message"`
		Path    []string `json:"path,omitempty"`
	} `json:"errors,omitempty"`
}

// eegMasterDataQuery is the exact GraphQL document we POST. Kept as a
// constant so that schema changes in the core surface as a single
// localised edit here, not scattered across the codebase.
const eegMasterDataQuery = `query EEGMasterData {
  eeg {
    id
    name
    rcNumber
    address { street streetNumber zip city }
    contact { email phone }
    accountInfo { iban owner bankName creditorId bic sepa }
  }
}`

// FetchEEGMasterData fires the GraphQL query against `<baseURL>/api/query` and
// returns the parsed EEG record. The bearer token must belong to a Keycloak
// identity whose tenant claim includes `tenant` (typically the admin's own
// JWT, forwarded from the Settings UI).
//
// The GraphQL endpoint lives on the same hostname as the existing REST
// endpoints — both are reached under the `/api/...` prefix served by the
// eegFaktura core reverse-proxy.
//
// Errors:
//   - ErrCoreNotConfigured   – CORE_BASE_URL is empty.
//   - ErrBearerTokenRequired – caller passed an empty bearer.
//   - ErrTenantRequired      – caller passed an empty tenant.
//   - ErrCoreTimeout         – HTTP client timed out.
//   - CoreHTTPError          – non-2xx response (4xx auth / 5xx core down).
//   - CoreParseError         – non-JSON or GraphQL `errors` array set.
func (c *HTTPCoreClient) FetchEEGMasterData(ctx context.Context, bearerToken, tenant string) (*EEGMasterData, error) {
	if c.baseURL == "" {
		return nil, ErrCoreNotConfigured
	}
	if bearerToken == "" {
		return nil, ErrBearerTokenRequired
	}
	if tenant == "" {
		return nil, ErrTenantRequired
	}

	body, err := json.Marshal(graphqlRequest{Query: eegMasterDataQuery})
	if err != nil {
		return nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/query", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
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
		return nil, fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &CoreHTTPError{StatusCode: resp.StatusCode, Body: truncate(string(respBody), 1000)}
	}
	if isHTMLResponse(resp.Header.Get("Content-Type"), respBody) {
		return nil, &CoreParseError{Detail: "core graphql returned HTML — CORE_BASE_URL likely points to a frontend or auth proxy, not the core API"}
	}

	var parsed graphqlEEGResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, &CoreParseError{Detail: fmt.Sprintf("unmarshal graphql response: %s", err.Error())}
	}
	if len(parsed.Errors) > 0 {
		// GraphQL semantics: business-level errors come back HTTP 200 but
		// with a populated `errors` array. We surface the first message
		// (commonly enough; multi-error responses are rare).
		return nil, &CoreParseError{Detail: fmt.Sprintf("graphql error: %s", parsed.Errors[0].Message)}
	}
	if parsed.Data == nil || parsed.Data.EEG == nil {
		return nil, &CoreParseError{Detail: "graphql response missing data.eeg"}
	}
	return parsed.Data.EEG, nil
}
