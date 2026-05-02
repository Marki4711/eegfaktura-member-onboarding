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
			reference_number, rc_number, status, started_at,
			member_type, titel, firstname, lastname, birth_date,
			company_name, uid_number, register_number,
			email, phone,
			resident_street, resident_street_number, resident_zip, resident_city,
			privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
			iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
			membership_start_date, persons_in_household, consumption_previous_year,
			consumption_forecast, feed_in_forecast, pv_power_kwp,
			heat_pump, electric_vehicle, electric_hot_water,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12,
			$13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22,
			$23, $24, $25, $26,
			$27, $28, $29,
			$30, $31, $32,
			$33, $34, $35,
			$36, $37
		) RETURNING id`

	now := app.CreatedAt
	args := []interface{}{
		app.ReferenceNumber, app.RCNumber, app.Status, app.StartedAt,
		app.MemberType, app.Titel, app.Firstname, app.Lastname, app.BirthDate,
		app.CompanyName, app.UIDNumber, app.RegisterNumber,
		app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity,
		app.PrivacyAccepted, app.PrivacyVersion, &now, app.AccuracyConfirmed,
		app.IBAN, app.AccountHolder, app.SepaMandateAccepted, app.SepaMandateAcceptedAt,
		app.MembershipStartDate, app.PersonsInHousehold, app.ConsumptionPreviousYear,
		app.ConsumptionForecast, app.FeedInForecast, app.PvPowerKwp,
		app.HeatPump, app.ElectricVehicle, app.ElectricHotWater,
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
			reference_number, rc_number, status, started_at,
			member_type, titel, firstname, lastname, birth_date,
			company_name, uid_number, register_number,
			email, phone,
			resident_street, resident_street_number, resident_zip, resident_city,
			privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
			iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
			membership_start_date, persons_in_household, consumption_previous_year,
			consumption_forecast, feed_in_forecast, pv_power_kwp,
			heat_pump, electric_vehicle, electric_hot_water,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12,
			$13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22,
			$23, $24, $25, $26,
			$27, $28, $29,
			$30, $31, $32,
			$33, $34, $35,
			$36, $37
		) RETURNING id`

	now := app.CreatedAt
	args := []interface{}{
		app.ReferenceNumber, app.RCNumber, app.Status, app.StartedAt,
		app.MemberType, app.Titel, app.Firstname, app.Lastname, app.BirthDate,
		app.CompanyName, app.UIDNumber, app.RegisterNumber,
		app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity,
		app.PrivacyAccepted, app.PrivacyVersion, &now, app.AccuracyConfirmed,
		app.IBAN, app.AccountHolder, app.SepaMandateAccepted, app.SepaMandateAcceptedAt,
		app.MembershipStartDate, app.PersonsInHousehold, app.ConsumptionPreviousYear,
		app.ConsumptionForecast, app.FeedInForecast, app.PvPowerKwp,
		app.HeatPump, app.ElectricVehicle, app.ElectricHotWater,
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
		SELECT id, reference_number, rc_number, status, started_at, submitted_at,
		       approved_at, rejected_at, imported_at,
		       member_type, titel, firstname, lastname, birth_date,
		       company_name, uid_number, register_number,
		       email, phone,
		       resident_street, resident_street_number, resident_zip, resident_city,
		       privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
		       iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
		       reviewed_by_user_id, admin_note, needs_info_reason, target_participant_id,
		       import_started_at, import_finished_at, import_error_message,
		       membership_start_date, persons_in_household, consumption_previous_year,
		       consumption_forecast, feed_in_forecast, pv_power_kwp,
		       heat_pump, electric_vehicle, electric_hot_water,
		       einzugsart, bank_name, mandate_reference, mandate_date,
		       member_number,
		       created_at, updated_at
		FROM member_onboarding.application
		WHERE id = $1`

	app := &shared.Application{}
	var phone, privacyVersion, iban, accountHolder, reviewedByUserID, adminNote, needsInfoReason, targetParticipantID, importErrorMessage sql.NullString
	var titel, firstname, lastname, companyName, uidNumber, registerNumber sql.NullString
	var bankName, mandateReference sql.NullString
	var birthDate, startedAt, submittedAt, approvedAt, rejectedAt, importedAt, privacyAcceptedAt, sepaMandateAcceptedAt, importStartedAt, importFinishedAt sql.NullTime
	var membershipStartDate, mandateDate sql.NullTime
	var personsInHousehold, consumptionPreviousYear, consumptionForecast, feedInForecast sql.NullInt64
	var pvPowerKwp sql.NullFloat64
	var heatPump, electricVehicle, electricHotWater sql.NullBool
	var memberNumber sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&app.ID, &app.ReferenceNumber, &app.RCNumber, &app.Status, &startedAt,
		&submittedAt, &approvedAt, &rejectedAt, &importedAt,
		&app.MemberType, &titel, &firstname, &lastname, &birthDate,
		&companyName, &uidNumber, &registerNumber,
		&app.Email, &phone,
		&app.ResidentStreet, &app.ResidentStreetNumber, &app.ResidentZip, &app.ResidentCity,
		&app.PrivacyAccepted, &privacyVersion, &privacyAcceptedAt, &app.AccuracyConfirmed,
		&iban, &accountHolder, &app.SepaMandateAccepted, &sepaMandateAcceptedAt,
		&reviewedByUserID, &adminNote, &needsInfoReason, &targetParticipantID, &importStartedAt, &importFinishedAt,
		&importErrorMessage,
		&membershipStartDate, &personsInHousehold, &consumptionPreviousYear,
		&consumptionForecast, &feedInForecast, &pvPowerKwp,
		&heatPump, &electricVehicle, &electricHotWater,
		&app.Einzugsart, &bankName, &mandateReference, &mandateDate,
		&memberNumber,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	if titel.Valid {
		app.Titel = &titel.String
	}
	if firstname.Valid {
		app.Firstname = &firstname.String
	}
	if lastname.Valid {
		app.Lastname = &lastname.String
	}
	if companyName.Valid {
		app.CompanyName = &companyName.String
	}
	if uidNumber.Valid {
		app.UIDNumber = &uidNumber.String
	}
	if registerNumber.Valid {
		app.RegisterNumber = &registerNumber.String
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
	if iban.Valid {
		app.IBAN = &iban.String
	}
	if accountHolder.Valid {
		app.AccountHolder = &accountHolder.String
	}
	if sepaMandateAcceptedAt.Valid {
		app.SepaMandateAcceptedAt = &sepaMandateAcceptedAt.Time
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
	if membershipStartDate.Valid {
		app.MembershipStartDate = &membershipStartDate.Time
	}
	if personsInHousehold.Valid {
		v := int(personsInHousehold.Int64)
		app.PersonsInHousehold = &v
	}
	if consumptionPreviousYear.Valid {
		v := consumptionPreviousYear.Int64
		app.ConsumptionPreviousYear = &v
	}
	if consumptionForecast.Valid {
		v := consumptionForecast.Int64
		app.ConsumptionForecast = &v
	}
	if feedInForecast.Valid {
		v := feedInForecast.Int64
		app.FeedInForecast = &v
	}
	if pvPowerKwp.Valid {
		app.PvPowerKwp = &pvPowerKwp.Float64
	}
	if heatPump.Valid {
		app.HeatPump = &heatPump.Bool
	}
	if electricVehicle.Valid {
		app.ElectricVehicle = &electricVehicle.Bool
	}
	if electricHotWater.Valid {
		app.ElectricHotWater = &electricHotWater.Bool
	}
	if bankName.Valid {
		app.BankName = &bankName.String
	}
	if mandateReference.Valid {
		app.MandateReference = &mandateReference.String
	}
	if mandateDate.Valid {
		app.MandateDate = &mandateDate.Time
	}
	if memberNumber.Valid {
		v := int(memberNumber.Int64)
		app.MemberNumber = &v
	}

	return app, nil
}

// AssignMemberNumberTx assigns the next available member number for the EEG to the
// given application, using a row lock on registration_entrypoint to prevent races.
// No-op when the application already has a member number.
func (r *ApplicationRepository) AssignMemberNumberTx(tx *sql.Tx, appID uuid.UUID, rcNumber string) error {
	var memberNumberStart int
	err := tx.QueryRow(`
		SELECT member_number_start
		FROM member_onboarding.registration_entrypoint
		WHERE rc_number = $1
		FOR UPDATE`, rcNumber).Scan(&memberNumberStart)
	if err != nil {
		return fmt.Errorf("failed to lock entrypoint for member number: %w", err)
	}

	var nextNumber int
	err = tx.QueryRow(`
		SELECT COALESCE(MAX(member_number), $1 - 1) + 1
		FROM member_onboarding.application
		WHERE rc_number = $2`, memberNumberStart, rcNumber).Scan(&nextNumber)
	if err != nil {
		return fmt.Errorf("failed to compute next member number: %w", err)
	}

	_, err = tx.Exec(`
		UPDATE member_onboarding.application
		SET member_number = $1
		WHERE id = $2 AND member_number IS NULL`, nextNumber, appID)
	if err != nil {
		return fmt.Errorf("failed to assign member number: %w", err)
	}
	return nil
}

// Update updates an application
func (r *ApplicationRepository) Update(app *shared.Application) error {
	query := `
		UPDATE member_onboarding.application SET
			member_type = $1,
			titel = $2, firstname = $3, lastname = $4, birth_date = $5,
			company_name = $6, uid_number = $7, register_number = $8,
			email = $9, phone = $10,
			resident_street = $11, resident_street_number = $12, resident_zip = $13,
			resident_city = $14, privacy_accepted = $15,
			privacy_version = $16, accuracy_confirmed = $17,
			iban = $18, account_holder = $19, sepa_mandate_accepted = $20, sepa_mandate_accepted_at = $21,
			membership_start_date = $22, persons_in_household = $23, consumption_previous_year = $24,
			consumption_forecast = $25, feed_in_forecast = $26, pv_power_kwp = $27,
			heat_pump = $28, electric_vehicle = $29, electric_hot_water = $30,
			updated_at = NOW()
		WHERE id = $31`

	_, err := r.db.Exec(query,
		app.MemberType,
		app.Titel, app.Firstname, app.Lastname, app.BirthDate,
		app.CompanyName, app.UIDNumber, app.RegisterNumber,
		app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity,
		app.PrivacyAccepted, app.PrivacyVersion, app.AccuracyConfirmed,
		app.IBAN, app.AccountHolder, app.SepaMandateAccepted, app.SepaMandateAcceptedAt,
		app.MembershipStartDate, app.PersonsInHousehold, app.ConsumptionPreviousYear,
		app.ConsumptionForecast, app.FeedInForecast, app.PvPowerKwp,
		app.HeatPump, app.ElectricVehicle, app.ElectricHotWater,
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
			member_type = $1,
			titel = $2, firstname = $3, lastname = $4, birth_date = $5,
			company_name = $6, uid_number = $7, register_number = $8,
			email = $9, phone = $10,
			resident_street = $11, resident_street_number = $12, resident_zip = $13,
			resident_city = $14, privacy_accepted = $15,
			privacy_version = $16, accuracy_confirmed = $17,
			iban = $18, account_holder = $19, sepa_mandate_accepted = $20, sepa_mandate_accepted_at = $21,
			membership_start_date = $22, persons_in_household = $23, consumption_previous_year = $24,
			consumption_forecast = $25, feed_in_forecast = $26, pv_power_kwp = $27,
			heat_pump = $28, electric_vehicle = $29, electric_hot_water = $30,
			updated_at = NOW()
		WHERE id = $31`

	_, err := tx.Exec(query,
		app.MemberType,
		app.Titel, app.Firstname, app.Lastname, app.BirthDate,
		app.CompanyName, app.UIDNumber, app.RegisterNumber,
		app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity,
		app.PrivacyAccepted, app.PrivacyVersion, app.AccuracyConfirmed,
		app.IBAN, app.AccountHolder, app.SepaMandateAccepted, app.SepaMandateAcceptedAt,
		app.MembershipStartDate, app.PersonsInHousehold, app.ConsumptionPreviousYear,
		app.ConsumptionForecast, app.FeedInForecast, app.PvPowerKwp,
		app.HeatPump, app.ElectricVehicle, app.ElectricHotWater,
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

// UpdateStatusTx updates application status inside an existing transaction.
func (r *ApplicationRepository) UpdateStatusTx(tx *sql.Tx, id uuid.UUID, status shared.ApplicationStatus, submittedAt *time.Time) error {
	query := `
		UPDATE member_onboarding.application SET
			status = $1, submitted_at = $2, updated_at = NOW()
		WHERE id = $3`

	_, err := tx.Exec(query, status, submittedAt, id)
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
	if filters.RCNumberFilter != nil {
		conditions = append(conditions, fmt.Sprintf("a.rc_number = $%d", n))
		args = append(args, *filters.RCNumberFilter)
		n++
	}
	if filters.RCNumbers != nil && len(*filters.RCNumbers) > 0 {
		placeholders := make([]string, len(*filters.RCNumbers))
		for i, rc := range *filters.RCNumbers {
			placeholders[i] = fmt.Sprintf("$%d", n)
			args = append(args, rc)
			n++
		}
		conditions = append(conditions, "a.rc_number IN ("+strings.Join(placeholders, ", ")+")")
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
		SELECT a.id, a.reference_number, a.rc_number, a.status,
		       a.member_type, a.firstname, a.lastname, a.company_name, a.email, a.submitted_at
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
		var firstname, lastname, companyName sql.NullString
		var submittedAt sql.NullTime
		if err := rows.Scan(
			&item.ID, &item.ReferenceNumber, &item.RCNumber, &item.Status,
			&item.MemberType, &firstname, &lastname, &companyName, &item.Email, &submittedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan application list item: %w", err)
		}
		if firstname.Valid {
			item.Firstname = &firstname.String
		}
		if lastname.Valid {
			item.Lastname = &lastname.String
		}
		if companyName.Valid {
			item.CompanyName = &companyName.String
		}
		if submittedAt.Valid {
			item.SubmittedAt = &submittedAt.Time
		}
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
			member_type = $1,
			titel = $2, firstname = $3, lastname = $4, birth_date = $5,
			company_name = $6, uid_number = $7, register_number = $8,
			email = $9, phone = $10,
			resident_street = $11, resident_street_number = $12, resident_zip = $13,
			resident_city = $14, admin_note = $15,
			iban = $16, account_holder = $17,
			einzugsart = $18, bank_name = $19, mandate_reference = $20, mandate_date = $21,
			member_number = $22,
			updated_at = NOW()
		WHERE id = $23`

	_, err := tx.Exec(query,
		app.MemberType,
		app.Titel, app.Firstname, app.Lastname, app.BirthDate,
		app.CompanyName, app.UIDNumber, app.RegisterNumber,
		app.Email, app.Phone,
		app.ResidentStreet, app.ResidentStreetNumber, app.ResidentZip, app.ResidentCity,
		app.AdminNote, app.IBAN, app.AccountHolder,
		app.Einzugsart, app.BankName, app.MandateReference, app.MandateDate,
		app.MemberNumber,
		app.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}
	return nil
}

// UpdateStatusAdminTx updates the status and related timestamp columns atomically.
// Columns not applicable to the transition are preserved via COALESCE.
// DeleteDraftsByRCNumbers deletes all draft applications belonging to the given RC numbers.
// Returns the number of deleted rows.
func (r *ApplicationRepository) DeleteDraftsByRCNumbers(rcNumbers []string) (int64, error) {
	if len(rcNumbers) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(rcNumbers))
	args := make([]interface{}, len(rcNumbers))
	for i, rc := range rcNumbers {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = rc
	}
	result, err := r.db.Exec(
		fmt.Sprintf(
			`DELETE FROM member_onboarding.application
			 WHERE status = 'draft'
			   AND rc_number IN (%s)`,
			strings.Join(placeholders, ", "),
		),
		args...,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete drafts: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

func (r *ApplicationRepository) Delete(id uuid.UUID) error {
	result, err := r.db.Exec(`DELETE FROM member_onboarding.application WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

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
