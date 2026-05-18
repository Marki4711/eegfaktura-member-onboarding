-- PROJ-52: Pro Richtung konfigurierbarer Zählpunkt-Prefix.
-- Wenn gesetzt, wird der Wert im Mitgliederformular als fixer Mask-Bestandteil
-- gerendert; das Mitglied muss nur die restlichen Stellen eintippen. Backend
-- validiert beim Submit, dass die Zählpunktnummer mit dem konfigurierten
-- Prefix beginnt. NULL = heutiges Verhalten (nur "AT" ist fix).
--
-- Format-Constraint: muss mit AT beginnen, max 33 Stellen, Stellen 3-13 nur
-- Ziffern (Netzbetreibernummer + PLZ nach E-Control-Spec), Stellen 14+
-- alphanumerisch. Längen-Check via DB-CHECK; Inhalts-Validierung im
-- Service-Layer.

ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN metering_point_prefix_consumption VARCHAR(33) NULL,
    ADD COLUMN metering_point_prefix_production  VARCHAR(33) NULL;

-- Defense-in-depth: leere Strings sind explizit verboten (entweder NULL
-- oder mindestens "AT"+Ziffern). Service-Layer normalisiert vor dem Insert.
ALTER TABLE member_onboarding.registration_entrypoint
    ADD CONSTRAINT registration_entrypoint_prefix_consumption_format
        CHECK (
            metering_point_prefix_consumption IS NULL
            OR metering_point_prefix_consumption ~ '^AT[0-9A-Z]{0,31}$'
        ),
    ADD CONSTRAINT registration_entrypoint_prefix_production_format
        CHECK (
            metering_point_prefix_production IS NULL
            OR metering_point_prefix_production ~ '^AT[0-9A-Z]{0,31}$'
        );
