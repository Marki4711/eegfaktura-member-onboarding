package http

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/coreclient"
	"github.com/your-org/eegfaktura-member-onboarding/internal/importing"
	"github.com/your-org/eegfaktura-member-onboarding/internal/logfields"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// eegMasterDataCacheTTL bounds how long CompareEEGSettingsWithCore reuses a
// previously-fetched EEG master-data payload before hitting the core again.
// The settings page reloads the comparison on every open; 30 s collapses the
// burst of "open settings, glance at banner, close" page-views into a single
// core call without making the drift indicator meaningfully stale.
const eegMasterDataCacheTTL = 30 * time.Second

type eegMasterDataCacheEntry struct {
	data      *coreclient.EEGMasterData
	fetchedAt time.Time
}

// eegMasterDataCache memoises FetchEEGMasterData by RC number. It is owned by
// AdminHandler (single in-process map; the deployment is single-replica). On
// sync the entry is overwritten with the just-fetched payload so the next
// compare-call doesn't re-hit the core to confirm what we just wrote.
type eegMasterDataCache struct {
	mu      sync.Mutex
	entries map[string]eegMasterDataCacheEntry
}

func (c *eegMasterDataCache) get(rcNumber string) *coreclient.EEGMasterData {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[rcNumber]
	if !ok {
		return nil
	}
	if time.Since(e.fetchedAt) > eegMasterDataCacheTTL {
		delete(c.entries, rcNumber)
		return nil
	}
	return e.data
}

func (c *eegMasterDataCache) put(rcNumber string, data *coreclient.EEGMasterData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]eegMasterDataCacheEntry{}
	}
	c.entries[rcNumber] = eegMasterDataCacheEntry{data: data, fetchedAt: time.Now()}
}

// AdminHandler handles admin-facing HTTP requests.
type AdminHandler struct {
	adminService      *application.AdminApplicationService
	entrypointRepo    *application.RegistrationEntrypointRepository
	apiKeyRepo        *application.ExternalAPIKeyRepository
	legalDocumentRepo *application.LegalDocumentRepository
	importService     *importing.ImportService
	// coreClient is shared with importService — used by PROJ-32 for the
	// master-data sync GraphQL call. nil when CORE_BASE_URL is not set.
	coreClient *coreclient.HTTPCoreClient
	// eegCache memoises FetchEEGMasterData calls from CompareEEGSettingsWithCore
	// for eegMasterDataCacheTTL. Always non-nil.
	eegCache *eegMasterDataCache
	// coreAuthMode selects how outgoing REST calls to the eegFaktura core are
	// authenticated. "direct" forwards the admin's session token as-is (works
	// once the Faktura backend whitelists our azp). "exchange" uses a separate
	// Faktura-side token that the frontend obtains via silent SSO against the
	// Faktura-frontend Keycloak-client and forwards in the X-Core-Authorization
	// header. Set via SetCoreAuthMode; default "direct".
	coreAuthMode string
	validate     *validator.Validate
	sanitizer    *bluemonday.Policy
}

// NewAdminHandler creates a new AdminHandler. Both importService and coreClient
// may be nil when CORE_BASE_URL is not configured — import endpoint and
// master-data-sync endpoints then return 503.
func NewAdminHandler(
	adminService *application.AdminApplicationService,
	entrypointRepo *application.RegistrationEntrypointRepository,
	apiKeyRepo *application.ExternalAPIKeyRepository,
	legalDocumentRepo *application.LegalDocumentRepository,
	importService *importing.ImportService,
	coreClient *coreclient.HTTPCoreClient,
) *AdminHandler {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "ul", "ol", "li")
	p.AllowAttrs("href", "target", "rel").OnElements("a")
	p.AllowURLSchemes("http", "https", "mailto")
	return &AdminHandler{
		adminService:      adminService,
		entrypointRepo:    entrypointRepo,
		apiKeyRepo:        apiKeyRepo,
		legalDocumentRepo: legalDocumentRepo,
		importService:     importService,
		coreClient:        coreClient,
		eegCache:          &eegMasterDataCache{},
		coreAuthMode:      "direct",
		validate:          validator.New(),
		sanitizer:         p,
	}
}

// SetCoreAuthMode configures whether outgoing core calls reuse the admin's
// session token ("direct") or expect a separate Faktura-side token forwarded
// in the X-Core-Authorization header ("exchange"). Unknown values fall back
// to "direct". Called once from main during startup.
func (h *AdminHandler) SetCoreAuthMode(mode string) {
	if mode == "exchange" {
		h.coreAuthMode = "exchange"
		return
	}
	h.coreAuthMode = "direct"
}

// coreBearerToken returns the token that should be used for an outgoing call
// to the eegFaktura core, based on the configured coreAuthMode:
//   - "direct"   — the admin's Onboarding session token (Authorization header).
//   - "exchange" — the Faktura-side token forwarded in X-Core-Authorization;
//     falls back to the session token when missing so that GraphQL endpoints
//     (PROJ-32 EEG-Stammdaten, PROJ-33 Logo), which sit on a different code
//     path in the Faktura backend that still accepts the Onboarding client,
//     keep working even when the frontend has no Faktura-token yet.
func (h *AdminHandler) coreBearerToken(r *http.Request) string {
	if h.coreAuthMode == "exchange" {
		raw := r.Header.Get("X-Core-Authorization")
		if strings.HasPrefix(raw, "Bearer ") {
			return strings.TrimPrefix(raw, "Bearer ")
		}
	}
	return extractBearerToken(r)
}

// parseRCAndCheck reads the rc_number query parameter and verifies that the
// authenticated tenant-admin is allowed to access it. Superusers pass through.
// On failure writes the appropriate error response and returns ("", false).
func (h *AdminHandler) parseRCAndCheck(w http.ResponseWriter, r *http.Request) (string, bool) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rc_number": "rc_number query parameter is required",
		})))
		return "", false
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return "", false
	}
	return rcNumber, true
}

// checkTenantAccess verifies that the authenticated tenant-admin is allowed to
// operate on the given application. Superusers are always allowed. Returns true
// if access is granted; writes 403 and returns false otherwise.
func (h *AdminHandler) checkTenantAccess(w http.ResponseWriter, r *http.Request, id uuid.UUID) bool {
	claims := ClaimsFromContext(r.Context())
	if claims == nil || claims.IsSuperuser() {
		return true
	}
	// Slim lookup: previously this loaded the full application detail
	// (app + metering points + status log + consents) just to read rc_number.
	// Every admin click went through that — now it's a single column read.
	rcNumber, err := h.adminService.GetRCNumberByID(id)
	if err != nil {
		h.handleServiceError(w, err)
		return false
	}
	if !containsRC(claims.Tenant, rcNumber) {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"code":    "forbidden",
			"message": "Kein Zugriff auf diesen Antrag.",
		})
		return false
	}
	return true
}

// GetFieldConfig handles GET /api/admin/settings/fields?rc_number=...
func (h *AdminHandler) GetFieldConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}

	rawConfig, err := h.adminService.GetFieldConfig(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	type fieldEntry struct {
		State      string  `json:"state"`
		AdminValue *string `json:"adminValue,omitempty"`
	}
	config := make(map[string]fieldEntry, len(rawConfig))
	for name, entry := range rawConfig {
		config[name] = fieldEntry{State: entry.State, AdminValue: entry.AdminValue}
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"rcNumber": rcNumber, "fieldConfig": config})
}

