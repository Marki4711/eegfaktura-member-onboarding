package application

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

const MaxLegalDocumentsPerEEG = 10

// LegalDocumentRepository handles DB access for legal_document.
type LegalDocumentRepository struct {
	db *sql.DB
}

// NewLegalDocumentRepository creates a new LegalDocumentRepository.
func NewLegalDocumentRepository(db *sql.DB) *LegalDocumentRepository {
	return &LegalDocumentRepository{db: db}
}

// GetByRCNumber returns all legal documents for an EEG ordered by sort_order.
func (r *LegalDocumentRepository) GetByRCNumber(rcNumber string) ([]shared.LegalDocument, error) {
	rows, err := r.db.Query(`
		SELECT id, rc_number, title, url, required, sort_order, created_at, updated_at
		FROM member_onboarding.legal_document
		WHERE rc_number = $1
		ORDER BY sort_order ASC`, rcNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query legal documents: %w", err)
	}
	defer rows.Close()

	var docs []shared.LegalDocument
	for rows.Next() {
		var d shared.LegalDocument
		if err := rows.Scan(&d.ID, &d.RCNumber, &d.Title, &d.URL, &d.Required, &d.SortOrder, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan legal document: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

// GetByID returns a single legal document by ID.
func (r *LegalDocumentRepository) GetByID(id uuid.UUID) (*shared.LegalDocument, error) {
	var d shared.LegalDocument
	err := r.db.QueryRow(`
		SELECT id, rc_number, title, url, required, sort_order, created_at, updated_at
		FROM member_onboarding.legal_document
		WHERE id = $1`, id).Scan(
		&d.ID, &d.RCNumber, &d.Title, &d.URL, &d.Required, &d.SortOrder, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, shared.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get legal document: %w", err)
	}
	return &d, nil
}

// CountByRCNumber returns the number of legal documents for an EEG.
func (r *LegalDocumentRepository) CountByRCNumber(rcNumber string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM member_onboarding.legal_document WHERE rc_number = $1`, rcNumber).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count legal documents: %w", err)
	}
	return count, nil
}

// Create inserts a new legal document and returns it with generated ID.
func (r *LegalDocumentRepository) Create(rcNumber, title, url string, required bool, sortOrder int) (*shared.LegalDocument, error) {
	now := time.Now()
	d := &shared.LegalDocument{
		ID:        uuid.New(),
		RCNumber:  rcNumber,
		Title:     title,
		URL:       url,
		Required:  required,
		SortOrder: sortOrder,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := r.db.Exec(`
		INSERT INTO member_onboarding.legal_document
		  (id, rc_number, title, url, required, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		d.ID, d.RCNumber, d.Title, d.URL, d.Required, d.SortOrder, d.CreatedAt, d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create legal document: %w", err)
	}
	return d, nil
}

// Update saves title, url, required for an existing document.
func (r *LegalDocumentRepository) Update(id uuid.UUID, title, url string, required bool) error {
	result, err := r.db.Exec(`
		UPDATE member_onboarding.legal_document
		SET title = $1, url = $2, required = $3, updated_at = NOW()
		WHERE id = $4`, title, url, required, id)
	if err != nil {
		return fmt.Errorf("failed to update legal document: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Delete removes a legal document by ID.
func (r *LegalDocumentRepository) Delete(id uuid.UUID) error {
	result, err := r.db.Exec(`DELETE FROM member_onboarding.legal_document WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete legal document: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Reorder updates sort_order for all documents of an EEG in a single transaction.
// ids must contain all document IDs for the EEG in the desired order.
func (r *LegalDocumentRepository) Reorder(rcNumber string, ids []uuid.UUID) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, id := range ids {
		_, err := tx.Exec(`
			UPDATE member_onboarding.legal_document
			SET sort_order = $1, updated_at = NOW()
			WHERE id = $2 AND rc_number = $3`, i, id, rcNumber)
		if err != nil {
			return fmt.Errorf("failed to update sort order: %w", err)
		}
	}
	return tx.Commit()
}
