-- +migrate Up

-- pg_trgm enables GIN indexes for ILIKE with leading wildcard ('%foo%')
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Composite indexes for the primary admin list access pattern:
-- tenant filter (rc_number) combined with sort (created_at DESC)
CREATE INDEX idx_application_rc_number_created_at
    ON member_onboarding.application(rc_number, created_at DESC);

-- Combined filter: rc_number + status filter + sort — avoids index intersection
CREATE INDEX idx_application_rc_number_status_created_at
    ON member_onboarding.application(rc_number, status, created_at DESC);

-- Date range filter on submitted_at (used by admin list filter)
CREATE INDEX idx_application_submitted_at
    ON member_onboarding.application(submitted_at);

-- GIN trigram indexes for ILIKE text search with leading wildcard
CREATE INDEX idx_application_lastname_trgm
    ON member_onboarding.application USING GIN (lastname gin_trgm_ops);

CREATE INDEX idx_application_email_trgm
    ON member_onboarding.application USING GIN (email gin_trgm_ops);

CREATE INDEX idx_application_reference_number_trgm
    ON member_onboarding.application USING GIN (reference_number gin_trgm_ops);

-- GIN trigram index for metering point ILIKE subquery in admin list filter
CREATE INDEX idx_metering_point_metering_point_trgm
    ON member_onboarding.metering_point USING GIN (metering_point gin_trgm_ops);
