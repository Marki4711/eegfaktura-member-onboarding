package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"html/template"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

//go:embed templates/*.html
var templateFS embed.FS

// MailService defines the contract for sending notification emails.
type MailService interface {
	SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, fieldConfig map[string]string, attachment []byte)
	SendMemberConfirmation(app *shared.Application, entrypoint *shared.RegistrationEntrypoint) error
	SendApprovalEmail(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, pdfBytes []byte, pdfFailed bool) error
}

// NoOpMailService silently drops all mail calls. Used when SMTP is not configured.
type NoOpMailService struct{}

func (n *NoOpMailService) SendSubmissionEmails(_ *shared.Application, _ []shared.MeteringPoint, _ *shared.RegistrationEntrypoint, _ map[string]string, _ []byte) {
}
func (n *NoOpMailService) SendMemberConfirmation(_ *shared.Application, _ *shared.RegistrationEntrypoint) error {
	return nil
}
func (n *NoOpMailService) SendApprovalEmail(_ *shared.Application, _ *shared.RegistrationEntrypoint, _ []byte, _ bool) error {
	return nil
}

// SMTPMailService sends HTML emails via SMTP.
type SMTPMailService struct {
	sender       Sender
	memberTpl    *template.Template
	eegTpl       *template.Template
	approvalTpl  *template.Template
	adminBaseURL string
}

// NewSMTPMailService parses the embedded templates and returns a ready service.
func NewSMTPMailService(sender Sender, adminBaseURL string) (*SMTPMailService, error) {
	memberTpl, err := template.ParseFS(templateFS, "templates/application_submitted_member.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse member template: %w", err)
	}
	eegTpl, err := template.ParseFS(templateFS, "templates/application_submitted_eeg.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse eeg template: %w", err)
	}
	approvalTpl, err := template.ParseFS(templateFS, "templates/application_approved_eeg.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse approval template: %w", err)
	}
	return &SMTPMailService{
		sender:       sender,
		memberTpl:    memberTpl,
		eegTpl:       eegTpl,
		approvalTpl:  approvalTpl,
		adminBaseURL: adminBaseURL,
	}, nil
}

type memberTemplateData struct {
	Firstname       string
	Lastname        string
	ReferenceNumber string
	HasSEPAMandate  bool
	// EEG-Daten (leer wenn nicht konfiguriert)
	EEGName         string
	EEGStreet       string
	EEGStreetNumber string
	EEGZip          string
	EEGCity         string
	CreditorID      string
	// Antragsdaten zur Überprüfung durch das Mitglied
	MemberType      string
	CompanyName     string
	UIDNumber       string
	RegisterNumber  string
	BirthDate       string
	Email           string
	Phone           string
	Street          string
	StreetNumber    string
	Zip             string
	City            string
	IBAN            string
	AccountHolder   string
	MeteringPoints  []meteringPointView
}

// meteringPointView is a resolved metering point with translated direction label.
type meteringPointView struct {
	MeteringPoint       string
	Direction           string
	ParticipationFactor int
}

// ConfigurableFieldDisplay is a resolved label-value entry for email and PDF templates.
type ConfigurableFieldDisplay struct {
	Label string
	Value string
}

type eegTemplateData struct {
	// Identifikation
	ReferenceNumber string
	SubmittedAt     string
	RCNumber        string

	// Mitgliedstyp
	MemberType string

	// Person (nur bei private / farmer)
	Firstname string
	Lastname  string
	BirthDate string

	// Unternehmen / Organisation
	CompanyName    string
	UIDNumber      string
	RegisterNumber string

	// Kontakt
	Email string
	Phone string

	// Adresse
	ResidentStreet       string
	ResidentStreetNumber string
	ResidentZip          string
	ResidentCity         string

	// Bankverbindung
	IBAN            string
	AccountHolder   string
	SepaMandateType string

	// Zählpunkte
	MeteringPoints []meteringPointView

	// Konfigurierbare Felder (gefiltert: nicht-hidden, nicht leer)
	ConfigurableFields []ConfigurableFieldDisplay

	// Admin-Link (leer wenn ADMIN_BASE_URL nicht konfiguriert)
	AdminDetailURL string
}

type approvedEEGTemplateData struct {
	MemberName      string
	ReferenceNumber string
	EEGName         string
	PDFFailed       bool
}

var configurableFieldLabels = map[string]string{
	"persons_in_household":      "Personen im Haushalt",
	"consumption_previous_year": "Verbrauch Vorjahr (kWh)",
	"consumption_forecast":      "Verbrauch Prognose (kWh)",
	"feed_in_forecast":          "Einspeisung Prognose (kWh)",
	"pv_power_kwp":              "PV-Leistung (kWp)",
	"heat_pump":                 "Wärmepumpe vorhanden",
	"electric_vehicle":          "Elektrofahrzeug vorhanden",
	"electric_hot_water":        "Warmwasser elektrisch",
	"membership_start_date":     "Beitrittsdatum",
}

