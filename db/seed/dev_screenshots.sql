-- ============================================================================
-- Dev seed for screenshot generation
-- ============================================================================
-- Inserts one registration_entrypoint (RC-DEMO) plus one application in each
-- of the eight application statuses, two metering points per application,
-- and a status_log trail for every non-draft.
--
-- All names / emails / phones / addresses are deliberately obvious placeholders
-- ("Mustermann", "@example.invalid" per RFC 2606, "Musterstraße") so the
-- screenshots never look like real personal data.
--
-- RC-DEMO is a dedicated screenshot tenant — separate from any RC numbers the
-- developer may already have in their local database. The screenshot bot user
-- in Keycloak is configured to see only this tenant.
--
-- Idempotent: wipes and reinserts the screenshot rows on every run, so
-- changes to the placeholder values propagate without manual cleanup.
-- Existing applications on OTHER rc_numbers are untouched.
--
-- Usage:
--   psql "$DATABASE_URL" -f db/seed/dev_screenshots.sql
--   # or via the Makefile:
--   make seed-dev
-- ============================================================================

BEGIN;

-- Wipe previous screenshot data. CASCADE on application removes child rows
-- (metering_point, status_log, document_consent). The registration_entrypoint
-- can only be deleted after its applications are gone (ON DELETE RESTRICT).
DELETE FROM member_onboarding.application WHERE reference_number LIKE 'R-DEMO-%';
DELETE FROM member_onboarding.registration_entrypoint WHERE rc_number IN ('RC-DEMO', 'RC123456-DEMO-OBSOLETE');

-- ── registration_entrypoint ────────────────────────────────────────────────
INSERT INTO member_onboarding.registration_entrypoint (
    id, rc_number, is_active, contact_email,
    eeg_name, eeg_street, eeg_street_number, eeg_zip, eeg_city,
    creditor_id, sepa_mandate_enabled, eeg_id, show_central_policy,
    member_number_start
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'RC-DEMO', TRUE, 'office@example.invalid',
    'Demo Energiegemeinschaft', 'Musterstraße', '1', '1010', 'Musterstadt',
    'AT00ZZZ00000000000', TRUE, 'demo-eeg-1', TRUE,
    1
)
ON CONFLICT (rc_number) DO NOTHING;

-- ── applications (one per status) ──────────────────────────────────────────
-- Common values factored into a CTE-ish pattern via repetition; PostgreSQL
-- doesn't allow CTEs to drive multi-row VALUES with mixed status semantics
-- in one INSERT, so we do eight explicit INSERTs.

