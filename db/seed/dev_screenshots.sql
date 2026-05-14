-- ============================================================================
-- Dev seed for screenshot generation
-- ============================================================================
-- Inserts one registration_entrypoint (RC123456) plus one application in each
-- of the eight application statuses, two metering points per application,
-- and a status_log trail for every non-draft.
--
-- Idempotent: uses deterministic UUIDs and ON CONFLICT DO NOTHING so re-running
-- this script after a partial failure or after a code reload does not duplicate
-- rows. Truncate-and-reseed semantics are intentionally NOT used — re-running
-- preserves any existing data.
--
-- Usage:
--   psql "$DATABASE_URL" -f db/seed/dev_screenshots.sql
--   # or via the Makefile:
--   make seed-dev
-- ============================================================================

BEGIN;

-- ── registration_entrypoint ────────────────────────────────────────────────
INSERT INTO member_onboarding.registration_entrypoint (
    id, rc_number, is_active, contact_email,
    eeg_name, eeg_street, eeg_street_number, eeg_zip, eeg_city,
    creditor_id, sepa_mandate_enabled, eeg_id, show_central_policy,
    member_number_start
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'RC123456', TRUE, 'office@demo-eeg.at',
    'Demo Energiegemeinschaft', 'Musterstraße', '1', '1010', 'Wien',
    'AT12ZZZ00000000001', TRUE, 'demo-eeg-1', TRUE,
    1
)
ON CONFLICT (rc_number) DO NOTHING;

-- ── applications (one per status) ──────────────────────────────────────────
-- Common values factored into a CTE-ish pattern via repetition; PostgreSQL
-- doesn't allow CTEs to drive multi-row VALUES with mixed status semantics
-- in one INSERT, so we do eight explicit INSERTs.

-- 1. draft — started, not yet submitted
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status, started_at,
    titel, firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted,
    member_type, einzugsart
) VALUES (
    '22222222-0001-0000-0000-000000000001',
    'R-DEMO-0001', 'RC123456', 'draft', NOW() - INTERVAL '6 hours',
    NULL, 'Lukas', 'Müller', '1985-03-12',
    'lukas.mueller@example.at', '+43 660 1234567',
    'Praterstraße', '15', '1020', 'Wien',
    FALSE, FALSE,
    NULL, NULL, FALSE,
    'private', 'core'
) ON CONFLICT (id) DO NOTHING;

-- 2. submitted — fresh submission, waiting for review
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status, started_at, submitted_at,
    titel, firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
    member_type, einzugsart, membership_start_date, persons_in_household
) VALUES (
    '22222222-0002-0000-0000-000000000002',
    'R-DEMO-0002', 'RC123456', 'submitted',
    NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days',
    NULL, 'Anna', 'Bauer', '1990-07-25',
    'anna.bauer@example.at', '+43 664 2345678',
    'Mariahilferstraße', '42', '1060', 'Wien',
    TRUE, 'v1', NOW() - INTERVAL '2 days', TRUE,
    'AT611904300234573201', 'Anna Bauer', TRUE, NOW() - INTERVAL '2 days',
    'private', 'core', '2026-06-01', 1
) ON CONFLICT (id) DO NOTHING;

-- 3. under_review — admin is currently looking at it
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status, started_at, submitted_at,
    firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
    member_type, einzugsart, membership_start_date, persons_in_household,
    admin_note
) VALUES (
    '22222222-0003-0000-0000-000000000003',
    'R-DEMO-0003', 'RC123456', 'under_review',
    NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days',
    'Familie', 'Steiner', '1978-11-03',
    'steiner@example.at', '+43 660 3456789',
    'Lerchenfelderstraße', '7', '1080', 'Wien',
    TRUE, 'v1', NOW() - INTERVAL '4 days', TRUE,
    'AT421100000123456789', 'Thomas Steiner', TRUE, NOW() - INTERVAL '4 days',
    'private', 'core', '2026-06-01', 4,
    'Wartet auf Rückmeldung bzgl. zweitem Zählpunkt.'
) ON CONFLICT (id) DO NOTHING;

-- 4. needs_info — admin asked for clarification
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status, started_at, submitted_at,
    firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
    member_type, einzugsart, membership_start_date, persons_in_household,
    needs_info_reason
) VALUES (
    '22222222-0004-0000-0000-000000000004',
    'R-DEMO-0004', 'RC123456', 'needs_info',
    NOW() - INTERVAL '10 days', NOW() - INTERVAL '8 days',
    'Johann', 'Huber', '1965-04-20',
    'j.huber@example.at', '+43 699 4567890',
    'Hauptstraße', '23', '3100', 'St. Pölten',
    TRUE, 'v1', NOW() - INTERVAL '8 days', TRUE,
    'AT151200000987654321', 'Johann Huber', TRUE, NOW() - INTERVAL '8 days',
    'private', 'core', '2026-07-01', 2,
    'Bitte Zählpunktnummer überprüfen — die angegebene Nummer ist nur 32-stellig.'
) ON CONFLICT (id) DO NOTHING;

