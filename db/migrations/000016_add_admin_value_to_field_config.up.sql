ALTER TABLE member_onboarding.field_config
  DROP CONSTRAINT field_config_state_check;

ALTER TABLE member_onboarding.field_config
  ADD COLUMN admin_value TEXT NULL,
  ADD CONSTRAINT field_config_state_check
    CHECK (state IN ('hidden', 'optional', 'required', 'admin_only'));
