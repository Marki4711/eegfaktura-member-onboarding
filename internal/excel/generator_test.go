package excel

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// openXlsx parses xlsx bytes with excelize and returns the file.
// Caller must call f.Close().
func openXlsx(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("excelize.OpenReader: %v", err)
	}
	return f
}

// cellValue returns the string value of a cell (e.g. "B10") in Sheet1.
func cellValue(t *testing.T, data []byte, cell string) string {
	t.Helper()
	f := openXlsx(t, data)
	defer f.Close()
	val, err := f.GetCellValue("Sheet1", cell)
	if err != nil {
		t.Fatalf("GetCellValue(%s): %v", cell, err)
	}
	return val
}

func strPtr(s string) *string { return &s }

func baseApp() *shared.Application {
	now := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	return &shared.Application{
		ReferenceNumber:      "MO-2026-000001",
		RCNumber:             "RC-EEG-001",
		Status:               shared.StatusApproved,
		MemberType:           shared.MemberTypePrivate,
		Firstname:            strPtr("Maria"),
		Lastname:             strPtr("Muster"),
		Email:                "maria@example.at",
		Phone:                strPtr("+43 699 12345678"),
		ResidentStreet:       "Hauptstraße",
		ResidentStreetNumber: "5",
		ResidentZip:          "4020",
		ResidentCity:         "Linz",
		IBAN:                 strPtr("AT12 3456 7890 1234 5678"),
		AccountHolder:        strPtr("Maria Muster"),
		MembershipStartDate:  &start,
		CreatedAt:            now,
	}
}

func baseMeteringPoint() shared.MeteringPoint {
	return shared.MeteringPoint{
		MeteringPoint:       "AT0010000000000000001234567890",
		Direction:           shared.DirectionConsumption,
		ParticipationFactor: 100,
	}
}

// --- happy path ---

