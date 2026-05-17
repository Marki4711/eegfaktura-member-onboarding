-- PROJ-49 follow-up: neues Boolean-Feld pro Zählpunkt, das vom Mitglied
-- beantwortet wird, wenn es einen Batteriespeicher angibt.
-- Frage: "Speichersteuerung über die EEG vorstellbar?"
-- Sinnvoll nur bei PRODUCTION + generation_type='pv' UND Mitglied hat
-- Batteriespeicher angegeben (Service-Layer cleart sonst).

ALTER TABLE member_onboarding.metering_point
    ADD COLUMN battery_control_acceptable BOOLEAN NULL;
