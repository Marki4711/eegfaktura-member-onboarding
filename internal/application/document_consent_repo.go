package application

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// DocumentConsentRepository handles DB access for document_consent.
type DocumentConsentRepository struct {
	db *sql.DB
}

// NewDocumentConsentRepository creates a new DocumentConsentRepository.
func NewDocumentConsentRepository(db *sql.DB) *DocumentConsentRepository {
	return &DocumentConsentRepository{db: db}
}

// CreateBulkTx inserts multiple consent snapshots inside an existing transaction.
func (r *DocumentConsentRepository) CreateBulkTx(tx *sql.Tx, consents []shared.DocumentConsent) error {
	for _, c := range consents {
		_, err := tx.Exec(`
			INSERT INTO member_onboarding.document_consent
			  (id, application_id, title, url, is_central_policy, consented_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			c.ID, c.ApplicationID, c.Title, c.URL, c.IsCentralPolicy, c.ConsentedAt)
		if err != nil {
			return fmt.Errorf("failed to insert document consent: %w", err)
		}
	}
	return nil
}

// GetByApplicationID returns all consent snapshots for an application.
func (r *DocumentConsentRepository) GetByApplicationID(applicationID uuid.UUID) ([]shared.DocumentConsent, error) {
	rows, err := r.db.Query(`
		SELECT id, application_id, title, url, is_central_policy, consented_at
		FROM member_onboarding.document_consent
		WHERE application_id = $1
		ORDER BY consented_at ASC`, applicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query document consents: %w", err)
	}
	defer rows.Close()

	var consents []shared.DocumentConsent
	for rows.Next() {
		var c shared.DocumentConsent
		if err := rows.Scan(&c.ID, &c.ApplicationID, &c.Title, &c.URL, &c.IsCentralPolicy, &c.ConsentedAt); err != nil {
			return nil, fmt.Errorf("failed to scan document consent: %w", err)
		}
		consents = append(consents, c)
	}
	return consents, rows.Err()
}