// SaveFieldConfig handles PUT /api/admin/settings/fields?rc_number=...
func (h *AdminHandler) SaveFieldConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}

	var rawConfig map[string]struct {
		State      string  `json:"state"`
		AdminValue *string `json:"adminValue,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&rawConfig); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	config := make(map[string]application.FieldConfigEntry, len(rawConfig))
	for name, v := range rawConfig {
		config[name] = application.FieldConfigEntry{State: v.State, AdminValue: v.AdminValue}
	}

	if err := h.adminService.SaveFieldConfig(rcNumber, config); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SyncEntrypoints handles POST /api/admin/sync
// Called once per session after login to ensure registration_entrypoint rows exist
// for all RC numbers in the Tenant-Admin's token. No-op for superusers.
func (h *AdminHandler) SyncEntrypoints(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil || claims.IsSuperuser() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.entrypointRepo.UpsertForRCNumbers(claims.Tenant); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"code":    "internal_error",
			"message": "Sync fehlgeschlagen.",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListApplications handles GET /api/admin/applications
//
// @Summary      List applications
// @Description  Returns a paginated, filterable list of member applications. Tenant-admins only see their own EEG's applications; superusers see all.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        status          query  string  false  "Filter by status (draft|submitted|under_review|needs_info|approved|rejected|imported|import_failed)"
// @Param        reference_number query string false "Filter by reference number (partial match)"
// @Param        name            query  string  false  "Filter by member name — partial match across firstname, lastname, and company_name"
// @Param        email           query  string  false  "Filter by email (partial match)"
// @Param        metering_point  query  string  false  "Filter by metering point ID"
// @Param        rc_number       query  string  false  "Filter by RC number (superuser only)"
// @Param        submitted_from  query  string  false  "Filter by submission date from (RFC3339)"
// @Param        submitted_to    query  string  false  "Filter by submission date to (RFC3339)"
// @Param        page            query  int     false  "Page number (default 1)"
// @Param        page_size       query  int     false  "Page size (default 20)"
// @Success      200  {object}  shared.ApplicationListResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications [get]
func (h *AdminHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filters := application.ApplicationListFilters{}
	if v := q.Get("status"); v != "" {
		filters.Status = &v
	}
	if v := q.Get("reference_number"); v != "" {
		filters.ReferenceNumber = &v
	}
	// `name` is the canonical filter name; `lastname` is accepted as a
	// legacy synonym so bookmarked URLs from before the rename keep working.
	if v := q.Get("name"); v != "" {
		filters.Name = &v
	} else if v := q.Get("lastname"); v != "" {
		filters.Name = &v
	}
	if v := q.Get("email"); v != "" {
		filters.Email = &v
	}
	if v := q.Get("metering_point"); v != "" {
		filters.MeteringPoint = &v
	}
	if v := q.Get("rc_number"); v != "" {
		filters.RCNumberFilter = &v
	}
	if v := q.Get("submitted_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filters.SubmittedFrom = &t
		}
	}
	if v := q.Get("submitted_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			filters.SubmittedTo = &t
		}
	}

	// Apply tenant scope: superuser sees everything; tenant-admin only sees own RC numbers.
	if claims := ClaimsFromContext(r.Context()); claims != nil && !claims.IsSuperuser() {
		rcNumbers := []string(claims.Tenant)
		filters.RCNumbers = &rcNumbers
	}

	filters.Sort = q.Get("sort")
	filters.Order = q.Get("order")

	page := intQueryParam(q.Get("page"), 1)
	pageSize := intQueryParam(q.Get("page_size"), 20)
	// Hard cap on deep pagination so a buggy/abusive client can't make the
	// DB scan-and-sort millions of rows behind a giant OFFSET. 10_000 pages
	// at any pageSize is far beyond legitimate admin browsing.
	const maxPage = 10_000
	if page > maxPage {
		page = maxPage
	}

	resp, err := h.adminService.ListApplications(filters, page, pageSize)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// GetApplicationDetail handles GET /api/admin/applications/{id}
//
// @Summary      Get application detail
// @Description  Returns full application data including metering points, status log, and document consents.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Application UUID"
// @Success      200  {object}  shared.AdminApplicationDetailResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id} [get]
func (h *AdminHandler) GetApplicationDetail(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	resp, err := h.adminService.GetApplicationDetail(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Tenant-admins may only access applications within their allowed RC numbers.
	adminUserID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		adminUserID = claims.Subject
		if !claims.IsSuperuser() && !containsRC(claims.Tenant, resp.RCNumber) {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"code":    "forbidden",
				"message": "Kein Zugriff auf diesen Antrag.",
			})
			return
		}
	}

	// DSGVO audit-trail: an admin actively pulled the full PII record
	// (IBAN, birth_date, address, contact-person). Pendant zu pii-export
	// für Download-Pfade. Loggt auf Info-Level (kein Fehler), Log-Shipper
	// kann auf classification=pii-read filtern und an die Compliance-
	// Archivierung routen.
	slog.Info("admin: pii-read",
		logfields.Classification, logfields.ClassPIIRead,
		logfields.ApplicationID, id,
		logfields.RCNumber, resp.RCNumber,
		logfields.AdminUserID, adminUserID,
	)

	h.writeJSON(w, http.StatusOK, resp)
}

// UpdateApplication handles PUT /api/admin/applications/{id}
//
// @Summary      Update application (admin)
// @Description  Allows admins to correct application data. All fields are optional; only provided fields are updated.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                            true  "Application UUID"
// @Param        body  body      shared.AdminUpdateApplicationRequest  true  "Fields to update"
// @Success      200   {object}  shared.AdminApplicationDetailResponse
// @Failure      400   {object}  shared.ErrorResponse  "Validation error"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      404   {object}  shared.ErrorResponse
// @Failure      500   {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id} [put]
func (h *AdminHandler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	var req shared.AdminUpdateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	if req.AdminNote != nil {
		sanitized := h.sanitizer.Sanitize(*req.AdminNote)
		req.AdminNote = &sanitized
	}

	resp, err := h.adminService.AdminUpdateApplication(id, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// ChangeStatus handles POST /api/admin/applications/{id}/status
//
// @Summary      Change application status
// @Description  Transitions the application to a new status. Only allowed transitions are accepted (see status model). Triggers email notifications for approved/rejected.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "Application UUID"
// @Param        body  body      shared.ChangeStatusRequest  true  "Target status and optional reason"
// @Success      200   {object}  shared.ChangeStatusResponse
// @Failure      400   {object}  shared.ErrorResponse  "Unknown status value"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      409   {object}  shared.ErrorResponse  "Invalid status transition"
// @Failure      500   {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id}/status [post]
func (h *AdminHandler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	var req shared.ChangeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	// Validate toStatus is a recognised value before passing to the service.
	// Unknown values return 400; disallowed transitions return 409.
	if !isKnownStatus(req.ToStatus) {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"toStatus": "unrecognised status value",
		})))
		return
	}

	actorID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
	}

	toStatus := shared.ApplicationStatus(req.ToStatus)
	resp, err := h.adminService.ChangeStatus(id, toStatus, req.Reason, actorID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// BulkAction handles POST /api/admin/applications/bulk-action
//
// @Summary      Bulk status action
// @Description  Applies a status transition to multiple applications in one request. Applications with invalid transitions or mismatching tenant are skipped without error.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      shared.BulkActionRequest   true  "Bulk action payload"
// @Success      200   {object}  shared.BulkActionResponse
// @Failure      400   {object}  shared.ErrorResponse  "Validation error"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      500   {object}  shared.ErrorResponse
// @Router       /api/admin/applications/bulk-action [post]
func (h *AdminHandler) BulkAction(w http.ResponseWriter, r *http.Request) {
	var req shared.BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}
	if req.Action == "reject" && req.Reason == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"reason": "a reason is required for bulk rejection",
		})))
		return
	}

	ids := make([]uuid.UUID, 0, len(req.IDs))
	for _, s := range req.IDs {
		parsed, err := uuid.Parse(s)
		if err != nil {
			h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
				"ids": "invalid UUID: " + s,
			})))
			return
		}
		ids = append(ids, parsed)
	}

	toStatus := shared.ApplicationStatus(map[string]string{
		"approve":      string(shared.StatusApproved),
		"reject":       string(shared.StatusRejected),
		"under_review": string(shared.StatusUnderReview),
	}[req.Action])

	var allowedRCNumbers []string
	actorID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
		if !claims.IsSuperuser() {
			allowedRCNumbers = []string(claims.Tenant)
		}
	}

	succeeded, skipped, err := h.adminService.BulkChangeStatus(ids, toStatus, req.Reason, actorID, allowedRCNumbers)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	succeededStrs := make([]string, len(succeeded))
	for i, id := range succeeded {
		succeededStrs[i] = id.String()
	}
	skippedStrs := make([]string, len(skipped))
	for i, id := range skipped {
		skippedStrs[i] = id.String()
	}
	h.writeJSON(w, http.StatusOK, shared.BulkActionResponse{
		Succeeded: succeededStrs,
		Skipped:   skippedStrs,
	})
}

// ResendMemberConfirmation handles POST /api/admin/applications/{id}/resend-confirmation
//
// @Summary      Resend member confirmation email
// @Description  Resends the submission confirmation email to the member. Useful when the original email was not received.
// @Tags         Admin
// @Security     BearerAuth
// @Param        id   path  string  true  "Application UUID"
// @Success      204  "Email resent"
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id}/resend-confirmation [post]
func (h *AdminHandler) ResendMemberConfirmation(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	if !h.checkTenantAccess(w, r, id) {
		return
	}
	if err := h.adminService.ResendMemberConfirmation(id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CompareEEGSettingsWithCore handles GET /api/admin/settings/eeg/core-comparison?rc_number=…
//
// @Summary      Compare local EEG master data with the eegFaktura core (PROJ-32)
// @Description  Fetches the EEG master data from the core via GraphQL and diffs it field-by-field against the local registration_entrypoint row. Used by the Settings UI to show a drift banner.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        rc_number  query  string  true  "RC number"
// @Success      200  {object}  shared.EEGSettingsComparisonResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      503  {object}  shared.ErrorResponse  "Core not configured or unreachable"
// @Router       /api/admin/settings/eeg/core-comparison [get]
func (h *AdminHandler) CompareEEGSettingsWithCore(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	if h.coreClient == nil {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "EEG-Stammdaten-Sync ist nicht konfiguriert (CORE_BASE_URL leer).",
		})
		return
	}
	bearerToken := extractBearerToken(r)
	if bearerToken == "" {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Vergleich erfordert eine authentifizierte Admin-Session (Keycloak).",
		})
		return
	}

	local, err := h.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	core := h.eegCache.get(rcNumber)
	if core == nil {
		fetched, fetchErr := h.coreClient.FetchEEGMasterData(r.Context(), bearerToken, rcNumber)
		if fetchErr != nil {
			// Surface as a graceful 200 with `coreReachable: false` so the
			// frontend can render a "Core nicht erreichbar"-Hinweis instead of
			// a hard error toast.
			h.writeJSON(w, http.StatusOK, shared.EEGSettingsComparisonResponse{
				CoreReachable:        false,
				CoreUnreachableError: fetchErr.Error(),
				LastSyncedAt:         local.LastSyncedFromCoreAt,
			})
			return
		}
		h.eegCache.put(rcNumber, fetched)
		core = fetched
	}

	resp := buildEEGSettingsComparison(local, core)
	h.writeJSON(w, http.StatusOK, resp)
}

// SyncEEGSettingsFromCore handles POST /api/admin/settings/eeg/sync?rc_number=…
//
// @Summary      Pull EEG master data from the eegFaktura core (PROJ-32)
// @Description  Fetches the latest EEG master data from the core via GraphQL and writes the synced columns on registration_entrypoint. Stamps last_synced_from_core_at. The synced fields (name, address, creditor-id, contact-email) are not user-editable elsewhere.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        rc_number  query  string  true  "RC number"
// @Success      200  {object}  shared.EEGSettingsComparisonResponse
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      502  {object}  shared.ErrorResponse  "Core returned an error"
// @Failure      503  {object}  shared.ErrorResponse  "Core not configured"
// @Router       /api/admin/settings/eeg/sync [post]
func (h *AdminHandler) SyncEEGSettingsFromCore(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	if h.coreClient == nil {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "EEG-Stammdaten-Sync ist nicht konfiguriert (CORE_BASE_URL leer).",
		})
		return
	}
	bearerToken := extractBearerToken(r)
	if bearerToken == "" {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Sync erfordert eine authentifizierte Admin-Session (Keycloak).",
		})
		return
	}

	// Sync always bypasses the cache — the admin explicitly asked for
	// fresh data. Store the result so a subsequent Compare-call doesn't
	// re-hit the core just to confirm what we wrote.
	core, fetchErr := h.coreClient.FetchEEGMasterData(r.Context(), bearerToken, rcNumber)
	if fetchErr != nil {
		h.writeJSON(w, http.StatusBadGateway, shared.ErrorResponse{
			Code:    "core_unreachable",
			Message: "eegFaktura konnte nicht abgefragt werden: " + fetchErr.Error(),
		})
		return
	}

	update := application.CoreMasterDataUpdate{
		EegID:      core.CommunityID,
		EEGName:    core.Name,
		CreditorID: nilIfAccount(core, func(a *coreclient.EEGMasterDataAccount) *string { return a.CreditorID }),
	}
	if core.Address != nil {
		update.EEGStreet = core.Address.Street
		update.EEGStreetNumber = core.Address.StreetNumber
		update.EEGZip = core.Address.Zip
		update.EEGCity = core.Address.City
	}
	if core.Contact != nil {
		update.ContactEmail = core.Contact.Email
	}

	if err := h.entrypointRepo.SyncFromCore(rcNumber, update); err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Warm the compare-cache with the just-fetched payload so a Compare-call
	// arriving within the TTL doesn't re-hit the core for the same data.
	h.eegCache.put(rcNumber, core)

	// PROJ-33: best-effort logo sync. Master-data sync already succeeded;
	// a logo failure (too large, wrong MIME, billing service down) becomes
	// a warning on the response, not a hard error. ErrLogoNotFound is the
	// "no logo configured" signal — we just skip silently in that case.
	logoWarning := ""
	if logoBytes, mime, logoErr := h.coreClient.FetchEEGLogo(r.Context(), bearerToken, rcNumber); logoErr == nil {
		if saveErr := h.entrypointRepo.SaveLogoFromCore(rcNumber, logoBytes, mime); saveErr != nil {
			slog.Warn("sync: logo bytes fetched but DB write failed",
				"rc_number", rcNumber, "error", saveErr)
			logoWarning = "Logo wurde geladen, konnte aber nicht gespeichert werden"
		}
	} else if !errors.Is(logoErr, coreclient.ErrLogoNotFound) {
		switch {
		case errors.Is(logoErr, coreclient.ErrLogoTooLarge):
			logoWarning = "Logo überschreitet 256 KB — bitte in eegFaktura ein kleineres hinterlegen"
		case errors.Is(logoErr, coreclient.ErrLogoUnsupportedMIME):
			logoWarning = "Logo-Format wird nicht unterstützt (nur PNG, JPEG, GIF)"
		default:
			logoWarning = "Logo konnte nicht aus eegFaktura geladen werden: " + logoErr.Error()
		}
		slog.Info("sync: logo fetch failed", "rc_number", rcNumber, "warning", logoWarning, "error", logoErr)
	}

	// Reload so the response carries the freshly stamped last_synced_at.
	local, err := h.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	resp := buildEEGSettingsComparison(local, core)
	resp.LogoSyncWarning = logoWarning
	resp.LogoSyncedAt = local.EEGLogoSyncedAt
	h.writeJSON(w, http.StatusOK, resp)
}

// GetEEGLogo handles GET /api/admin/settings/eeg/logo?rc_number=…
//
// @Summary      Serve the cached EEG logo for inline preview (PROJ-33)
// @Description  Returns the BYTEA logo bytes pulled from the eegFaktura-billing service on the last successful sync, with the correct Content-Type. 404 when no logo has been synced yet (or the EEG has no logo configured).
// @Tags         Admin
// @Produce      image/png
// @Produce      image/jpeg
// @Produce      image/gif
// @Security     BearerAuth
// @Param        rc_number  query  string  true  "RC number"
// @Success      200  {file}    file
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse  "No logo synced yet"
// @Router       /api/admin/settings/eeg/logo [get]
func (h *AdminHandler) GetEEGLogo(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	logoBytes, mime, err := h.entrypointRepo.GetLogo(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	if len(logoBytes) == 0 || mime == "" {
		h.writeJSON(w, http.StatusNotFound, shared.ErrorResponse{
			Code:    "not_found",
			Message: "Noch kein Logo aus eegFaktura geladen",
		})
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Content-Length", strconv.Itoa(len(logoBytes)))
	_, _ = w.Write(logoBytes)
}

// nilIfAccount is a tiny helper for the SyncEEGSettingsFromCore path that
// avoids a nil-deref when core.AccountInfo is missing from the GraphQL
// response.
func nilIfAccount(core *coreclient.EEGMasterData, pick func(*coreclient.EEGMasterDataAccount) *string) *string {
	if core == nil || core.AccountInfo == nil {
		return nil
	}
	return pick(core.AccountInfo)
}

// buildEEGSettingsComparison computes the per-field diff between the local
// registration_entrypoint row and a freshly fetched core response. Used by
// both the comparison endpoint (read-only) and the sync endpoint (which
// runs the comparison again against the just-written values, so the front-
// end can immediately render the new "synchron"-Status).
func buildEEGSettingsComparison(local *shared.RegistrationEntrypoint, core *coreclient.EEGMasterData) shared.EEGSettingsComparisonResponse {
	resp := shared.EEGSettingsComparisonResponse{
		CoreReachable: true,
		LastSyncedAt:  local.LastSyncedFromCoreAt,
	}
	var coreStreet, coreStreetNumber, coreZip, coreCity *string
	if core.Address != nil {
		coreStreet = core.Address.Street
		coreStreetNumber = core.Address.StreetNumber
		coreZip = core.Address.Zip
		coreCity = core.Address.City
	}
	var coreContactEmail *string
	if core.Contact != nil {
		coreContactEmail = core.Contact.Email
	}
	var coreCreditorID *string
	if core.AccountInfo != nil {
		coreCreditorID = core.AccountInfo.CreditorID
	}

	add := func(field, label string, localVal, coreVal *string) {
		if !stringPointersEqual(localVal, coreVal) {
			resp.DifferingFields = append(resp.DifferingFields, shared.EEGSettingsFieldDiff{
				Field:     field,
				Label:     label,
				LocalValue:  derefStringPointer(localVal),
				CoreValue: derefStringPointer(coreVal),
			})
		}
	}
	add("eegId", "Gemeinschafts-ID", local.EegID, core.CommunityID)
	add("eegName", "EEG-Name", local.EEGName, core.Name)
	add("eegStreet", "Straße", local.EEGStreet, coreStreet)
	add("eegStreetNumber", "Hausnummer", local.EEGStreetNumber, coreStreetNumber)
	add("eegZip", "PLZ", local.EEGZip, coreZip)
	add("eegCity", "Ort", local.EEGCity, coreCity)
	add("creditorId", "Creditor-ID", local.CreditorID, coreCreditorID)
	add("contactEmail", "Kontakt-E-Mail", local.ContactEmail, coreContactEmail)

	resp.InSync = len(resp.DifferingFields) == 0
	return resp
}

func stringPointersEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func derefStringPointer(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ResendEmailConfirmation handles POST /api/admin/applications/{id}/resend-email-confirmation
//
// @Summary      Resend e-mail confirmation link (PROJ-31)
// @Description  Rotates the confirmation token and re-sends the member confirmation mail with a fresh link. Throttled to one resend every 5 minutes per application.
// @Tags         Admin
// @Security     BearerAuth
// @Param        id   path  string  true  "Application UUID"
// @Success      204  "Email resent"
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      409  {object}  shared.ErrorResponse  "Throttled, wrong status, or EEG opt-out"
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id}/resend-email-confirmation [post]
func (h *AdminHandler) ResendEmailConfirmation(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	if !h.checkTenantAccess(w, r, id) {
		return
	}
	if err := h.adminService.ResendEmailConfirmation(id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteApplication handles DELETE /api/admin/applications/{id}
//
// @Summary      Delete application
// @Description  Permanently deletes an application and all its metering points and status log entries.
// @Tags         Admin
// @Security     BearerAuth
// @Param        id   path  string  true  "Application UUID"
// @Success      204  "Application deleted"
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id} [delete]
func (h *AdminHandler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	claims := ClaimsFromContext(r.Context())
	userID := ""
	if claims != nil {
		userID = claims.Subject
	}
	slog.Info("admin: application deleted", "application_id", id, "user_id", userID)

	if err := h.adminService.DeleteApplication(id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteDraftApplications handles DELETE /api/admin/applications/drafts
//
// @Summary      Delete all draft applications
// @Description  Deletes all applications in status `draft` for the calling admin's EEGs. Superusers delete all drafts across all EEGs.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]int  "deleted count"
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/drafts [delete]
func (h *AdminHandler) DeleteDraftApplications(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Optional rc_number filter narrows the deletion to drafts of one EEG.
	// Used by the multi-EEG admin UI so the "delete all drafts" button
	// respects the currently active rc_number filter instead of nuking
	// drafts across every accessible EEG. The tenant-scope check below
	// still applies: a tenant-admin cannot delete in foreign RC numbers.
	rcFilter := strings.TrimSpace(r.URL.Query().Get("rc_number"))

	var rcNumbers []string
	if claims.IsSuperuser() {
		if rcFilter != "" {
			rcNumbers = []string{rcFilter}
		}
		// else: empty list signals "all RCs" → use DeleteAllDrafts below
	} else {
		tenants := []string(claims.Tenant)
		if rcFilter != "" {
			if !containsRC(tenants, rcFilter) {
				h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
				return
			}
			rcNumbers = []string{rcFilter}
		} else {
			rcNumbers = tenants
		}
	}

	var (
		n   int64
		err error
	)
	if len(rcNumbers) == 0 {
		// Only reachable for superusers with no rc_number filter.
		n, err = h.adminService.DeleteAllDrafts()
	} else {
		n, err = h.adminService.DeleteDrafts(rcNumbers)
	}
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	slog.Info("admin: draft applications deleted",
		"count", n,
		"user_id", claims.Subject,
		"superuser", claims.IsSuperuser(),
		"rc_filter", rcFilter,
	)
	h.writeJSON(w, http.StatusOK, map[string]int64{"deleted": n})
}

// ExportApplicationExcel handles GET /api/admin/applications/{id}/export/excel
//
// @Summary      Export application as Excel
// @Description  Downloads an Excel file (.xlsx) with the full application data formatted for eegFaktura import.
// @Tags         Admin
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        id   path  string  true  "Application UUID"
// @Success      200  {file}    binary   "Excel file"
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id}/export/excel [get]
func (h *AdminHandler) ExportApplicationExcel(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	data, filename, err := h.adminService.ExportApplicationExcel(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// DSGVO audit-trail: single-application Excel-Export carries full PII
	// (IBAN, birth_date, address). Same classification as PROJ-60 batch
	// exports so log-shippers can route both to the compliance archive.
	adminUserID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		adminUserID = claims.Subject
	}
	slog.Info("admin: pii-export",
		logfields.Classification, logfields.ClassPIIExport,
		logfields.ApplicationID, id,
		logfields.AdminUserID, adminUserID,
		"format", "xlsx",
	)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// DownloadApprovalPDF handles GET /api/admin/applications/{id}/approval-pdf
//
// @Summary      Download approval PDF
// @Description  Generates and downloads the Beitrittsbestätigung PDF for an approved application. Available for status approved, imported, import_failed.
// @Tags         Admin
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        id   path  string  true  "Application UUID"
// @Success      200  {file}    binary   "PDF file"
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      409  {object}  shared.ErrorResponse
// @Failure      500  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id}/approval-pdf [get]
func (h *AdminHandler) DownloadApprovalPDF(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	data, filename, err := h.adminService.GenerateApprovalPDF(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ImportApplication handles POST /api/admin/applications/{id}/import
//
// @Summary      Import an approved application into eegFaktura core
// @Description  Triggers a synchronous import of an approved application into the eegFaktura core service. On success, the application transitions to status `imported` and `targetParticipantId` is populated. On failure, the status is set to `import_failed` and the error is recorded.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Application UUID"
// @Success      200  {object}  map[string]interface{}  "Import succeeded"
// @Failure      400  {object}  shared.ErrorResponse  "Application has no metering points"
// @Failure      401  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      409  {object}  shared.ErrorResponse  "Application not in approved status"
// @Failure      500  {object}  shared.ErrorResponse  "Core service error or DB error"
// @Failure      503  {object}  shared.ErrorResponse  "Core service not configured"
// @Router       /api/admin/applications/{id}/import [post]
func (h *AdminHandler) ImportApplication(w http.ResponseWriter, r *http.Request) {
	if h.importService == nil {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Import endpoint not configured (CORE_BASE_URL is empty).",
		})
		return
	}

	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	bearerToken := h.coreBearerToken(r)
	if bearerToken == "" {
		// Dev mode without Keycloak: forwarding to the core would fail anyway,
		// because the core enforces JWT auth. Reject early with a clear message.
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Import requires an authenticated admin session (Keycloak).",
		})
		return
	}

	var actorID string
	var allowedTenants []string
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
		if !claims.IsSuperuser() {
			allowedTenants = []string(claims.Tenant)
		}
	}

	// Tariff selection + member number from the import dialog. memberNumber is
	// now required: the onboarding no longer auto-assigns it at submit time;
	// the admin picks it (pre-filled from the core's max+1 suggestion) in the
	// import dialog.
	selection := importing.TariffSelection{MeterTariffIDs: map[string]string{}}
	var body shared.ImportApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(body); err != nil {
		h.writeValidationError(w, err)
		return
	}
	selection.MemberTariffID = body.TariffID
	if body.MeterTariffs != nil {
		selection.MeterTariffIDs = body.MeterTariffs
	}

	result, err := h.importService.Import(r.Context(), id, bearerToken, actorID, allowedTenants, selection, *body.MemberNumber)
	if err != nil {
		// Pre-import typed errors — application untouched on disk.
		var validationErr shared.ValidationError
		var conflictErr shared.ConflictError
		switch {
		case errors.As(err, &validationErr), errors.As(err, &conflictErr):
			h.handleServiceError(w, err)
			return
		case errors.Is(err, shared.ErrNotFound),
			errors.Is(err, shared.ErrConflict),
			errors.Is(err, shared.ErrForbidden):
			h.handleServiceError(w, err)
			return
		}

		// Core call failed and bookkeeping recorded import_failed.
		if result != nil && result.Status == shared.StatusImportFailed {
			slog.Error("import: core call failed", "application_id", id, "error", err)
			h.writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success":       false,
				"applicationId": id,
				"status":        string(result.Status),
				"message":       result.ErrorMessage, // already normalized + bounded
			})
			return
		}

		// Bookkeeping failure after a successful core insert. The participant
		// exists in the core but our DB couldn't persist it — surface enough
		// info to allow manual cleanup, but avoid leaking the raw DB error.
		if result != nil && result.TargetParticipantID != "" {
			slog.Error("import: orphan participant created in core",
				"application_id", id,
				"target_participant_id", result.TargetParticipantID,
				"error", err,
			)
			h.writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success":             false,
				"applicationId":       id,
				"status":              string(result.Status),
				"targetParticipantId": result.TargetParticipantID,
				"message":             result.ErrorMessage,
			})
			return
		}

		slog.Error("import: unexpected error", "application_id", id, "error", err)
		h.writeError(w, shared.NewErrorResponse(shared.ErrInternal))
		return
	}

	slog.Info("admin: application imported", "application_id", id, "target_participant_id", result.TargetParticipantID, "user_id", actorID)

	// PROJ-46 Stage B: kick off the post-import notification (member welcome
	// mail with PDF + EEG copy) asynchronously. Best-effort — failures are
	// logged inside the helper; we don't block the HTTP response on SMTP.
	go h.adminService.SendPostImportNotification(id)

	resp := map[string]interface{}{
		"success":             true,
		"applicationId":       result.ApplicationID,
		"status":              string(result.Status),
		"targetParticipantId": result.TargetParticipantID,
	}
	if result.MemberTariffWarning != "" {
		resp["memberTariffWarning"] = result.MemberTariffWarning
	}
	h.writeJSON(w, http.StatusOK, resp)
}

// CheckActivation (PROJ-46 Stage D) handles POST /api/admin/applications/check-activation
// — a batch check that asks the core which of our ready_for_activation
// applications are now actually ACTIVE there, and transitions those to
// `activated`.
//
// @Summary      Check core for activation status of pending applications
// @Description  Iterates over all applications in status `ready_for_activation` (filtered by the admin's tenant claim) and queries the eegFaktura core's GET /participant per tenant. Transitions matching applications to `activated` when the core participant status is ACTIVE. Returns a summary { checked, activated, errors }.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  importing.ActivationCheckResult
// @Failure      503  {object}  shared.ErrorResponse
// @Router       /api/admin/applications/check-activation [post]
func (h *AdminHandler) CheckActivation(w http.ResponseWriter, r *http.Request) {
	if h.importService == nil {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Activation check requires Core integration (CORE_BASE_URL is empty).",
		})
		return
	}
	bearerToken := h.coreBearerToken(r)
	if bearerToken == "" {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Activation check requires an authenticated admin session (Keycloak).",
		})
		return
	}

	var allowedTenants []string
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		if !claims.IsSuperuser() {
			allowedTenants = []string(claims.Tenant)
		}
	}

	result, err := h.importService.CheckActivations(r.Context(), bearerToken, allowedTenants)
	if err != nil {
		slog.Error("activation-check: batch failed", "error", err)
		h.writeError(w, shared.NewErrorResponse(shared.ErrInternal))
		return
	}
	slog.Info("activation-check: batch ran", "checked", result.Checked, "activated", result.Activated, "errors", len(result.Errors))
	h.writeJSON(w, http.StatusOK, result)
}

// ListTariffs handles GET /api/admin/tariffs?rcNumber=<RC>
//
// @Summary      List tariffs available for an EEG (PROJ-27)
// @Description  Proxies the eegFaktura core's GET /eeg/tariff for the import-time tariff selection dialog. The admin's bearer token is forwarded; tenant isolation is enforced via the rcNumber query parameter (must be in the JWT's Tenants claim, or caller must be superuser).
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        rcNumber  query  string  true  "EEG RC number"
// @Success      200       {object}  map[string]interface{}  "{ tariffs: [...] }"
// @Failure      400       {object}  shared.ErrorResponse  "rcNumber missing"
// @Failure      401       {object}  shared.ErrorResponse
// @Failure      403       {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      503       {object}  shared.ErrorResponse  "Core service unavailable"
// @Router       /api/admin/tariffs [get]
func (h *AdminHandler) ListTariffs(w http.ResponseWriter, r *http.Request) {
	if h.importService == nil {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Core service not configured (CORE_BASE_URL is empty).",
		})
		return
	}

	rcNumber := r.URL.Query().Get("rcNumber")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rcNumber": "rcNumber query parameter is required",
		})))
		return
	}

	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}

	bearerToken := h.coreBearerToken(r)
	if bearerToken == "" {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Tariff lookup requires an authenticated admin session (Keycloak).",
		})
		return
	}

	tariffs, err := h.importService.ListTariffs(r.Context(), bearerToken, rcNumber)
	if err != nil {
		slog.Warn("admin: tariff lookup failed", "rc_number", rcNumber, "error", err)
		// Surface as 503 so the frontend can fall back to "import without tariffs".
		// The raw core error (HTTP status, response-header summary, parse detail)
		// is included verbatim so the admin sees in the UI whether it was an auth
		// problem, a proxy-rewrite, a schema drift, etc. — instead of always the
		// same generic "Tarife konnten nicht aus dem Core geladen werden."
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Tarife konnten nicht aus dem Core geladen werden: " + err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"tariffs": tariffs})
}

// SuggestNextMemberNumber handles GET /api/admin/applications/{id}/next-member-number
//
// @Summary      Suggest next member number for an application
// @Description  Returns max(numeric participantNumber) + 1 over the existing participants in the application's tenant. Used by the import dialog to pre-fill the member number field; the admin may override it before submitting the import.
// @Tags         Admin
// @Produce      json
// @Param        id  path  string  true  "Application ID"
// @Success      200  {object}  map[string]int  "next_member_number"
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Failure      503  {object}  shared.ErrorResponse  "Core not configured / lookup failed"
// @Security     BearerAuth
// @Router       /api/admin/applications/{id}/next-member-number [get]
func (h *AdminHandler) SuggestNextMemberNumber(w http.ResponseWriter, r *http.Request) {
	if h.importService == nil {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Core service not configured (CORE_BASE_URL is empty).",
		})
		return
	}

	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	if !h.checkTenantAccess(w, r, id) {
		return
	}

	rcNumber, err := h.adminService.GetRCNumberByID(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	bearerToken := h.coreBearerToken(r)
	if bearerToken == "" {
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Member-number lookup requires an authenticated admin session (Keycloak).",
		})
		return
	}

	next, err := h.importService.SuggestNextMemberNumber(r.Context(), bearerToken, rcNumber)
	if err != nil {
		slog.Warn("admin: next-member-number lookup failed", "application_id", id, "rc_number", rcNumber, "error", err)
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Nächste Mitgliedsnummer konnte nicht aus dem Core ermittelt werden: " + err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"next_member_number": next})
}

// UpdateAdminNote handles PATCH /api/admin/applications/{id}/admin-note
//
// @Summary      Update admin note
// @Description  Replaces only the admin_note column. Does not touch member type, metering points, participation factors, or any other field — by design, so saving a note from the admin UI cannot accidentally reset application data.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id    path      string                          true  "Application ID"
// @Param        body  body      shared.UpdateAdminNoteRequest  true  "New note (empty string clears it)"
// @Success      204  {string}  string  "no content"
// @Failure      400  {object}  shared.ErrorResponse
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Security     BearerAuth
// @Router       /api/admin/applications/{id}/admin-note [patch]
func (h *AdminHandler) UpdateAdminNote(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	var req shared.UpdateAdminNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	if err := h.adminService.UpdateAdminNote(id, req.Note); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ResetImport handles POST /api/admin/applications/{id}/reset-import
//
// @Summary      Reset an imported application back to approved (PROJ-30)
// @Description  Transitions an application from `imported` back to `approved` so it can be re-imported after the eegFaktura admin deleted the participant in the core. Clears `target_participant_id` and all `import_*` bookkeeping fields. A reason is mandatory and recorded in the status_log; the previous `target_participant_id` is archived in the same log entry. The eegFaktura core is NOT contacted — admin verifies the deletion manually.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string                       true  "Application UUID"
// @Param        body  body  shared.ResetImportRequest    true  "Reason for the reset"
// @Success      200   {object}  shared.AdminApplicationDetailResponse
// @Failure      400   {object}  shared.ErrorResponse  "Validation failed"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      404   {object}  shared.ErrorResponse
// @Failure      409   {object}  shared.ErrorResponse  "Application not in imported status"
// @Failure      500   {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id}/reset-import [post]
func (h *AdminHandler) ResetImport(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	var req shared.ResetImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	actorID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
	}

	app, err := h.adminService.ResetImport(id, req.Reason, actorID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	detail, err := h.adminService.GetApplicationDetail(id)
	if err != nil {
		// Fallback: return the bare application; UI can still re-render header.
		slog.Warn("admin: reset-import succeeded but detail fetch failed",
			"application_id", id, "error", err)
		h.writeJSON(w, http.StatusOK, app)
		return
	}
	h.writeJSON(w, http.StatusOK, detail)
}

// MarkImportedManually handles POST /api/admin/applications/{id}/mark-imported-manually
//
// @Summary      Manually close a stuck import (PROJ-34)
// @Description  Recovery for the orphan scenario where the core created the participant but the onboarding bookkeeping failed and left the application stuck in `approved` with an in-flight slot. Admin reads the participant UUID + member-number from eegFaktura and submits them; status transitions to `imported`. Refused with 409 when the application is not in the stuck state.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string                            true  "Application UUID"
// @Param        body  body  shared.MarkImportedManuallyRequest true  "Participant UUID + member-number from eegFaktura"
// @Success      200   {object}  shared.AdminApplicationDetailResponse
// @Failure      400   {object}  shared.ErrorResponse  "Validation failed"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      404   {object}  shared.ErrorResponse
// @Failure      409   {object}  shared.ErrorResponse  "Application not in stuck import state"
// @Router       /api/admin/applications/{id}/mark-imported-manually [post]
func (h *AdminHandler) MarkImportedManually(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	if !h.checkTenantAccess(w, r, id) {
		return
	}
	var req shared.MarkImportedManuallyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}
	actorID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
	}
	if _, err := h.adminService.MarkImportedManually(id, req.TargetParticipantID, req.MemberNumber, req.Reason, actorID); err != nil {
		h.handleServiceError(w, err)
		return
	}
	detail, err := h.adminService.GetApplicationDetail(id)
	if err != nil {
		slog.Warn("admin: mark-imported-manually succeeded but detail fetch failed",
			"application_id", id, "error", err)
		h.writeJSON(w, http.StatusOK, map[string]string{"status": "imported"})
		return
	}
	h.writeJSON(w, http.StatusOK, detail)
}

// MarkActivated handles POST /api/admin/applications/{id}/mark-activated (PROJ-53)
//
// @Summary      Manuelle Aktivierung (Import übersprungen)
// @Description  Setzt eine Anwendung direkt von `approved` auf `activated`, ohne den eegFaktura-Core-Import. Ausnahmefall: das Mitglied existiert im Core bereits (Faktura kann Mitglieder nicht löschen) und wurde dort manuell mit den Onboarding-Daten überschrieben. Der Admin gibt die Mitgliedsnummer mit, damit die Beitrittsbestätigungs-Mail (wird hier ebenfalls versandt) die korrekte Referenz enthält. Nur aus Status `approved` zulässig.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string                       true  "Application UUID"
// @Param        body  body  shared.MarkActivatedRequest  true  "Mitgliedsnummer"
// @Success      200   {object}  shared.AdminApplicationDetailResponse
// @Failure      400   {object}  shared.ErrorResponse  "Validation failed"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse
// @Failure      404   {object}  shared.ErrorResponse
// @Failure      409   {object}  shared.ErrorResponse  "Application not in approved status, or member-number conflict"
// @Router       /api/admin/applications/{id}/mark-activated [post]
func (h *AdminHandler) MarkActivated(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	if !h.checkTenantAccess(w, r, id) {
		return
	}
	var req shared.MarkActivatedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}
	actorID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
	}
	if _, err := h.adminService.MarkActivatedSkipImport(id, req.MemberNumber, actorID); err != nil {
		h.handleServiceError(w, err)
		return
	}
	detail, err := h.adminService.GetApplicationDetail(id)
	if err != nil {
		slog.Warn("admin: mark-activated succeeded but detail fetch failed",
			"application_id", id, "error", err)
		h.writeJSON(w, http.StatusOK, map[string]string{"status": "activated"})
		return
	}
	h.writeJSON(w, http.StatusOK, detail)
}

// ClearImportLock handles POST /api/admin/applications/{id}/clear-import-lock
//
// @Summary      Release a stuck import lock for retry (PROJ-34)
// @Description  Clears the in-flight slot on a stuck application without changing its status. Useful when the admin wants to retry the import — at the explicit risk of producing a duplicate in the core if the original attempt had already inserted there. A reason is mandatory and recorded in the status_log together with the previous target_participant_id (if any).
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string                          true  "Application UUID"
// @Param        body  body  shared.ClearImportLockRequest   true  "Reason for the lock release"
// @Success      200   {object}  shared.AdminApplicationDetailResponse
// @Failure      400   {object}  shared.ErrorResponse  "Validation failed"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      404   {object}  shared.ErrorResponse
// @Failure      409   {object}  shared.ErrorResponse  "Application not in stuck import state"
// @Router       /api/admin/applications/{id}/clear-import-lock [post]
func (h *AdminHandler) ClearImportLock(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	if !h.checkTenantAccess(w, r, id) {
		return
	}
	var req shared.ClearImportLockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}
	actorID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
	}
	if _, err := h.adminService.ClearImportLock(id, req.Reason, actorID); err != nil {
		h.handleServiceError(w, err)
		return
	}
	detail, err := h.adminService.GetApplicationDetail(id)
	if err != nil {
		slog.Warn("admin: clear-import-lock succeeded but detail fetch failed",
			"application_id", id, "error", err)
		h.writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
		return
	}
	h.writeJSON(w, http.StatusOK, detail)
}

// GetIntroText handles GET /api/admin/settings/intro-text?rc_number=...
func (h *AdminHandler) GetIntroText(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}

	ep, err := h.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"rcNumber": rcNumber, "introText": ep.IntroText})
}

// SaveIntroText handles PUT /api/admin/settings/intro-text?rc_number=...
func (h *AdminHandler) SaveIntroText(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}

	var body struct {
		IntroText *string `json:"introText"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	var sanitized *string
	if body.IntroText != nil {
		s := h.sanitizer.Sanitize(*body.IntroText)
		if s == "" {
			sanitized = nil
		} else {
			sanitized = &s
		}
	}

	if err := h.entrypointRepo.SaveIntroText(rcNumber, sanitized); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEEGSettings handles GET /api/admin/settings/eeg?rc_number=...
func (h *AdminHandler) GetEEGSettings(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}

	ep, err := h.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"rcNumber":                rcNumber,
		"registrationActive":      ep.IsActive,
		"eegId":                   ep.EegID,
		"eegName":                 ep.EEGName,
		"eegStreet":               ep.EEGStreet,
		"eegStreetNumber":         ep.EEGStreetNumber,
		"eegZip":                  ep.EEGZip,
		"eegCity":                 ep.EEGCity,
		"creditorId":              ep.CreditorID,
		"contactEmail":            ep.ContactEmail,
		"lastSyncedFromCoreAt":    ep.LastSyncedFromCoreAt,
		"eegLogoSyncedAt":         ep.EEGLogoSyncedAt,
		"sepaMandateEnabled":      ep.SEPAMandateEnabled,
		"useCompanySEPAMandate":   ep.UseCompanySEPAMandate,
		"sepaMandateAtImport":     ep.SEPAMandateAtImport,
		"showCentralPolicy":       ep.ShowCentralPolicy,
		"memberNumberStart":       ep.MemberNumberStart,
		"requireEmailConfirmation": ep.RequireEmailConfirmation,
		// PROJ-52: pro Richtung konfigurierbarer Zählpunkt-Prefix.
		"meteringPointPrefixConsumption": ep.MeteringPointPrefixConsumption,
		"meteringPointPrefixProduction":  ep.MeteringPointPrefixProduction,
		// PROJ-53 Aktivierungs-Modus
		"activationMode":                 ep.ActivationMode,
		// PROJ-37 Genossenschaftsanteile
		"cooperativeSharesEnabled":    ep.CooperativeSharesEnabled,
		"cooperativeRequiredShares":   ep.CooperativeRequiredShares,
		"cooperativeShareAmountCents": ep.CooperativeShareAmountCents,
	})
}

