package application

import (
	"database/sql"
	"fmt"
	"strings"
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

// List returns a paginated, filtered list of applications for the admin view.
func (r *ApplicationRepository) List(filters ApplicationListFilters, page, pageSize int) ([]shared.ApplicationListItem, int, error) {
	conditions := []string{}
	args := []interface{}{}
	n := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", n))
		args = append(args, *filters.Status)
		n++
	}
	if filters.EEGID != nil {
		conditions = append(conditions, fmt.Sprintf("a.eeg_id = $%d", n))
		args = append(args, *filters.EEGID)
		n++
	}
	if filters.ReferenceNumber != nil {
		conditions = append(conditions, fmt.Sprintf("a.reference_number ILIKE $%d", n))
		args = append(args, "%"+*filters.ReferenceNumber+"%")
		n++
	}
	if filters.Lastname != nil {
		conditions = append(conditions, fmt.Sprintf("a.lastname ILIKE $%d", n))
		args = append(args, "%"+*filters.Lastname+"%")
		n++
	}
	if filters.Email != nil {
		conditions = append(conditions, fmt.Sprintf("a.email ILIKE $%d", n))
		args = append(args, "%"+*filters.Email+"%")
		n++
	}
	if filters.MeteringPoint != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM member_onboarding.metering_point mp WHERE mp.application_id = a.id AND mp.metering_point ILIKE $%d)", n,
		))
		args = append(args, "%"+*filters.MeteringPoint+"%")
		n++
	}
	if filters.SubmittedFrom != nil {
		conditions = append(conditions, fmt.Sprintf("a.submitted_at >= $%d", n))
		args = append(args, *filters.SubmittedFrom)
		n++
	}
	if filters.SubmittedTo != nil {
		conditions = append(conditions, fmt.Sprintf("a.submitted_at <= $%d", n))
		args = append(args, *filters.SubmittedTo)
		n++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM member_onboarding.application a %s`, where)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count applications: %w", err)
	}

	offset := (page - 1) * pageSize
	listArgs := append(args, pageSize, offset)
	listQuery := fmt.Sprintf(`
		SELECT a.id, a.reference_number, a.eeg_id, a.rc_number, a.status,
		       a.firstname, a.lastname, a.email, a.submitted_at
		FROM member_onboarding.application a
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d`, where, n, n+1)

	rows, err := r.db.Query(listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list applications: %w", err)
	}
	defer rows.Close()

	items := []shared.ApplicationListItem{}
	for rows.Next() {
		var item shared.ApplicationListItem
		var eegID sql.NullString
		var submittedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.ReferenceNumber, &eegID, &item.RCNumber, &item.Status,
			&item.Firstname, &item.Lastname, &item.Email, &submittedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan application list item: %w", err)
		}
		if eegID.Valid {
			item.EEGID = &eegID.String
		}
		if submittedAt.Valid {
			item.SubmittedAt = &submittedAt.Time
		}
		item.MeteringPoints = []string{}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating applications: %w", err)
	}

	return items, total, nil
}

// UpdateAdminTx updates application fields (including admin_note) using an existing transaction.
func (r *ApplicationRepository) UpdateAdminTx(tx *sql.Tx, app *shared.Application) error {
	query := `
		UPDATE member_onboarding.application SET
			firstname = $1, lastname = $2, birth_date = $3, email = $4, phone = $5,
			resident_street = $6, resident_street_number = $7, resident_zip = $8,
			resident_city = $9, resident_country = $10, admin_note = $11,
			updated_at = NOW()
		WHERE id = $12`

	_, err := tx.Exec(query,
		app.Firstname, app.Lastname, app.BirthDate, app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity, app.ResidentCountry,
		app.AdminNote,
		app.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}
	return nil
}

// UpdateStatusAdminTx updates the status and related timestamp columns atomically.
// Columns not applicable to the transition are preserved via COALESCE.
func (r *ApplicationRepository) UpdateStatusAdminTx(
	tx *sql.Tx,
	id uuid.UUID,
	toStatus shared.ApplicationStatus,
	submittedAt, approvedAt, rejectedAt *time.Time,
	needsInfoReason, reviewedByUserID *string,
) error {
	query := `
		UPDATE member_onboarding.application SET
			status              = $1,
			submitted_at        = COALESCE($2, submitted_at),
			approved_at         = COALESCE($3, approved_at),
			rejected_at         = COALESCE($4, rejected_at),
			needs_info_reason   = COALESCE($5, needs_info_reason),
			reviewed_by_user_id = COALESCE($6, reviewed_by_user_id),
			updated_at          = NOW()
		WHERE id = $7`

	_, err := tx.Exec(query, toStatus, submittedAt, approvedAt, rejectedAt, needsInfoReason, reviewedByUserID, id)
	if err != nil {
		return fmt.Errorf("failed to update application status: %w", err)
	}
	return nil
}