var memberTypeLabels = map[string]string{
	"private":      "Privatperson",
	"farmer":       "Landwirt",
	"company":      "Unternehmen",
	"municipality": "Gemeinde",
	"association":  "Verein",
}

// buildConfigurableFields returns the display list for non-hidden configurable fields with values.
func buildConfigurableFields(app *shared.Application, fieldConfig map[string]string) []ConfigurableFieldDisplay {
	var result []ConfigurableFieldDisplay

	add := func(name, value string) {
		label, hasLabel := configurableFieldLabels[name]
		if !hasLabel {
			return
		}
		state := fieldConfig[name]
		if state == "hidden" || state == "" {
			return
		}
		if value == "" {
			return
		}
		result = append(result, ConfigurableFieldDisplay{Label: label, Value: value})
	}

	if app.HeatPump != nil {
		v := "Nein"
		if *app.HeatPump {
			v = "Ja"
		}
		add("heat_pump", v)
	}
	if app.ElectricVehicle != nil {
		v := "Nein"
		if *app.ElectricVehicle {
			v = "Ja"
		}
		add("electric_vehicle", v)
	}
	if app.ElectricHotWater != nil {
		v := "Nein"
		if *app.ElectricHotWater {
			v = "Ja"
		}
		add("electric_hot_water", v)
	}
	if app.PersonsInHousehold != nil {
		add("persons_in_household", fmt.Sprintf("%d", *app.PersonsInHousehold))
	}
	if app.ConsumptionPreviousYear != nil {
		add("consumption_previous_year", fmt.Sprintf("%d", *app.ConsumptionPreviousYear))
	}
	if app.ConsumptionForecast != nil {
		add("consumption_forecast", fmt.Sprintf("%d", *app.ConsumptionForecast))
	}
	if app.FeedInForecast != nil {
		add("feed_in_forecast", fmt.Sprintf("%d", *app.FeedInForecast))
	}
	if app.PvPowerKwp != nil {
		add("pv_power_kwp", fmt.Sprintf("%.2f", *app.PvPowerKwp))
	}
	if app.MembershipStartDate != nil {
		add("membership_start_date", app.MembershipStartDate.Format("02.01.2006"))
	}
	return result
}

func resolveSepaMandateType(app *shared.Application, ep *shared.RegistrationEntrypoint) string {
	if !app.SepaMandateAccepted {
		return "Per E-Mail"
	}
	if ep.UseCompanySEPAMandate &&
		(app.MemberType == shared.MemberTypeCompany || app.MemberType == shared.MemberTypeAssociation) {
		return "Firmenlastschrift"
	}
	return "Basislastschrift"
}

func memberDisplayName(app *shared.Application) string {
	switch app.MemberType {
	case shared.MemberTypePrivate, shared.MemberTypeFarmer:
		return strings.TrimSpace(derefString(app.Firstname) + " " + derefString(app.Lastname))
	default:
		return derefString(app.CompanyName)
	}
}