// SaveEEGSettings handles PUT /api/admin/settings/eeg?rc_number=...
func (h *AdminHandler) SaveEEGSettings(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}

	var body struct {
		RegistrationActive       *bool  `json:"registrationActive"`
		SEPAMandateEnabled       bool   `json:"sepaMandateEnabled"`
		UseCompanySEPAMandate    bool   `json:"useCompanySEPAMandate"`
		SEPAMandateAtImport      bool   `json:"sepaMandateAtImport"`
		ShowCentralPolicy        *bool  `json:"showCentralPolicy"`
		MemberNumberStart        *int   `json:"memberNumberStart"`
		RequireEmailConfirmation *bool  `json:"requireEmailConfirmation"`
		// PROJ-52: zwei optionale Prefixes — jedes Feld ist ein
		// **PATCH**-Signal: nil = unverändert lassen, leerer String = clearen,
		// Wert = setzen (nach Normalisierung).
		MeteringPointPrefixConsumption *string `json:"meteringPointPrefixConsumption"`
		MeteringPointPrefixProduction  *string `json:"meteringPointPrefixProduction"`
		// Flag, ob die Prefix-Felder im Body überhaupt mitgeliefert wurden.
		// Wir brauchen das, weil json.Decode nil von "Feld nicht im Body"
		// nicht unterscheiden kann — also macht der Frontend ein explizites
		// Mitsenden (nil ⇒ clear, leerer String ⇒ clear, Wert ⇒ set).
		MeteringPointPrefixesPresent bool `json:"meteringPointPrefixesPresent"`
		// PROJ-53: Aktivierungs-Modus. nil = unverändert lassen,
		// "participant_active" oder "any_meter_registration_started" = setzen.
		ActivationMode *string `json:"activationMode"`
		// PROJ-37 Genossenschaftsanteile
		CooperativeSharesEnabled    bool   `json:"cooperativeSharesEnabled"`
		CooperativeRequiredShares   *int   `json:"cooperativeRequiredShares"`
		CooperativeShareAmountCents *int64 `json:"cooperativeShareAmountCents"`
		// Fields that are now Core-mastered (PROJ-32: eegId, eegName,
		// address, creditorId, contactEmail) are deliberately NOT accepted
		// here. A legacy admin client that still sends them won't 400 —
		// json.Decode just ignores unknown fields.
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	// PROJ-37: cooperative-shares cross-field validation. When the toggle
	// is on, both config values must be present and positive; when off,
	// the two value fields are forcibly cleared so a re-enable later
	// starts from a clean slate.
	coopRequired := body.CooperativeRequiredShares
	coopAmount := body.CooperativeShareAmountCents
	if body.CooperativeSharesEnabled {
		fields := map[string]string{}
		if coopRequired == nil || *coopRequired <= 0 {
			fields["cooperativeRequiredShares"] = "Pflichtanteile je Standort sind erforderlich (mindestens 1)"
		}
		if coopAmount == nil || *coopAmount <= 0 {
			fields["cooperativeShareAmountCents"] = "Anteilswert ist erforderlich und muss größer 0 sein"
		}
		if len(fields) > 0 {
			h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", fields)))
			return
		}
	} else {
		coopRequired = nil
		coopAmount = nil
	}

	if err := h.entrypointRepo.SaveEEGSettings(
		rcNumber,
		body.SEPAMandateEnabled,
		body.UseCompanySEPAMandate,
		body.SEPAMandateAtImport,
		body.CooperativeSharesEnabled,
		coopRequired,
		coopAmount,
	); err != nil {
		h.handleServiceError(w, err)
		return
	}

	if body.RegistrationActive != nil {
		if err := h.entrypointRepo.SaveIsActive(rcNumber, *body.RegistrationActive); err != nil {
			h.handleServiceError(w, err)
			return
		}
	}

	if body.ShowCentralPolicy != nil {
		if err := h.entrypointRepo.SaveShowCentralPolicy(rcNumber, *body.ShowCentralPolicy); err != nil {
			h.handleServiceError(w, err)
			return
		}
	}

	if body.MemberNumberStart != nil {
		if err := h.entrypointRepo.SaveMemberNumberStart(rcNumber, *body.MemberNumberStart); err != nil {
			h.handleServiceError(w, err)
			return
		}
	}

	if body.RequireEmailConfirmation != nil {
		if err := h.entrypointRepo.SaveRequireEmailConfirmation(rcNumber, *body.RequireEmailConfirmation); err != nil {
			h.handleServiceError(w, err)
			return
		}
	}

	// PROJ-52: nur speichern wenn der Frontend das Feld explizit
	// mitsendet. Normalisierung (Whitespace + Dots entfernen, uppercase)
	// und Format-Check (^AT[0-9A-Z]{0,31}$) laufen vor dem Save.
	if body.MeteringPointPrefixesPresent {
		consumption := application.NormalizeMeteringPointPrefix(body.MeteringPointPrefixConsumption)
		production := application.NormalizeMeteringPointPrefix(body.MeteringPointPrefixProduction)
		fields := map[string]string{}
		if err := application.ValidateMeteringPointPrefix(consumption); err != nil {
			fields["meteringPointPrefixConsumption"] = err.Error()
		}
		if err := application.ValidateMeteringPointPrefix(production); err != nil {
			fields["meteringPointPrefixProduction"] = err.Error()
		}
		if len(fields) > 0 {
			h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", fields)))
			return
		}
		if err := h.entrypointRepo.SaveMeteringPointPrefixes(rcNumber, consumption, production); err != nil {
			h.handleServiceError(w, err)
			return
		}
	}

	// PROJ-53: Aktivierungs-Modus. Patch-Semantik wie bei den anderen
	// optionalen Settings — nur speichern, wenn der Frontend das Feld
	// explizit mitsendet. Validierung gegen das Enum, sonst 400.
	if body.ActivationMode != nil {
		mode := *body.ActivationMode
		if !shared.IsValidActivationMode(mode) {
			h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
				"activationMode": "ungültiger Wert (erlaubt: participant_active, any_meter_registration_started)",
			})))
			return
		}
		if err := h.entrypointRepo.SaveActivationMode(rcNumber, mode); err != nil {
			h.handleServiceError(w, err)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAPIKeyStatus handles GET /api/admin/settings/api-key?rc_number=...
func (h *AdminHandler) GetAPIKeyStatus(w http.ResponseWriter, r *http.Request) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rc_number": "rc_number query parameter is required",
		})))
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}

	active, lastGenAt, err := h.apiKeyRepo.GetStatus(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	var lastGenStr *string
	if lastGenAt != nil {
		s := lastGenAt.UTC().Format(time.RFC3339)
		lastGenStr = &s
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"active":          active,
		"lastGeneratedAt": lastGenStr,
	})
}