-- Placeholder addresses, phone numbers, and bank details below are deliberately
-- non-functional: Musterstraße / Musterweg, +43 1 234567X, AT-format IBANs
-- with all-zero body, and @example.invalid (RFC 2606) emails.

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
    'R-DEMO-0001', 'RC-DEMO', 'draft', NOW() - INTERVAL '6 hours',
    NULL, 'Max', 'Musterperson', '1985-03-12',
    'demo1@example.invalid', '+43 1 2345671',
    'Musterstraße', '1', '1010', 'Musterstadt',
    FALSE, FALSE,
    NULL, NULL, FALSE,
    'private', 'core'
);

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
    'R-DEMO-0002', 'RC-DEMO', 'submitted',
    NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days',
    NULL, 'Erika', 'Musterperson', '1990-07-25',
    'demo2@example.invalid', '+43 1 2345672',
    'Musterstraße', '2', '1020', 'Musterstadt',
    TRUE, 'v1', NOW() - INTERVAL '2 days', TRUE,
    'AT000000000000000000', 'Erika Musterperson', TRUE, NOW() - INTERVAL '2 days',
    'private', 'core', '2026-06-01', 1
);

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
    'R-DEMO-0003', 'RC-DEMO', 'under_review',
    NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days',
    'Otto', 'Musterperson', '1978-11-03',
    'demo3@example.invalid', '+43 1 2345673',
    'Musterstraße', '3', '1030', 'Musterstadt',
    TRUE, 'v1', NOW() - INTERVAL '4 days', TRUE,
    'AT000000000000000003', 'Otto Musterperson', TRUE, NOW() - INTERVAL '4 days',
    'private', 'core', '2026-06-01', 4,
    'Wartet auf Rückmeldung bzgl. zweitem Zählpunkt.'
);

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
    'R-DEMO-0004', 'RC-DEMO', 'needs_info',
    NOW() - INTERVAL '10 days', NOW() - INTERVAL '8 days',
    'Anna', 'Musterperson', '1965-04-20',
    'demo4@example.invalid', '+43 1 2345674',
    'Musterstraße', '4', '4020', 'Musterstadt',
    TRUE, 'v1', NOW() - INTERVAL '8 days', TRUE,
    'AT000000000000000004', 'Anna Musterperson', TRUE, NOW() - INTERVAL '8 days',
    'private', 'core', '2026-07-01', 2,
    'Bitte Zählpunktnummer überprüfen — die angegebene Nummer ist nur 32-stellig.'
);

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
    'R-DEMO-0005', 'RC-DEMO', 'approved',
    NOW() - INTERVAL '14 days', NOW() - INTERVAL '12 days', NOW() - INTERVAL '1 day',
    'Hans', 'Musterperson', '1982-09-15',
    'demo5@example.invalid', '+43 1 2345675',
    'Musterstraße', '5', '8010', 'Musterstadt',
    TRUE, 'v1', NOW() - INTERVAL '12 days', TRUE,
    'AT000000000000000005', 'Hans Musterperson', TRUE, NOW() - INTERVAL '12 days',
    'private', 'core', '2026-06-15', 3,
    4200, 3800, 5.5
);

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
    'R-DEMO-0006', 'RC-DEMO', 'rejected',
    NOW() - INTERVAL '20 days', NOW() - INTERVAL '18 days', NOW() - INTERVAL '15 days',
    'Maria', 'Musterperson', '1972-01-08',
    'demo6@example.invalid', '+43 1 2345676',
    'Musterstraße', '6', '5020', 'Musterstadt',
    TRUE, 'v1', NOW() - INTERVAL '18 days', TRUE,
    'AT000000000000000006', 'Maria Musterperson', TRUE, NOW() - INTERVAL '18 days',
    'private', 'core',
    'Zählpunkte liegen außerhalb des EEG-Versorgungsgebiets.'
);

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
    'R-DEMO-0007', 'RC-DEMO', 'imported',
    NOW() - INTERVAL '40 days', NOW() - INTERVAL '38 days',
    NOW() - INTERVAL '35 days', NOW() - INTERVAL '34 days',
    'Karl', 'Musterperson', '1995-12-19',
    'demo7@example.invalid', '+43 1 2345677',
    'Musterstraße', '7', '6020', 'Musterstadt',
    TRUE, 'v1', NOW() - INTERVAL '38 days', TRUE,
    'AT000000000000000007', 'Karl Musterperson', TRUE, NOW() - INTERVAL '38 days',
    'private', 'core', '2026-05-01', 2,
    'A005', 'core-part-A005-uuid',
    NOW() - INTERVAL '34 days', NOW() - INTERVAL '34 days'
);

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
    'R-DEMO-0008', 'RC-DEMO', 'import_failed',
    NOW() - INTERVAL '7 days', NOW() - INTERVAL '5 days', NOW() - INTERVAL '2 days',
    'Stefanie', 'Musterperson', '1988-08-30',
    'demo8@example.invalid', '+43 1 2345678',
    'Musterstraße', '8', '9020', 'Musterstadt',
    TRUE, 'v1', NOW() - INTERVAL '5 days', TRUE,
    'AT000000000000000008', 'Stefanie Musterperson', TRUE, NOW() - INTERVAL '5 days',
    'private', 'core', '2026-06-01',
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day',
    'eegFaktura-Core: 503 service unavailable (timeout nach 30s)'
);

-- ── metering_points (two per application) ──────────────────────────────────
INSERT INTO member_onboarding.metering_point (application_id, metering_point, direction, participation_factor)
SELECT a.id, 'AT0010000000000000000000000' || lpad((row_number() OVER (ORDER BY a.reference_number) * 2 - 1)::text, 6, '0'), 'CONSUMPTION', 100
FROM member_onboarding.application a
WHERE a.rc_number = 'RC-DEMO' AND a.reference_number LIKE 'R-DEMO-%';

INSERT INTO member_onboarding.metering_point (application_id, metering_point, direction, participation_factor)
SELECT a.id, 'AT0010000000000000000000000' || lpad((row_number() OVER (ORDER BY a.reference_number) * 2)::text, 6, '0'), 'PRODUCTION', 100
FROM member_onboarding.application a
WHERE a.rc_number = 'RC-DEMO' AND a.reference_number LIKE 'R-DEMO-%';