func TestGenerateExcel_HappyPath_PrivateMember(t *testing.T) {
	app := baseApp()
	mp := baseMeteringPoint()
	data, err := GenerateExcel(app, []shared.MeteringPoint{mp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty xlsx data")
	}
	// Verify it's a valid xlsx (parseable by excelize)
	f := openXlsx(t, data)
	defer f.Close()
}

func TestGenerateExcel_MultipleMetringPoints_ProducesOneRowEach(t *testing.T) {
	app := baseApp()
	mp1 := baseMeteringPoint()
	mp2 := baseMeteringPoint()
	mp2.MeteringPoint = "AT0010000000000000009876543210"
	mp2.Direction = shared.DirectionProduction

	data, err := GenerateExcel(app, []shared.MeteringPoint{mp1, mp2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty xlsx data")
	}
}

// --- no metering points ---

func TestGenerateExcel_NoMeteringPoints_ReturnsError(t *testing.T) {
	app := baseApp()
	_, err := GenerateExcel(app, []shared.MeteringPoint{})
	if err == nil {
		t.Fatal("expected error for empty metering points")
	}
	if !strings.Contains(err.Error(), "no metering points") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- business role mapping ---

func TestMapBusinessRole_Private(t *testing.T) {
	if got := mapBusinessRole(shared.MemberTypePrivate); got != "privat" {
		t.Errorf("private: want 'privat', got %q", got)
	}
}

func TestMapBusinessRole_Farmer(t *testing.T) {
	if got := mapBusinessRole(shared.MemberTypeFarmer); got != "privat" {
		t.Errorf("farmer: want 'privat', got %q", got)
	}
}

func TestMapBusinessRole_Company(t *testing.T) {
	if got := mapBusinessRole(shared.MemberTypeCompany); got != "business" {
		t.Errorf("company: want 'business', got %q", got)
	}
}

func TestMapBusinessRole_Association(t *testing.T) {
	if got := mapBusinessRole(shared.MemberTypeAssociation); got != "business" {
		t.Errorf("association: want 'business', got %q", got)
	}
}

func TestMapBusinessRole_Municipality(t *testing.T) {
	if got := mapBusinessRole(shared.MemberTypeMunicipality); got != "business" {
		t.Errorf("municipality: want 'business', got %q", got)
	}
}

// --- direction mapping ---

func TestDirectionMapping_Production_WritesGENERATION(t *testing.T) {
	app := baseApp()
	mp := baseMeteringPoint()
	mp.Direction = shared.DirectionProduction

	data, err := GenerateExcel(app, []shared.MeteringPoint{mp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Column M in first data row (row 10)
	if got := cellValue(t, data, "M10"); got != "GENERATION" {
		t.Errorf("direction: want 'GENERATION', got %q", got)
	}
}

func TestDirectionMapping_Consumption_WritesCONSUMPTION(t *testing.T) {
	app := baseApp()
	mp := baseMeteringPoint()
	mp.Direction = shared.DirectionConsumption

	data, err := GenerateExcel(app, []shared.MeteringPoint{mp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cellValue(t, data, "M10"); got != "CONSUMPTION" {
		t.Errorf("direction: want 'CONSUMPTION', got %q", got)
	}
}

// --- date formatting ---

func TestFormatDate(t *testing.T) {
	cases := []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), "1.4.2026"},
		{time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), "31.12.2026"},
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), "1.1.2000"},
	}
	for _, tc := range cases {
		got := formatDate(tc.t)
		if got != tc.want {
			t.Errorf("formatDate(%v) = %q, want %q", tc.t, got, tc.want)
		}
	}
}

// --- optional fields absent (nil pointer safety) ---

func TestGenerateExcel_OptionalFieldsNil_NoError(t *testing.T) {
	app := &shared.Application{
		ReferenceNumber:      "MO-2026-000002",
		RCNumber:             "RC-EEG-001",
		MemberType:           shared.MemberTypeCompany,
		CompanyName:          strPtr("Test GmbH"),
		Email:                "test@company.at",
		ResidentStreet:       "Industriestr.",
		ResidentStreetNumber: "10",
		ResidentZip:          "4020",
		ResidentCity:         "Linz",
		// all optional fields nil: Phone, IBAN, AccountHolder, UIDNumber,
		// MembershipStartDate, Firstname, Lastname
		CreatedAt: time.Now(),
	}
	mp := baseMeteringPoint()
	mp.Transformer = nil
	mp.InstallationName = nil

	_, err := GenerateExcel(app, []shared.MeteringPoint{mp})
	if err != nil {
		t.Fatalf("unexpected error with nil optional fields: %v", err)
	}
}

// --- company name takes precedence for Name 1 ---

func TestGenerateExcel_CompanyMember_Name1IsCompanyName(t *testing.T) {
	app := baseApp()
	app.MemberType = shared.MemberTypeCompany
	app.CompanyName = strPtr("Acme EEG GmbH")
	app.Firstname = nil
	app.Lastname = nil

	data, err := GenerateExcel(app, []shared.MeteringPoint{baseMeteringPoint()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Column U row 10 is Name 1 (first data row)
	if got := cellValue(t, data, "U10"); got != "Acme EEG GmbH" {
		t.Errorf("Name 1: want 'Acme EEG GmbH', got %q", got)
	}
	// Column V row 10 is Name 2 — should be empty for business
	if got := cellValue(t, data, "V10"); got != "" {
		t.Errorf("Name 2: want empty for company, got %q", got)
	}
}

// --- template structure ---

func TestGenerateExcel_TemplateHeader_MarkerRows(t *testing.T) {
	app := baseApp()
	data, err := GenerateExcel(app, []shared.MeteringPoint{baseMeteringPoint()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Rows 1–6 and 8–9 must have the importer marker in column A
	for _, cell := range []string{"A1", "A2", "A3", "A4", "A5", "A6", "A8", "A9"} {
		if got := cellValue(t, data, cell); !strings.Contains(got, "Leerzeile") {
			t.Errorf("marker row %s: want string containing 'Leerzeile', got %q", cell, got)
		}
	}
}

func TestGenerateExcel_TemplateHeader_HeaderRowAt7(t *testing.T) {
	app := baseApp()
	data, err := GenerateExcel(app, []shared.MeteringPoint{baseMeteringPoint()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Row 7 must be the header row
	if got := cellValue(t, data, "A7"); got != "Netzbetreiber" {
		t.Errorf("header row A7: want 'Netzbetreiber', got %q", got)
	}
	if got := cellValue(t, data, "B7"); got != "Gemeinschafts-ID" {
		t.Errorf("header row B7: want 'Gemeinschafts-ID', got %q", got)
	}
	if got := cellValue(t, data, "AJ7"); got != "Meter Codes" {
		t.Errorf("header row AJ7: want 'Meter Codes', got %q", got)
	}
}

func TestGenerateExcel_TemplateHeader_ErforderlichAnnotations(t *testing.T) {
	app := baseApp()
	data, err := GenerateExcel(app, []shared.MeteringPoint{baseMeteringPoint()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Row 4 must have Erforderlich in required columns
	for _, cell := range []string{"B4", "C4", "D4", "E4", "F4", "G4", "L4", "M4", "U4", "V4", "AH4"} {
		if got := cellValue(t, data, cell); got != "Erforderlich" {
			t.Errorf("required annotation %s: want 'Erforderlich', got %q", cell, got)
		}
	}
}

func TestGenerateExcel_DataStartsAtRow10(t *testing.T) {
	app := baseApp()
	mp := baseMeteringPoint()

	data, err := GenerateExcel(app, []shared.MeteringPoint{mp})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// First data row: metering point in column L, row 10
	if got := cellValue(t, data, "L10"); got != mp.MeteringPoint {
		t.Errorf("L10: want %q, got %q", mp.MeteringPoint, got)
	}
	// Row 9 (last marker row) must not contain metering point data
	if got := cellValue(t, data, "L9"); got != "" {
		t.Errorf("L9 should be empty (marker row), got %q", got)
	}
}

func TestGenerateExcel_MultipleMetringPoints_SecondRowAt11(t *testing.T) {
	app := baseApp()
	mp1 := baseMeteringPoint()
	mp2 := baseMeteringPoint()
	mp2.MeteringPoint = "AT0010000000000000009876543210"
	mp2.Direction = shared.DirectionProduction

	data, err := GenerateExcel(app, []shared.MeteringPoint{mp1, mp2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cellValue(t, data, "L10"); got != mp1.MeteringPoint {
		t.Errorf("L10: want %q, got %q", mp1.MeteringPoint, got)
	}
	if got := cellValue(t, data, "L11"); got != mp2.MeteringPoint {
		t.Errorf("L11: want %q, got %q", mp2.MeteringPoint, got)
	}
	if got := cellValue(t, data, "M11"); got != "GENERATION" {
		t.Errorf("M11 direction: want 'GENERATION', got %q", got)
	}
}
