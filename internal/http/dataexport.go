package http

import (
	"encoding/json"
	"errors"
	"log/slog"
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

// ListPlugins lists all registered data-export plugins with their standard templates.
// @Summary      List registered data-export plugins (PROJ-60)
// @Description  Returns all plugins compiled into the backend with their built-in standard templates. Global — no rc_number filter.
// @Tags         data-export
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/plugins [get]
func (h *DataExportHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plugins": dataexport.PluginInfos(),
	})
}

// =====================================================================
// CONFIG CRUD
// =====================================================================

// ListConfigs returns all non-deleted data-export configurations for the EEG.
// @Summary      List data-export configurations (PROJ-60)
// @Tags         data-export
// @Security     BearerAuth
// @Produce      json
// @Param        rc_number  query     string  true  "EEG RC number (must be in the admin's tenant claim)"
// @Success      200        {object}  map[string]interface{}
// @Failure      400        {object}  shared.ErrorResponse
// @Failure      403        {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/configs [get]
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

// CreateConfig creates a new data-export configuration for the EEG.
// @Summary      Create data-export configuration (PROJ-60)
// @Description  Plugin-specific config validated by Plugin.ValidateConfig. Enforced limits: max 20 non-deleted configs per EEG, unique name per EEG across plugin types.
// @Tags         data-export
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        rc_number  query     string                         true  "EEG RC number"
// @Param        request    body      shared.DataExportConfigRequest  true  "Plugin type, name, plugin-specific config"
// @Success      201        {object}  shared.DataExportConfigResponse
// @Failure      400        {object}  shared.ErrorResponse  "validation_error — field-level errors under `fields` (e.g. `columns[1].header`)"
// @Failure      403        {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/configs [post]
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

// GetConfig returns one data-export configuration by ID.
// @Summary      Get data-export configuration (PROJ-60)
// @Tags         data-export
// @Security     BearerAuth
// @Produce      json
// @Param        id         path      string  true  "Config UUID"
// @Param        rc_number  query     string  true  "EEG RC number"
// @Success      200        {object}  shared.DataExportConfigResponse
// @Failure      403        {object}  shared.ErrorResponse
// @Failure      404        {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/configs/{id} [get]
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

// UpdateConfig updates an existing data-export configuration.
// @Summary      Update data-export configuration (PROJ-60)
// @Description  Same shape as create. pluginType cannot be changed. Obsolete configs are read-only.
// @Tags         data-export
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id         path      string                         true  "Config UUID"
// @Param        rc_number  query     string                         true  "EEG RC number"
// @Param        request    body      shared.DataExportConfigRequest  true  "Updated plugin config"
// @Success      200        {object}  shared.DataExportConfigResponse
// @Failure      400        {object}  shared.ErrorResponse
// @Failure      403        {object}  shared.ErrorResponse
// @Failure      404        {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/configs/{id} [put]
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

// DeleteConfig soft-deletes a data-export configuration.
// @Summary      Soft-delete data-export configuration (PROJ-60)
// @Description  Sets deleted_at = NOW. Running jobs keep their snapshot. Hard-delete via cleanup CronJob after 7 years (DSGVO § 132 BAO).
// @Tags         data-export
// @Security     BearerAuth
// @Param        id         path  string  true  "Config UUID"
// @Param        rc_number  query string  true  "EEG RC number"
// @Success      204
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/configs/{id} [delete]
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

// PreviewConfig runs a plugin config against the EEG's latest 5 post-imported members.
// @Summary      Live-preview a data-export config (PROJ-60)
// @Description  Returns headers + rows as JSON (not file bytes) — used by the editor for instant feedback. Falls back to the plugin's synthetic sample when the EEG has no imported members yet.
// @Tags         data-export
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        rc_number  query     string                          true  "EEG RC number"
// @Param        request    body      shared.DataExportPreviewRequest  true  "Plugin type + config to preview"
// @Success      200        {object}  shared.DataExportPreviewResponse
// @Failure      400        {object}  shared.ErrorResponse
// @Failure      403        {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/configs/preview [post]
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

// TriggerJob enqueues a new data-export job for the given config + application IDs.
// @Summary      Trigger data-export job (PROJ-60)
// @Description  Limits: 1..1000 application IDs, max 3 active jobs per EEG (soft, tolerates 4-5 burst). Config snapshot is frozen at trigger time so subsequent config edits don't affect this run. Returns 409 during graceful shutdown.
// @Tags         data-export
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        rc_number  query     string                              true  "EEG RC number"
// @Param        request    body      shared.DataExportJobTriggerRequest  true  "Config ID + application IDs"
// @Success      202        {object}  shared.DataExportJobResponse
// @Failure      400        {object}  shared.ErrorResponse
// @Failure      403        {object}  shared.ErrorResponse
// @Failure      404        {object}  shared.ErrorResponse  "Config not found"
// @Failure      409        {object}  shared.ErrorResponse  "Config obsolete or server shutting down"
// @Router       /api/admin/data-export/jobs [post]
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

