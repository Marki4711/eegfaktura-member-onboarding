package importing

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestBuildPayload_BillingEqualsResident(t *testing.T) {
	now := time.Now()
	app := &shared.Application{
		ID:                   uuid.New(),
		RCNumber:             "RC101665",
		Status:               shared.StatusApproved,
		Firstname:            strPtr("Anna"),
		Lastname:             strPtr("Beispiel"),
		Email:                "anna@example.com",
		ResidentStreet:       "Hauptstraße",
		ResidentStreetNumber: "12",
		ResidentZip:          "1010",
		ResidentCity:         "Wien",
	}

	got := BuildPayload(app, nil, now, nil)

	if got.ResidentAddress.Type != "RESIDENCE" {
		t.Errorf("resident type = %q, want RESIDENCE", got.ResidentAddress.Type)
	}
	if got.BillingAddress.Type != "BILLING" {
		t.Errorf("billing type = %q, want BILLING", got.BillingAddress.Type)
	}
	if got.BillingAddress.Street != got.ResidentAddress.Street ||
		got.BillingAddress.City != got.ResidentAddress.City ||
		got.BillingAddress.Zip != got.ResidentAddress.Zip ||
		got.BillingAddress.StreetNumber != got.ResidentAddress.StreetNumber {
		t.Errorf("billing address must equal resident in V1\nresident=%+v\nbilling=%+v", got.ResidentAddress, got.BillingAddress)
	}
}

func TestBuildPayload_MetersUseResidentAddress(t *testing.T) {
	now := time.Now()
	app := &shared.Application{
		Email:                "x@example.com",
		ResidentStreet:       "Hauptstraße",
		ResidentStreetNumber: "12",
		ResidentZip:          "1010",
		ResidentCity:         "Wien",
	}
	mps := []shared.MeteringPoint{
		{ID: uuid.New(), MeteringPoint: "AT0010000000000000001000000000001", Direction: shared.DirectionConsumption, ParticipationFactor: 100},
		{ID: uuid.New(), MeteringPoint: "AT0010000000000000001000000000002", Direction: shared.DirectionProduction, ParticipationFactor: 75},
	}

	got := BuildPayload(app, mps, now, nil)

	if len(got.Meters) != 2 {
		t.Fatalf("got %d meters, want 2", len(got.Meters))
	}
	for i, m := range got.Meters {
		if m.City != app.ResidentCity || m.Zip != app.ResidentZip || m.Street != app.ResidentStreet || m.StreetNumber != app.ResidentStreetNumber {
			t.Errorf("meter %d: address must equal resident, got street=%q, city=%q", i, m.Street, m.City)
		}
		if m.Status != "INIT" {
			t.Errorf("meter %d status = %q, want INIT", i, m.Status)
		}
		if m.ProcessState != "NEW" {
			t.Errorf("meter %d processState = %q, want NEW", i, m.ProcessState)
		}
		if !m.RegisteredSince.Equal(now) {
			t.Errorf("meter %d registeredSince = %v, want %v", i, m.RegisteredSince, now)
		}
	}
	if got.Meters[0].Direction != "CONSUMPTION" {
		t.Errorf("consumption meter direction = %q, want CONSUMPTION", got.Meters[0].Direction)
	}
	if got.Meters[1].Direction != "GENERATION" {
		t.Errorf("production meter direction = %q, want GENERATION (core enum, not PRODUCTION)", got.Meters[1].Direction)
	}
	if got.Meters[0].PartFact != 100 {
		t.Errorf("consumption meter partFact = %d, want 100", got.Meters[0].PartFact)
	}
	if got.Meters[1].PartFact != 75 {
		t.Errorf("production meter partFact = %d, want 75 (from application data)", got.Meters[1].PartFact)
	}
}