-- 5. approved — ready for import (triggers admin-import-action.png)
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status,
    started_at, submitted_at, approved_at,
    firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
    member_type, einzugsart, membership_start_date, persons_in_household,
    consumption_previous_year, consumption_forecast, pv_power_kwp
) VALUES (
    '22222222-0005-0000-0000-000000000005',
    'R-DEMO-0005', 'RC123456', 'approved',
    NOW() - INTERVAL '14 days', NOW() - INTERVAL '12 days', NOW() - INTERVAL '1 day',
    'Maria', 'Wagner', '1982-09-15',
    'maria.wagner@example.at', '+43 676 5678901',
    'Schönbrunnerstraße', '88', '1120', 'Wien',
    TRUE, 'v1', NOW() - INTERVAL '12 days', TRUE,
    'AT021420020010938765', 'Maria Wagner', TRUE, NOW() - INTERVAL '12 days',
    'private', 'core', '2026-06-15', 3,
    4200, 3800, 5.5
) ON CONFLICT (id) DO NOTHING;

-- 6. rejected — declined
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status,
    started_at, submitted_at, rejected_at,
    firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
    member_type, einzugsart, admin_note
) VALUES (
    '22222222-0006-0000-0000-000000000006',
    'R-DEMO-0006', 'RC123456', 'rejected',
    NOW() - INTERVAL '20 days', NOW() - INTERVAL '18 days', NOW() - INTERVAL '15 days',
    'Karl', 'Maier', '1972-01-08',
    'k.maier@example.at', '+43 660 6789012',
    'Ringstraße', '5', '8010', 'Graz',
    TRUE, 'v1', NOW() - INTERVAL '18 days', TRUE,
    'AT483200000111222333', 'Karl Maier', TRUE, NOW() - INTERVAL '18 days',
    'private', 'core',
    'Zählpunkte liegen außerhalb des EEG-Versorgungsgebiets.'
) ON CONFLICT (id) DO NOTHING;

-- 7. imported — already in eegFaktura (triggers admin-reset-import.png)
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status,
    started_at, submitted_at, approved_at, imported_at,
    firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
    member_type, einzugsart, membership_start_date, persons_in_household,
    member_number, target_participant_id,
    import_started_at, import_finished_at
) VALUES (
    '22222222-0007-0000-0000-000000000007',
    'R-DEMO-0007', 'RC123456', 'imported',
    NOW() - INTERVAL '40 days', NOW() - INTERVAL '38 days',
    NOW() - INTERVAL '35 days', NOW() - INTERVAL '34 days',
    'Stefanie', 'Pichler', '1995-12-19',
    's.pichler@example.at', '+43 664 7890123',
    'Landstraßer Hauptstraße', '99', '1030', 'Wien',
    TRUE, 'v1', NOW() - INTERVAL '38 days', TRUE,
    'AT741750000123456789', 'Stefanie Pichler', TRUE, NOW() - INTERVAL '38 days',
    'private', 'core', '2026-05-01', 2,
    'A005', 'core-part-A005-uuid',
    NOW() - INTERVAL '34 days', NOW() - INTERVAL '34 days'
) ON CONFLICT (id) DO NOTHING;

-- 8. import_failed — approved but core call failed
INSERT INTO member_onboarding.application (
    id, reference_number, rc_number, status,
    started_at, submitted_at, approved_at,
    firstname, lastname, birth_date,
    email, phone,
    resident_street, resident_street_number, resident_zip, resident_city,
    privacy_accepted, privacy_version, privacy_accepted_at, accuracy_confirmed,
    iban, account_holder, sepa_mandate_accepted, sepa_mandate_accepted_at,
    member_type, einzugsart, membership_start_date,
    import_started_at, import_finished_at, import_error_message
) VALUES (
    '22222222-0008-0000-0000-000000000008',
    'R-DEMO-0008', 'RC123456', 'import_failed',
    NOW() - INTERVAL '7 days', NOW() - INTERVAL '5 days', NOW() - INTERVAL '2 days',
    'Robert', 'Gruber', '1988-08-30',
    'robert.gruber@example.at', '+43 699 8901234',
    'Mozartgasse', '12', '4020', 'Linz',
    TRUE, 'v1', NOW() - INTERVAL '5 days', TRUE,
    'AT091500000444555666', 'Robert Gruber', TRUE, NOW() - INTERVAL '5 days',
    'private', 'core', '2026-06-01',
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day',
    'eegFaktura-Core: 503 service unavailable (timeout nach 30s)'
) ON CONFLICT (id) DO NOTHING;

-- ── metering_points (two per application) ──────────────────────────────────
INSERT INTO member_onboarding.metering_point (application_id, metering_point, direction, participation_factor)
SELECT a.id, 'AT0010000000000000000000000' || lpad((row_number() OVER (ORDER BY a.reference_number) * 2 - 1)::text, 6, '0'), 'CONSUMPTION', 100
FROM member_onboarding.application a
WHERE a.rc_number = 'RC123456' AND a.reference_number LIKE 'R-DEMO-%'
ON CONFLICT (application_id, metering_point) DO NOTHING;

