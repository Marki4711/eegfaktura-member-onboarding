package shared

import "testing"

// TestIsValidActivationMode (PROJ-53) is the source-of-truth gate that
// keeps the HTTP layer in sync with the DB CHECK constraint defined in
// migration 000048. If you add a new mode, add a constant in models.go,
// extend this validator, AND adjust the migration.
func TestIsValidActivationMode(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{ActivationModeParticipantActive, true},
		{ActivationModeAnyMeterRegistrationStarted, true},
		{"", false},
		{"PARTICIPANT_ACTIVE", false}, // case-sensitive on purpose
		{"any_meter_active", false},
		{"unknown", false},
	}
	for _, tc := range cases {
		if got := IsValidActivationMode(tc.in); got != tc.want {
			t.Errorf("IsValidActivationMode(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
