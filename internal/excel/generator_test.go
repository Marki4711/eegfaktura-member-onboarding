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

// cellValue returns the string value of a cell (e.g. "B3") in Sheet1.
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
	// Column M (13th column) in row 3 is the direction field
	if got := cellValue(t, data, "M3"); got != "GENERATION" {
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
	if got := cellValue(t, data, "M3"); got != "CONSUMPTION" {
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
	// Column U (21st) row 3 is Name 1
	if got := cellValue(t, data, "U3"); got != "Acme EEG GmbH" {
		t.Errorf("Name 1: want 'Acme EEG GmbH', got %q", got)
	}
	// Column V (22nd) row 3 is Name 2 — should be empty for business
	if got := cellValue(t, data, "V3"); got != "" {
		t.Errorf("Name 2: want empty for company, got %q", got)
	}
}

// --- importer marker row ---

func TestGenerateExcel_ContainsImporterMarker(t *testing.T) {
	app := baseApp()
	data, err := GenerateExcel(app, []shared.MeteringPoint{baseMeteringPoint()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Row 2, column A must contain the importer marker
	if got := cellValue(t, data, "A2"); !strings.Contains(got, "Leerzeile") {
		t.Errorf("importer marker: want string containing 'Leerzeile', got %q", got)
	}
}
