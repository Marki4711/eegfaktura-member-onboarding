-- PROJ-56: Netzbetreiber-Info-Seite im Beitrittsbestätigungs-PDF.
-- Zwei optionale Felder pro Antrag, die im Public-Formular sichtbar werden,
-- wenn die Vollmacht-Checkbox aktiviert wird. NULL = nicht angegeben
-- (auch wenn die Vollmacht erteilt wurde — der User darf die Felder leer
-- lassen, sofern die EEG-field_config sie nicht auf required setzt).

ALTER TABLE member_onboarding.application
    ADD COLUMN network_operator_customer_number TEXT NULL,
    ADD COLUMN meter_inventory_number           TEXT NULL;
