-- Local development seed data.
-- Run once after migrations to make the happy path testable:
--
--   PowerShell:
--     $env:PGPASSWORD="password"
--     psql -h localhost -U postgres -d member_onboarding -f db/seeds/dev_seed.sql
--
--   bash / Make:
--     PGPASSWORD=password psql -h localhost -U postgres -d member_onboarding -f db/seeds/dev_seed.sql

INSERT INTO member_onboarding.registration_entrypoint (rc_number, is_active)
VALUES ('RC123456', TRUE)
ON CONFLICT (rc_number) DO NOTHING;
