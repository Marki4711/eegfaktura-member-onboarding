-- PROJ-57: Ansprechperson für Org-Mitgliedstypen (company, association,
-- municipality). Vier neue Spalten auf application — has_contact_person
-- als expliziter Toggle, damit "leer + nein" und "leer + ja" semantisch
-- unterscheidbar bleiben. Service-Layer cleart die drei TEXT-Felder auf
-- NULL, wenn der Toggle false ist oder der Mitgliedstyp nicht in der
-- Org-Liste liegt.

ALTER TABLE member_onboarding.application
    ADD COLUMN has_contact_person   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN contact_person_name  TEXT NULL,
    ADD COLUMN contact_person_email TEXT NULL,
    ADD COLUMN contact_person_phone TEXT NULL;