// GenerateAPIKey handles POST /api/admin/settings/api-key?rc_number=...
// Returns the plaintext key exactly once.
func (h *AdminHandler) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rc_number": "rc_number query parameter is required",
		})))
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}

	rawKey, err := generateAPIKeyString()
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	keyHash := hashAPIKey(rawKey)
	if err := h.apiKeyRepo.Upsert(rcNumber, keyHash); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]string{
		"apiKey": rawKey,
	})
}

// RevokeAPIKey handles DELETE /api/admin/settings/api-key?rc_number=...
func (h *AdminHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rc_number": "rc_number query parameter is required",
		})))
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}

	if err := h.apiKeyRepo.Revoke(rcNumber); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const apiKeyAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateAPIKeyString returns a cryptographically random key in the format moak_<32 chars>.
func generateAPIKeyString() (string, error) {
	b := make([]byte, 32)
	alphabetLen := big.NewInt(int64(len(apiKeyAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		b[i] = apiKeyAlphabet[n.Int64()]
	}
	return "moak_" + string(b), nil
}

// --- helpers ---

func (h *AdminHandler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, error) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid application ID", nil)))
		return uuid.Nil, err
	}
	return id, nil
}

func (h *AdminHandler) writeValidationError(w http.ResponseWriter, err error) {
	fields := make(map[string]string)
	for _, verr := range err.(validator.ValidationErrors) {
		field := verr.Field()
		if _, exists := fields[field]; !exists {
			fields[field] = validationMessage(verr)
		}
	}
	h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", fields)))
}

