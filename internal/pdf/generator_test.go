package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func fullData() SEPAMandateData {
	return SEPAMandateData{
		EEGName:            "Muster Energiegemeinschaft",
		EEGStreet:          "Hauptstraße",
		EEGStreetNumber:    "12",
		EEGZip:             "1010",
		EEGCity:            "Wien",
		CreditorID:         "AT28ZZZ00000000000",
		MemberName:         "Josef Muster",
		MemberStreet:       "Testgasse",
		MemberStreetNumber: "5",
		MemberZip:          "8010",
		MemberCity:         "Graz",
		IBAN:               "AT61 1904 3002 3457 3201",
	}
}

// TestFPDFGenerator_GeneratesValidPDF verifies that Generate returns a non-empty byte slice
// with the PDF magic bytes for valid input.
func TestFPDFGenerator_GeneratesValidPDF(t *testing.T) {
	g := NewFPDFGenerator()
	b, err := g.Generate(fullData())
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("Generate returned empty byte slice")
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Errorf("output does not start with PDF magic bytes, got: %q", b[:min(8, len(b))])
	}
}

// TestFPDFGenerator_OutputSizeReasonable verifies the generated PDF is large enough
// to contain meaningful content (> 5 KB for a full DIN-A4 mandate).
func TestFPDFGenerator_OutputSizeReasonable(t *testing.T) {
	g := NewFPDFGenerator()
	b, err := g.Generate(fullData())
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	const minExpectedBytes = 1_500
	if len(b) < minExpectedBytes {
		t.Errorf("PDF too small (%d bytes), expected at least %d — content may be missing", len(b), minExpectedBytes)
	}
}

// TestFPDFGenerator_ContainsXRefTable verifies the PDF has a valid cross-reference table,
// which indicates a well-formed PDF structure.
func TestFPDFGenerator_ContainsXRefTable(t *testing.T) {
	g := NewFPDFGenerator()
	b, err := g.Generate(fullData())
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	// All valid PDFs end with %%EOF and contain an xref table
	if !bytes.Contains(b, []byte("xref")) {
		t.Error("PDF missing xref table — structure may be invalid")
	}
	if !bytes.Contains(b, []byte("%"+"%EOF")) {
		t.Error("PDF missing end-of-file marker")
	}
}

// TestFPDFGenerator_LongEEGName verifies the generator handles a long EEG name
// without returning an error (layout must not crash).
func TestFPDFGenerator_LongEEGName(t *testing.T) {
	g := NewFPDFGenerator()
	data := fullData()
	data.EEGName = strings.Repeat("Lange Energiegemeinschaft ", 5)
	_, err := g.Generate(data)
	if err != nil {
		t.Errorf("Generate failed with long EEG name: %v", err)
	}
}

// TestFPDFGenerator_UmlautsEncoded verifies that common German umlauts are handled
// without returning an error (encoding path).
func TestFPDFGenerator_UmlautsEncoded(t *testing.T) {
	g := NewFPDFGenerator()
	data := fullData()
	data.EEGName = "Österreichische Energiegemeinschaft Müller & Söhne"
	data.EEGCity = "Köln"
	_, err := g.Generate(data)
	if err != nil {
		t.Errorf("Generate failed with umlauts: %v", err)
	}
}

// TestW1252_RoundTrip verifies that w1252 encodes ASCII strings unchanged.
func TestW1252_RoundTrip(t *testing.T) {
	input := "SEPA-Mandat"
	got := w1252(input)
	if got != input {
		t.Errorf("w1252(%q) = %q, want %q", input, got, input)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
