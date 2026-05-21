ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS network_operator_customer_number,
    DROP COLUMN IF EXISTS meter_inventory_number;
