package dataexport

import (
	"strings"
	"testing"
)

// =====================================================================
// sanitiseFilenameSegment (Filename-Spec fix)
// =====================================================================

func TestSanitiseFilenameSegment(t *testing.T) {
	cases := map[string]string{
		"AT00001":              "AT00001",
		"Newsletter":           "Newsletter",
		"Newsletter 2026":      "Newsletter_2026", // space → underscore
		"Crm-Stammdaten":       "Crm-Stammdaten",
		"../../etc/passwd":     "______etc_passwd", // path traversal stripped (5 dots + 2 slashes → underscores, but adjacent dots collapse since '.' → '_')
		"weird\"name":          "weird_name",         // quote stripped (Content-Disposition safety)
		"":                     "export",             // empty → fallback
	}
	for in, expected := range cases {
		got := sanitiseFilenameSegment(in)
		if got != expected {
			t.Errorf("sanitiseFilenameSegment(%q): expected %q, got %q", in, expected, got)
		}
	}
}

func TestSanitiseFilenameSegment_Truncates(t *testing.T) {
	in := strings.Repeat("a", 200)
	got := sanitiseFilenameSegment(in)
	if len(got) != 64 {
		t.Errorf("expected truncation to 64, got %d", len(got))
	}
}

// =====================================================================
// detectSensitiveFields (DSGVO audit-log fix)
// =====================================================================

func TestDetectSensitiveFields_FindsIBANAndBirthDate(t *testing.T) {
	cfg := map[string]interface{}{
		"format": "xlsx",
		"columns": []interface{}{
			map[string]interface{}{"field": "firstname"},
			map[string]interface{}{"field": "iban"},
			map[string]interface{}{"field": "birth_date"},
		},
	}
	got := detectSensitiveFields(cfg)
	if len(got) != 2 {
		t.Fatalf("expected 2 sensitive fields, got %v", got)
	}
}

func TestDetectSensitiveFields_NoneWhenAbsent(t *testing.T) {
	cfg := map[string]interface{}{
		"format": "xlsx",
		"columns": []interface{}{
			map[string]interface{}{"field": "firstname"},
			map[string]interface{}{"field": "email"},
		},
	}
	if got := detectSensitiveFields(cfg); len(got) != 0 {
		t.Errorf("expected no sensitive fields, got %v", got)
	}
}

func TestDetectSensitiveFields_HandlesMalformedConfig(t *testing.T) {
	// Non-excel config shape — must not panic.
	if got := detectSensitiveFields(map[string]interface{}{"foo": "bar"}); got != nil {
		t.Errorf("expected nil for malformed config, got %v", got)
	}
}
