// Package excel implements the data-export plugin that produces XLSX or CSV
// files from selected applications. Registered with the dataexport.Registry
// via init() — pulled in via side-effect import in cmd/server/main.go.
package excel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/your-org/eegfaktura-member-onboarding/internal/dataexport"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

const (
	PluginType  = "excel"
	DisplayName = "Excel/CSV-Export"

	FormatXLSX = "xlsx"
	FormatCSV  = "csv"

	maxColumns = shared.DataExportMaxColumnsPerExcel
)

// init registers the Excel plugin globally.
func init() {
	dataexport.Register(&Plugin{})
}

// Plugin implements the dataexport.Plugin interface for Excel/CSV exports.
type Plugin struct{}

// Type returns the persistent plugin identifier.
func (p *Plugin) Type() string { return PluginType }

// DisplayName is the admin-UI label.
func (p *Plugin) DisplayName() string { return DisplayName }

// excelConfig is the typed view of a plugin config (decoded from the JSONB).
type excelConfig struct {
	Format  string         `json:"format"`
	Columns []columnConfig `json:"columns"`
}

type columnConfig struct {
	Header string `json:"header"`
	Field  string `json:"field"`
	Format string `json:"format"`
}

// ValidateConfig checks that the config is well-formed: format is one of
// xlsx/csv, columns has between 1 and maxColumns entries, each column has
// a non-empty header + a known field + a valid format for that field type.
func (p *Plugin) ValidateConfig(raw map[string]interface{}) error {
	cfg, err := decodeConfig(raw)
	if err != nil {
		return shared.NewValidationError("Validation failed", map[string]string{
			"config": "ungültiges Format: " + err.Error(),
		})
	}

	fields := map[string]string{}
	if cfg.Format != FormatXLSX && cfg.Format != FormatCSV {
		fields["format"] = "muss 'xlsx' oder 'csv' sein"
	}
	if len(cfg.Columns) == 0 {
		fields["columns"] = "mindestens eine Spalte erforderlich"
	}
	if len(cfg.Columns) > maxColumns {
		fields["columns"] = fmt.Sprintf("maximal %d Spalten erlaubt", maxColumns)
	}

	seenHeaders := map[string]bool{}
	for i, col := range cfg.Columns {
		key := fmt.Sprintf("columns[%d]", i)
		if col.Header == "" {
			fields[key+".header"] = "Header darf nicht leer sein"
		}
		if len(col.Header) > 200 {
			fields[key+".header"] = "Header zu lang (max 200 Zeichen)"
		}
		if seenHeaders[col.Header] {
			fields[key+".header"] = "doppelter Header: " + col.Header
		}
		seenHeaders[col.Header] = true

		fieldDef, ok := AvailableFields[col.Field]
		if !ok {
			fields[key+".field"] = "unbekanntes Feld: " + col.Field
			continue
		}
		if !fieldDef.SupportsFormat(col.Format) {
			fields[key+".format"] = fmt.Sprintf("Format %q nicht zulässig für Feld %q", col.Format, col.Field)
		}
	}

	if len(fields) > 0 {
		return shared.NewValidationError("Validation failed", fields)
	}
	return nil
}

// Process generates the actual XLSX/CSV file.
func (p *Plugin) Process(
	ctx context.Context,
	configSnapshot map[string]interface{},
	apps []dataexport.ApplicationSnapshot,
	progress dataexport.ProgressFn,
) (dataexport.Result, error) {
	cfg, err := decodeConfig(configSnapshot)
	if err != nil {
		return nil, fmt.Errorf("decode config snapshot: %w", err)
	}

	// Generate file based on format.
	var bytes []byte
	var fileName string
	var mimeType string

	if cfg.Format == FormatCSV {
		bytes, err = renderCSV(cfg, apps, progress)
		mimeType = "text/csv; charset=utf-8"
		fileName = fmt.Sprintf("export-%d.csv", len(apps))
	} else {
		bytes, err = renderXLSX(cfg, apps, progress)
		mimeType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		fileName = fmt.Sprintf("export-%d.xlsx", len(apps))
	}
	if err != nil {
		return nil, err
	}

	return dataexport.DownloadResult{
		FileName:  fileName,
		MimeType:  mimeType,
		Bytes:     bytes,
		ItemCount: len(apps),
	}, nil
}

// decodeConfig is a JSON round-trip from map → typed struct, used by
// both ValidateConfig and Process to keep the parse logic in one place.
func decodeConfig(raw map[string]interface{}) (excelConfig, error) {
	var cfg excelConfig
	b, err := json.Marshal(raw)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