func (h *AdminHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case shared.ValidationError:
		h.writeError(w, shared.NewErrorResponse(e))
	case shared.ConflictError:
		h.writeError(w, shared.NewErrorResponse(e))
	case shared.UnprocessableEntityError:
		h.writeError(w, shared.NewErrorResponse(e))
	default:
		switch err {
		case shared.ErrNotFound:
			h.writeError(w, shared.NewErrorResponse(shared.ErrNotFound))
		case shared.ErrGone:
			h.writeError(w, shared.NewErrorResponse(shared.ErrGone))
		case shared.ErrConflict:
			h.writeError(w, shared.NewErrorResponse(shared.ErrConflict))
		default:
			slog.Error("internal error", "error", err)
			h.writeError(w, shared.NewErrorResponse(shared.ErrInternal))
		}
	}
}

func (h *AdminHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *AdminHandler) writeError(w http.ResponseWriter, errorResp shared.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(errorResp.Code))
	json.NewEncoder(w).Encode(errorResp)
}

func intQueryParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}

func containsRC(tenants []string, rc string) bool {
	for _, t := range tenants {
		if t == rc {
			return true
		}
	}
	return false
}

// ListLegalDocuments handles GET /api/admin/legal-documents?rc_number=...
func (h *AdminHandler) ListLegalDocuments(w http.ResponseWriter, r *http.Request) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rc_number": "rc_number query parameter is required",
		})))
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}
	docs, err := h.legalDocumentRepo.GetByRCNumber(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	if docs == nil {
		docs = []shared.LegalDocument{}
	}
	h.writeJSON(w, http.StatusOK, docs)
}

