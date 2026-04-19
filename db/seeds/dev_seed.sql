-- Local development seed data.
-- Run once after migrations to make the happy path testable:
--
--   PowerShell:
--     $env:PGPASSWORD="password"
--     psql -h localhost -U postgres -d member_onboarding -f db/seeds/dev_seed.sql
--
--   bash / Make:
--     PGPASSWORD=password psql -h localhost -U postgres -d member_onboarding -f db/seeds/dev_seed.sql

INSERT INTO member_onboarding.registration_entrypoint (eeg_id, rc_number, is_active)
VALUES ('00000000-0000-0000-0000-000000000001', 'RC123456', TRUE)
ON CONFLICT (rc_number) DO NOTHING;
