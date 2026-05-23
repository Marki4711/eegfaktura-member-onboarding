package excel

import (
	"github.com/your-org/eegfaktura-member-onboarding/internal/dataexport"
)

// StandardConfigs returns 3 read-only templates that admins clone via
// the "Aus Vorlage erstellen"-button: Newsletter, CRM-Stammdaten,
// Buchhaltungs-Export.
func (p *Plugin) StandardConfigs() []dataexport.StandardConfig {
	return []dataexport.StandardConfig{
		{
			Name: "Newsletter-Adressliste",
			Config: map[string]interface{}{
				"format": FormatXLSX,
				"columns": []map[string]interface{}{
					{"header": "Anrede", "field": "titel", "format": "string"},
					{"header": "Vorname", "field": "firstname", "format": "string"},
					{"header": "Nachname", "field": "lastname", "format": "string"},
					{"header": "Firma", "field": "company_name", "format": "string"},
					{"header": "E-Mail", "field": "email", "format": "string"},
				},
			},
		},
		{
			Name: "CRM-Stammdaten",
			Config: map[string]interface{}{
				"format": FormatXLSX,
				"columns": []map[string]interface{}{
					{"header": "Mitgliedstyp", "field": "member_type", "format": "enum_label"},
					{"header": "Vorname", "field": "firstname", "format": "string"},
					{"header": "Nachname", "field": "lastname", "format": "string"},
					{"header": "Firma", "field": "company_name", "format": "string"},
					{"header": "E-Mail", "field": "email", "format": "string"},
					{"header": "Telefon", "field": "phone", "format": "string"},
					{"header": "Straße", "field": "resident_street", "format": "string"},
					{"header": "Hausnummer", "field": "resident_street_number", "format": "string"},
					{"header": "PLZ", "field": "resident_zip", "format": "string"},
					{"header": "Ort", "field": "resident_city", "format": "string"},
					{"header": "Mitgliedsnummer", "field": "member_number", "format": "string"},
					{"header": "Beitrittsdatum", "field": "membership_start_date", "format": "date_dmy"},
				},
			},
		},
		{
			Name: "Buchhaltungs-Export",
			Config: map[string]interface{}{
				"format": FormatXLSX,
				"columns": []map[string]interface{}{
					{"header": "Mitgliedsnummer", "field": "member_number", "format": "string"},
					{"header": "Vorname", "field": "firstname", "format": "string"},
					{"header": "Nachname", "field": "lastname", "format": "string"},
					{"header": "Firma", "field": "company_name", "format": "string"},
					{"header": "Rechnungs-Straße", "field": "resident_street", "format": "string"},
					{"header": "Rechnungs-Hausnummer", "field": "resident_street_number", "format": "string"},
					{"header": "Rechnungs-PLZ", "field": "resident_zip", "format": "string"},
					{"header": "Rechnungs-Ort", "field": "resident_city", "format": "string"},
					{"header": "IBAN", "field": "iban", "format": "string"},
					{"header": "UID-Nummer", "field": "uid_number", "format": "string"},
					{"header": "Einzugsart", "field": "einzugsart", "format": "enum_label"},
				},
			},
		},
	}
}

// PreviewSample returns an anonymised sample data set for the live preview
// when the EEG has no real members yet. Currently returns nil — the
// service layer falls back to "no preview available, use real data".
func (p *Plugin) PreviewSample() []dataexport.ApplicationSnapshot {
	return nil
}

// BuildPreviewTable implements dataexport.PreviewBuilder for live structured
// preview in the admin UI. Same field-extraction + format pipeline as the
// renderer, but emits []map[header]→value instead of file bytes.
func (p *Plugin) BuildPreviewTable(config map[string]interface{}, apps []dataexport.ApplicationSnapshot) ([]string, []map[string]interface{}, error) {
	cfg, err := decodeConfig(config)
	if err != nil {
		return nil, nil, err
	}
	headers := make([]string, len(cfg.Columns))
	for i, col := range cfg.Columns {
		headers[i] = col.Header
	}
	rows := make([]map[string]interface{}, len(apps))
	for i, app := range apps {
		row := make(map[string]interface{}, len(cfg.Columns))
		for _, col := range cfg.Columns {
			row[col.Header] = extractAndFormat(col, app)
		}
		rows[i] = row
	}
	return headers, rows, nil
}
