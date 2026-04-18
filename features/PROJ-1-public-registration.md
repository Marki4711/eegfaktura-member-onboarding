# PROJ-1: Public Registration

## Overview
Enable potential EEG members to register themselves through a public web interface. Members can access a registration link, fill out their personal and metering point information, and submit their application for admin review.

## User Story
As a potential new EEG member, I want to register myself through a web form so that I can submit my membership application without manual admin data entry.

## Scope
This feature covers the complete public registration flow for V1:

- Load registration entry point via fixed link per EEG
- Create new application with member master data and metering points
- Update application data before submission
- Submit application for admin review

The feature includes:
- Client-side form validation
- Server-side data validation and persistence
- Status tracking and logging
- Email confirmation (async)

## Non-Goals
- Admin review interface
- Keycloak authentication integration
- Import to eegFaktura core system
- Tariff or role management
- Document upload/handling
- Multi-language support
- Advanced form features (save drafts, file uploads)

## Acceptance Criteria

### Load Registration Entry Point
- [ ] When I access `/register/{registration_slug}`, the system loads the registration configuration
- [ ] The page displays the EEG-specific title and registration form
- [ ] If the registration slug is invalid, I see a 404 error page
- [ ] If the registration is inactive, I see a 410 error page

### Create Application
- [ ] I can fill out the registration form with required personal information
- [ ] I can add one or more metering points with meter numbers and directions
- [ ] The form validates data client-side before submission
- [ ] Upon successful creation, I receive an application ID and reference number
- [ ] The application status is set to "draft"

### Update Application
- [ ] I can modify my application data while it's in "draft" status
- [ ] I can add, remove, or modify metering points
- [ ] All changes are validated before saving
- [ ] The application remains in "draft" status until submitted

### Submit Application
- [ ] I can submit my completed application
- [ ] The system validates all required data server-side
- [ ] Upon successful submission, status changes to "submitted"
- [ ] I receive a confirmation message with reference number
- [ ] An email confirmation is sent asynchronously
- [ ] The submission is logged in the status history

### Form Validation
- [ ] Required fields: firstname, lastname, email, resident address, at least one metering point
- [ ] Email format validation
- [ ] Phone number format validation (optional)
- [ ] Postal code validation for the country
- [ ] Metering point uniqueness within the application
- [ ] Privacy consent and accuracy confirmation required
- [ ] Maximum 10 metering points per application

### Data Persistence
- [ ] All application data is stored in the database
- [ ] Metering points are stored separately linked to the application
- [ ] Status changes are logged with timestamps
- [ ] Data integrity is maintained with proper constraints

## Edge Cases

### Network Issues
- If network fails during form submission, user data is preserved in the form
- Clear error messages guide users to retry
- No duplicate applications created on retry

### Invalid Data
- Server-side validation catches client-side bypass attempts
- Detailed error messages specify which fields need correction
- Form state preserved when validation fails

### Concurrent Updates
- If user has multiple tabs open, last save wins
- No data corruption from simultaneous operations

### Large Applications
- Applications with maximum metering points handled efficiently
- Form performance remains acceptable

### Email Delivery Issues
- Application submission succeeds even if email delivery fails
- Email retry logic handled asynchronously
- No user-facing errors for email failures

### Invalid Registration Links
- Clear error pages for unknown or inactive registration slugs
- No information leakage about valid slugs

## Dependencies
- None (this is the first feature)

## Affected Tables
- `member_onboarding.application` - stores application data and status
- `member_onboarding.metering_point` - stores metering point data
- `member_onboarding.status_log` - tracks status changes

## Affected API Endpoints
- `GET /api/public/registration/{registration_slug}` - load registration config
- `POST /api/public/applications` - create new application
- `PUT /api/public/applications/{id}` - update existing application
- `POST /api/public/applications/{id}/submit` - submit application

## Definition of Done
- [ ] Registration entry point loads correctly
- [ ] Application creation works with all required fields
- [ ] Application update functionality works
- [ ] Application submission validates and changes status
- [ ] All form validations work client and server-side
- [ ] Data persistence works correctly
- [ ] Status logging works for all operations
- [ ] Email confirmation is sent
- [ ] Error handling works for all edge cases
- [ ] API endpoints return correct responses
- [ ] Database constraints and indexes are in place