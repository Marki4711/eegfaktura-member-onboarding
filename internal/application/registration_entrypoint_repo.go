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

// GetByRCNumber fetches the entrypoint for the given RC number.
// Returns shared.ErrNotFound when no row matches.
func (r *RegistrationEntrypointRepository) GetByRCNumber(rcNumber string) (*shared.RegistrationEntrypoint, error) {
	query := `
		SELECT id, eeg_id, rc_number, is_active, created_at, updated_at
		FROM member_onboarding.registration_entrypoint
		WHERE rc_number = $1`

	ep := &shared.RegistrationEntrypoint{}
	err := r.db.QueryRow(query, rcNumber).Scan(
		&ep.ID, &ep.EEGID, &ep.RCNumber, &ep.IsActive, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get registration entrypoint: %w", err)
	}
	return ep, nil
}
