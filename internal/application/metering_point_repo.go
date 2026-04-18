package application

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

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
			application_id, metering_point, direction, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, point := range points {
		_, err = stmt.Exec(applicationID, point.MeteringPoint, point.Direction, point.CreatedAt, point.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert metering point: %w", err)
		}
	}

	return tx.Commit()
}

// GetByApplicationID gets all metering points for an application
func (r *MeteringPointRepository) GetByApplicationID(applicationID uuid.UUID) ([]shared.MeteringPoint, error) {
	query := `
		SELECT id, application_id, metering_point, direction, created_at, updated_at
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
		err := rows.Scan(&point.ID, &point.ApplicationID, &point.MeteringPoint, &point.Direction, &point.CreatedAt, &point.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metering point: %w", err)
		}
		points = append(points, point)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metering points: %w", err)
	}

	return points, nil
}

// ValidateUniqueMeteringPoints checks if metering points are unique within the application
func (r *MeteringPointRepository) ValidateUniqueMeteringPoints(applicationID uuid.UUID, points []shared.MeteringPoint) error {
	// Check for duplicates in the input
	pointMap := make(map[string]bool)
	for _, point := range points {
		if pointMap[point.MeteringPoint] {
			return fmt.Errorf("duplicate metering point in request: %s", point.MeteringPoint)
		}
		pointMap[point.MeteringPoint] = true
	}

	// Check against existing points in database (for updates)
	if applicationID != uuid.Nil {
		existingPoints, err := r.GetByApplicationID(applicationID)
		if err != nil {
			return fmt.Errorf("failed to get existing metering points: %w", err)
		}

		for _, existing := range existingPoints {
			if pointMap[existing.MeteringPoint] {
				return fmt.Errorf("metering point already exists: %s", existing.MeteringPoint)
			}
		}
	}

	return nil
}