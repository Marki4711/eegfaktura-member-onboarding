-- PROJ-35: per-EEG, per-year counter for the new reference-number format
-- `<RC>-<Year>-<NNNN>`. The old global sequence
-- `application_reference_number_seq` from migration 25 stays in place but is
-- no longer used for new applications. Old references (format
-- `MO-YYYY-NNNNNN`) remain untouched so links in already-sent mails stay
-- valid.
--
-- Atomic increment via INSERT ... ON CONFLICT DO UPDATE ... RETURNING.

CREATE TABLE IF NOT EXISTS member_onboarding.reference_number_counter (
    rc_number  TEXT NOT NULL,
    year       INT  NOT NULL,
    last_value INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (rc_number, year)
);