// validateLegalDocumentFields checks title/url constraints shared by create and update.
// Returns a validation error response or nil.
func validateLegalDocumentFields(title, rawURL string) error {
	if title == "" || rawURL == "" {
		return shared.NewValidationError("Validation failed", map[string]string{
			"title": "title and url are required",
		})
	}
	if len(title) > 500 {
		return shared.NewValidationError("Validation failed", map[string]string{
			"title": "title must not exceed 500 characters",
		})
	}
	if len(rawURL) > 2048 {
		return shared.NewValidationError("Validation failed", map[string]string{
			"url": "url must not exceed 2048 characters",
		})
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return shared.NewValidationError("Validation failed", map[string]string{
			"url": "url must use http or https scheme",
		})
	}
	return nil
}

// CreateLegalDocument handles POST /api/admin/legal-documents?rc_number=...
func (h *AdminHandler) CreateLegalDocument(w http.ResponseWriter, r *http.Request) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rc_number": "rc_number query parameter is required",
		})))
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}

	var body struct {
		Title    string `json:"title"`
		URL      string `json:"url"`
		Required bool   `json:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := validateLegalDocumentFields(body.Title, body.URL); err != nil {
		h.writeError(w, shared.NewErrorResponse(err))
		return
	}

	if err := h.entrypointRepo.UpsertForRCNumbers([]string{rcNumber}); err != nil {
		h.handleServiceError(w, err)
		return
	}

	count, err := h.legalDocumentRepo.CountByRCNumber(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	if count >= application.MaxLegalDocumentsPerEEG {
		h.writeError(w, shared.NewErrorResponse(shared.NewConflictError("maximum number of legal documents reached")))
		return
	}

	doc, err := h.legalDocumentRepo.Create(rcNumber, body.Title, body.URL, body.Required, count)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, doc)
}

// UpdateLegalDocument handles PUT /api/admin/legal-documents/{id}
func (h *AdminHandler) UpdateLegalDocument(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	existing, err := h.legalDocumentRepo.GetByID(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, existing.RCNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}

	var body struct {
		Title    string `json:"title"`
		URL      string `json:"url"`
		Required bool   `json:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := validateLegalDocumentFields(body.Title, body.URL); err != nil {
		h.writeError(w, shared.NewErrorResponse(err))
		return
	}

	if err := h.legalDocumentRepo.Update(id, body.Title, body.URL, body.Required); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteLegalDocument handles DELETE /api/admin/legal-documents/{id}
func (h *AdminHandler) DeleteLegalDocument(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}
	existing, err := h.legalDocumentRepo.GetByID(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, existing.RCNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}
	if err := h.legalDocumentRepo.Delete(id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ReorderLegalDocuments handles PUT /api/admin/legal-documents/reorder?rc_number=...
func (h *AdminHandler) ReorderLegalDocuments(w http.ResponseWriter, r *http.Request) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"rc_number": "rc_number query parameter is required",
		})))
		return
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		h.writeError(w, shared.NewErrorResponse(shared.ErrForbidden))
		return
	}

	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	ids := make([]uuid.UUID, 0, len(body.IDs))
	for _, s := range body.IDs {
		parsed, err := uuid.Parse(s)
		if err != nil {
			h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
				"ids": "invalid UUID: " + s,
			})))
			return
		}
		ids = append(ids, parsed)
	}

	if err := h.legalDocumentRepo.Reorder(rcNumber, ids); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ReassignEEG handles POST /api/admin/applications/{id}/reassign-eeg (PROJ-40).