-- ── status_log trails ──────────────────────────────────────────────────────
-- For each non-draft application, log the transitions in chronological order.
-- The earlier wipe-and-reinsert already removed status_log rows via the
-- application CASCADE; the entries below are inserted fresh against the
-- newly created application rows.

-- submitted: draft → submitted
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES ('22222222-0002-0000-0000-000000000002', 'draft', 'submitted', 'system', NOW() - INTERVAL '2 days')
;

-- under_review: draft → submitted → under_review
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES
    ('22222222-0003-0000-0000-000000000003', 'draft', 'submitted', 'system', NOW() - INTERVAL '4 days'),
    ('22222222-0003-0000-0000-000000000003', 'submitted', 'under_review', 'admin-demo', NOW() - INTERVAL '3 days')
;

-- needs_info trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, reason, created_at)
VALUES
    ('22222222-0004-0000-0000-000000000004', 'draft', 'submitted', 'system', NULL, NOW() - INTERVAL '8 days'),
    ('22222222-0004-0000-0000-000000000004', 'submitted', 'under_review', 'admin-demo', NULL, NOW() - INTERVAL '7 days'),
    ('22222222-0004-0000-0000-000000000004', 'under_review', 'needs_info', 'admin-demo',
     'Bitte Zählpunktnummer überprüfen — die angegebene Nummer ist nur 32-stellig.',
     NOW() - INTERVAL '6 days')
;

-- approved trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES
    ('22222222-0005-0000-0000-000000000005', 'draft', 'submitted', 'system', NOW() - INTERVAL '12 days'),
    ('22222222-0005-0000-0000-000000000005', 'submitted', 'under_review', 'admin-demo', NOW() - INTERVAL '10 days'),
    ('22222222-0005-0000-0000-000000000005', 'under_review', 'approved', 'admin-demo', NOW() - INTERVAL '1 day')
;

-- rejected trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, reason, created_at)
VALUES
    ('22222222-0006-0000-0000-000000000006', 'draft', 'submitted', 'system', NULL, NOW() - INTERVAL '18 days'),
    ('22222222-0006-0000-0000-000000000006', 'submitted', 'under_review', 'admin-demo', NULL, NOW() - INTERVAL '17 days'),
    ('22222222-0006-0000-0000-000000000006', 'under_review', 'rejected', 'admin-demo',
     'Zählpunkte liegen außerhalb des EEG-Versorgungsgebiets.', NOW() - INTERVAL '15 days')
;

-- imported trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, created_at)
VALUES
    ('22222222-0007-0000-0000-000000000007', 'draft', 'submitted', 'system', NOW() - INTERVAL '38 days'),
    ('22222222-0007-0000-0000-000000000007', 'submitted', 'under_review', 'admin-demo', NOW() - INTERVAL '36 days'),
    ('22222222-0007-0000-0000-000000000007', 'under_review', 'approved', 'admin-demo', NOW() - INTERVAL '35 days'),
    ('22222222-0007-0000-0000-000000000007', 'approved', 'imported', 'admin-demo', NOW() - INTERVAL '34 days')
;

-- import_failed trail
INSERT INTO member_onboarding.status_log (application_id, from_status, to_status, changed_by_user_id, reason, created_at)
VALUES
    ('22222222-0008-0000-0000-000000000008', 'draft', 'submitted', 'system', NULL, NOW() - INTERVAL '5 days'),
    ('22222222-0008-0000-0000-000000000008', 'submitted', 'under_review', 'admin-demo', NULL, NOW() - INTERVAL '4 days'),
    ('22222222-0008-0000-0000-000000000008', 'under_review', 'approved', 'admin-demo', NULL, NOW() - INTERVAL '2 days'),
    ('22222222-0008-0000-0000-000000000008', 'approved', 'import_failed', 'admin-demo',
     'eegFaktura-Core: 503 service unavailable (timeout nach 30s)', NOW() - INTERVAL '1 day')
;

COMMIT;

-- Summary (for visibility when running interactively)
\echo '--------------------------------------------------------------------'
\echo 'dev_screenshots.sql complete.'
\echo 'Applications by status:'
SELECT status, count(*) FROM member_onboarding.application WHERE rc_number = 'RC-DEMO' GROUP BY status ORDER BY status;
