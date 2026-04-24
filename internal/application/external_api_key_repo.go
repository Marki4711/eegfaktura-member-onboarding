package application

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ExternalAPIKey represents a row in member_onboarding.external_api_key.
type ExternalAPIKey struct {
	ID               uuid.UUID
	RCNumber         string
	KeyHash          string
	RevokedAt        *time.Time
	LastGeneratedAt  time.Time
	CreatedAt        time.Time
	DailyCount       int
	QuotaDate        *time.Time
}

// ExternalAPIKeyRepository handles DB access for external_api_key.
type ExternalAPIKeyRepository struct {
	db *sql.DB
}

// NewExternalAPIKeyRepository creates a new ExternalAPIKeyRepository.
func NewExternalAPIKeyRepository(db *sql.DB) *ExternalAPIKeyRepository {
	return &ExternalAPIKeyRepository{db: db}
}

// Upsert inserts or replaces the API key for an EEG.
// Resets revoked_at, daily_count, and quota_date on each call.
func (r *ExternalAPIKeyRepository) Upsert(rcNumber, keyHash string) error {
	_, err := r.db.Exec(`
		INSERT INTO member_onboarding.external_api_key
		    (rc_number, key_hash, revoked_at, last_generated_at, created_at, daily_count, quota_date)
		VALUES ($1, $2, NULL, NOW(), NOW(), 0, NULL)
		ON CONFLICT (rc_number) DO UPDATE
		    SET key_hash          = EXCLUDED.key_hash,
		        revoked_at        = NULL,
		        last_generated_at = NOW(),
		        daily_count       = 0,
		        quota_date        = NULL
	`, rcNumber, keyHash)
	return err
}

// Revoke marks the active key for an EEG as revoked.
// Returns ErrNotFound when no active key exists.
func (r *ExternalAPIKeyRepository) Revoke(rcNumber string) error {
	res, err := r.db.Exec(`
		UPDATE member_onboarding.external_api_key
		   SET revoked_at = NOW()
		 WHERE rc_number = $1 AND revoked_at IS NULL
	`, rcNumber)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// GetStatus returns whether an active key exists and when it was last generated.
// Returns (false, nil, nil) when no row exists for the EEG.
func (r *ExternalAPIKeyRepository) GetStatus(rcNumber string) (active bool, lastGeneratedAt *time.Time, err error) {
	var row struct {
		RevokedAt       *time.Time
		LastGeneratedAt time.Time
	}
	e := r.db.QueryRow(`
		SELECT revoked_at, last_generated_at
		  FROM member_onboarding.external_api_key
		 WHERE rc_number = $1
	`, rcNumber).Scan(&row.RevokedAt, &row.LastGeneratedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil, nil
	}
	if e != nil {
		return false, nil, e
	}
	t := row.LastGeneratedAt
	return row.RevokedAt == nil, &t, nil
}

// GetByKeyHash looks up an active (non-revoked) key by its SHA-256 hash.
// Returns ErrNotFound when the hash is unknown or the key has been revoked.
func (r *ExternalAPIKeyRepository) GetByKeyHash(keyHash string) (*ExternalAPIKey, error) {
	row := &ExternalAPIKey{}
	err := r.db.QueryRow(`
		SELECT id, rc_number, key_hash, revoked_at, last_generated_at, created_at,
		       daily_count, quota_date
		  FROM member_onboarding.external_api_key
		 WHERE key_hash = $1 AND revoked_at IS NULL
	`, keyHash).Scan(
		&row.ID, &row.RCNumber, &row.KeyHash,
		&row.RevokedAt, &row.LastGeneratedAt, &row.CreatedAt,
		&row.DailyCount, &row.QuotaDate,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// IncrementDailyCount atomically increments the daily counter.
// Resets to 1 when quota_date differs from today. Returns the new count.
func (r *ExternalAPIKeyRepository) IncrementDailyCount(id uuid.UUID) (int, error) {
	today := time.Now().UTC().Format("2006-01-02")
	var newCount int
	err := r.db.QueryRow(`
		UPDATE member_onboarding.external_api_key
		   SET daily_count = CASE
		                         WHEN quota_date = $2::date THEN daily_count + 1
		                         ELSE 1
		                     END,
		       quota_date  = $2::date
		 WHERE id = $1
		 RETURNING daily_count
	`, id, today).Scan(&newCount)
	return newCount, err
}
