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

// CreateBulkTx inserts multiple consent snapshots inside an existing
// transaction. Each entry's ConsentType is persisted as-is — empty falls
// back to 'explicit' via the column default, but callers should always
// populate it explicitly for clarity (shared.ConsentTypeExplicit /
// shared.ConsentTypeInformational).
func (r *DocumentConsentRepository) CreateBulkTx(tx *sql.Tx, consents []shared.DocumentConsent) error {
	for _, c := range consents {
		consentType := string(c.ConsentType)
		if consentType == "" {
			consentType = string(shared.ConsentTypeExplicit)
		}
		_, err := tx.Exec(`
			INSERT INTO member_onboarding.document_consent
			  (id, application_id, title, url, is_central_policy, consented_at, consent_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			c.ID, c.ApplicationID, c.Title, c.URL, c.IsCentralPolicy, c.ConsentedAt, consentType)
		if err != nil {
			return fmt.Errorf("failed to insert document consent: %w", err)
		}
	}
	return nil
}

// GetByApplicationID returns all consent snapshots for an application.
func (r *DocumentConsentRepository) GetByApplicationID(applicationID uuid.UUID) ([]shared.DocumentConsent, error) {
	rows, err := r.db.Query(`
		SELECT id, application_id, title, url, is_central_policy, consented_at, consent_type
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
		var consentType string
		if err := rows.Scan(&c.ID, &c.ApplicationID, &c.Title, &c.URL, &c.IsCentralPolicy, &c.ConsentedAt, &consentType); err != nil {
			return nil, fmt.Errorf("failed to scan document consent: %w", err)
		}
		c.ConsentType = shared.ConsentType(consentType)
		consents = append(consents, c)
	}
	return consents, rows.Err()
}
