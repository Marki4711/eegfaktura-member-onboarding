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
// Missing rows are inserted with is_active=false; existing rows are left untouched.
// Activation must be done explicitly by the admin via the settings page.
func (r *RegistrationEntrypointRepository) UpsertForRCNumbers(rcNumbers []string) error {
	for _, rc := range rcNumbers {
		_, err := r.db.Exec(`
			INSERT INTO member_onboarding.registration_entrypoint (rc_number, is_active)
			VALUES ($1, FALSE)
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
		       sepa_mandate_at_import,
		       show_central_policy, member_number_start, require_email_confirmation,
		       metering_point_prefix_consumption, metering_point_prefix_production,
		       last_synced_from_core_at, eeg_logo_synced_at,
		       cooperative_shares_enabled, cooperative_required_shares,
		       cooperative_share_amount_cents,
		       created_at, updated_at
		FROM member_onboarding.registration_entrypoint
		WHERE rc_number = $1`

	ep := &shared.RegistrationEntrypoint{}
	err := r.db.QueryRow(query, rcNumber).Scan(
		&ep.ID, &ep.RCNumber, &ep.EegID, &ep.IsActive, &ep.ContactEmail, &ep.IntroText,
		&ep.EEGName, &ep.EEGStreet, &ep.EEGStreetNumber, &ep.EEGZip, &ep.EEGCity,
		&ep.CreditorID, &ep.SEPAMandateEnabled, &ep.UseCompanySEPAMandate,
		&ep.SEPAMandateAtImport,
		&ep.ShowCentralPolicy, &ep.MemberNumberStart, &ep.RequireEmailConfirmation,
		&ep.MeteringPointPrefixConsumption, &ep.MeteringPointPrefixProduction,
		&ep.LastSyncedFromCoreAt, &ep.EEGLogoSyncedAt,
		&ep.CooperativeSharesEnabled, &ep.CooperativeRequiredShares,
		&ep.CooperativeShareAmountCents,
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

// SaveEEGSettings persists the Onboarding-owned settings for the given RC
// number. Since PROJ-32 the EEG master data (eeg_id / community-id, name,
// address, creditor-id, contact-email) is **not** written here anymore —
// those fields are mastered by the eegFaktura core and only modified via
// SyncFromCore. This function writes the two SEPA toggles plus the three
// PROJ-37 cooperative-shares settings (Onboarding-only).
//
// cooperativeRequiredShares / cooperativeShareAmountCents are written as-is.
// The caller (admin service) is responsible for the cross-field
// validation (enabled ⇒ both non-nil > 0). When the toggle is disabled,
// pass nil for both — the existing values get cleared so a re-enable
// later starts from a clean slate.
func (r *RegistrationEntrypointRepository) SaveEEGSettings(
	rcNumber string,
	sepaMandateEnabled bool,
	useCompanySEPAMandate bool,
	sepaMandateAtImport bool,
	cooperativeSharesEnabled bool,
	cooperativeRequiredShares *int,
	cooperativeShareAmountCents *int64,
) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET sepa_mandate_enabled = $1,
		    use_company_sepa_mandate = $2,
		    sepa_mandate_at_import = $3,
		    cooperative_shares_enabled = $4,
		    cooperative_required_shares = $5,
		    cooperative_share_amount_cents = $6,
		    updated_at = NOW()
		WHERE rc_number = $7`,
		sepaMandateEnabled, useCompanySEPAMandate, sepaMandateAtImport,
		cooperativeSharesEnabled, cooperativeRequiredShares, cooperativeShareAmountCents,
		rcNumber)
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

// SaveIsActive sets the is_active flag for the given RC number.
func (r *RegistrationEntrypointRepository) SaveIsActive(rcNumber string, active bool) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET is_active = $1, updated_at = NOW()
		WHERE rc_number = $2`, active, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save is_active for %s: %w", rcNumber, err)
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

// SaveShowCentralPolicy toggles whether the central privacy policy is shown
// in the public registration form for the given RC number.
func (r *RegistrationEntrypointRepository) SaveShowCentralPolicy(rcNumber string, show bool) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET show_central_policy = $1, updated_at = NOW()
		WHERE rc_number = $2`, show, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save show_central_policy for %s: %w", rcNumber, err)
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

// SyncFromCore overwrites the Core-mastered fields with values pulled from
// the eegFaktura core (PROJ-32). The Onboarding does NOT own these values
// — they are mirrored here so PDF/Mail render code can keep reading the
// registration_entrypoint table unchanged. last_synced_from_core_at is
// stamped with NOW() inside the same UPDATE so the admin UI can show a
// reliable "Stand vom" timestamp.
//
// nil values overwrite the local column with NULL. That is the intended
// behaviour: if the Core has no value (e.g. creditor_id not configured),
// we should reflect that, not retain a stale local value.
type CoreMasterDataUpdate struct {
	EegID           *string
	EEGName         *string
	EEGStreet       *string
	EEGStreetNumber *string
	EEGZip          *string
	EEGCity         *string
	CreditorID      *string
	ContactEmail    *string
}

func (r *RegistrationEntrypointRepository) SyncFromCore(rcNumber string, u CoreMasterDataUpdate) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET eeg_id = $1,
		    eeg_name = $2,
		    eeg_street = $3,
		    eeg_street_number = $4,
		    eeg_zip = $5,
		    eeg_city = $6,
		    creditor_id = $7,
		    contact_email = $8,
		    last_synced_from_core_at = NOW(),
		    updated_at = NOW()
		WHERE rc_number = $9`,
		u.EegID, u.EEGName, u.EEGStreet, u.EEGStreetNumber, u.EEGZip, u.EEGCity,
		u.CreditorID, u.ContactEmail, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to sync from core for %s: %w", rcNumber, err)
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

// SaveLogoFromCore persists the EEG logo bytes + MIME type pulled from the
// eegFaktura-billing service (PROJ-33). Pure overwrite — there is no "merge"
// semantic: every successful sync replaces the previous bytes. Stamps
// eeg_logo_synced_at = NOW() for the admin UI's "Stand vom" indicator.
//
// Called only from the Sync handler after a successful FetchEEGLogo. If the
// caller observed ErrLogoNotFound it should NOT call this method — keep the
// old cached bytes around (the core having lost its logo doesn't mean we
// want to drop ours, because there might be a transient billing-config
// reshuffle on the other side).
func (r *RegistrationEntrypointRepository) SaveLogoFromCore(rcNumber string, logoBytes []byte, mime string) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET eeg_logo_bytes = $1,
		    eeg_logo_mime = $2,
		    eeg_logo_synced_at = NOW(),
		    updated_at = NOW()
		WHERE rc_number = $3`,
		logoBytes, mime, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save eeg logo for %s: %w", rcNumber, err)
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

// GetLogo reads only the logo columns for the given RC number. Separate from
// GetByRCNumber so that the typical settings-page read doesn't pull the
// BYTEA into memory. Returns (nil, "", nil) when the entrypoint exists but
// has no logo synced yet — the caller renders a 404 in that case.
//
// Returns shared.ErrNotFound when the entrypoint row itself doesn't exist.
func (r *RegistrationEntrypointRepository) GetLogo(rcNumber string) ([]byte, string, error) {
	var logoBytes []byte
	var mime *string
	err := r.db.QueryRow(`
		SELECT eeg_logo_bytes, eeg_logo_mime
		FROM member_onboarding.registration_entrypoint
		WHERE rc_number = $1`,
		rcNumber).Scan(&logoBytes, &mime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", shared.ErrNotFound
		}
		return nil, "", fmt.Errorf("failed to get logo for %s: %w", rcNumber, err)
	}
	if mime == nil {
		return nil, "", nil
	}
	return logoBytes, *mime, nil
}

// SaveRequireEmailConfirmation toggles whether new applications for this EEG
// require an e-mail confirmation click before they become reviewable by the
// admin (PROJ-31).
func (r *RegistrationEntrypointRepository) SaveRequireEmailConfirmation(rcNumber string, require bool) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET require_email_confirmation = $1, updated_at = NOW()
		WHERE rc_number = $2`, require, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save require_email_confirmation for %s: %w", rcNumber, err)
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

// SaveMeteringPointPrefixes (PROJ-52) persists the per-direction Zählpunkt
// prefix configuration. Both values are optional — pass nil to clear. The
// caller is responsible for normalisation (whitespace + dots stripped,
// uppercase) and the high-level format check; the DB CHECK constraint
// (^AT[0-9A-Z]{0,31}$) is the final guard.
func (r *RegistrationEntrypointRepository) SaveMeteringPointPrefixes(rcNumber string, consumption, production *string) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET metering_point_prefix_consumption = $1,
		    metering_point_prefix_production  = $2,
		    updated_at = NOW()
		WHERE rc_number = $3`, consumption, production, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save metering-point prefixes for %s: %w", rcNumber, err)
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

// SaveMemberNumberStart persists the per-EEG starting value for member number auto-increment.
func (r *RegistrationEntrypointRepository) SaveMemberNumberStart(rcNumber string, start int) error {
	if start < 1 {
		return fmt.Errorf("member_number_start must be >= 1")
	}
	result, err := r.db.Exec(`
		UPDATE member_onboarding.registration_entrypoint
		SET member_number_start = $1, updated_at = NOW()
		WHERE rc_number = $2`, start, rcNumber)
	if err != nil {
		return fmt.Errorf("failed to save member_number_start for %s: %w", rcNumber, err)
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
