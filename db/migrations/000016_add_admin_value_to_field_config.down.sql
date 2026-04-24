ALTER TABLE member_onboarding.field_config
  DROP CONSTRAINT field_config_state_check;

ALTER TABLE member_onboarding.field_config
  DROP COLUMN IF EXISTS admin_value,
  ADD CONSTRAINT field_config_state_check
    CHECK (state IN ('hidden', 'optional', 'required'));