INSERT INTO member_onboarding.metering_point (application_id, metering_point, direction, participation_factor)
SELECT a.id, 'AT0010000000000000000000000' || lpad((row_number() OVER (ORDER BY a.reference_number) * 2)::text, 6, '0'), 'PRODUCTION', 100
FROM member_onboarding.application a
WHERE a.rc_number = 'RC123456' AND a.reference_number LIKE 'R-DEMO-%'
ON CONFLICT (application_id, metering_point) DO NOTHING;

-- ── status_log trails ──────────────────────────────────────────────────────
-- For each non-draft application, log the transitions in chronological order.
-- status_log has no natural unique key, so we wipe-and-reinsert for the
-- demo applications to keep this script idempotent.
DELETE FROM member_onboarding.status_log
WHERE application_id IN (
    '22222222-0002-0000-0000-000000000002',
    '22222222-0003-0000-0000-000000000003',
    '22222222-0004-0000-0000-000000000004',
    '22222222-0005-0000-0000-000000000005',
    '22222222-0006-0000-0000-000000000006',
    '22222222-0007-0000-0000-000000000007',
    '22222222-0008-0000-0000-000000000008'
);

-- submitted: draft → submitted
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES ('22222222-0002-0000-0000-000000000002', 'draft', 'submitted', 'system', NOW() - INTERVAL '2 days')
ON CONFLICT DO NOTHING;

-- under_review: draft → submitted → under_review
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES
    ('22222222-0003-0000-0000-000000000003', 'draft', 'submitted', 'system', NOW() - INTERVAL '4 days'),
    ('22222222-0003-0000-0000-000000000003', 'submitted', 'under_review', 'admin-demo', NOW() - INTERVAL '3 days')
ON CONFLICT DO NOTHING;

-- needs_info trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, reason, created_at)
VALUES
    ('22222222-0004-0000-0000-000000000004', 'draft', 'submitted', 'system', NULL, NOW() - INTERVAL '8 days'),
    ('22222222-0004-0000-0000-000000000004', 'submitted', 'under_review', 'admin-demo', NULL, NOW() - INTERVAL '7 days'),
    ('22222222-0004-0000-0000-000000000004', 'under_review', 'needs_info', 'admin-demo',
     'Bitte Zählpunktnummer überprüfen — die angegebene Nummer ist nur 32-stellig.',
     NOW() - INTERVAL '6 days')
ON CONFLICT DO NOTHING;

-- approved trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES
    ('22222222-0005-0000-0000-000000000005', 'draft', 'submitted', 'system', NOW() - INTERVAL '12 days'),
    ('22222222-0005-0000-0000-000000000005', 'submitted', 'under_review', 'admin-demo', NOW() - INTERVAL '10 days'),
    ('22222222-0005-0000-0000-000000000005', 'under_review', 'approved', 'admin-demo', NOW() - INTERVAL '1 day')
ON CONFLICT DO NOTHING;

-- rejected trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, reason, created_at)
VALUES
    ('22222222-0006-0000-0000-000000000006', 'draft', 'submitted', 'system', NULL, NOW() - INTERVAL '18 days'),
    ('22222222-0006-0000-0000-000000000006', 'submitted', 'under_review', 'admin-demo', NULL, NOW() - INTERVAL '17 days'),
    ('22222222-0006-0000-0000-000000000006', 'under_review', 'rejected', 'admin-demo',
     'Zählpunkte liegen außerhalb des EEG-Versorgungsgebiets.', NOW() - INTERVAL '15 days')
ON CONFLICT DO NOTHING;

-- imported trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES
    ('22222222-0007-0000-0000-000000000007', 'draft', 'submitted', 'system', NOW() - INTERVAL '38 days'),
    ('22222222-0007-0000-0000-000000000007', 'submitted', 'under_review', 'admin-demo', NOW() - INTERVAL '36 days'),
    ('22222222-0007-0000-0000-000000000007', 'under_review', 'approved', 'admin-demo', NOW() - INTERVAL '35 days'),
    ('22222222-0007-0000-0000-000000000007', 'approved', 'imported', 'admin-demo', NOW() - INTERVAL '34 days')
ON CONFLICT DO NOTHING;

-- import_failed trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, reason, created_at)
VALUES
    ('22222222-0008-0000-0000-000000000008', 'draft', 'submitted', 'system', NULL, NOW() - INTERVAL '5 days'),
    ('22222222-0008-0000-0000-000000000008', 'submitted', 'under_review', 'admin-demo', NULL, NOW() - INTERVAL '4 days'),
    ('22222222-0008-0000-0000-000000000008', 'under_review', 'approved', 'admin-demo', NULL, NOW() - INTERVAL '2 days'),
    ('22222222-0008-0000-0000-000000000008', 'approved', 'import_failed', 'admin-demo',
     'eegFaktura-Core: 503 service unavailable (timeout nach 30s)', NOW() - INTERVAL '1 day')
ON CONFLICT DO NOTHING;

COMMIT;

-- Summary (for visibility when running interactively)
\echo '--------------------------------------------------------------------'
\echo 'dev_screenshots.sql complete.'
\echo 'Applications by status:'
SELECT status, count(*) FROM member_onboarding.application WHERE rc_number = 'RC123456' GROUP BY status ORDER BY status;