//
// @Summary      Reassign application to a different EEG
// @Description  Moves an application from its current EEG to a different EEG. The admin must be authorized for BOTH source and target (or be a superuser). The reference number is regenerated from the target EEG's per-year counter. Old rc_number + old reference_number are archived in the status_log. Only reassignable while status ∈ {submitted, email_confirmed, under_review, needs_info}.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                       true  "Application ID"
// @Param        body  body      shared.ReassignEEGRequest    true  "Target RC + reason"
// @Success      200   {object}  shared.AdminApplicationDetailResponse
// @Failure      400   {object}  shared.ErrorResponse  "Validation error"
// @Failure      403   {object}  shared.ErrorResponse  "Tenant mismatch on source or target"
// @Failure      404   {object}  shared.ErrorResponse  "Application or target RC not found"
// @Failure      409   {object}  shared.ErrorResponse  "Status not reassignable, source==target, or target inactive"
// @Router       /api/admin/applications/{id}/reassign-eeg [post]
func (h *AdminHandler) ReassignEEG(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	// Source-side tenant check first — admin must own the application's
	// current RC. Target-side check is done inside ReassignEEG using the
	// passed allowedRCNumbers, so we don't need to look it up here.
	if !h.checkTenantAccess(w, r, id) {
		return
	}

	var req shared.ReassignEEGRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	actorID := ""
	var allowedRCNumbers []string
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Subject
		if !claims.IsSuperuser() {
			allowedRCNumbers = []string(claims.Tenant)
		}
	}

	app, err := h.adminService.ReassignEEG(id, req.TargetRCNumber, req.Reason, actorID, allowedRCNumbers)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	detail, err := h.adminService.GetApplicationDetail(id)
	if err != nil {
		slog.Warn("admin: reassign-eeg succeeded but detail fetch failed",
			"application_id", id, "error", err)
		h.writeJSON(w, http.StatusOK, app)
		return
	}
	h.writeJSON(w, http.StatusOK, detail)
}

func isKnownStatus(s string) bool {
	switch shared.ApplicationStatus(s) {
	case shared.StatusDraft, shared.StatusSubmitted, shared.StatusEmailConfirmed,
		shared.StatusUnderReview, shared.StatusNeedsInfo, shared.StatusApproved,
		shared.StatusRejected, shared.StatusImported, shared.StatusImportFailed:
		return true
	}
	return false
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	default:
		return "Invalid value"
	}
}
