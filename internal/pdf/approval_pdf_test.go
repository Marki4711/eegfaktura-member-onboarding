package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func approvalData() ApprovalPDFData {
	bd := time.Date(1985, 3, 15, 0, 0, 0, 0, time.UTC)
	return ApprovalPDFData{
		EEGName:              "Muster Energiegemeinschaft",
		RCNumber:             "RC123456",
		ApprovedAt:           time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC),
		ReferenceNumber:      "REF-2026-001",
		MemberType:           "Privatperson",
		Firstname:            "Josef",
		Lastname:             "Muster",
		BirthDate:            &bd,
		Email:                "max.mustermann@example.org",
		Phone:                "+43 650 1234567",
		ResidentStreet:       "Testgasse",
		ResidentStreetNumber: "5",
		ResidentZip:          "8010",
		ResidentCity:         "Graz",
		IBAN:                 "AT61 1904 3002 3457 3201",
		SepaMandateType:      "Basislastschrift",
		MeteringPoints: []MeteringPointPDF{
			{MeteringPoint: "AT0031000000000000000000990022105", Direction: "Verbrauch", ParticipationFactor: 100},
		},
		Consents: []ConsentPDF{
			{Title: "Datenschutzerklärung", URL: "https://example.at/datenschutz", ConsentedAt: time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC)},
		},
		StatusLog: []StatusLogPDF{
			{FromStatus: "", ToStatus: "submitted", Timestamp: time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC)},
			{FromStatus: "submitted", ToStatus: "approved", Timestamp: time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)},
		},
	}
}