func TestBuildPayload_OptionalFields(t *testing.T) {
	app := &shared.Application{
		Email:                "x@example.com",
		ResidentStreet:       "S",
		ResidentStreetNumber: "1",
		ResidentZip:          "1",
		ResidentCity:         "C",
		Firstname:            strPtr("First"),
		Lastname:             strPtr("Last"),
		Phone:                strPtr("+43 1 234"),
		Titel:                strPtr("Dr."),
		IBAN:                 strPtr("AT00 0000 0000 0000 0000"),
		AccountHolder:        strPtr("First Last"),
		MemberNumber:         strPtr("42"),
		UIDNumber:            strPtr("ATU12345678"),
		RegisterNumber:       strPtr("FN 12345 a"),
	}

	got := BuildPayload(app, nil, time.Now(), nil)

	if got.FirstName != "First" || got.LastName != "Last" {
		t.Errorf("name not mapped: got first=%q last=%q", got.FirstName, got.LastName)
	}
	if got.Contact.Phone != "+43 1 234" {
		t.Errorf("phone not mapped: %q", got.Contact.Phone)
	}
	if got.TitleBefore != "Dr." {
		t.Errorf("title not mapped: %q", got.TitleBefore)
	}
	if got.BankAccount.Iban == "" || got.BankAccount.Owner == "" {
		t.Errorf("bank info not mapped: %+v", got.BankAccount)
	}
	if got.ParticipantNumber != "42" {
		t.Errorf("participantNumber = %q, want \"42\"", got.ParticipantNumber)
	}
	if got.VatNumber != "ATU12345678" {
		t.Errorf("vatNumber = %q", got.VatNumber)
	}
	if got.CompanyRegister != "FN 12345 a" {
		t.Errorf("companyRegisterNumber = %q", got.CompanyRegister)
	}
}

func TestBuildPayload_NonPrivateMemberUsesCompanyNameInFirstNameOnly(t *testing.T) {
	app := &shared.Application{
		Email:                "office@stnikolaus.example",
		ResidentStreet:       "S",
		ResidentStreetNumber: "1",
		ResidentZip:          "1",
		ResidentCity:         "C",
		MemberType:           shared.MemberTypeMunicipality,
		Firstname:            nil, // not collected for company-type members
		Lastname:             nil,
		CompanyName:          strPtr("Gemeinde St. Nikolaus"),
	}

	got := BuildPayload(app, nil, time.Now(), nil)

	if got.FirstName != "Gemeinde St. Nikolaus" {
		t.Errorf("FirstName = %q, want %q (mapped from companyName)", got.FirstName, "Gemeinde St. Nikolaus")
	}
	if got.LastName != "" {
		t.Errorf("LastName = %q, want empty (eegFaktura convention: company name only in firstName)", got.LastName)
	}
}

func TestBuildPayload_NaturalPersonsKeepRealName(t *testing.T) {
	// Both private and farmer are natural persons — companyName must not override.
	cases := []shared.MemberType{shared.MemberTypePrivate, shared.MemberTypeFarmer}
	for _, mt := range cases {
		t.Run(string(mt), func(t *testing.T) {
			app := &shared.Application{
				Email:                "anna@example.com",
				ResidentStreet:       "S",
				ResidentStreetNumber: "1",
				ResidentZip:          "1",
				ResidentCity:         "C",
				MemberType:           mt,
				Firstname:            strPtr("Anna"),
				Lastname:             strPtr("Beispiel"),
				CompanyName:          strPtr("ignored for natural persons"),
			}
			got := BuildPayload(app, nil, time.Now(), nil)
			if got.FirstName != "Anna" || got.LastName != "Beispiel" {
				t.Errorf("%s: name not preserved: got %q %q", mt, got.FirstName, got.LastName)
			}
		})
	}
}

func TestBuildPayload_NonPrivateWithContactPerson(t *testing.T) {
	// If onboarding collected a contact person for a company, keep it.
	app := &shared.Application{
		Email:                "x@example.com",
		ResidentStreet:       "S",
		ResidentStreetNumber: "1",
		ResidentZip:          "1",
		ResidentCity:         "C",
		MemberType:           shared.MemberTypeCompany,
		Firstname:            strPtr("Harald"),
		Lastname:             strPtr("Geissler"),
		CompanyName:          strPtr("Acme GmbH"),
	}

	got := BuildPayload(app, nil, time.Now(), nil)

	if got.FirstName != "Harald" || got.LastName != "Geissler" {
		t.Errorf("contact person should be preserved: got %q %q", got.FirstName, got.LastName)
	}
}

