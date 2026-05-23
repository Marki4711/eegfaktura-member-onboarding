package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/dataexport"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// writeValidationErrorDE is a local helper that mirrors AdminHandler.writeValidationError
// without requiring a method receiver. Uses the existing validationMessage formatter.
func writeValidationErrorDE(w http.ResponseWriter, err error) {
	fields := make(map[string]string)
	if verrs, ok := err.(validator.ValidationErrors); ok {
		for _, verr := range verrs {
			field := verr.Field()
			if _, exists := fields[field]; !exists {
				fields[field] = validationMessage(verr)
			}
		}
	}
	writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Validation failed", fields)))
}

// DataExportHandler exposes PROJ-60 endpoints for plugin-driven data
// forwarding (Excel/CSV-Plugin in V1, CRM-Plugins later).
type DataExportHandler struct {
	configService *dataexport.ConfigService
	jobService    *dataexport.JobService
	validate      *validator.Validate
}

// NewDataExportHandler wires up the handler. Mailer is optional and lives
// inside the Worker, not here.
func NewDataExportHandler(cs *dataexport.ConfigService, js *dataexport.JobService) *DataExportHandler {
	return &DataExportHandler{
		configService: cs,
		jobService:    js,
		validate:      validator.New(),
	}
}

// =====================================================================
// PLUGIN LISTING
// =====================================================================

// ListPlugins handles GET /api/admin/data-export/plugins.
// No RC-number needed; the list is global.
func (h *DataExportHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plugins": dataexport.PluginInfos(),
	})
}

// =====================================================================
// CONFIG CRUD
// =====================================================================

// ListConfigs handles GET /api/admin/data-export/configs?rc_number=...
func (h *DataExportHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	configs, err := h.configService.ListConfigs(rcNumber)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	out := make([]shared.DataExportConfigResponse, len(configs))
	for i, c := range configs {
		out[i] = configToResponse(c)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"configs": out})
}

// CreateConfig handles POST /api/admin/data-export/configs?rc_number=...
func (h *DataExportHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	var req shared.DataExportConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrorDE(w, err)
		return
	}
	cfg, err := h.configService.CreateConfig(rcNumber, req)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, configToResponse(*cfg))
}

// GetConfig handles GET /api/admin/data-export/configs/{id}?rc_number=...
func (h *DataExportHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid ID", nil)))
		return
	}
	cfg, err := h.configService.GetConfig(id, rcNumber)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, configToResponse(*cfg))
}

// UpdateConfig handles PUT /api/admin/data-export/configs/{id}?rc_number=...
func (h *DataExportHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid ID", nil)))
		return
	}
	var req shared.DataExportConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrorDE(w, err)
		return
	}
	cfg, err := h.configService.UpdateConfig(id, rcNumber, req)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, configToResponse(*cfg))
}

// DeleteConfig handles DELETE /api/admin/data-export/configs/{id}?rc_number=...
func (h *DataExportHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid ID", nil)))
		return
	}
	if err := h.configService.DeleteConfig(id, rcNumber); err != nil {
		handleDataExportError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PreviewConfig handles POST /api/admin/data-export/configs/preview?rc_number=...
func (h *DataExportHandler) PreviewConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	var req shared.DataExportPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrorDE(w, err)
		return
	}
	preview, err := h.configService.Preview(r.Context(), rcNumber, req.PluginType, req.Config)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

// =====================================================================
// JOB CRUD
// =====================================================================

// TriggerJob handles POST /api/admin/data-export/jobs?rc_number=...
func (h *DataExportHandler) TriggerJob(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	var req shared.DataExportJobTriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeValidationErrorDE(w, err)
		return
	}
	configID, err := uuid.Parse(req.ConfigID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{"configId": "invalid UUID"})))
		return
	}
	appIDs := make([]uuid.UUID, len(req.ApplicationIDs))
	for i, s := range req.ApplicationIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{"applicationIds": "invalid UUID: " + s})))
			return
		}
		appIDs[i] = id
	}

	adminUserID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		adminUserID = claims.Subject
	}

	job, err := h.jobService.TriggerJob(rcNumber, configID, appIDs, adminUserID)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, jobToResponse(*job, false, nil, nil))
}

// GetJob handles GET /api/admin/data-export/jobs/{id}?rc_number=...
func (h *DataExportHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid ID", nil)))
		return
	}
	job, err := h.jobService.GetJob(id, rcNumber)
	if err != nil {
		handleDataExportError(w, err)
		return
	}

	// Resolve has-result by checking the result table (lightweight metadata only).
	hasResult, fileName, fileSize := false, "", 0
	if job.Status == shared.DataExportJobStatusDone {
		// LoadResult bumps downloaded_at, which we don't want for status pings.
		// Use the metadata-only query on the result repo indirectly via JobService.
		fn, fs, exists := h.jobService.GetResultMetadata(id)
		hasResult = exists
		fileName = fn
		fileSize = fs
	}
	var fnPtr *string
	var fsPtr *int
	if hasResult {
		fnPtr = &fileName
		fsPtr = &fileSize
	}
	writeJSON(w, http.StatusOK, jobToResponse(*job, hasResult, fnPtr, fsPtr))
}