// TestFPDFApprovalGenerator_GeneratesValidPDF verifies that GenerateApproval returns a non-empty
// byte slice with valid PDF magic bytes for complete input.
func TestFPDFApprovalGenerator_GeneratesValidPDF(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	b, err := g.GenerateApproval(approvalData())
	if err != nil {
		t.Fatalf("GenerateApproval returned error: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("GenerateApproval returned empty byte slice")
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Errorf("output does not start with PDF magic bytes, got: %q", b[:min(8, len(b))])
	}
}

// TestFPDFApprovalGenerator_OutputSizeReasonable verifies the generated approval PDF is
// large enough to contain meaningful content.
func TestFPDFApprovalGenerator_OutputSizeReasonable(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	b, err := g.GenerateApproval(approvalData())
	if err != nil {
		t.Fatalf("GenerateApproval returned error: %v", err)
	}
	const minExpectedBytes = 1_500
	if len(b) < minExpectedBytes {
		t.Errorf("approval PDF too small (%d bytes), expected at least %d — content may be missing", len(b), minExpectedBytes)
	}
}

// TestFPDFApprovalGenerator_ContainsXRefTable verifies a well-formed PDF structure.
func TestFPDFApprovalGenerator_ContainsXRefTable(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	b, err := g.GenerateApproval(approvalData())
	if err != nil {
		t.Fatalf("GenerateApproval returned error: %v", err)
	}
	if !bytes.Contains(b, []byte("xref")) {
		t.Error("approval PDF missing xref table — structure may be invalid")
	}
	if !bytes.Contains(b, []byte("%"+"%EOF")) {
		t.Error("approval PDF missing end-of-file marker")
	}
}

// TestFPDFApprovalGenerator_EmptyOptionalFields verifies no crash when optional fields are zero.
func TestFPDFApprovalGenerator_EmptyOptionalFields(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	data := ApprovalPDFData{
		EEGName:    "Test EEG",
		RCNumber:   "RC0001",
		ApprovedAt: time.Now(),
		MemberType: "Privatperson",
		Firstname:  "Anna",
		Lastname:   "Beispiel",
		Email:      "anna@example.org",
		// No IBAN, no consents, no configurable fields, no phone, no birthdate
		MeteringPoints: []MeteringPointPDF{
			{MeteringPoint: "AT001234567", Direction: "Verbrauch", ParticipationFactor: 100},
		},
		StatusLog: []StatusLogPDF{
			{FromStatus: "", ToStatus: "approved", Timestamp: time.Now()},
		},
	}
	_, err := g.GenerateApproval(data)
	if err != nil {
		t.Errorf("GenerateApproval crashed on minimal data: %v", err)
	}
}

// TestFPDFApprovalGenerator_CompanyMember verifies company member data renders without error.
func TestFPDFApprovalGenerator_CompanyMember(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	data := approvalData()
	data.MemberType = "Unternehmen"
	data.Firstname = ""
	data.Lastname = ""
	data.CompanyName = "Muster GmbH"
	data.UIDNumber = "ATU12345678"
	data.RegisterNumber = "FN 123456 a"

	_, err := g.GenerateApproval(data)
	if err != nil {
		t.Errorf("GenerateApproval failed for company member: %v", err)
	}
}

// TestFPDFApprovalGenerator_UmlautsEncoded verifies German umlauts do not cause errors.
func TestFPDFApprovalGenerator_UmlautsEncoded(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	data := approvalData()
	data.EEGName = "Österreichische Energiegemeinschaft Müller & Söhne"
	data.ResidentCity = "Köln"
	data.Consents[0].Title = "Datenschutzerklärung & Nutzungsbedingungen"

	_, err := g.GenerateApproval(data)
	if err != nil {
		t.Errorf("GenerateApproval failed with umlauts: %v", err)
	}
}

// TestFPDFApprovalGenerator_WithConfigurableFields verifies configurable fields section renders.
func TestFPDFApprovalGenerator_WithConfigurableFields(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	data := approvalData()
	data.ConfigurableFields = []ConfigurableFieldPDF{
		{Label: "Wärmepumpe vorhanden", Value: "Ja"},
		{Label: "Personen im Haushalt", Value: "3"},
	}
	_, err := g.GenerateApproval(data)
	if err != nil {
		t.Errorf("GenerateApproval failed with configurable fields: %v", err)
	}
}

// TestFPDFApprovalGenerator_ReApprovalStatusLog verifies a re-approval status log
// (with import_failed entry) renders without error.
func TestFPDFApprovalGenerator_ReApprovalStatusLog(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	data := approvalData()
	data.StatusLog = []StatusLogPDF{
		{FromStatus: "", ToStatus: "submitted", Timestamp: time.Now().Add(-48 * 3600 * 1e9)},
		{FromStatus: "submitted", ToStatus: "approved", Timestamp: time.Now().Add(-24 * 3600 * 1e9)},
		{FromStatus: "approved", ToStatus: "import_failed", Timestamp: time.Now().Add(-12 * 3600 * 1e9), Reason: "Import-Fehler"},
		{FromStatus: "import_failed", ToStatus: "approved", Timestamp: time.Now()},
	}
	_, err := g.GenerateApproval(data)
	if err != nil {
		t.Errorf("GenerateApproval failed with re-approval status log: %v", err)
	}
}

// TestFPDFApprovalGenerator_DifferentFromSEPA verifies the approval PDF differs from a SEPA mandate PDF.
func TestFPDFApprovalGenerator_DifferentFromSEPA(t *testing.T) {
	sepa := NewFPDFGenerator()
	approval := NewFPDFApprovalGenerator()

	sepaBytes, err := sepa.Generate(fullData())
	if err != nil {
		t.Fatalf("SEPA Generate failed: %v", err)
	}
	approvalBytes, err := approval.GenerateApproval(approvalData())
	if err != nil {
		t.Fatalf("Approval GenerateApproval failed: %v", err)
	}
	if bytes.Equal(sepaBytes, approvalBytes) {
		t.Error("approval PDF and SEPA PDF are identical — they should differ")
	}
}

// TestFPDFApprovalGenerator_LargeStatusLog verifies multi-page rendering works for long status logs.
func TestFPDFApprovalGenerator_LargeStatusLog(t *testing.T) {
	g := NewFPDFApprovalGenerator()
	data := approvalData()
	// Add enough status log entries to force a page break
	for i := 0; i < 30; i++ {
		data.StatusLog = append(data.StatusLog, StatusLogPDF{
			FromStatus: "needs_info",
			ToStatus:   strings.Repeat("under_review", 1),
			Timestamp:  time.Now(),
			Reason:     strings.Repeat("Rückfrage beantwortet", 1),
		})
	}
	_, err := g.GenerateApproval(data)
	if err != nil {
		t.Errorf("GenerateApproval crashed with large status log: %v", err)
	}
}
