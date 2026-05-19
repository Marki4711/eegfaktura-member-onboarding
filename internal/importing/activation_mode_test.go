package importing

import (
	"testing"

	"github.com/your-org/eegfaktura-member-onboarding/internal/coreclient"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// TestShouldActivate covers the PROJ-53 mode-A vs mode-B logic in the
// activation-check batch. Verifiziert insbesondere die EDA-Stadien
// PENDING/APPROVED/ACTIVE als "Anmeldung gestartet" — gemessen am Live-
// Sample am 2026-05-19.
func TestShouldActivate(t *testing.T) {
	cases := []struct {
		name string
		mode string
		p    coreclient.CoreParticipantSummary
		want bool
	}{
		// Mode A: classic — participant.status == ACTIVE
		{
			name: "modeA: participant ACTIVE → activate",
			mode: shared.ActivationModeParticipantActive,
			p:    coreclient.CoreParticipantSummary{Status: "ACTIVE"},
			want: true,
		},
		{
			name: "modeA: participant PENDING → skip",
			mode: shared.ActivationModeParticipantActive,
			p:    coreclient.CoreParticipantSummary{Status: "PENDING"},
			want: false,
		},
		{
			name: "modeA: participant NEW → skip",
			mode: shared.ActivationModeParticipantActive,
			p:    coreclient.CoreParticipantSummary{Status: "NEW"},
			want: false,
		},
		{
			name: "modeA: meters irrelevant in modeA",
			mode: shared.ActivationModeParticipantActive,
			p: coreclient.CoreParticipantSummary{
				Status: "PENDING",
				Meters: []coreclient.CoreMeterSummary{{ProcessState: "ACTIVE"}},
			},
			want: false,
		},

		// Mode B: any meter with processState in {PENDING, APPROVED, ACTIVE}
		{
			name: "modeB: one meter ACTIVE → activate",
			mode: shared.ActivationModeAnyMeterRegistrationStarted,
			p: coreclient.CoreParticipantSummary{
				Status: "PENDING",
				Meters: []coreclient.CoreMeterSummary{{ProcessState: "ACTIVE"}},
			},
			want: true,
		},
		{
			name: "modeB: one meter APPROVED → activate (ZUSTIMMUNG_ECON case)",
			mode: shared.ActivationModeAnyMeterRegistrationStarted,
			p: coreclient.CoreParticipantSummary{
				Status: "NEW",
				Meters: []coreclient.CoreMeterSummary{
					{ProcessState: "INVALID"},
					{ProcessState: "APPROVED"},
				},
			},
			want: true,
		},
		{
			name: "modeB: one meter PENDING → activate (ANTWORT_ECON case)",
			mode: shared.ActivationModeAnyMeterRegistrationStarted,
			p: coreclient.CoreParticipantSummary{
				Meters: []coreclient.CoreMeterSummary{{ProcessState: "PENDING"}},
			},
			want: true,
		},
		{
			name: "modeB: all meters INVALID → skip (registration not started)",
			mode: shared.ActivationModeAnyMeterRegistrationStarted,
			p: coreclient.CoreParticipantSummary{
				Meters: []coreclient.CoreMeterSummary{
					{ProcessState: "INVALID"},
					{ProcessState: "INVALID"},
				},
			},
			want: false,
		},
		{
			name: "modeB: all meters INACTIVE → skip",
			mode: shared.ActivationModeAnyMeterRegistrationStarted,
			p: coreclient.CoreParticipantSummary{
				Meters: []coreclient.CoreMeterSummary{{ProcessState: "INACTIVE"}},
			},
			want: false,
		},
		{
			name: "modeB: no meters at all → skip",
			mode: shared.ActivationModeAnyMeterRegistrationStarted,
			p:    coreclient.CoreParticipantSummary{Status: "ACTIVE"},
			want: false,
		},

		// Unknown mode falls back to mode A (defensive).
		{
			name: "unknown mode → falls back to participant_active behaviour",
			mode: "garbage_value_from_db",
			p:    coreclient.CoreParticipantSummary{Status: "ACTIVE"},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldActivate(tc.mode, tc.p)
			if got != tc.want {
				t.Fatalf("shouldActivate(%q, %+v) = %v, want %v",
					tc.mode, tc.p, got, tc.want)
			}
		})
	}
}
