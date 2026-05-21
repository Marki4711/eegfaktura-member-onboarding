package application

import (
	"database/sql"
	"fmt"
)

// knownConfigurableFields maps field name → default state.
var knownConfigurableFields = map[string]string{
	"phone":                    "optional",
	"birth_date":               "optional",
	// bank_name: bisher fix in der Bankverbindungs-Section. Seit
	// dieser Erweiterung pro EEG konfigurierbar. Default "optional"
	// bewahrt das heutige Verhalten (immer sichtbar, nicht erzwungen).
	"bank_name":                "optional",
	"membership_start_date":    "hidden",
	"persons_in_household":     "hidden",
	"heat_pump":                "hidden",
	"electric_vehicle":         "hidden",
	"electric_vehicle_count":   "hidden",
	"electric_vehicle_annual_km": "hidden",
	"electric_hot_water":       "hidden",
	"transformer":              "hidden",
	"installation_number":      "hidden",
	"installation_name":        "hidden",
	// Teilnahmefaktor (Metering-Point-Scope). Default `optional` bewahrt
	// das heutige Verhalten (Feld sichtbar im Formular). Wenn EEG es auf
	// `hidden` oder `admin_only` stellt, wird der Wert serverseitig auf
	// 100 % defaulted; PDF/Mail/Excel zeigen das Feld unverändert in
	// allen Modi.
	"participation_factor":     "optional",
	// PROJ-44: Netzbetreiber-Vollmacht (Application-Scope).
	"network_operator_authorization": "hidden",
	// PROJ-56: Netzbetreiber-Info-Felder (Application-Scope). Im Public-
	// Formular nur sichtbar, wenn (a) hier nicht hidden UND (b) die
	// Vollmacht-Checkbox aktiv ist. Service-Layer cleart sonst auf NULL.
	"network_operator_customer_number": "hidden",
	"meter_inventory_number":           "hidden",
	// PROJ-57: Ansprechperson-Block (Toggle + Name-Pflicht) für die Org-
	// Mitgliedstypen company/association/municipality. Master-Switch:
	//   - hidden: ganzer Block (Checkbox + alle Felder) verschwindet
	//   - optional/required: Block sichtbar; Toggle aktiviert Name-Pflicht
	"contact_person": "hidden",
	// Feinere Steuerung der Email/Telefon-Pflicht pro EEG. Greifen nur,
	// wenn contact_person != hidden UND HasContactPerson=true.
	// Default required = bisheriges Verhalten (alle 3 Felder Pflicht).
	"contact_person_email": "required",
	"contact_person_phone": "required",
	// PROJ-58: Abweichende Rechnungs-E-Mail. Per-EEG steuerbar.
	// Default hidden — neue EEGs müssen explizit aktivieren.
	"billing_email": "hidden",
	// PROJ-45: Batterie + Wechselrichter (Metering-Point-Scope, nur bei
	// generation_type='pv' rendern — Service cleart sonst).
	"battery_size_kwh":      "hidden",
	"inverter_manufacturer": "hidden",
	// PROJ-49: Energie-Felder pro Zählpunkt. CONSUMPTION-only:
	//   consumption_previous_year, consumption_forecast
	// PRODUCTION-only (alle direction-gegated im Service-Layer):
	//   feed_in_forecast (alle Erzeugungsformen)
	//   pv_power_kwp, feed_in_limit_kw (nur generation_type='pv')
	"consumption_previous_year": "hidden",
	"consumption_forecast":      "hidden",
	"feed_in_forecast":          "hidden",
	"pv_power_kwp":              "hidden",
	"feed_in_limit_kw":          "hidden",
	"inverter_power_kw":         "hidden",
	// PROJ-49 follow-up: Mitglied-Einverständnis „Speichersteuerung im
	// Sinne der EEG vorstellbar?". Nur sinnvoll, wenn Mitglied
	// Batterie-Parameter angegeben hat — Service cleart sonst.
	"battery_control_acceptable": "hidden",
}

var validFieldStates = map[string]bool{
	"hidden":     true,
	"optional":   true,
	"required":   true,
	"admin_only": true,
}

// FieldConfigEntry holds the state and optional admin-provided default value for a field.
type FieldConfigEntry struct {
	State      string
	AdminValue *string
}

// effectiveState resolves the configured state for a field.
// Falls back to the registered default, then "hidden".
func effectiveState(fieldConfig map[string]FieldConfigEntry, fieldName string) string {
	if entry, ok := fieldConfig[fieldName]; ok {
		return entry.State
	}
	if def, ok := knownConfigurableFields[fieldName]; ok {
		return def
	}
	return "hidden"
}

// FieldConfigRepository handles database operations for field configuration.
type FieldConfigRepository struct {
	db *sql.DB
}

// NewFieldConfigRepository creates a new FieldConfigRepository.
func NewFieldConfigRepository(db *sql.DB) *FieldConfigRepository {
	return &FieldConfigRepository{db: db}
}

// Get returns all stored field_config entries for an RC number as map[fieldName]FieldConfigEntry.
func (r *FieldConfigRepository) Get(rcNumber string) (map[string]FieldConfigEntry, error) {
	rows, err := r.db.Query(
		`SELECT field_name, state, admin_value FROM member_onboarding.field_config WHERE rc_number = $1`,
		rcNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load field config: %w", err)
	}
	defer rows.Close()

	result := make(map[string]FieldConfigEntry)
	for rows.Next() {
		var name, state string
		var adminValue sql.NullString
		if err := rows.Scan(&name, &state, &adminValue); err != nil {
			return nil, fmt.Errorf("failed to scan field config row: %w", err)
		}
		entry := FieldConfigEntry{State: state}
		if adminValue.Valid {
			v := adminValue.String
			entry.AdminValue = &v
		}
		result[name] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating field config rows: %w", err)
	}
	return result, nil
}

// Save replaces all field_config entries for an RC number atomically.
// Only entries with valid field names and states are written; unknown entries are silently skipped.
func (r *FieldConfigRepository) Save(rcNumber string, config map[string]FieldConfigEntry) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`DELETE FROM member_onboarding.field_config WHERE rc_number = $1`, rcNumber,
	); err != nil {
		return fmt.Errorf("failed to delete existing field config: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO member_onboarding.field_config (rc_number, field_name, state, admin_value, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (rc_number, field_name) DO UPDATE
		  SET state = EXCLUDED.state, admin_value = EXCLUDED.admin_value, updated_at = NOW()`)
	if err != nil {
		return fmt.Errorf("failed to prepare field config insert: %w", err)
	}
	defer stmt.Close()

	for name, entry := range config {
		if _, ok := knownConfigurableFields[name]; !ok {
			continue
		}
		if !validFieldStates[entry.State] {
			continue
		}
		var adminVal interface{}
		if entry.AdminValue != nil {
			adminVal = *entry.AdminValue
		}
		if _, err := stmt.Exec(rcNumber, name, entry.State, adminVal); err != nil {
			return fmt.Errorf("failed to insert field config entry: %w", err)
		}
	}

	return tx.Commit()
}
