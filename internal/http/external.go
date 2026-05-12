package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ExternalHandler handles requests from external API integrations.
type ExternalHandler struct {
	applicationService *application.ApplicationService
	validate           *validator.Validate
}

// NewExternalHandler creates a new ExternalHandler.
func NewExternalHandler(svc *application.ApplicationService) *ExternalHandler {
	return &ExternalHandler{
		applicationService: svc,
		validate:           validator.New(),
	}
}

// externalApplicationRequest is the body for POST /api/external/v1/applications.
// Differs from CreateApplicationRequest: no rcNumber (from API key), no privacyVersion
// or accuracyConfirmed (implied by operator submitting on behalf of member).
type externalApplicationRequest struct {
	MemberType           string                       `json:"memberType"           validate:"required,oneof=private sole_proprietor farmer municipality company association"`
	Titel                *string                      `json:"titel,omitempty"      validate:"omitempty,max=50"`
	Firstname            *string                      `json:"firstname,omitempty"  validate:"omitempty,min=1,max=100"`
	Lastname             *string                      `json:"lastname,omitempty"   validate:"omitempty,min=1,max=100"`
	BirthDate            *string                      `json:"birthDate,omitempty"  validate:"omitempty,len=10"`
	CompanyName          *string                      `json:"companyName,omitempty" validate:"omitempty,min=1,max=150"`
	UIDNumber            *string                      `json:"uidNumber,omitempty"   validate:"omitempty,max=50"`
	RegisterNumber       *string                      `json:"registerNumber,omitempty" validate:"omitempty,max=50"`
	Email                string                       `json:"email"                validate:"required,email"`
	Phone                *string                      `json:"phone,omitempty"      validate:"omitempty,max=50"`
	ResidentStreet       string                       `json:"residentStreet"       validate:"required,min=1,max=255"`
	ResidentStreetNumber string                       `json:"residentStreetNumber" validate:"required,min=1,max=50"`
	ResidentZip          string                       `json:"residentZip"          validate:"required,min=1,max=20"`
	ResidentCity         string                       `json:"residentCity"         validate:"required,min=1,max=255"`
	IBAN                 string                       `json:"iban"                 validate:"required,min=15,max=50"`
	AccountHolder        string                       `json:"accountHolder"        validate:"required,min=1,max=150"`
	PrivacyAccepted      bool                         `json:"privacyAccepted"      validate:"required"`
	SepaMandateAccepted  bool                         `json:"sepaMandateAccepted"  validate:"required"`
	MeteringPoints       []shared.CreateMeteringPointRequest `json:"meteringPoints"  validate:"required,min=1,max=10,dive"`
	// Configurable fields (PROJ-8)
	MembershipStartDate     *string  `json:"membershipStartDate,omitempty" validate:"omitempty,len=10"`
	PersonsInHousehold      *int     `json:"personsInHousehold,omitempty"      validate:"omitempty,min=0"`
	ConsumptionPreviousYear *int64   `json:"consumptionPreviousYear,omitempty" validate:"omitempty,min=0"`
	ConsumptionForecast     *int64   `json:"consumptionForecast,omitempty"     validate:"omitempty,min=0"`
	FeedInForecast          *int64   `json:"feedInForecast,omitempty"          validate:"omitempty,min=0"`
	PvPowerKwp              *float64 `json:"pvPowerKwp,omitempty"              validate:"omitempty,min=0"`
	HeatPump                *bool    `json:"heatPump,omitempty"`
	ElectricVehicle         *bool    `json:"electricVehicle,omitempty"`
	ElectricHotWater        *bool    `json:"electricHotWater,omitempty"`
}

