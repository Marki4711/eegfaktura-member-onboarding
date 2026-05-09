// Package importing orchestrates the import of an approved onboarding
// application into the eegFaktura core. See features/PROJ-4-core-import.md.
package importing

import (
	"strconv"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// CoreParticipantPayload is the body posted to the eegFaktura core
// `POST /participant` endpoint. The shape follows
// github.com/eegfaktura/eegfaktura-backend/model/participant.go.
type CoreParticipantPayload struct {
	ParticipantNumber string                 `json:"participantNumber,omitempty"`
	BusinessRole      string                 `json:"businessRole"`
	Role              string                 `json:"role"`
	FirstName         string                 `json:"firstname"`
	LastName          string                 `json:"lastname"`
	TitleBefore       string                 `json:"titleBefore"`
	TitleAfter        string                 `json:"titleAfter"`
	ParticipantSince  time.Time              `json:"participantSince"`
	VatNumber         string                 `json:"vatNumber"`
	TaxNumber         string                 `json:"taxNumber"`
	CompanyRegister   string                 `json:"companyRegisterNumber"`
	Status            string                 `json:"status"`
	Contact           CoreContact            `json:"contact"`
	BillingAddress    CoreAddress            `json:"billingAddress"`
	ResidentAddress   CoreAddress            `json:"residentAddress"`
	BankAccount       CoreBankInfo           `json:"accountInfo"`
	MeteringPoint     []CoreMeteringPoint    `json:"meters"`
}

// CoreContact maps to model.ContactInfo on the core.
type CoreContact struct {
	Phone string `json:"phone,omitempty"`
	Email string `json:"email"`
}

// CoreAddress maps to model.Address on the core.
// Type is "RESIDENCE" for the resident address and "BILLING" for the billing
// address. In V1, billing == resident.
type CoreAddress struct {
	Type         string `json:"type"`
	Street       string `json:"street"`
	StreetNumber string `json:"streetNumber"`
	Zip          string `json:"zip"`
	City         string `json:"city"`
}

// CoreBankInfo maps to model.BankInfo on the core. Both fields nullable.
type CoreBankInfo struct {
	Iban  string `json:"iban,omitempty"`
	Owner string `json:"owner,omitempty"`
}

// CoreMeteringPoint maps to model.MeteringPoint on the core. The address
// fields default to the member's resident address (V1 rule from
// docs/import-mapping.md — separate metering-point addresses are not
// managed in onboarding).
type CoreMeteringPoint struct {
	MeteringPoint   string    `json:"meteringPoint"`
	Direction       string    `json:"direction"`
	Status          string    `json:"status"`
	EquipmentNumber string    `json:"equipmentNumber,omitempty"`
	EquipmentName   string    `json:"equipmentName,omitempty"`
	InverterID      string    `json:"inverterId,omitempty"`
	Street          string    `json:"street"`
	StreetNumber    string    `json:"streetNumber"`
	City            string    `json:"city"`
	Zip             string    `json:"zip"`
	RegisteredSince time.Time `json:"registeredSince"`
}

// BuildPayload converts an onboarding application + its metering points into
// the core participant payload. participantSince is the timestamp at which
// the import is performed.
func BuildPayload(app *shared.Application, meteringPoints []shared.MeteringPoint, participantSince time.Time) CoreParticipantPayload {
	residentAddress := CoreAddress{
		Type:         "RESIDENCE",
		Street:       app.ResidentStreet,
		StreetNumber: app.ResidentStreetNumber,
		Zip:          app.ResidentZip,
		City:         app.ResidentCity,
	}
	billingAddress := residentAddress
	billingAddress.Type = "BILLING"

	meters := make([]CoreMeteringPoint, 0, len(meteringPoints))
	for _, mp := range meteringPoints {
		meter := CoreMeteringPoint{
			MeteringPoint:   mp.MeteringPoint,
			Direction:       string(mp.Direction),
			Status:          "INIT",
			Street:          app.ResidentStreet,
			StreetNumber:    app.ResidentStreetNumber,
			City:            app.ResidentCity,
			Zip:             app.ResidentZip,
			RegisteredSince: participantSince,
		}
		if mp.InstallationNumber != nil {
			meter.EquipmentNumber = *mp.InstallationNumber
		}
		if mp.InstallationName != nil {
			meter.EquipmentName = *mp.InstallationName
		}
		meters = append(meters, meter)
	}

	firstName, lastName := mapPersonName(app)

	payload := CoreParticipantPayload{
		FirstName:        firstName,
		LastName:         lastName,
		TitleBefore:      derefString(app.Titel),
		ParticipantSince: participantSince,
		Status:           "NEW",
		Contact: CoreContact{
			Email: app.Email,
			Phone: derefString(app.Phone),
		},
		ResidentAddress: residentAddress,
		BillingAddress:  billingAddress,
		BankAccount: CoreBankInfo{
			Iban:  derefString(app.IBAN),
			Owner: derefString(app.AccountHolder),
		},
		MeteringPoint: meters,
	}

	if app.MemberNumber != nil {
		payload.ParticipantNumber = strconv.Itoa(*app.MemberNumber)
	}
	if app.UIDNumber != nil {
		payload.VatNumber = *app.UIDNumber
	}
	if app.RegisterNumber != nil {
		payload.CompanyRegister = *app.RegisterNumber
	}

	return payload
}

// mapPersonName produces (firstName, lastName) for the core participant
// payload. Natural-person member types (private, farmer) keep their actual
// firstname/lastname. For non-natural-person member types (company,
// association, municipality), eegFaktura's convention is to place the
// organisation name in firstName only and leave lastName empty.
func mapPersonName(app *shared.Application) (firstName, lastName string) {
	firstName = derefString(app.Firstname)
	lastName = derefString(app.Lastname)

	if isNaturalPerson(app.MemberType) {
		return firstName, lastName
	}

	// Non-natural-person: organisation name goes into firstName only.
	if companyName := derefString(app.CompanyName); companyName != "" && firstName == "" {
		firstName = companyName
	}
	return firstName, lastName
}

func isNaturalPerson(t shared.MemberType) bool {
	return t == shared.MemberTypePrivate || t == shared.MemberTypeFarmer
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

