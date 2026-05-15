ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS cooperative_shares_count;

ALTER TABLE member_onboarding.registration_entrypoint
    DROP COLUMN IF EXISTS cooperative_share_amount_cents,
    DROP COLUMN IF EXISTS cooperative_required_shares,
    DROP COLUMN IF EXISTS cooperative_shares_enabled;
