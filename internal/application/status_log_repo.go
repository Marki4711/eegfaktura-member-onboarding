package application

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// StatusLogRepository handles database operations for status logs
type StatusLogRepository struct {
	db *sql.DB
}

// NewStatusLogRepository creates a new status log repository
func NewStatusLogRepository(db *sql.DB) *StatusLogRepository {
	return &StatusLogRepository{db: db}
}

// Create creates a new status log entry
func (r *StatusLogRepository) Create(entry *shared.StatusLogEntry) error {
	query := `
		INSERT INTO member_onboarding.status_log (
			application_id, from_status, to_status, changed_by_user_id, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err := r.db.QueryRow(query,
		entry.ApplicationID, entry.FromStatus, entry.ToStatus,
		entry.ChangedByUserID, entry.Reason, entry.CreatedAt,
	).Scan(&entry.ID)

	if err != nil {
		return fmt.Errorf("failed to create status log entry: %w", err)
	}

	return nil
}

// GetByApplicationID gets all status log entries for an application
func (r *StatusLogRepository) GetByApplicationID(applicationID uuid.UUID) ([]shared.StatusLogEntry, error) {
	query := `
		SELECT id, application_id, from_status, to_status, changed_by_user_id, reason, created_at
		FROM member_onboarding.status_log
		WHERE application_id = $1
		ORDER BY created_at`

	rows, err := r.db.Query(query, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status log: %w", err)
	}
	defer rows.Close()

	var entries []shared.StatusLogEntry
	for rows.Next() {
		var entry shared.StatusLogEntry
		var fromStatus, changedByUserID, reason sql.NullString

		err := rows.Scan(&entry.ID, &entry.ApplicationID, &fromStatus, &entry.ToStatus,
			&changedByUserID, &reason, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status log entry: %w", err)
		}

		if fromStatus.Valid {
			entry.FromStatus = &fromStatus.String
		}
		if changedByUserID.Valid {
			entry.ChangedByUserID = &changedByUserID.String
		}
		if reason.Valid {
			entry.Reason = &reason.String
		}

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status log entries: %w", err)
	}

	return entries, nil
}