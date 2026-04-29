package http

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health returns the service health status.
//
// @Summary      Health check
// @Description  Returns `{"status":"ok"}` when the service and database are reachable. Returns 503 when the database is unavailable.
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]string  "status: ok"
// @Failure      503  {object}  map[string]string  "status: degraded"
// @Router       /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := h.db.PingContext(r.Context()); err != nil {
		slog.Error("health check: db ping failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "degraded", "error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
