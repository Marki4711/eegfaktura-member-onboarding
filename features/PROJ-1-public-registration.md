# PROJ-1: Public Registration

## Overview
Enable potential EEG members to register themselves through a public web form. The form collects member master data and metering point information, then stores it in the onboarding database for admin review.

## User Stories

### Primary User: Potential EEG Member
As a potential new EEG member, I want to access a registration link so that I can start the registration process for my specific EEG.

As a potential new EEG member, I want to fill out a form with my personal and contact information so that I can provide the required member master data.

As a potential new EEG member, I want to add one or more metering points so that I can register all my electricity meters.

As a potential new EEG member, I want to submit my application so that it gets stored for admin review.

As a potential new EEG member, I want to see confirmation that my application was submitted successfully so that I know the process worked.

## Acceptance Criteria

### Registration Link Access
- [ ] When I access a fixed registration URL for a specific EEG, I see the registration form
- [ ] The URL format is `/register/[eeg-identifier]` where eeg-identifier is provided by the EEG admin
- [ ] If the EEG identifier is invalid, I see an error message

### Member Data Collection
- [ ] I can enter my full name (first name, last name)
- [ ] I can enter my contact information (email, phone number)
- [ ] I can enter my address (street, house number, postal code, city)
- [ ] I can enter my date of birth
- [ ] All required fields are clearly marked
- [ ] I see real-time validation feedback for invalid data

### Metering Points Collection
- [ ] I can add my first metering point with meter number and address
- [ ] I can add additional metering points (up to 10)
- [ ] Each metering point requires a unique meter number
- [ ] I can remove metering points I added by mistake
- [ ] At least one metering point is required

### Form Submission
- [ ] When I submit the form with valid data, my application is saved to the database
- [ ] I see a success message with an application reference number
- [ ] I receive an email confirmation with the reference number
- [ ] The application status is set to "Submitted" for admin review

### Form Validation
- [ ] Required fields cannot be empty
- [ ] Email addresses must be valid format
- [ ] Phone numbers must be valid format
- [ ] Postal codes must be valid for the country
- [ ] Meter numbers must be unique within the application
- [ ] Date of birth must be a valid past date

## Edge Cases

### Network Issues
- If the form submission fails due to network issues, I see an error message and can retry
- My entered data is preserved if submission fails

### Duplicate Submissions
- If I submit the same application twice, I see a warning but the second submission is allowed
- Duplicate detection is based on email + meter numbers combination

### Invalid Data
- If I enter invalid data, I see specific error messages for each field
- I cannot submit until all validation errors are resolved

### Large Number of Metering Points
- If I try to add more than 10 metering points, I see an error message
- The form prevents adding more than the maximum allowed

### Browser Compatibility
- The form works in modern browsers (Chrome, Firefox, Safari, Edge)
- Mobile devices are supported with responsive design

## Dependencies
- None (this is the first feature)

## Technical Notes
- Form data is stored in `member_onboarding.application` and `member_onboarding.metering_point` tables
- Application gets a unique reference number for tracking
- Email confirmation is sent asynchronously
- No user authentication required (public form)

## Definition of Done
- [ ] Form is accessible via registration link
- [ ] All required fields are collected
- [ ] Form validation works correctly
- [ ] Multiple metering points can be added
- [ ] Successful submission saves data and shows confirmation
- [ ] Email confirmation is sent
- [ ] Application appears in admin review interface