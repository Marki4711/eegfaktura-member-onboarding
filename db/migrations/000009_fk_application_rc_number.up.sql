ALTER TABLE member_onboarding.application
    ADD CONSTRAINT fk_application_rc_number
    FOREIGN KEY (rc_number)
    REFERENCES member_onboarding.registration_entrypoint (rc_number)
    ON DELETE RESTRICT;
