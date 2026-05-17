-- PROJ-48: EEG-Setting für den Zeitpunkt der Mandat-Übermittlung.
--
-- FALSE (Default = heutiges Verhalten):
--   Basis-Mandat-PDF wird bei Submit angehängt (ohne Mandatsreferenz —
--   Platzhalter "wird von EEG ausgefüllt"). B2B-Mandat (PROJ-47) erst
--   beim Import mit eingedruckter Mandatsreferenz.
--
-- TRUE:
--   Bei Submit wird KEIN Mandat-PDF versendet. Beim Import wird das
--   Mandat (Basis ODER B2B, je nach einzugsart) mit eingedruckter
--   Mandatsreferenz = Mitgliedsnummer an die Import-Mail angehängt.
--
-- Migration ist additiv + Bestands-Default FALSE = keine Verhaltens-
-- änderung für existierende EEGs.

ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN sepa_mandate_at_import BOOLEAN NOT NULL DEFAULT FALSE;
