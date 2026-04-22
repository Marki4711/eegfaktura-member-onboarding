package application

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// MeteringPointRepository handles database operations for metering points
type MeteringPointRepository struct {
	db *sql.DB
}

// NewMeteringPointRepository creates a new metering point repository
func NewMeteringPointRepository(db *sql.DB) *MeteringPointRepository {
	return &MeteringPointRepository{db: db}
}

// CreateBulk creates multiple metering points for an application
func (r *MeteringPointRepository) CreateBulk(applicationID uuid.UUID, points []shared.MeteringPoint) error {
	if len(points) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// First, delete existing metering points for this application
	_, err = tx.Exec(`DELETE FROM member_onboarding.metering_point WHERE application_id = $1`, applicationID)
	if err != nil {
		return fmt.Errorf("failed to delete existing metering points: %w", err)
	}

	// Insert new metering points
	stmt, err := tx.Prepare(`
		INSERT INTO member_onboarding.metering_point (
			application_id, metering_point, direction, participation_factor,
			transformer, installation_number, installation_name,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, point := range points {
		_, err = stmt.Exec(
			applicationID, point.MeteringPoint, point.Direction, point.ParticipationFactor,
			point.Transformer, point.InstallationNumber, point.InstallationName,
			point.CreatedAt, point.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert metering point: %w", err)
		}
	}

	return tx.Commit()
}

// CreateBulkTx replaces all metering points for an application using an existing transaction.
func (r *MeteringPointRepository) CreateBulkTx(tx *sql.Tx, applicationID uuid.UUID, points []shared.MeteringPoint) error {
	_, err := tx.Exec(`DELETE FROM member_onboarding.metering_point WHERE application_id = $1`, applicationID)
	if err != nil {
		return fmt.Errorf("failed to delete existing metering points: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO member_onboarding.metering_point (
			application_id, metering_point, direction, participation_factor,
			transformer, installation_number, installation_name,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, point := range points {
		if _, err = stmt.Exec(
			applicationID, point.MeteringPoint, point.Direction, point.ParticipationFactor,
			point.Transformer, point.InstallationNumber, point.InstallationName,
			point.CreatedAt, point.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to insert metering point: %w", err)
		}
	}
	return nil
}

// GetByApplicationID gets all metering points for an application
func (r *MeteringPointRepository) GetByApplicationID(applicationID uuid.UUID) ([]shared.MeteringPoint, error) {
	query := `
		SELECT id, application_id, metering_point, direction, participation_factor,
		       transformer, installation_number, installation_name,
		       created_at, updated_at
		FROM member_onboarding.metering_point
		WHERE application_id = $1
		ORDER BY created_at`

	rows, err := r.db.Query(query, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query metering points: %w", err)
	}
	defer rows.Close()

	var points []shared.MeteringPoint
	for rows.Next() {
		var point shared.MeteringPoint
		var transformer, installationNumber, installationName sql.NullString
		err := rows.Scan(
			&point.ID, &point.ApplicationID, &point.MeteringPoint, &point.Direction, &point.ParticipationFactor,
			&transformer, &installationNumber, &installationName,
			&point.CreatedAt, &point.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metering point: %w", err)
		}
		if transformer.Valid {
			point.Transformer = &transformer.String
		}
		if installationNumber.Valid {
			point.InstallationNumber = &installationNumber.String
		}
		if installationName.Valid {
			point.InstallationName = &installationName.String
		}
		points = append(points, point)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metering points: %w", err)
	}

	return points, nil
}

// ValidateUniqueMeteringPoints checks that no metering point appears twice in the request.
// Cross-checking against DB rows is intentionally omitted: CreateBulk/CreateBulkTx
// always DELETE existing rows before inserting, so the only constraint that matters
// is uniqueness within the incoming set.
func (r *MeteringPointRepository) ValidateUniqueMeteringPoints(_ uuid.UUID, points []shared.MeteringPoint) error {
	seen := make(map[string]bool)
	for _, point := range points {
		if seen[point.MeteringPoint] {
			return fmt.Errorf("duplicate metering point in request: %s", point.MeteringPoint)
		}
		seen[point.MeteringPoint] = true
	}
	return nil
}

// GetNumbersByApplicationIDs fetches metering point numbers grouped by application ID.
// Used by the admin list endpoint to enrich list items in a single query.
func (r *MeteringPointRepository) GetNumbersByApplicationIDs(ids []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(ids) == 0 {
		return map[uuid.UUID][]string{}, nil
	}

	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = id.String()
	}

	query := `
		SELECT application_id, metering_point
		FROM member_onboarding.metering_point
		WHERE application_id = ANY($1::uuid[])
		ORDER BY application_id, created_at`

	rows, err := r.db.Query(query, pq.Array(strIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to query metering points by application IDs: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]string)
	for rows.Next() {
		var appID uuid.UUID
		var mp string
		if err := rows.Scan(&appID, &mp); err != nil {
			return nil, fmt.Errorf("failed to scan metering point row: %w", err)
		}
		result[appID] = append(result[appID], mp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metering point rows: %w", err)
	}

	return result, nil
}