package importing

import (
	"context"
	"testing"

	"github.com/your-org/eegfaktura-member-onboarding/internal/coreclient"
)

type stubCoreClient struct {
	numbers []string
}

func (s *stubCoreClient) CreateParticipant(_ context.Context, _ any, _, _ string) (string, error) {
	return "", nil
}
func (s *stubCoreClient) ListTariffs(_ context.Context, _, _ string) ([]coreclient.CoreTariff, error) {
	return nil, nil
}
func (s *stubCoreClient) UpdateParticipantField(_ context.Context, _, _, _, _ string, _ any) error {
	return nil
}
func (s *stubCoreClient) ListParticipants(_ context.Context, _, _ string) ([]coreclient.CoreParticipantSummary, error) {
	out := make([]coreclient.CoreParticipantSummary, len(s.numbers))
	for i, n := range s.numbers {
		v := n
		out[i] = coreclient.CoreParticipantSummary{ID: "p", ParticipantNumber: &v}
	}
	return out, nil
}

func TestSplitTrailingDigits(t *testing.T) {
	cases := []struct {
		in     string
		prefix string
		digits string
	}{
		{"", "", ""},
		{"123", "", "123"},
		{"A005", "A", "005"},
		{"M-12", "M-", "12"},
		{"ABC", "ABC", ""},
		{"X1Y2", "X1Y", "2"},
	}
	for _, tc := range cases {
		p, d := splitTrailingDigits(tc.in)
		if p != tc.prefix || d != tc.digits {
			t.Errorf("splitTrailingDigits(%q) = (%q,%q), want (%q,%q)",
				tc.in, p, d, tc.prefix, tc.digits)
		}
	}
}

// suggestImpl exercises the pure-computation core of SuggestNextMemberNumber
// without needing a CoreClient. The actual method is a thin wrapper around
// ListParticipants + this logic.
//
// We test it via the public path with a fake CoreClient because that's the
// integration boundary; here we just sanity-check the trailing-digits split.

// TestSuggestNextMemberNumber_PatternDetection drives the full algorithm
// through a stub coreClient to lock in the dominant-pattern behaviour.
func TestSuggestNextMemberNumber_PatternDetection(t *testing.T) {
	cases := []struct {
		name     string
		existing []string
		want     string
	}{
		{"empty", []string{}, "1"},
		{"plain numeric", []string{"1", "2", "3"}, "4"},
		{"zero-padded plain", []string{"001", "002", "003"}, "004"},
		{"single A-prefix", []string{"A005"}, "A006"},
		{"A-prefix group", []string{"A001", "A002", "A005"}, "A006"},
		{"mixed: dominant wins", []string{"A001", "A002", "B999"}, "A003"},
		{"dash prefix", []string{"M-12", "M-13"}, "M-14"},
		{"non-numeric ignored", []string{"foo", "bar", "5"}, "6"},
		{"all unparseable", []string{"foo", "bar"}, "1"},
		{"grows padding", []string{"01", "99"}, "100"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &ImportService{coreClient: &stubCoreClient{numbers: tc.existing}}
			got, err := svc.SuggestNextMemberNumber(nil, "tok", "RC1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("existing=%v: got %q, want %q", tc.existing, got, tc.want)
			}
		})
	}
}
