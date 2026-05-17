package application

import (
	"database/sql"
	"fmt"
)

// knownConfigurableFields maps field name → default state.
var knownConfigurableFields = map[string]string{
	"phone":                    "optional",
	"birth_date":               "optional",
	"membership_start_date":    "hidden",
	"persons_in_household":     "hidden",
	"consumption_previous_year": "hidden",
	"consumption_forecast":     "hidden",
	"feed_in_forecast":         "hidden",
	"pv_power_kwp":             "hidden",
	"heat_pump":                "hidden",
	"electric_vehicle":         "hidden",
	"electric_vehicle_count":   "hidden",
	"electric_vehicle_annual_km": "hidden",
	"electric_hot_water":       "hidden",
	"transformer":              "hidden",
	"installation_number":      "hidden",
	"installation_name":        "hidden",
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
