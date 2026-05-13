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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/importing"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// AdminHandler handles admin-facing HTTP requests.
type AdminHandler struct {
	adminService      *application.AdminApplicationService
	entrypointRepo    *application.RegistrationEntrypointRepository
	apiKeyRepo        *application.ExternalAPIKeyRepository
	legalDocumentRepo *application.LegalDocumentRepository
	importService     *importing.ImportService
	validate          *validator.Validate
	sanitizer         *bluemonday.Policy
}

// NewAdminHandler creates a new AdminHandler. importService may be nil when
// CORE_BASE_URL is not configured — the import endpoint then returns 503.
func NewAdminHandler(
	adminService *application.AdminApplicationService,
	entrypointRepo *application.RegistrationEntrypointRepository,
	apiKeyRepo *application.ExternalAPIKeyRepository,
	legalDocumentRepo *application.LegalDocumentRepository,
	importService *importing.ImportService,
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
		validate:          validator.New(),
		sanitizer:         p,
	}
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
// ListApplications handles GET /api/admin/applications
//
// @Summary      List applications
// @Description  Returns a paginated, filterable list of member applications. Tenant-admins only see their own EEG's applications; superusers see all.
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        status          query  string  false  "Filter by status (draft|submitted|under_review|needs_info|approved|rejected|imported|import_failed)"
// @Param        reference_number query string false "Filter by reference number (partial match)"
// @Param        lastname        query  string  false  "Filter by member lastname (partial match)"
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
	if v := q.Get("lastname"); v != "" {
		filters.Lastname = &v
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
	if claims := ClaimsFromContext(r.Context()); claims != nil && !claims.IsSuperuser() {
		if !containsRC(claims.Tenant, resp.RCNumber) {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"code":    "forbidden",
				"message": "Kein Zugriff auf diesen Antrag.",
			})
			return
		}
	}

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

	var (
		n   int64
		err error
	)
	if claims.IsSuperuser() {
		n, err = h.adminService.DeleteAllDrafts()
	} else {
		n, err = h.adminService.DeleteDrafts([]string(claims.Tenant))
	}
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	slog.Info("admin: draft applications deleted", "count", n, "user_id", claims.Subject, "superuser", claims.IsSuperuser())
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

	bearerToken := extractBearerToken(r)
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

	// PROJ-27: optional tariff selection sent by the import dialog. Body is
	// fully optional for backward compatibility — an empty body means "no
	// tariffs", same as the legacy import behaviour.
	selection := importing.TariffSelection{MeterTariffIDs: map[string]string{}}
	if r.ContentLength > 0 {
		var body shared.ImportApplicationRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
			return
		}
		selection.MemberTariffID = body.TariffID
		if body.MeterTariffs != nil {
			selection.MeterTariffIDs = body.MeterTariffs
		}
	}

	result, err := h.importService.Import(r.Context(), id, bearerToken, actorID, allowedTenants, selection)
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

	bearerToken := extractBearerToken(r)
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
		h.writeJSON(w, http.StatusServiceUnavailable, shared.ErrorResponse{
			Code:    "service_unavailable",
			Message: "Tarife konnten nicht aus dem Core geladen werden.",
		})
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"tariffs": tariffs})
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
// @Success      200   {object}  shared.AdminApplicationDetail
// @Failure      400   {object}  shared.ErrorResponse  "Validation failed"
// @Failure      401   {object}  shared.ErrorResponse
// @Failure      403   {object}  shared.ErrorResponse  "Tenant mismatch"
// @Failure      404   {object}  shared.ErrorResponse
// @Failure      409   {object}  shared.ErrorResponse  "Application not in imported status"
// @Failure      500   {object}  shared.ErrorResponse
// @Router       /api/admin/applications/{id}/reset-import [post]
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

// GetIntroText handles GET /api/admin/settings/intro-text?rc_number=...
func (h *AdminHandler) GetIntroText(w http.ResponseWriter, r *http.Request) {
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

	ep, err := h.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{"rcNumber": rcNumber, "introText": ep.IntroText})
}

// SaveIntroText handles PUT /api/admin/settings/intro-text?rc_number=...
func (h *AdminHandler) SaveIntroText(w http.ResponseWriter, r *http.Request) {
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
		"sepaMandateEnabled":      ep.SEPAMandateEnabled,
		"useCompanySEPAMandate":   ep.UseCompanySEPAMandate,
		"showCentralPolicy":       ep.ShowCentralPolicy,
		"memberNumberStart":       ep.MemberNumberStart,
	})
}

// SaveEEGSettings handles PUT /api/admin/settings/eeg?rc_number=...
func (h *AdminHandler) SaveEEGSettings(w http.ResponseWriter, r *http.Request) {
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
		RegistrationActive    *bool   `json:"registrationActive"`
		EegID                 *string `json:"eegId"`
		EEGName               *string `json:"eegName"`
		EEGStreet             *string `json:"eegStreet"`
		EEGStreetNumber       *string `json:"eegStreetNumber"`
		EEGZip                *string `json:"eegZip"`
		EEGCity               *string `json:"eegCity"`
		CreditorID            *string `json:"creditorId"`
		SEPAMandateEnabled    bool    `json:"sepaMandateEnabled"`
		UseCompanySEPAMandate bool    `json:"useCompanySEPAMandate"`
		ShowCentralPolicy     *bool   `json:"showCentralPolicy"`
		MemberNumberStart     *int    `json:"memberNumberStart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	// Normalise: empty string → nil
	nilIfEmpty := func(s *string) *string {
		if s == nil || *s == "" {
			return nil
		}
		return s
	}

	if err := h.entrypointRepo.SaveEEGSettings(
		rcNumber,
		nilIfEmpty(body.EegID),
		nilIfEmpty(body.EEGName),
		nilIfEmpty(body.EEGStreet),
		nilIfEmpty(body.EEGStreetNumber),
		nilIfEmpty(body.EEGZip),
		nilIfEmpty(body.EEGCity),
		nilIfEmpty(body.CreditorID),
		body.SEPAMandateEnabled,
		body.UseCompanySEPAMandate,
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

func isKnownStatus(s string) bool {
	switch shared.ApplicationStatus(s) {
	case shared.StatusDraft, shared.StatusSubmitted, shared.StatusUnderReview,
		shared.StatusNeedsInfo, shared.StatusApproved, shared.StatusRejected,
		shared.StatusImported, shared.StatusImportFailed:
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