// SendSubmissionEmails sends the member confirmation and EEG notification emails.
// Errors are logged but never propagate to the caller.
func (s *SMTPMailService) SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, fieldConfig map[string]string, attachment []byte) {
	slog.Info("mail: sending submission emails", "application_id", app.ID, "ref", app.ReferenceNumber, "to", app.Email)

	memberMpViews := make([]meteringPointView, len(meteringPoints))
	for i, mp := range meteringPoints {
		dir := "Verbrauch"
		if mp.Direction == shared.DirectionProduction {
			dir = "Einspeisung"
		}
		memberMpViews[i] = meteringPointView{
			MeteringPoint:       mp.MeteringPoint,
			Direction:           dir,
			ParticipationFactor: mp.ParticipationFactor,
		}
	}

	memberBirthDate := ""
	if app.BirthDate != nil {
		memberBirthDate = app.BirthDate.Format("02.01.2006")
	}
	memberTypeLabel := memberTypeLabels[string(app.MemberType)]
	if memberTypeLabel == "" {
		memberTypeLabel = string(app.MemberType)
	}

	var memberBuf bytes.Buffer
	if err := s.memberTpl.Execute(&memberBuf, memberTemplateData{
		Firstname:       derefString(app.Firstname),
		Lastname:        derefString(app.Lastname),
		ReferenceNumber: app.ReferenceNumber,
		HasSEPAMandate:  len(attachment) > 0,
		EEGName:         derefString(entrypoint.EEGName),
		EEGStreet:       derefString(entrypoint.EEGStreet),
		EEGStreetNumber: derefString(entrypoint.EEGStreetNumber),
		EEGZip:          derefString(entrypoint.EEGZip),
		EEGCity:         derefString(entrypoint.EEGCity),
		CreditorID:      derefString(entrypoint.CreditorID),
		MemberType:      memberTypeLabel,
		CompanyName:     derefString(app.CompanyName),
		UIDNumber:       derefString(app.UIDNumber),
		RegisterNumber:  derefString(app.RegisterNumber),
		BirthDate:       memberBirthDate,
		Email:           app.Email,
		Phone:           derefString(app.Phone),
		Street:          app.ResidentStreet,
		StreetNumber:    app.ResidentStreetNumber,
		Zip:             app.ResidentZip,
		City:            app.ResidentCity,
		IBAN:            derefString(app.IBAN),
		AccountHolder:   derefString(app.AccountHolder),
		MeteringPoints:  memberMpViews,
	}); err != nil {
		slog.Error("mail: failed to render member template", "application_id", app.ID, "error", err)
	} else {
		subject := fmt.Sprintf("Ihre Beitrittserklärung wurde eingereicht (%s)", app.ReferenceNumber)
		memberHTML := memberBuf.String()
		memberPlain := htmlToText(memberHTML)
		var sendErr error
		if len(attachment) > 0 {
			sendErr = s.sender.SendWithAttachment(app.Email, subject, memberHTML, memberPlain, "sepa-lastschriftmandat.pdf", attachment)
		} else {
			sendErr = s.sender.Send(app.Email, subject, memberHTML, memberPlain)
		}
		if sendErr != nil {
			slog.Error("mail: failed to send member confirmation", "application_id", app.ID, "to", app.Email, "error", sendErr)
		} else {
			slog.Info("mail: member confirmation sent", "application_id", app.ID, "to", app.Email, "has_attachment", len(attachment) > 0)
		}
	}

	// EEG notification — only when contact_email is set
	if entrypoint.ContactEmail == nil || *entrypoint.ContactEmail == "" {
		slog.Info("mail: skipping EEG notification (no contact_email)", "application_id", app.ID, "rc_number", entrypoint.RCNumber)
		return
	}

	// Build metering point views (translated direction labels)
	mpViews := make([]meteringPointView, len(meteringPoints))
	for i, mp := range meteringPoints {
		dir := "Verbrauch"
		if mp.Direction == shared.DirectionProduction {
			dir = "Einspeisung"
		}
		mpViews[i] = meteringPointView{
			MeteringPoint:       mp.MeteringPoint,
			Direction:           dir,
			ParticipationFactor: mp.ParticipationFactor,
		}
	}

	// Admin detail link (optional)
	adminDetailURL := ""
	if s.adminBaseURL != "" {
		adminDetailURL = s.adminBaseURL + "/admin/applications/" + app.ID.String()
	}

	// Member type label (reused for EEG template)
	memberTypeLabel = memberTypeLabels[string(app.MemberType)]
	if memberTypeLabel == "" {
		memberTypeLabel = string(app.MemberType)
	}

	birthDate := ""
	if app.BirthDate != nil {
		birthDate = app.BirthDate.Format("02.01.2006")
	}

	iban := ""
	if app.IBAN != nil {
		iban = *app.IBAN
	}

	accountHolder := ""
	if app.AccountHolder != nil {
		accountHolder = *app.AccountHolder
	}

	submittedAt := time.Now().Format("02.01.2006 15:04")
	if app.SubmittedAt != nil {
		submittedAt = app.SubmittedAt.Format("02.01.2006 15:04")
	}

	tplData := eegTemplateData{
		ReferenceNumber:      app.ReferenceNumber,
		SubmittedAt:          submittedAt,
		RCNumber:             app.RCNumber,
		MemberType:           memberTypeLabel,
		Firstname:            derefString(app.Firstname),
		Lastname:             derefString(app.Lastname),
		BirthDate:            birthDate,
		CompanyName:          derefString(app.CompanyName),
		UIDNumber:            derefString(app.UIDNumber),
		RegisterNumber:       derefString(app.RegisterNumber),
		Email:                app.Email,
		Phone:                derefString(app.Phone),
		ResidentStreet:       app.ResidentStreet,
		ResidentStreetNumber: app.ResidentStreetNumber,
		ResidentZip:          app.ResidentZip,
		ResidentCity:         app.ResidentCity,
		IBAN:                 iban,
		AccountHolder:        accountHolder,
		SepaMandateType:      resolveSepaMandateType(app, entrypoint),
		MeteringPoints:       mpViews,
		ConfigurableFields:   buildConfigurableFields(app, fieldConfig),
		AdminDetailURL:       adminDetailURL,
	}

	var eegBuf bytes.Buffer
	if err := s.eegTpl.Execute(&eegBuf, tplData); err != nil {
		slog.Error("mail: failed to render EEG template", "application_id", app.ID, "error", err)
		return
	}

	displayName := memberDisplayName(app)
	subject := fmt.Sprintf("Neuer Beitrittsantrag: %s (%s)", displayName, app.ReferenceNumber)
	eegHTML := eegBuf.String()
	if err := s.sender.Send(*entrypoint.ContactEmail, subject, eegHTML, htmlToText(eegHTML)); err != nil {
		slog.Error("mail: failed to send EEG notification", "application_id", app.ID, "to", *entrypoint.ContactEmail, "error", err)
	} else {
		slog.Info("mail: EEG notification sent", "application_id", app.ID, "to", *entrypoint.ContactEmail)
	}
}

