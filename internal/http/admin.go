package http

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// AdminHandler handles admin-facing HTTP requests.
type AdminHandler struct {
	adminService   *application.AdminApplicationService
	entrypointRepo *application.RegistrationEntrypointRepository
	apiKeyRepo     *application.ExternalAPIKeyRepository
	validate       *validator.Validate
	sanitizer      *bluemonday.Policy
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	adminService *application.AdminApplicationService,
	entrypointRepo *application.RegistrationEntrypointRepository,
	apiKeyRepo *application.ExternalAPIKeyRepository,
) *AdminHandler {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "ul", "ol", "li")
	p.AllowAttrs("href", "target", "rel").OnElements("a")
	p.AllowURLSchemes("http", "https", "mailto")
	return &AdminHandler{
		adminService:   adminService,
		entrypointRepo: entrypointRepo,
		apiKeyRepo:     apiKeyRepo,
		validate:       validator.New(),
		sanitizer:      p,
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
	detail, err := h.adminService.GetApplicationDetail(id)
	if err != nil {
		h.handleServiceError(w, err)
		return false
	}
	if !containsRC(claims.Tenant, detail.RCNumber) {
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

	resp, err := h.adminService.AdminUpdateApplication(id, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// ChangeStatus handles POST /api/admin/applications/{id}/status
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

// ResendMemberConfirmation handles POST /api/admin/applications/{id}/resend-confirmation
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
func (h *AdminHandler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(w, r)
	if err != nil {
		return
	}

	if !h.checkTenantAccess(w, r, id) {
		return
	}

	if err := h.adminService.DeleteApplication(id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		"eegName":                 ep.EEGName,
		"eegStreet":               ep.EEGStreet,
		"eegStreetNumber":         ep.EEGStreetNumber,
		"eegZip":                  ep.EEGZip,
		"eegCity":                 ep.EEGCity,
		"creditorId":              ep.CreditorID,
		"sepaMandateEnabled":      ep.SEPAMandateEnabled,
		"useCompanySEPAMandate":   ep.UseCompanySEPAMandate,
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
		EEGName               *string `json:"eegName"`
		EEGStreet             *string `json:"eegStreet"`
		EEGStreetNumber       *string `json:"eegStreetNumber"`
		EEGZip                *string `json:"eegZip"`
		EEGCity               *string `json:"eegCity"`
		CreditorID            *string `json:"creditorId"`
		SEPAMandateEnabled    bool    `json:"sepaMandateEnabled"`
		UseCompanySEPAMandate bool    `json:"useCompanySEPAMandate"`
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
	default:
		switch err {
		case shared.ErrNotFound:
			h.writeError(w, shared.NewErrorResponse(shared.ErrNotFound))
		case shared.ErrGone:
			h.writeError(w, shared.NewErrorResponse(shared.ErrGone))
		case shared.ErrConflict:
			h.writeError(w, shared.NewErrorResponse(shared.ErrConflict))
		default:
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
