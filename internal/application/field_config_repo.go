package application

import (
	"database/sql"
	"fmt"
)

// knownConfigurableFields maps field name → default state.
// "optional" = shown but not required by default.
// "hidden"   = not shown by default (admin must enable).
var knownConfigurableFields = map[string]string{
	"phone":                    "optional",
	"birth_date":               "optional",
	"uid_number":               "optional",
	"membership_start_date":    "hidden",
	"persons_in_household":     "hidden",
	"consumption_previous_year": "hidden",
	"consumption_forecast":     "hidden",
	"feed_in_forecast":         "hidden",
	"pv_power_kwp":             "hidden",
	"heat_pump":                "hidden",
	"electric_vehicle":         "hidden",
	"electric_hot_water":       "hidden",
	"transformer":              "hidden",
	"installation_number":      "hidden",
	"installation_name":        "hidden",
}

var validFieldStates = map[string]bool{
	"hidden":   true,
	"optional": true,
	"required": true,
}

// effectiveState resolves the configured state for a field.
// Falls back to the registered default, then "hidden".
func effectiveState(fieldConfig map[string]string, fieldName string) string {
	if state, ok := fieldConfig[fieldName]; ok {
		return state
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

// Get returns all stored field_config entries for an RC number as map[fieldName]state.
// Missing entries are not included; callers should use effectiveState for defaults.
func (r *FieldConfigRepository) Get(rcNumber string) (map[string]string, error) {
	rows, err := r.db.Query(
		`SELECT field_name, state FROM member_onboarding.field_config WHERE rc_number = $1`,
		rcNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load field config: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var name, state string
		if err := rows.Scan(&name, &state); err != nil {
			return nil, fmt.Errorf("failed to scan field config row: %w", err)
		}
		result[name] = state
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating field config rows: %w", err)
	}
	return result, nil
}

// Save replaces all field_config entries for an RC number atomically.
// Only entries with valid field names and states are written; unknown entries
// are silently skipped.
func (r *FieldConfigRepository) Save(rcNumber string, config map[string]string) error {
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
		INSERT INTO member_onboarding.field_config (rc_number, field_name, state, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (rc_number, field_name) DO UPDATE SET state = EXCLUDED.state, updated_at = NOW()`)
	if err != nil {
		return fmt.Errorf("failed to prepare field config insert: %w", err)
	}
	defer stmt.Close()

	for name, state := range config {
		if _, ok := knownConfigurableFields[name]; !ok {
			continue
		}
		if !validFieldStates[state] {
			continue
		}
		if _, err := stmt.Exec(rcNumber, name, state); err != nil {
			return fmt.Errorf("failed to insert field config entry: %w", err)
		}
	}

	return tx.Commit()
}