// GetJob returns a job's current status — polled by the frontend modal.
// @Summary      Get data-export job status (PROJ-60)
// @Description  Polled every 2-5 seconds while the job is queued/running. errorMessage is always user-safe text (never contains stack traces or DB internals).
// @Tags         data-export
// @Security     BearerAuth
// @Produce      json
// @Param        id         path      string  true  "Job UUID"
// @Param        rc_number  query     string  true  "EEG RC number"
// @Success      200        {object}  shared.DataExportJobResponse
// @Failure      403        {object}  shared.ErrorResponse
// @Failure      404        {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/jobs/{id} [get]
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

// ListJobs returns the BackOffice job overview with optional filter + cursor pagination.
// @Summary      List data-export jobs (BackOffice, PROJ-60)
// @Description  Cursor-based pagination via created_at. Includes failedLast7Days for the red-badge counter.
// @Tags         data-export
// @Security     BearerAuth
// @Produce      json
// @Param        rc_number  query     string  true   "EEG RC number"
// @Param        status     query     string  false  "queued | running | done | failed | expired"
// @Param        since      query     string  false  "Filter created_at >= RFC3339"
// @Param        until      query     string  false  "Filter created_at < RFC3339"
// @Param        cursor     query     string  false  "Pagination cursor (RFC3339Nano of last item)"
// @Param        limit      query     int     false  "Page size (default 50, max 200)"
// @Success      200        {object}  map[string]interface{}
// @Failure      403        {object}  shared.ErrorResponse
// @Router       /api/admin/data-export/jobs [get]
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

	// Batch-load metadata for all done-jobs in one query instead of N
	// (was a per-job lookup; with limit=50 that's 50 extra DB round-trips).
	doneIDs := make([]uuid.UUID, 0, len(jobs))
	for _, j := range jobs {
		if j.Status == shared.DataExportJobStatusDone {
			doneIDs = append(doneIDs, j.ID)
		}
	}
	metaByJob := h.jobService.GetResultMetadataBatch(doneIDs)

	out := make([]shared.DataExportJobResponse, len(jobs))
	for i, j := range jobs {
		hasResult, fileName, fileSize := false, "", 0
		if j.Status == shared.DataExportJobStatusDone {
			if m, ok := metaByJob[j.ID]; ok {
				hasResult = true
				fileName = m.FileName
				fileSize = m.FileSize
			}
		}
		var fnPtr *string
		var fsPtr *int
		if hasResult {
			fnPtr = &fileName
			fsPtr = &fileSize
		}
		out[i] = jobToResponse(j, hasResult, fnPtr, fsPtr)
	}

	// Also fetch failed-jobs-count for the last 7 days (UI badge). Failure
	// just means the red badge stays at 0 — we still log so the operator
	// can spot persistent failures in slog instead of silent zeros.
	failedSince := time.Now().Add(-7 * 24 * time.Hour)
	failedCount, err := h.jobService.CountFailedSince(rcNumber, failedSince)
	if err != nil {
		slog.Warn("dataexport: CountFailedSince failed (badge will render 0)",
			"rc_number", rcNumber, "error", err)
	}

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

// DownloadResult streams the result file (Excel/CSV) for a done job.
// @Summary      Download data-export result (PROJ-60)
// @Description  Filename pattern `{rc_number}-{config_name}-{YYYY-MM-DD}.{xlsx|csv}` with path-traversal sanitization. CSV includes UTF-8 BOM + semicolon (DACH-Excel convention). All cell values whose first non-whitespace char is `=+-@\t\r` are prefixed with `'` to defang CSV/Excel-injection.
// @Tags         data-export
// @Security     BearerAuth
// @Produce      application/octet-stream
// @Param        id         path  string  true  "Job UUID"
// @Param        rc_number  query string  true  "EEG RC number"
// @Success      200  "Binary stream with Content-Disposition: attachment"
// @Failure      403  {object}  shared.ErrorResponse
// @Failure      404  {object}  shared.ErrorResponse  "Job unknown or result expired"
// @Failure      409  {object}  shared.ErrorResponse  "Job not in done status"
// @Router       /api/admin/data-export/jobs/{id}/download [get]
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

// RetryJob creates a new queued job with the same snapshot as the original.
// @Summary      Retry a data-export job (PROJ-60)
// @Description  Original job is unaffected (audit-trail preserved). Returns 409 during graceful shutdown.
// @Tags         data-export
// @Security     BearerAuth
// @Produce      json
// @Param        id         path      string  true  "Original job UUID"
// @Param        rc_number  query     string  true  "EEG RC number"
// @Success      202        {object}  shared.DataExportJobResponse
// @Failure      403        {object}  shared.ErrorResponse
// @Failure      404        {object}  shared.ErrorResponse
// @Failure      409        {object}  shared.ErrorResponse  "Server shutting down"
// @Router       /api/admin/data-export/jobs/{id}/retry [post]
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
