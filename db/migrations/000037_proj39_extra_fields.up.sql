-- PROJ-39: Drei unabhängige Erweiterungen am Public-Form.
--
-- 1. application.titel_nach — optionaler Titel nach dem Namen
--    (z.B. "BSc", "MSc", "MBA"). Das bestehende Feld `titel` bleibt
--    erhalten und repräsentiert implizit den Titel VOR dem Namen.
--
-- 2. bank_name ist bereits vorhanden — keine DB-Änderung nötig, nur
--    Frontend/Public-Form-Erweiterung (siehe app code).
--
-- 3. metering_point.address_{street, street_number, zip, city} —
--    optionale abweichende Adresse je Zählpunkt. Wenn alle vier NULL,
--    gilt die Mitgliederadresse als implizierter Default. Wenn ≥1
--    gesetzt, müssen alle vier gesetzt sein (Validierung in Service-
--    Layer, nicht via DB-Constraint, damit Migration alter Daten
--    schmerzfrei bleibt).
--
-- Bricht die V1-Architektur-Entscheidung "all metering points use
-- the same address as the member" aus CLAUDE.md. Dokumentation
-- entsprechend aktualisiert.

ALTER TABLE member_onboarding.application
    ADD COLUMN titel_nach VARCHAR(50);

ALTER TABLE member_onboarding.metering_point
    ADD COLUMN address_street        VARCHAR(255),
    ADD COLUMN address_street_number VARCHAR(50),
    ADD COLUMN address_zip           VARCHAR(20),
    ADD COLUMN address_city          VARCHAR(255);
