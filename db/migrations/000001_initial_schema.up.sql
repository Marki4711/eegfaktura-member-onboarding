-- +migrate Up
-- Create member_onboarding schema
CREATE SCHEMA IF NOT EXISTS member_onboarding;

-- Create application table
CREATE TABLE member_onboarding.application (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_number VARCHAR(50) UNIQUE NOT NULL,
    eeg_id VARCHAR(255),
    registration_slug VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'under_review', 'needs_info', 'approved', 'rejected', 'imported', 'import_failed')),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    submitted_at TIMESTAMP WITH TIME ZONE,
    approved_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    imported_at TIMESTAMP WITH TIME ZONE,
    firstname VARCHAR(255) NOT NULL,
    lastname VARCHAR(255) NOT NULL,
    birth_date DATE,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    resident_street VARCHAR(255) NOT NULL,
    resident_street_number VARCHAR(50) NOT NULL,
    resident_zip VARCHAR(20) NOT NULL,
    resident_city VARCHAR(255) NOT NULL,
    resident_country VARCHAR(10) NOT NULL DEFAULT 'AT',
    privacy_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    privacy_version VARCHAR(50),
    privacy_accepted_at TIMESTAMP WITH TIME ZONE,
    accuracy_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    communication_consent BOOLEAN NOT NULL DEFAULT FALSE,
    reviewed_by_user_id VARCHAR(255),
    admin_note TEXT,
    needs_info_reason TEXT,
    target_participant_id VARCHAR(255),
    import_started_at TIMESTAMP WITH TIME ZONE,
    import_finished_at TIMESTAMP WITH TIME ZONE,
    import_error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for application table
CREATE INDEX idx_application_reference_number ON member_onboarding.application(reference_number);
CREATE INDEX idx_application_registration_slug ON member_onboarding.application(registration_slug);
CREATE INDEX idx_application_status ON member_onboarding.application(status);
CREATE INDEX idx_application_eeg_id ON member_onboarding.application(eeg_id);

-- Create metering_point table
CREATE TABLE member_onboarding.metering_point (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES member_onboarding.application(id) ON DELETE CASCADE,
    metering_point VARCHAR(255) NOT NULL,
    direction VARCHAR(50) NOT NULL DEFAULT 'CONSUMPTION' CHECK (direction IN ('CONSUMPTION', 'PRODUCTION')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(application_id, metering_point)
);

-- Create index for metering_point table
CREATE INDEX idx_metering_point_application_id ON member_onboarding.metering_point(application_id);

-- Create status_log table
CREATE TABLE member_onboarding.status_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES member_onboarding.application(id) ON DELETE CASCADE,
    from_status VARCHAR(50),
    to_status VARCHAR(50) NOT NULL,
    changed_by_user_id VARCHAR(255),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index for status_log table
CREATE INDEX idx_status_log_application_id ON member_onboarding.status_log(application_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION member_onboarding.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_application_updated_at BEFORE UPDATE ON member_onboarding.application FOR EACH ROW EXECUTE FUNCTION member_onboarding.update_updated_at_column();
CREATE TRIGGER update_metering_point_updated_at BEFORE UPDATE ON member_onboarding.metering_point FOR EACH ROW EXECUTE FUNCTION member_onboarding.update_updated_at_column();

-- +migrate Down
-- Drop triggers
DROP TRIGGER IF EXISTS update_metering_point_updated_at ON member_onboarding.metering_point;
DROP TRIGGER IF EXISTS update_application_updated_at ON member_onboarding.application;

-- Drop function
DROP FUNCTION IF EXISTS member_onboarding.update_updated_at_column();

-- Drop tables
DROP TABLE IF EXISTS member_onboarding.status_log;
DROP TABLE IF EXISTS member_onboarding.metering_point;
DROP TABLE IF EXISTS member_onboarding.application;

-- Drop schema
DROP SCHEMA IF EXISTS member_onboarding;