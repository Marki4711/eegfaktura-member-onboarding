-- PROJ-53: Pro EEG konfigurierbarer Aktivierungs-Modus. Steuert, wann der
-- Activation-Check-Batch eine Anwendung von `ready_for_activation` auf
-- `activated` setzt.
--
-- Werte:
--  - 'participant_active' (Default, heutige Lösung): Core-Teilnehmer-Status
--    muss `ACTIVE` sein.
--  - 'any_meter_registration_started': Mindestens ein Zählpunkt im Core
--    muss processState in (PENDING, APPROVED, ACTIVE) haben — sprich der
--    Netzbetreiber hat die Online-Registrierung zumindest bestätigt.
--
-- Default 'participant_active' ist rückwärtskompatibel — keine Bestands-EEG
-- ändert ihr Verhalten ohne explizites Umstellen.

ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN activation_mode VARCHAR(40) NOT NULL DEFAULT 'participant_active';

ALTER TABLE member_onboarding.registration_entrypoint
    ADD CONSTRAINT registration_entrypoint_activation_mode_valid
        CHECK (activation_mode IN ('participant_active', 'any_meter_registration_started'));

COMMENT ON COLUMN member_onboarding.registration_entrypoint.activation_mode IS
    'PROJ-53: Trigger fuer ready_for_activation->activated. participant_active = Core-Teilnehmer ACTIVE (default); any_meter_registration_started = min. ein Meter mit processState in PENDING/APPROVED/ACTIVE.';
