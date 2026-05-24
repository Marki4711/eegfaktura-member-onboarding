// Package dataexport implements the PROJ-60 plugin framework for forwarding
// member data to external systems. Phase 1 ships the Excel/CSV plugin;
// Phase 2 will add CRM plugins (Zoho, HubSpot, etc.) on the same framework
// without refactoring.
package dataexport

import (
	"context"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// Plugin is the interface every data-export adapter must implement.
//
// Registration happens via Go side-effect imports in cmd/server/main.go,
// e.g. `import _ "...//dataexport/excel"` — the package's init() registers
// itself with the global Registry. Pattern follows sql.Driver.
type Plugin interface {
	// Type returns the persistent identifier (stored in config.plugin_type
	// and job.plugin_type). Must be stable across releases.
	Type() string

	// DisplayName is the human-readable name shown in the admin UI.
	DisplayName() string

	// ValidateConfig checks a plugin-specific config payload (JSON-decoded
	// into a generic map). Called when a config is created or updated.
	// Must return a *shared.ValidationError for user-facing input errors
	// and a plain error only for unexpected failures.
	ValidateConfig(config map[string]interface{}) error

	// Process executes the plugin against the given applications using
	// the configSnapshot (NOT the live config — snapshot was taken at
	// job creation to immunise against concurrent edits).
	//
	// The ProgressFn is called periodically to update job.processed_count.
	// Implementations should call it at least every ~50 items or every
	// ~5 seconds, whichever is more frequent at the chosen granularity.
	Process(ctx context.Context, configSnapshot map[string]interface{}, apps []ApplicationSnapshot, progress ProgressFn) (Result, error)

	// StandardConfigs returns read-only templates that admins can clone
	// when creating a new config. Returned objects are snapshotted into
	// new configs at clone time, so future plugin updates do not affect
	// already-cloned configs.
	StandardConfigs() []StandardConfig

	// PreviewSample returns a small synthesised data set for the admin
	// UI to render a live preview when the EEG has no real members yet.
	// Plugins that do not support preview return nil.
	PreviewSample() []ApplicationSnapshot
}

// ProgressFn is invoked by plugins to report incremental progress.
// processed is the number of applications fully handled so far.
type ProgressFn func(processed int)

// StandardConfig is a built-in template made available in the admin UI.
type StandardConfig struct {
	Name   string
	Config map[string]interface{}
}

// ApplicationSnapshot is the read-only view of an application that
// plugins receive. Contains application data plus its metering points
// and the EEG's master data (Entrypoint) — all already loaded from
// the database. Entrypoint is shared across all snapshots in one job
// (one job = one RC number = one entrypoint row).
type ApplicationSnapshot struct {
	Application    *shared.Application
	MeteringPoints []shared.MeteringPoint
	Entrypoint     *shared.RegistrationEntrypoint
}

// Result is the polymorphic outcome of Plugin.Process. Either a
// DownloadResult (file stream stored as BLOB, fetched later via
// download endpoint) or a SyncResult (per-application push outcome,
// summarised in job.result_summary).
type Result interface {
	// Summary returns a JSON-serialisable map that becomes job.result_summary.
	Summary() map[string]interface{}
}

// DownloadResult is the Result type for download-style plugins (Excel/CSV).
// The Bytes are stored as a BLOB in data_export_result with a TTL.
type DownloadResult struct {
	FileName  string
	MimeType  string
	Bytes     []byte
	SummaryV  map[string]interface{} // optional override; default is {"downloaded": <total>}
	ItemCount int
}

// Summary implements Result.
func (r DownloadResult) Summary() map[string]interface{} {
	if r.SummaryV != nil {
		return r.SummaryV
	}
	return map[string]interface{}{
		"downloaded": r.ItemCount,
		"file_size":  len(r.Bytes),
	}
}

// SyncResult is the Result type for push-style plugins (Zoho, HubSpot, …).
// The per-application status is summarised here; individual diagnostics
// can be embedded in PerAppDetails for the admin to inspect.
type SyncResult struct {
	Synced   []uuid.UUID
	Failed   []uuid.UUID
	Skipped  []uuid.UUID
	Details  map[string]interface{} // optional extra diagnostics
}

// Summary implements Result.
func (r SyncResult) Summary() map[string]interface{} {
	out := map[string]interface{}{
		"synced":  len(r.Synced),
		"failed":  len(r.Failed),
		"skipped": len(r.Skipped),
	}
	if r.Details != nil {
		out["details"] = r.Details
	}
	return out
}
