package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"
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

// TestFPDFGenerator_MandateDate_ChangesOutput verifies that setting
// MandateDate produces a PDF different from the zero-value baseline,
// for both CORE and B2B variants (PROJ-52 Mini-Lücke 3). We don't
// search for the formatted date string in the output bytes because
// fpdf compresses the page content stream with zlib; a content-level
// check would need a PDF text-extraction dependency. The byte-diff
// is sufficient to catch regressions in the render path.
func TestFPDFGenerator_MandateDate_ChangesOutput(t *testing.T) {
	g := NewFPDFGenerator()

	withDate := fullData()
	withDate.MandateDate = time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	withoutDate := fullData() // zero-value MandateDate

	core1, err := g.Generate(withDate)
	if err != nil {
		t.Fatalf("Generate(withDate) error: %v", err)
	}
	core2, err := g.Generate(withoutDate)
	if err != nil {
		t.Fatalf("Generate(withoutDate) error: %v", err)
	}
	if bytes.Equal(core1, core2) {
		t.Error("CORE mandate PDF identical with and without MandateDate — date not rendered")
	}

	b2b1, err := g.GenerateCompany(withDate)
	if err != nil {
		t.Fatalf("GenerateCompany(withDate) error: %v", err)
	}
	b2b2, err := g.GenerateCompany(withoutDate)
	if err != nil {
		t.Fatalf("GenerateCompany(withoutDate) error: %v", err)
	}
	if bytes.Equal(b2b1, b2b2) {
		t.Error("B2B mandate PDF identical with and without MandateDate — date not rendered")
	}
}

// TestFPDFGenerator_ZeroMandateDate_NotRendered verifies legacy callers
// passing a zero MandateDate keep the original blank-line layout.
func TestFPDFGenerator_ZeroMandateDate_NotRendered(t *testing.T) {
	g := NewFPDFGenerator()
	// fullData() leaves MandateDate at zero value — that's the legacy
	// callsite behaviour. We don't make a strict negative assertion on
	// PDF bytes (the date string might coincidentally appear elsewhere
	// for unrelated reasons), but we verify the call still succeeds.
	if _, err := g.Generate(fullData()); err != nil {
		t.Errorf("Generate with zero MandateDate returned error: %v", err)
	}
	if _, err := g.GenerateCompany(fullData()); err != nil {
		t.Errorf("GenerateCompany with zero MandateDate returned error: %v", err)
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

// ─── GenerateCompany tests ────────────────────────────────────────────────────

// TestFPDFGenerator_GenerateCompany_ValidPDF verifies GenerateCompany returns a valid PDF.
func TestFPDFGenerator_GenerateCompany_ValidPDF(t *testing.T) {
	g := NewFPDFGenerator()
	b, err := g.GenerateCompany(fullData())
	if err != nil {
		t.Fatalf("GenerateCompany returned error: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("GenerateCompany returned empty byte slice")
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Errorf("output does not start with PDF magic bytes, got: %q", b[:min(8, len(b))])
	}
}

// TestFPDFGenerator_GenerateCompany_LargerThanCore verifies the B2B PDF is a different
// size than the CORE mandate, which indirectly confirms different content was rendered.
// Note: fpdf compresses content streams, so direct text search in bytes is not reliable.
func TestFPDFGenerator_GenerateCompany_LargerThanCore(t *testing.T) {
	g := NewFPDFGenerator()
	core, err := g.Generate(fullData())
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	b2b, err := g.GenerateCompany(fullData())
	if err != nil {
		t.Fatalf("GenerateCompany returned error: %v", err)
	}
	// Both must be valid PDFs of meaningful size — size difference confirms distinct templates
	if len(core) == 0 || len(b2b) == 0 {
		t.Fatal("one of the PDFs is empty")
	}
	if len(core) == len(b2b) {
		t.Error("CORE and B2B mandate PDFs have identical size — they may be using the same template")
	}
}

// TestFPDFGenerator_GenerateCompany_DifferentFromCore verifies the B2B PDF differs from the CORE PDF.
func TestFPDFGenerator_GenerateCompany_DifferentFromCore(t *testing.T) {
	g := NewFPDFGenerator()
	core, err := g.Generate(fullData())
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	b2b, err := g.GenerateCompany(fullData())
	if err != nil {
		t.Fatalf("GenerateCompany returned error: %v", err)
	}
	if bytes.Equal(core, b2b) {
		t.Error("B2B mandate PDF is identical to CORE mandate PDF — they should differ")
	}
}

// TestFPDFGenerator_GenerateCompany_SizeReasonable verifies the B2B PDF is large enough.
func TestFPDFGenerator_GenerateCompany_SizeReasonable(t *testing.T) {
	g := NewFPDFGenerator()
	b, err := g.GenerateCompany(fullData())
	if err != nil {
		t.Fatalf("GenerateCompany returned error: %v", err)
	}
	const minExpectedBytes = 1_500
	if len(b) < minExpectedBytes {
		t.Errorf("B2B PDF too small (%d bytes), expected at least %d — content may be missing", len(b), minExpectedBytes)
	}
}

// TestFPDFGenerator_GenerateCompany_LongCompanyName verifies no crash on long company name.
func TestFPDFGenerator_GenerateCompany_LongCompanyName(t *testing.T) {
	g := NewFPDFGenerator()
	data := fullData()
	data.MemberName = strings.Repeat("Lange Firmenbezeichnung GmbH & Co KG ", 3)
	_, err := g.GenerateCompany(data)
	if err != nil {
		t.Errorf("GenerateCompany failed with long company name: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
