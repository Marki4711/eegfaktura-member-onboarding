package application

import (
	"testing"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// --- helpers ---

func strPtr(s string) *string { return &s }

func baseApp(memberType shared.MemberType) *shared.Application {
	return &shared.Application{
		MemberType:           memberType,
		Email:                "test@example.at",
		ResidentStreet:       "Teststr.",
		ResidentStreetNumber: "1",
		ResidentZip:          "4020",
		ResidentCity:         "Linz",
	}
}

// --- validateMemberTypeFields ---

func TestValidateMemberTypeFields_Private_Valid(t *testing.T) {
	app := baseApp(shared.MemberTypePrivate)
	app.Firstname = strPtr("Max")
	app.Lastname = strPtr("Mustermann")
	bd := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	app.BirthDate = &bd
	if err := validateMemberTypeFields(app); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateMemberTypeFields_Private_MissingFirstname(t *testing.T) {
	app := baseApp(shared.MemberTypePrivate)
	app.Lastname = strPtr("Mustermann")
	err := validateMemberTypeFields(app)
	if err == nil {
		t.Fatal("expected validation error for missing firstname")
	}
	ve, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, hasField := ve.Fields["firstname"]; !hasField {
		t.Errorf("expected 'firstname' field error, got fields: %v", ve.Fields)
	}
}

func TestValidateMemberTypeFields_Private_MissingLastname(t *testing.T) {
	app := baseApp(shared.MemberTypePrivate)
	app.Firstname = strPtr("Max")
	err := validateMemberTypeFields(app)
	if err == nil {
		t.Fatal("expected validation error for missing lastname")
	}
}

func TestValidateMemberTypeFields_Private_MissingBirthDate(t *testing.T) {
	app := baseApp(shared.MemberTypePrivate)
	app.Firstname = strPtr("Max")
	app.Lastname = strPtr("Mustermann")
	// BirthDate intentionally nil
	if err := validateMemberTypeFields(app); err == nil {
		t.Fatal("expected validation error for missing birthDate (BUG-1 regression)")
	}
}

func TestValidateMemberTypeFields_Private_EmptyFirstname(t *testing.T) {
	app := baseApp(shared.MemberTypePrivate)
	app.Firstname = strPtr("   ")
	app.Lastname = strPtr("Mustermann")
	if err := validateMemberTypeFields(app); err == nil {
		t.Fatal("expected validation error for whitespace-only firstname")
	}
}

func TestValidateMemberTypeFields_Farmer_Valid(t *testing.T) {
	app := baseApp(shared.MemberTypeFarmer)
	app.Firstname = strPtr("Anna")
	app.Lastname = strPtr("Bauer")
	bd := time.Date(1975, 6, 15, 0, 0, 0, 0, time.UTC)
	app.BirthDate = &bd
	if err := validateMemberTypeFields(app); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateMemberTypeFields_Municipality_Valid(t *testing.T) {
	app := baseApp(shared.MemberTypeMunicipality)
	app.CompanyName = strPtr("Gemeinde Musterort")
	if err := validateMemberTypeFields(app); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateMemberTypeFields_Municipality_UIDOptional(t *testing.T) {
	app := baseApp(shared.MemberTypeMunicipality)
	app.CompanyName = strPtr("Gemeinde Musterort")
	// No UID — must still pass
	if err := validateMemberTypeFields(app); err != nil {
		t.Fatalf("municipality UID should be optional, got: %v", err)
	}
}

func TestValidateMemberTypeFields_Municipality_MissingCompanyName(t *testing.T) {
	app := baseApp(shared.MemberTypeMunicipality)
	if err := validateMemberTypeFields(app); err == nil {
		t.Fatal("expected validation error for missing companyName")
	}
}

func TestValidateMemberTypeFields_Company_Valid(t *testing.T) {
	app := baseApp(shared.MemberTypeCompany)
	app.CompanyName = strPtr("Muster GmbH")
	app.UIDNumber = strPtr("ATU12345678")
	app.RegisterNumber = strPtr("FN 123456 a")
	if err := validateMemberTypeFields(app); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateMemberTypeFields_Company_MissingUID(t *testing.T) {
	app := baseApp(shared.MemberTypeCompany)
	app.CompanyName = strPtr("Muster GmbH")
	app.RegisterNumber = strPtr("FN 123456 a")
	if err := validateMemberTypeFields(app); err == nil {
		t.Fatal("expected validation error for missing UID")
	}
}

func TestValidateMemberTypeFields_Company_MissingRegisterNumber(t *testing.T) {
	app := baseApp(shared.MemberTypeCompany)
	app.CompanyName = strPtr("Muster GmbH")
	app.UIDNumber = strPtr("ATU12345678")
	if err := validateMemberTypeFields(app); err == nil {
		t.Fatal("expected validation error for missing registerNumber")
	}
}

func TestValidateMemberTypeFields_Company_MissingCompanyName(t *testing.T) {
	app := baseApp(shared.MemberTypeCompany)
	app.UIDNumber = strPtr("ATU12345678")
	app.RegisterNumber = strPtr("FN 123456 a")
	if err := validateMemberTypeFields(app); err == nil {
		t.Fatal("expected validation error for missing companyName")
	}
}

func TestValidateMemberTypeFields_InvalidType(t *testing.T) {
	app := baseApp(shared.MemberType("unknown"))
	if err := validateMemberTypeFields(app); err == nil {
		t.Fatal("expected validation error for unknown memberType")
	}
}

// --- clearMemberTypeFields ---

func TestClearMemberTypeFields_PersonType_ClearsOrgFields(t *testing.T) {
	app := baseApp(shared.MemberTypePrivate)
	app.Firstname = strPtr("Max")
	app.Lastname = strPtr("Muster")
	app.CompanyName = strPtr("leftover")
	app.UIDNumber = strPtr("ATU99999999")
	app.RegisterNumber = strPtr("FN 999 a")

	clearMemberTypeFields(app)

	if app.CompanyName != nil {
		t.Error("CompanyName should be nil for private type")
	}
	if app.UIDNumber != nil {
		t.Error("UIDNumber should be nil for private type")
	}
	if app.RegisterNumber != nil {
		t.Error("RegisterNumber should be nil for private type")
	}
	if app.Firstname == nil {
		t.Error("Firstname should be preserved for private type")
	}
}

func TestClearMemberTypeFields_OrgType_ClearsPersonFields(t *testing.T) {
	app := baseApp(shared.MemberTypeCompany)
	app.CompanyName = strPtr("Muster GmbH")
	app.Firstname = strPtr("leftover")
	app.Lastname = strPtr("leftover")

	clearMemberTypeFields(app)

	if app.Firstname != nil {
		t.Error("Firstname should be nil for company type")
	}
	if app.Lastname != nil {
		t.Error("Lastname should be nil for company type")
	}
	if app.CompanyName == nil {
		t.Error("CompanyName should be preserved for company type")
	}
}

func TestClearMemberTypeFields_Municipality_ClearsPersonFields(t *testing.T) {
	app := baseApp(shared.MemberTypeMunicipality)
	app.Firstname = strPtr("leftover")
	app.Lastname = strPtr("leftover")
	app.CompanyName = strPtr("Gemeinde")

	clearMemberTypeFields(app)

	if app.Firstname != nil {
		t.Error("Firstname should be nil for municipality type")
	}
	if app.CompanyName == nil {
		t.Error("CompanyName should be preserved for municipality type")
	}
}

// --- validateIBAN ---

func TestValidateIBAN_ValidAustrian(t *testing.T) {
	if !validateIBAN("AT611904300234573201") {
		t.Error("expected valid Austrian IBAN")
	}
}

func TestValidateIBAN_Invalid(t *testing.T) {
	cases := []string{"", "AT00000000000000000000", "INVALID", "AT61190430023457320X"}
	for _, c := range cases {
		if validateIBAN(c) {
			t.Errorf("expected invalid IBAN for %q", c)
		}
	}
}

func TestNormalizeIBAN_StripSpacesAndUppercase(t *testing.T) {
	got := normalizeIBAN("at61 1904 3002 3457 3201")
	want := "AT611904300234573201"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
