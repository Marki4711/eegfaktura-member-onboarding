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
		SELECT id, rc_number, eeg_id, is_active, contact_email, intro_text,
		       eeg_name, eeg_street, eeg_street_number, eeg_zip, eeg_city,
		       creditor_id, sepa_mandate_enabled, use_company_sepa_mandate,
		       created_at, updated_at
		FROM member_onboarding.registration_entrypoint
		WHERE rc_number = $1`

	ep := &shared.RegistrationEntrypoint{}
	err := r.db.QueryRow(query, rcNumber).Scan(
		&ep.ID, &ep.RCNumber, &ep.EegID, &ep.IsActive, &ep.ContactEmail, &ep.IntroText,
		&ep.EEGName, &ep.EEGStreet, &ep.EEGStreetNumber, &ep.EEGZip, &ep.EEGCity,
		&ep.CreditorID, &ep.SEPAMandateEnabled, &ep.UseCompanySEPAMandate,
		&ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get registration entrypoint: %w", err)
	}
	return ep, nil
}

// SaveEEGSettings persists the EEG master data and SEPA mandate toggles for the given RC number.
func (r *RegistrationEntrypointRepository) SaveEEGSettings(
	rcNumber string,
	eegID *string,
	eegName, eegStreet, eegStreetNumber, eegZip, eegCity, creditorID *string,
	sepaMandateEnabled bool,
	useCompanySEPAMandate bool,
) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET eeg_id = $1, eeg_name = $2, eeg_street = $3, eeg_street_number = $4,
		    eeg_zip = $5, eeg_city = $6, creditor_id = $7,
		    sepa_mandate_enabled = $8, use_company_sepa_mandate = $9,
		    updated_at = NOW()
		WHERE rc_number = $10`,
		eegID, eegName, eegStreet, eegStreetNumber, eegZip, eegCity, creditorID,
		sepaMandateEnabled, useCompanySEPAMandate, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save EEG settings for %s: %w", rcNumber, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// SaveIntroText persists the sanitized intro_text for the given RC number.
// Returns shared.ErrNotFound when no row matches.
func (r *RegistrationEntrypointRepository) SaveIntroText(rcNumber string, introText *string) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET intro_text = $1, updated_at = NOW()
		WHERE rc_number = $2`, introText, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save intro text for %s: %w", rcNumber, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return shared.ErrNotFound
	}
	return nil
}
