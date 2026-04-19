package application

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationRepository handles database operations for applications
type ApplicationRepository struct {
	db *sql.DB
}

// NewApplicationRepository creates a new application repository
func NewApplicationRepository(db *sql.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

// Create creates a new application
func (r *ApplicationRepository) Create(app *shared.Application) error {
	query := `
		INSERT INTO member_onboarding.application (
			reference_number, eeg_id, rc_number, status, started_at,
			firstname, lastname, birth_date, email, phone,
			resident_street, resident_street_number, resident_zip, resident_city, resident_country,
			privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed, communication_consent,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20,
			$21, $22
		) RETURNING id`

	now := app.CreatedAt
	args := []interface{}{
		app.ReferenceNumber, app.EEGID, app.RCNumber, app.Status, app.StartedAt,
		app.Firstname, app.Lastname, app.BirthDate, app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity, app.ResidentCountry,
		app.PrivacyAccepted, app.PrivacyVersion, &now, app.AccuracyConfirmed, app.CommunicationConsent,
		app.CreatedAt, app.UpdatedAt,
	}

	err := r.db.QueryRow(query, args...).Scan(&app.ID)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	return nil
}

// CreateTx inserts a new application using an existing transaction.
func (r *ApplicationRepository) CreateTx(tx *sql.Tx, app *shared.Application) error {
	query := `
		INSERT INTO member_onboarding.application (
			reference_number, eeg_id, rc_number, status, started_at,
			firstname, lastname, birth_date, email, phone,
			resident_street, resident_street_number, resident_zip, resident_city, resident_country,
			privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed, communication_consent,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20,
			$21, $22
		) RETURNING id`

	now := app.CreatedAt
	args := []interface{}{
		app.ReferenceNumber, app.EEGID, app.RCNumber, app.Status, app.StartedAt,
		app.Firstname, app.Lastname, app.BirthDate, app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity, app.ResidentCountry,
		app.PrivacyAccepted, app.PrivacyVersion, &now, app.AccuracyConfirmed, app.CommunicationConsent,
		app.CreatedAt, app.UpdatedAt,
	}

	err := tx.QueryRow(query, args...).Scan(&app.ID)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	return nil
}

// GetByID gets an application by ID
func (r *ApplicationRepository) GetByID(id uuid.UUID) (*shared.Application, error) {
	query := `
		SELECT id, reference_number, eeg_id, rc_number, status, started_at, submitted_at,
		       approved_at, rejected_at, imported_at, firstname, lastname, birth_date, email, phone,
		       resident_street, resident_street_number, resident_zip, resident_city, resident_country,
		       privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed, communication_consent,
		       reviewed_by_user_id, admin_note, needs_info_reason, target_participant_id,
		       import_started_at, import_finished_at, import_error_message, created_at, updated_at
		FROM member_onboarding.application
		WHERE id = $1`

	app := &shared.Application{}
	var eegID, phone, privacyVersion, reviewedByUserID, adminNote, needsInfoReason, targetParticipantID, importErrorMessage sql.NullString
	var birthDate, startedAt, submittedAt, approvedAt, rejectedAt, importedAt, privacyAcceptedAt, importStartedAt, importFinishedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&app.ID, &app.ReferenceNumber, &eegID, &app.RCNumber, &app.Status, &startedAt,
		&submittedAt, &approvedAt, &rejectedAt, &importedAt, &app.Firstname, &app.Lastname, &birthDate,
		&app.Email, &phone, &app.ResidentStreet, &app.ResidentStreetNumber, &app.ResidentZip,
		&app.ResidentCity, &app.ResidentCountry, &app.PrivacyAccepted, &privacyVersion,
		&privacyAcceptedAt, &app.AccuracyConfirmed, &app.CommunicationConsent, &reviewedByUserID,
		&adminNote, &needsInfoReason, &targetParticipantID, &importStartedAt, &importFinishedAt,
		&importErrorMessage, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// Handle nullable fields
	if eegID.Valid {
		app.EEGID = &eegID.String
	}
	if phone.Valid {
		app.Phone = &phone.String
	}
	if privacyVersion.Valid {
		app.PrivacyVersion = &privacyVersion.String
	}
	if reviewedByUserID.Valid {
		app.ReviewedByUserID = &reviewedByUserID.String
	}
	if adminNote.Valid {
		app.AdminNote = &adminNote.String
	}
	if needsInfoReason.Valid {
		app.NeedsInfoReason = &needsInfoReason.String
	}
	if targetParticipantID.Valid {
		app.TargetParticipantID = &targetParticipantID.String
	}
	if importErrorMessage.Valid {
		app.ImportErrorMessage = &importErrorMessage.String
	}
	if birthDate.Valid {
		app.BirthDate = &birthDate.Time
	}
	if startedAt.Valid {
		app.StartedAt = &startedAt.Time
	}
	if submittedAt.Valid {
		app.SubmittedAt = &submittedAt.Time
	}
	if approvedAt.Valid {
		app.ApprovedAt = &approvedAt.Time
	}
	if rejectedAt.Valid {
		app.RejectedAt = &rejectedAt.Time
	}
	if importedAt.Valid {
		app.ImportedAt = &importedAt.Time
	}
	if privacyAcceptedAt.Valid {
		app.PrivacyAcceptedAt = &privacyAcceptedAt.Time
	}
	if importStartedAt.Valid {
		app.ImportStartedAt = &importStartedAt.Time
	}
	if importFinishedAt.Valid {
		app.ImportFinishedAt = &importFinishedAt.Time
	}

	return app, nil
}

// Update updates an application
func (r *ApplicationRepository) Update(app *shared.Application) error {
	query := `
		UPDATE member_onboarding.application SET
			firstname = $1, lastname = $2, birth_date = $3, email = $4, phone = $5,
			resident_street = $6, resident_street_number = $7, resident_zip = $8,
			resident_city = $9, resident_country = $10, privacy_accepted = $11,
			privacy_version = $12, accuracy_confirmed = $13, communication_consent = $14,
			updated_at = NOW()
		WHERE id = $15`

	_, err := r.db.Exec(query,
		app.Firstname, app.Lastname, app.BirthDate, app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity, app.ResidentCountry,
		app.PrivacyAccepted, app.PrivacyVersion, app.AccuracyConfirmed, app.CommunicationConsent,
		app.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	return nil
}

// UpdateTx updates an application using an existing transaction.
func (r *ApplicationRepository) UpdateTx(tx *sql.Tx, app *shared.Application) error {
	query := `
		UPDATE member_onboarding.application SET
			firstname = $1, lastname = $2, birth_date = $3, email = $4, phone = $5,
			resident_street = $6, resident_street_number = $7, resident_zip = $8,
			resident_city = $9, resident_country = $10, privacy_accepted = $11,
			privacy_version = $12, accuracy_confirmed = $13, communication_consent = $14,
			updated_at = NOW()
		WHERE id = $15`

	_, err := tx.Exec(query,
		app.Firstname, app.Lastname, app.BirthDate, app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity, app.ResidentCountry,
		app.PrivacyAccepted, app.PrivacyVersion, app.AccuracyConfirmed, app.CommunicationConsent,
		app.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of an application
func (r *ApplicationRepository) UpdateStatus(id uuid.UUID, status shared.ApplicationStatus, submittedAt *time.Time) error {
	query := `
		UPDATE member_onboarding.application SET
			status = $1, submitted_at = $2, updated_at = NOW()
		WHERE id = $3`

	_, err := r.db.Exec(query, status, submittedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update application status: %w", err)
	}

	return nil
}