// PROJ-28: SoleProprietor (Kleinunternehmer) — company name always wins over
// any incoming firstname (Q5). Public form does not collect firstname for
// this type, but the external API could send one; it MUST be ignored.
func TestBuildPayload_SoleProprietor_CompanyNameAlwaysInFirstName(t *testing.T) {
	app := &shared.Application{
		Email:                "x@example.com",
		ResidentStreet:       "S",
		ResidentStreetNumber: "1",
		ResidentZip:          "1",
		ResidentCity:         "C",
		MemberType:           shared.MemberTypeSoleProprietor,
		Firstname:            nil,
		Lastname:             nil,
		CompanyName:          strPtr("Maier IT"),
	}

	got := BuildPayload(app, nil, time.Now(), nil)

	if got.FirstName != "Maier IT" {
		t.Errorf("FirstName = %q, want %q (companyName for sole_proprietor)", got.FirstName, "Maier IT")
	}
	if got.LastName != "" {
		t.Errorf("LastName = %q, want empty for sole_proprietor", got.LastName)
	}
	if got.BusinessRole != "EEG_BUSINESS" {
		t.Errorf("BusinessRole = %q, want EEG_BUSINESS for sole_proprietor", got.BusinessRole)
	}
}

func TestBuildPayload_SoleProprietor_IncomingFirstnameIsIgnored(t *testing.T) {
	// Q5: incoming firstname (e.g. via the external API) must never override
	// the company name for sole_proprietor — unlike for `company` where a
	// contact person's firstname is preserved.
	app := &shared.Application{
		Email:                "x@example.com",
		ResidentStreet:       "S",
		ResidentStreetNumber: "1",
		ResidentZip:          "1",
		ResidentCity:         "C",
		MemberType:           shared.MemberTypeSoleProprietor,
		Firstname:            strPtr("Eva"),
		Lastname:             strPtr("Maier"),
		CompanyName:          strPtr("Maier IT"),
	}

	got := BuildPayload(app, nil, time.Now(), nil)

	if got.FirstName != "Maier IT" {
		t.Errorf("FirstName = %q, want %q (sole_proprietor must always use companyName)", got.FirstName, "Maier IT")
	}
	if got.LastName != "" {
		t.Errorf("LastName = %q, want empty (sole_proprietor must drop lastname)", got.LastName)
	}
}

func TestBuildPayload_BusinessRoleAndRole(t *testing.T) {
	cases := []struct {
		memberType        shared.MemberType
		wantBusinessRole  string
	}{
		{shared.MemberTypePrivate, "EEG_PRIVATE"},
		{shared.MemberTypeFarmer, "EEG_PRIVATE"},
		{shared.MemberTypeSoleProprietor, "EEG_BUSINESS"},
		{shared.MemberTypeCompany, "EEG_BUSINESS"},
		{shared.MemberTypeAssociation, "EEG_BUSINESS"},
		{shared.MemberTypeMunicipality, "EEG_BUSINESS"},
	}
	for _, tc := range cases {
		t.Run(string(tc.memberType), func(t *testing.T) {
			app := &shared.Application{
				Email:                "x@example.com",
				ResidentStreet:       "S",
				ResidentStreetNumber: "1",
				ResidentZip:          "1",
				ResidentCity:         "C",
				MemberType:           tc.memberType,
				Firstname:            strPtr("a"),
				Lastname:             strPtr("b"),
				CompanyName:          strPtr("c"),
			}
			got := BuildPayload(app, nil, time.Now(), nil)
			if got.BusinessRole != tc.wantBusinessRole {
				t.Errorf("%s: BusinessRole = %q, want %q", tc.memberType, got.BusinessRole, tc.wantBusinessRole)
			}
			if got.Role != "EEG_USER" {
				t.Errorf("%s: Role = %q, want EEG_USER", tc.memberType, got.Role)
			}
		})
	}
}

func TestBuildPayload_NilOptionalsAreEmpty(t *testing.T) {
	app := &shared.Application{
		Email:                "x@example.com",
		ResidentStreet:       "S",
		ResidentStreetNumber: "1",
		ResidentZip:          "1",
		ResidentCity:         "C",
	}

	got := BuildPayload(app, nil, time.Now(), nil)

	if got.ParticipantNumber != "" || got.VatNumber != "" || got.CompanyRegister != "" {
		t.Errorf("optional fields must be empty when nil, got %+v", got)
	}
	if got.BankAccount.Iban != "" || got.BankAccount.Owner != "" {
		t.Errorf("nil bank info must serialize to empty strings, got %+v", got.BankAccount)
	}
	if got.Contact.Phone != "" {
		t.Errorf("nil phone must be empty, got %q", got.Contact.Phone)
	}
}