// SendMemberConfirmation sends only the member confirmation email and returns any error.
func (s *SMTPMailService) SendMemberConfirmation(app *shared.Application, entrypoint *shared.RegistrationEntrypoint) error {
	var buf bytes.Buffer
	if err := s.memberTpl.Execute(&buf, memberTemplateData{
		Firstname:       derefString(app.Firstname),
		Lastname:        derefString(app.Lastname),
		ReferenceNumber: app.ReferenceNumber,
		EEGName:         derefString(entrypoint.EEGName),
		EEGStreet:       derefString(entrypoint.EEGStreet),
		EEGStreetNumber: derefString(entrypoint.EEGStreetNumber),
		EEGZip:          derefString(entrypoint.EEGZip),
		EEGCity:         derefString(entrypoint.EEGCity),
		CreditorID:      derefString(entrypoint.CreditorID),
	}); err != nil {
		return fmt.Errorf("render member template: %w", err)
	}
	subject := fmt.Sprintf("Ihre Beitrittserklärung wurde eingereicht (%s)", app.ReferenceNumber)
	htmlBody := buf.String()
	return s.sender.Send(app.Email, subject, htmlBody, htmlToText(htmlBody))
}

// SendApprovalEmail sends the approval notification with optional PDF attachment to the EEG contact.
// Returns nil immediately when contact_email is not configured.
func (s *SMTPMailService) SendApprovalEmail(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, pdfBytes []byte, pdfFailed bool) error {
	if entrypoint.ContactEmail == nil || *entrypoint.ContactEmail == "" {
		return nil
	}

	eegName := ""
	if entrypoint.EEGName != nil {
		eegName = *entrypoint.EEGName
	}

	memberName := memberDisplayName(app)

	var buf bytes.Buffer
	if err := s.approvalTpl.Execute(&buf, approvedEEGTemplateData{
		MemberName:      memberName,
		ReferenceNumber: app.ReferenceNumber,
		EEGName:         eegName,
		PDFFailed:       pdfFailed,
	}); err != nil {
		return fmt.Errorf("render approval template: %w", err)
	}

	subject := fmt.Sprintf("Mitgliedsantrag genehmigt – %s (%s)", memberName, app.ReferenceNumber)
	filename := fmt.Sprintf("beitrittsbestaetigung-%s.pdf", app.ReferenceNumber)
	approvalHTML := buf.String()
	approvalPlain := htmlToText(approvalHTML)

	if len(pdfBytes) > 0 {
		return s.sender.SendWithAttachment(*entrypoint.ContactEmail, subject, approvalHTML, approvalPlain, filename, pdfBytes)
	}
	return s.sender.Send(*entrypoint.ContactEmail, subject, approvalHTML, approvalPlain)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var (
	reBlock    = regexp.MustCompile(`(?i)<(br\s*/?>|/?(p|h[1-6]|tr|li|div|blockquote|hr)[^>]*)>`)
	reListItem = regexp.MustCompile(`(?i)<li[^>]*>`)
	reTag      = regexp.MustCompile(`<[^>]+>`)
	reSpaces   = regexp.MustCompile(`[ \t]+`)
	reNewlines = regexp.MustCompile(`\n{3,}`)
)

// htmlToText converts an HTML email body to a plain-text alternative.
// It replaces block elements with newlines, strips remaining tags,
// and decodes HTML entities.
func htmlToText(h string) string {
	s := reListItem.ReplaceAllString(h, "\n- ")
	s = reBlock.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	// Normalise whitespace per line, then collapse excessive blank lines.
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = reSpaces.ReplaceAllString(strings.TrimSpace(l), " ")
	}
	s = strings.Join(lines, "\n")
	s = reNewlines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
