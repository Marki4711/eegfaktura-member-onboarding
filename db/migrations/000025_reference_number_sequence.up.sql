-- +migrate Up
-- Replace random reference number generation with a DB sequence.
-- Prevents birthday-paradox collisions that become near-certain at scale
-- (e.g. 1000 EEGs × 50 members already gives ~100% collision probability
-- with the previous 900,000-value random pool).
CREATE SEQUENCE IF NOT EXISTS member_onboarding.application_reference_number_seq
    START WITH 1
    INCREMENT BY 1
    NO MAXVALUE
    NO CYCLE
    CACHE 1;
