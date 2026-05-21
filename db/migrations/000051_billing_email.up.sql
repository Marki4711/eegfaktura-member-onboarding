-- PROJ-58: Abweichende Rechnungs-E-Mail für Org-Mitgliedstypen (company,
-- association, municipality). has_billing_email als expliziter Toggle,
-- damit "leer + nein" und "leer + ja" semantisch unterscheidbar bleiben.
-- Service-Layer cleart billing_email auf NULL, wenn der Toggle false
-- ist oder der Mitgliedstyp nicht in der Org-Liste liegt.

ALTER TABLE member_onboarding.application
    ADD COLUMN has_billing_email BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN billing_email     TEXT NULL;
