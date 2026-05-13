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

// Health returns the combined service health status (legacy endpoint).
// Kept for compatibility; new K8s probes should target /livez and /readyz.
//
// @Summary      Health check (legacy combined)
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

// Livez is the K8s livenessProbe target — succeeds as long as the process is
// alive and serving HTTP. Deliberately does NOT touch the DB so a transient
// Postgres blip doesn't cause kubelet to restart the pod (which would extend
// the outage). Use Readyz for "is the service useful right now" semantics.
//
// @Summary      Liveness probe
// @Description  Always returns 200 once the HTTP server is up. Intended for K8s livenessProbe.
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]string  "status: alive"
// @Router       /livez [get]
func (h *HealthHandler) Livez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// Readyz is the K8s readinessProbe target — pings the DB. On failure the pod
// is dropped from the Service endpoints (no restart) so traffic stops while
// the DB recovers. K8s liveness should NOT target this.
//
// @Summary      Readiness probe
// @Description  Returns 200 when the service is ready to serve traffic (DB reachable). 503 otherwise.
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]string  "status: ready"
// @Failure      503  {object}  map[string]string  "status: not_ready"
// @Router       /readyz [get]
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := h.db.PingContext(r.Context()); err != nil {
		slog.Warn("readiness probe: db ping failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "not_ready"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