// ListJobs handles GET /api/admin/data-export/jobs?rc_number=...&status=...&since=...&until=...&cursor=...&limit=...
func (h *DataExportHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	since := parseTimeParam(q.Get("since"))
	until := parseTimeParam(q.Get("until"))
	cursor := parseTimeParam(q.Get("cursor"))
	limit := 50
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	jobs, err := h.jobService.ListJobs(rcNumber, status, since, until, cursor, limit)
	if err != nil {
		handleDataExportError(w, err)
		return
	}

	// For each job in done-status, attach result metadata.
	out := make([]shared.DataExportJobResponse, len(jobs))
	for i, j := range jobs {
		hasResult, fileName, fileSize := false, "", 0
		if j.Status == shared.DataExportJobStatusDone {
			fn, fs, exists := h.jobService.GetResultMetadata(j.ID)
			hasResult = exists
			fileName = fn
			fileSize = fs
		}
		var fnPtr *string
		var fsPtr *int
		if hasResult {
			fnPtr = &fileName
			fsPtr = &fileSize
		}
		out[i] = jobToResponse(j, hasResult, fnPtr, fsPtr)
	}

	// Also fetch failed-jobs-count for the last 7 days (UI badge).
	failedSince := time.Now().Add(-7 * 24 * time.Hour)
	failedCount, _ := h.jobService.CountFailedSince(rcNumber, failedSince)

	resp := map[string]interface{}{
		"jobs":              out,
		"failedLast7Days":   failedCount,
	}
	if len(jobs) == limit {
		// Provide next cursor.
		resp["nextCursor"] = jobs[len(jobs)-1].CreatedAt.Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, resp)
}

// DownloadResult handles GET /api/admin/data-export/jobs/{id}/download?rc_number=...
// Streams the result BLOB to the client with Content-Disposition.
func (h *DataExportHandler) DownloadResult(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid ID", nil)))
		return
	}
	res, err := h.jobService.LoadResult(id, rcNumber)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	w.Header().Set("Content-Type", res.MimeType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+res.FileName+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(res.FileSize))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res.FileBytes)
}

// RetryJob handles POST /api/admin/data-export/jobs/{id}/retry?rc_number=...
func (h *DataExportHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
	rcNumber, ok := h.parseRCAndCheck(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Invalid ID", nil)))
		return
	}
	adminUserID := ""
	if claims := ClaimsFromContext(r.Context()); claims != nil {
		adminUserID = claims.Subject
	}
	job, err := h.jobService.Retry(id, rcNumber, adminUserID)
	if err != nil {
		handleDataExportError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, jobToResponse(*job, false, nil, nil))
}

// =====================================================================
// Helpers
// =====================================================================

func (h *DataExportHandler) parseRCAndCheck(w http.ResponseWriter, r *http.Request) (string, bool) {
	rcNumber := r.URL.Query().Get("rc_number")
	if rcNumber == "" {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{"rc_number": "required"})))
		return "", false
	}
	claims := ClaimsFromContext(r.Context())
	if claims != nil && !claims.IsSuperuser() && !containsRC(claims.Tenant, rcNumber) {
		writeJSON(w, http.StatusForbidden, shared.NewErrorResponse(shared.ErrForbidden))
		return "", false
	}
	return rcNumber, true
}

func handleDataExportError(w http.ResponseWriter, err error) {
	if errors.Is(err, shared.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, shared.NewErrorResponse(err))
		return
	}
	if errors.Is(err, shared.ErrForbidden) {
		writeJSON(w, http.StatusForbidden, shared.NewErrorResponse(err))
		return
	}
	var ve shared.ValidationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusBadRequest, shared.NewErrorResponse(ve))
		return
	}
	var ce shared.ConflictError
	if errors.As(err, &ce) {
		writeJSON(w, http.StatusConflict, shared.NewErrorResponse(ce))
		return
	}
	writeJSON(w, http.StatusInternalServerError, shared.NewErrorResponse(err))
}

func parseTimeParam(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return nil
	}
	return &t
}

func configToResponse(c shared.DataExportConfig) shared.DataExportConfigResponse {
	var configMap map[string]interface{}
	if len(c.Config) > 0 {
		_ = json.Unmarshal(c.Config, &configMap)
	}
	return shared.DataExportConfigResponse{
		ID:         c.ID.String(),
		RCNumber:   c.RCNumber,
		PluginType: c.PluginType,
		Name:       c.Name,
		Config:     configMap,
		IsObsolete: c.IsObsolete,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.Format(time.RFC3339),
	}
}

func jobToResponse(j shared.DataExportJob, hasResult bool, fileName *string, fileSize *int) shared.DataExportJobResponse {
	resp := shared.DataExportJobResponse{
		ID:             j.ID.String(),
		RCNumber:       j.RCNumber,
		PluginType:     j.PluginType,
		Status:         string(j.Status),
		AdminUserID:    j.AdminUserID,
		ProcessedCount: j.ProcessedCount,
		TotalCount:     j.TotalCount,
		ErrorMessage:   j.ErrorMessage,
		RetryCount:     j.RetryCount,
		HasResult:      hasResult,
		ResultFileName: fileName,
		ResultFileSize: fileSize,
		CreatedAt:      j.CreatedAt.Format(time.RFC3339),
	}
	if j.ConfigID != nil {
		s := j.ConfigID.String()
		resp.ConfigID = &s
	}
	if j.StartedAt != nil {
		s := j.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if j.FinishedAt != nil {
		s := j.FinishedAt.Format(time.RFC3339)
		resp.FinishedAt = &s
	}
	if len(j.ResultSummary) > 0 {
		var sum map[string]interface{}
		if err := json.Unmarshal(j.ResultSummary, &sum); err == nil {
			resp.ResultSummary = sum
		}
	}
	return resp
}