// SubmitExternalApplication handles POST /api/external/v1/applications.
// The RC number is read from the request context (set by APIKeyMiddleware).
//
// @Summary      Submit application (external)
// @Description  Creates and immediately submits a member application in a single request. Intended for ERP/third-party integrations. The EEG RC number is derived from the API key — no rc_number field in the body.
// @Tags         External
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        body  body     externalApplicationRequest  true  "Member application data"
// @Success      201   {object} map[string]string           "id and referenceNumber of the created application"
// @Failure      404   {object} shared.ErrorResponse  "RC number not found"
// @Failure      409   {object} shared.ErrorResponse  "Duplicate metering point"
// @Failure      410   {object} shared.ErrorResponse  "Registration deactivated"
// @Failure      422   {object} shared.ErrorResponse  "Validation error"
// @Failure      500   {object} shared.ErrorResponse
// @Router       /api/external/v1/applications [post]
func (h *ExternalHandler) SubmitExternalApplication(w http.ResponseWriter, r *http.Request) {
	rcNumber := ExternalRCNumberFromContext(r.Context())
	if rcNumber == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"code":    "internal_error",
			"message": "RC-Nummer konnte nicht aus dem Kontext gelesen werden.",
		})
		return
	}

	var req externalApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, shared.NewErrorResponse(
			shared.NewValidationError("Ungültiges JSON", nil),
		))
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	if !req.PrivacyAccepted {
		writeJSON(w, http.StatusUnprocessableEntity, shared.NewErrorResponse(
			shared.NewValidationError("Validation failed", map[string]string{
				"privacyAccepted": "Datenschutzerklärung muss akzeptiert werden",
			}),
		))
		return
	}
	if !req.SepaMandateAccepted {
		writeJSON(w, http.StatusUnprocessableEntity, shared.NewErrorResponse(
			shared.NewValidationError("Validation failed", map[string]string{
				"sepaMandateAccepted": "SEPA-Lastschriftmandat muss akzeptiert werden",
			}),
		))
		return
	}

	// Build CreateApplicationRequest — RC number from API key context.
	privacyVersion := "external-v1"
	accuracyConfirmed := true
	createReq := shared.CreateApplicationRequest{
		RCNumber:             strings.ToUpper(rcNumber),
		MemberType:           req.MemberType,
		Titel:                req.Titel,
		Firstname:            req.Firstname,
		Lastname:             req.Lastname,
		BirthDate:            req.BirthDate,
		CompanyName:          req.CompanyName,
		UIDNumber:            req.UIDNumber,
		RegisterNumber:       req.RegisterNumber,
		Email:                req.Email,
		Phone:                req.Phone,
		ResidentStreet:       req.ResidentStreet,
		ResidentStreetNumber: req.ResidentStreetNumber,
		ResidentZip:          req.ResidentZip,
		ResidentCity:         req.ResidentCity,
		IBAN:                 req.IBAN,
		AccountHolder:        req.AccountHolder,
		PrivacyAccepted:      true,
		PrivacyVersion:       privacyVersion,
		AccuracyConfirmed:    accuracyConfirmed,
		SepaMandateAccepted:  true,
		MeteringPoints:       req.MeteringPoints,
		// Configurable fields
		MembershipStartDate:     req.MembershipStartDate,
		PersonsInHousehold:      req.PersonsInHousehold,
		ConsumptionPreviousYear: req.ConsumptionPreviousYear,
		ConsumptionForecast:     req.ConsumptionForecast,
		FeedInForecast:          req.FeedInForecast,
		PvPowerKwp:              req.PvPowerKwp,
		HeatPump:                req.HeatPump,
		ElectricVehicle:         req.ElectricVehicle,
		ElectricHotWater:        req.ElectricHotWater,
	}

	app, err := h.applicationService.CreateApplication(createReq)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	submitted, err := h.applicationService.SubmitApplication(app.ID, nil)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":              app.ID.String(),
		"referenceNumber": submitted.ReferenceNumber,
	})
}

func (h *ExternalHandler) writeValidationError(w http.ResponseWriter, err error) {
	fields := make(map[string]string)
	if verrs, ok := err.(validator.ValidationErrors); ok {
		for _, verr := range verrs {
			field := verr.Field()
			if _, exists := fields[field]; !exists {
				fields[field] = validationMessage(verr)
			}
		}
	}
	writeJSON(w, http.StatusUnprocessableEntity, shared.NewErrorResponse(
		shared.NewValidationError("Validation failed", fields),
	))
}

func (h *ExternalHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case shared.ValidationError:
		writeJSON(w, http.StatusUnprocessableEntity, shared.NewErrorResponse(e))
	case shared.ConflictError:
		writeJSON(w, http.StatusConflict, shared.NewErrorResponse(e))
	default:
		switch err {
		case shared.ErrGone:
			writeJSON(w, http.StatusGone, shared.NewErrorResponse(shared.ErrGone))
		case shared.ErrNotFound:
			writeJSON(w, http.StatusNotFound, shared.NewErrorResponse(shared.ErrNotFound))
		default:
			slog.Error("internal error", "error", err)
			writeJSON(w, http.StatusInternalServerError, shared.NewErrorResponse(shared.ErrInternal))
		}
	}
}
