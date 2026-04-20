package application

import (
	"database/sql"
	"fmt"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// RegistrationEntrypointRepository handles database access for registration_entrypoint.
type RegistrationEntrypointRepository struct {
	db *sql.DB
}

// NewRegistrationEntrypointRepository creates a new RegistrationEntrypointRepository.
func NewRegistrationEntrypointRepository(db *sql.DB) *RegistrationEntrypointRepository {
	return &RegistrationEntrypointRepository{db: db}
}

// UpsertForRCNumbers ensures a registration_entrypoint row exists for each RC number.
// Missing rows are inserted with is_active=true; existing rows are left untouched.
// This is called once per login session for Tenant-Admins.
func (r *RegistrationEntrypointRepository) UpsertForRCNumbers(rcNumbers []string) error {
	for _, rc := range rcNumbers {
		_, err := r.db.Exec(`
			INSERT INTO member_onboarding.registration_entrypoint (rc_number, is_active)
			VALUES ($1, TRUE)
			ON CONFLICT (rc_number) DO NOTHING`, rc)
		if err != nil {
			return fmt.Errorf("failed to upsert registration entrypoint for %s: %w", rc, err)
		}
	}
	return nil
}

// GetByRCNumber fetches the entrypoint for the given RC number.
// Returns shared.ErrNotFound when no row matches.
func (r *RegistrationEntrypointRepository) GetByRCNumber(rcNumber string) (*shared.RegistrationEntrypoint, error) {
	query := `
		SELECT id, rc_number, is_active, contact_email, created_at, updated_at
		FROM member_onboarding.registration_entrypoint
		WHERE rc_number = $1`

	ep := &shared.RegistrationEntrypoint{}
	err := r.db.QueryRow(query, rcNumber).Scan(
		&ep.ID, &ep.RCNumber, &ep.IsActive, &ep.ContactEmail, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get registration entrypoint: %w", err)
	}
	return ep, nil
}
