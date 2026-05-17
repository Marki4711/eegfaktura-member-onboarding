package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"html/template"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/metrics"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

//go:embed templates/*.html
var templateFS embed.FS

// MailService defines the contract for sending notification emails.
//
// SendSubmissionEmails:
//   - Always sends the member-facing confirmation mail.
//   - When emailConfirmationURL is non-empty (PROJ-31), the mail carries the
//     confirmation button and the EEG-notification mail is **deferred** until
//     the member clicks the button (the confirm-email handler then invokes
//     SendEEGNotification).
//   - When emailConfirmationURL is empty (the default / pre-PROJ-31 flow),
//     both mails are sent immediately.
type MailService interface {
	SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, fieldConfig map[string]string, attachment []byte, consents []shared.DocumentConsent, emailConfirmationURL string)
	SendEEGNotification(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, fieldConfig map[string]string)
	SendMemberConfirmation(app *shared.Application, entrypoint *shared.RegistrationEntrypoint) error
	// PROJ-41: Mail an Mitglied bei Ablehnung. Reason wird 1:1 in den
	// Mail-Body übernommen (von der Admin-Oberfläche eingegeben).
	SendRejectedNotification(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, reason string) error
	// PROJ-43: Mail an Mitglied bei Info-Anfrage. Reason ist die
	// Rückfrage des EEG-Admins, geht 1:1 in den Mail-Body.
	SendNeedsInfoNotification(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, reason string) error
	// PROJ-46 Stage B: nach erfolgreichem Import an Mitglied + EEG-Contact.
	// Member bekommt die Beitrittsbestätigungs-PDF; bei b2b ergänzt die Mail
	// einen Bank-Hinweis. EEG-Contact bekommt eine Kopie der Mail/PDF und
	// einen Status-Hinweis (warten auf Bank-Bestätigung vs. ready).
	//
	// PROJ-47: b2bMandatePDF ist optional; wenn gesetzt (nur bei
	// einzugsart=b2b), wird das B2B-Firmenlastschrift-Mandat mit eingedruckter
	// Mandatsreferenz=Mitgliedsnummer als zweiter Anhang mitgeschickt, damit
	// der Member es ausdrucken und an seine Bank weiterreichen kann.
	SendImportedNotification(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, pdfBytes []byte, pdfFailed bool, b2bMandatePDF []byte) error
	// PROJ-46 Stage B: an Mitglied beim Übergang auf 'activated'.
	SendActivatedNotification(app *shared.Application, entrypoint *shared.RegistrationEntrypoint) error
}

// NoOpMailService silently drops all mail calls. Used when SMTP is not configured.
type NoOpMailService struct{}

func (n *NoOpMailService) SendSubmissionEmails(_ *shared.Application, _ []shared.MeteringPoint, _ *shared.RegistrationEntrypoint, _ map[string]string, _ []byte, _ []shared.DocumentConsent, _ string) {
}
func (n *NoOpMailService) SendEEGNotification(_ *shared.Application, _ []shared.MeteringPoint, _ *shared.RegistrationEntrypoint, _ map[string]string) {
}
func (n *NoOpMailService) SendMemberConfirmation(_ *shared.Application, _ *shared.RegistrationEntrypoint) error {
	return nil
}
func (n *NoOpMailService) SendRejectedNotification(_ *shared.Application, _ *shared.RegistrationEntrypoint, _ string) error {
	return nil
}
func (n *NoOpMailService) SendNeedsInfoNotification(_ *shared.Application, _ *shared.RegistrationEntrypoint, _ string) error {
	return nil
}
func (n *NoOpMailService) SendImportedNotification(_ *shared.Application, _ *shared.RegistrationEntrypoint, _ []byte, _ bool, _ []byte) error {
	return nil
}
func (n *NoOpMailService) SendActivatedNotification(_ *shared.Application, _ *shared.RegistrationEntrypoint) error {
	return nil
}

// SMTPMailService sends HTML emails via SMTP.
type SMTPMailService struct {
	sender             Sender
	memberTpl          *template.Template
	eegTpl             *template.Template
	rejectedTpl        *template.Template
	needsInfoTpl       *template.Template
	importedMemberTpl  *template.Template
	importedEEGTpl     *template.Template
	activatedTpl       *template.Template
	adminBaseURL       string
}

// templateFuncs exposes display-timezone-aware formatters to every mail
// template so {{fmtDateTime .X}} / {{fmtDate .X}} render Europe/Vienna times.
var templateFuncs = template.FuncMap{
	"fmtDateTime": shared.FmtDateTime,
	"fmtDate":     shared.FmtDate,
}

// NewSMTPMailService parses the embedded templates and returns a ready service.
func NewSMTPMailService(sender Sender, adminBaseURL string) (*SMTPMailService, error) {
	memberTpl, err := template.New("application_submitted_member.html").Funcs(templateFuncs).ParseFS(templateFS, "templates/application_submitted_member.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse member template: %w", err)
	}
	eegTpl, err := template.New("application_submitted_eeg.html").Funcs(templateFuncs).ParseFS(templateFS, "templates/application_submitted_eeg.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse eeg template: %w", err)
	}
	rejectedTpl, err := template.New("application_rejected_member.html").Funcs(templateFuncs).ParseFS(templateFS, "templates/application_rejected_member.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse rejected template: %w", err)
	}
	needsInfoTpl, err := template.New("application_needs_info_member.html").Funcs(templateFuncs).ParseFS(templateFS, "templates/application_needs_info_member.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse needs-info template: %w", err)
	}
	// PROJ-46 Stage B templates.
	importedMemberTpl, err := template.New("application_imported_member.html").Funcs(templateFuncs).ParseFS(templateFS, "templates/application_imported_member.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse imported-member template: %w", err)
	}
	importedEEGTpl, err := template.New("application_imported_eeg.html").Funcs(templateFuncs).ParseFS(templateFS, "templates/application_imported_eeg.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse imported-eeg template: %w", err)
	}
	activatedTpl, err := template.New("application_activated_member.html").Funcs(templateFuncs).ParseFS(templateFS, "templates/application_activated_member.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse activated template: %w", err)
	}
	return &SMTPMailService{
		sender:            sender,
		memberTpl:         memberTpl,
		eegTpl:            eegTpl,
		rejectedTpl:       rejectedTpl,
		needsInfoTpl:      needsInfoTpl,
		importedMemberTpl: importedMemberTpl,
		importedEEGTpl:    importedEEGTpl,
		activatedTpl:      activatedTpl,
		adminBaseURL:      adminBaseURL,
	}, nil
}

type memberTemplateData struct {
	Titel           string
	TitelNach       string
	Firstname       string
	Lastname        string
	ReferenceNumber string
	HasSEPAMandate  bool
	// ShowB2BHint (PROJ-48): bei Mitgliedstyp company/municipality wird
	// in der Submit-Mail ein Hinweis-Block eingeblendet, dass bei Bedarf
	// auf Firmenlastschrift (B2B) umgestellt werden kann und die EEG
	// sich diesbezüglich meldet.
	ShowB2BHint     bool
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
	BankName        string
	MeteringPoints  []meteringPointView
	// Zustimmungen
	PrivacyAccepted     bool
	PrivacyVersion      string
	AccuracyConfirmed   bool
	SepaMandateAccepted bool
	SEPAMandateEnabled  bool
	DocumentConsents    []shared.DocumentConsent
	// E-Mail-Bestätigung (PROJ-31). Non-empty triggers the conditional
	// confirmation-button block in the mail template.
	EmailConfirmationURL string
}

// meteringPointView is a resolved metering point with translated direction label.
type meteringPointView struct {
	MeteringPoint       string
	Direction           string
	ParticipationFactor int
	// PROJ-39: abweichende Adresse je Zählpunkt (leer wenn = Mitgliederadresse).
	AddressLine string
	// PROJ-45: Erzeugungs-Zeile (Form + Speicher + Wechselrichter) für
	// PRODUCTION-Zählpunkte; leer für CONSUMPTION.
	GenerationLine string
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
	// EmailConfirmedAt (PROJ-31, optional). Wenn die EEG `require_email_confirmation=true`
	// hat, wird die EEG-Notification erst NACH dem Member-Klick versendet —
	// dann kann SubmittedAt deutlich älter sein als der Mailversand. Das
	// Template macht den Zeitversatz transparent.
	EmailConfirmedAt string
	RCNumber         string

	// Mitgliedstyp
	MemberType string

	// Person (nur bei private / farmer)
	Titel     string
	TitelNach string
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
	BankName        string
	SepaMandateType string

	// Zählpunkte
	MeteringPoints []meteringPointView

	// Konfigurierbare Felder (gefiltert: nicht-hidden, nicht leer)
	ConfigurableFields []ConfigurableFieldDisplay

	// Admin-Link (leer wenn ADMIN_BASE_URL nicht konfiguriert)
	AdminDetailURL string
}

var configurableFieldLabels = map[string]string{
	"persons_in_household":            "Personen im Haushalt",
	// PROJ-49: consumption_*, feed_in_forecast, pv_power_kwp wandern in die
	// per-MP-Tabelle (siehe FormatGenerationLine), nicht mehr hier.
	"heat_pump":                       "Wärmepumpe vorhanden",
	"electric_vehicle":                "Elektrofahrzeug vorhanden",
	"electric_vehicle_count":          "Anzahl E-Fahrzeuge",
	"electric_vehicle_annual_km":      "Jahres-Kilometer (E-Fahrzeuge)",
	"electric_hot_water":              "Warmwasser elektrisch",
	"membership_start_date":           "Beitrittsdatum",
	"network_operator_authorization":  "Netzbetreiber-Vollmacht erteilt",
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
	if app.ElectricVehicleCount != nil {
		add("electric_vehicle_count", fmt.Sprintf("%d", *app.ElectricVehicleCount))
	}
	if app.ElectricVehicleAnnualKm != nil {
		add("electric_vehicle_annual_km", fmt.Sprintf("%d km", *app.ElectricVehicleAnnualKm))
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
	// PROJ-49: consumption_*, feed_in_forecast, pv_power_kwp werden über
	// FormatGenerationLine pro Zählpunkt gerendert, nicht hier.
	if app.MembershipStartDate != nil {
		add("membership_start_date", app.MembershipStartDate.Format("02.01.2006"))
	}
	// PROJ-44: nur "Ja" rendern, default-FALSE wird unterdrückt.
	if app.NetworkOperatorAuthorization {
		add("network_operator_authorization", "Ja")
	}
	return result
}

// resolveSepaMandateType returns the human-readable SEPA-Variante for the
// EEG-submission mail. Seit PROJ-48 richtet sich die Variante allein nach
// `app.einzugsart` (Admin-Entscheidung), nicht mehr nach Mitgliedstyp +
// useCompanySEPAMandate.
func resolveSepaMandateType(app *shared.Application, ep *shared.RegistrationEntrypoint) string {
	if !ep.SEPAMandateEnabled || !app.SepaMandateAccepted {
		return "Per E-Mail"
	}
	switch app.Einzugsart {
	case "b2b":
		return "Firmenlastschrift"
	case "kein_sepa":
		return "Kein SEPA"
	default:
		return "Basislastschrift"
	}
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
//
// When emailConfirmationURL is non-empty (PROJ-31), the member mail carries
// the confirmation button and the EEG-notification mail is deferred — the
// confirm-email handler invokes SendEEGNotification once the member clicks.
func (s *SMTPMailService) SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, fieldConfig map[string]string, attachment []byte, consents []shared.DocumentConsent, emailConfirmationURL string) {
	slog.Info("mail: sending submission emails", "application_id", app.ID, "ref", app.ReferenceNumber, "to", app.Email, "confirmation_pending", emailConfirmationURL != "")

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
			AddressLine:         formatMeteringPointAddress(&meteringPoints[i]),
			GenerationLine:      FormatGenerationLine(&meteringPoints[i]),
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
		Titel:           derefString(app.Titel),
		TitelNach:       derefString(app.TitelNach),
		Firstname:       derefString(app.Firstname),
		Lastname:        derefString(app.Lastname),
		ReferenceNumber: app.ReferenceNumber,
		HasSEPAMandate:  len(attachment) > 0,
		ShowB2BHint:     app.MemberType == shared.MemberTypeCompany || app.MemberType == shared.MemberTypeMunicipality,
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
		IBAN:                derefString(app.IBAN),
		AccountHolder:       derefString(app.AccountHolder),
		BankName:            derefString(app.BankName),
		MeteringPoints:      memberMpViews,
		PrivacyAccepted:     app.PrivacyAccepted,
		PrivacyVersion:      derefString(app.PrivacyVersion),
		AccuracyConfirmed:   app.AccuracyConfirmed,
		SepaMandateAccepted:  app.SepaMandateAccepted,
		SEPAMandateEnabled:   entrypoint.SEPAMandateEnabled,
		DocumentConsents:     consents,
		EmailConfirmationURL: emailConfirmationURL,
	}); err != nil {
		slog.Error("mail: failed to render member template", "application_id", app.ID, "error", err)
	} else {
		subject := fmt.Sprintf("Ihre Beitrittserklärung wurde eingereicht (%s)", app.ReferenceNumber)
		memberHTML := memberBuf.String()
		memberPlain := htmlToText(memberHTML)
		// Reply-To = EEG contact so the member's "Reply" lands at their EEG,
		// not at the unmonitored noreply address.
		memberOpts := transactionalOpts(derefString(entrypoint.ContactEmail))
		var sendErr error
		if len(attachment) > 0 {
			sendErr = s.sender.SendWithAttachment(memberOpts, app.Email, subject, memberHTML, memberPlain, "sepa-lastschriftmandat.pdf", attachment)
		} else {
			sendErr = s.sender.Send(memberOpts, app.Email, subject, memberHTML, memberPlain)
		}
		if sendErr != nil {
			metrics.MailSentTotal.WithLabelValues("member_confirmation", "failed").Inc()
			slog.Error("mail: failed to send member confirmation", "application_id", app.ID, "to", app.Email, "error", sendErr)
		} else {
			metrics.MailSentTotal.WithLabelValues("member_confirmation", "success").Inc()
			slog.Info("mail: member confirmation sent", "application_id", app.ID, "to", app.Email, "has_attachment", len(attachment) > 0)
		}
	}

	// EEG notification is deferred when an e-mail-confirmation is pending —
	// it gets triggered by the confirm-email handler once the member clicks.
	if emailConfirmationURL != "" {
		slog.Info("mail: deferring EEG notification until e-mail confirmation", "application_id", app.ID, "rc_number", entrypoint.RCNumber)
		return
	}
	s.SendEEGNotification(app, meteringPoints, entrypoint, fieldConfig)
}

// SendEEGNotification renders + sends the EEG-facing "new application" mail.
// Called either immediately by SendSubmissionEmails (legacy flow) or by the
// confirm-email handler after the member has confirmed (PROJ-31).
func (s *SMTPMailService) SendEEGNotification(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, fieldConfig map[string]string) {
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
			AddressLine:         formatMeteringPointAddress(&meteringPoints[i]),
			GenerationLine:      FormatGenerationLine(&meteringPoints[i]),
		}
	}

	// Admin detail link (optional)
	adminDetailURL := ""
	if s.adminBaseURL != "" {
		adminDetailURL = s.adminBaseURL + "/admin/applications/" + app.ID.String()
	}

	memberTypeLabel := memberTypeLabels[string(app.MemberType)]
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

	submittedAt := shared.FmtDateTime(time.Now())
	if app.SubmittedAt != nil {
		submittedAt = shared.FmtDateTime(*app.SubmittedAt)
	}
	emailConfirmedAt := ""
	if app.EmailConfirmedAt != nil {
		emailConfirmedAt = shared.FmtDateTime(*app.EmailConfirmedAt)
	}

	tplData := eegTemplateData{
		ReferenceNumber:      app.ReferenceNumber,
		SubmittedAt:          submittedAt,
		EmailConfirmedAt:     emailConfirmedAt,
		RCNumber:             app.RCNumber,
		MemberType:           memberTypeLabel,
		Titel:                derefString(app.Titel),
		TitelNach:            derefString(app.TitelNach),
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
		BankName:             derefString(app.BankName),
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
	// Reply-To = applicant email so the EEG admin can answer the applicant
	// directly from their inbox instead of having to copy-paste the address.
	eegOpts := transactionalOpts(app.Email)
	if err := s.sender.Send(eegOpts, *entrypoint.ContactEmail, subject, eegHTML, htmlToText(eegHTML)); err != nil {
		metrics.MailSentTotal.WithLabelValues("eeg_notification", "failed").Inc()
		slog.Error("mail: failed to send EEG notification", "application_id", app.ID, "to", *entrypoint.ContactEmail, "error", err)
	} else {
		metrics.MailSentTotal.WithLabelValues("eeg_notification", "success").Inc()
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
		ShowB2BHint:     app.MemberType == shared.MemberTypeCompany || app.MemberType == shared.MemberTypeMunicipality,
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
	opts := transactionalOpts(derefString(entrypoint.ContactEmail))
	err := s.sender.Send(opts, app.Email, subject, htmlBody, htmlToText(htmlBody))
	if err != nil {
		metrics.MailSentTotal.WithLabelValues("resend", "failed").Inc()
	} else {
		metrics.MailSentTotal.WithLabelValues("resend", "success").Inc()
	}
	return err
}

// statusChangeTemplateData backs both the rejected and the needs-info
// member-facing mail templates (PROJ-41 + PROJ-43). Symmetric on purpose
// so the templates stay simple and the field shape obvious.
type statusChangeTemplateData struct {
	Firstname       string
	Lastname        string
	ReferenceNumber string
	Reason          string
	EEGName         string
	EEGStreet       string
	EEGStreetNumber string
	EEGZip          string
	EEGCity         string
}

func buildStatusChangeData(app *shared.Application, ep *shared.RegistrationEntrypoint, reason string) statusChangeTemplateData {
	return statusChangeTemplateData{
		Firstname:       derefString(app.Firstname),
		Lastname:        derefString(app.Lastname),
		ReferenceNumber: app.ReferenceNumber,
		Reason:          reason,
		EEGName:         derefString(ep.EEGName),
		EEGStreet:       derefString(ep.EEGStreet),
		EEGStreetNumber: derefString(ep.EEGStreetNumber),
		EEGZip:          derefString(ep.EEGZip),
		EEGCity:         derefString(ep.EEGCity),
	}
}

// SendRejectedNotification sends the PROJ-41 mail to the applicant
// after the admin rejected the application. Reason is the admin's
// free-text rejection reason, rendered 1:1 into the mail body.
func (s *SMTPMailService) SendRejectedNotification(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, reason string) error {
	var buf bytes.Buffer
	if err := s.rejectedTpl.Execute(&buf, buildStatusChangeData(app, entrypoint, reason)); err != nil {
		metrics.MailSentTotal.WithLabelValues("member_rejection", "failed").Inc()
		return fmt.Errorf("render rejected template: %w", err)
	}
	subject := fmt.Sprintf("Ihr Beitrittsantrag wurde abgelehnt (%s)", app.ReferenceNumber)
	htmlBody := buf.String()
	// Reply-To = EEG contact so the member's "Reply" goes to the EEG, not
	// to the noreply mailbox. Same pattern as the welcome mail.
	opts := transactionalOpts(derefString(entrypoint.ContactEmail))
	if err := s.sender.Send(opts, app.Email, subject, htmlBody, htmlToText(htmlBody)); err != nil {
		metrics.MailSentTotal.WithLabelValues("member_rejection", "failed").Inc()
		return err
	}
	metrics.MailSentTotal.WithLabelValues("member_rejection", "success").Inc()
	return nil
}

// SendNeedsInfoNotification sends the PROJ-43 mail to the applicant
// after the admin requested additional information. Reason is the
// admin's free-text request, rendered 1:1 into the mail body.
func (s *SMTPMailService) SendNeedsInfoNotification(app *shared.Application, entrypoint *shared.RegistrationEntrypoint, reason string) error {
	var buf bytes.Buffer
	if err := s.needsInfoTpl.Execute(&buf, buildStatusChangeData(app, entrypoint, reason)); err != nil {
		metrics.MailSentTotal.WithLabelValues("member_needs_info", "failed").Inc()
		return fmt.Errorf("render needs-info template: %w", err)
	}
	subject := fmt.Sprintf("Rückfragen zu Ihrem Beitrittsantrag (%s)", app.ReferenceNumber)
	htmlBody := buf.String()
	opts := transactionalOpts(derefString(entrypoint.ContactEmail))
	if err := s.sender.Send(opts, app.Email, subject, htmlBody, htmlToText(htmlBody)); err != nil {
		metrics.MailSentTotal.WithLabelValues("member_needs_info", "failed").Inc()
		return err
	}
	metrics.MailSentTotal.WithLabelValues("member_needs_info", "success").Inc()
	return nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// importedTemplateData backs both PROJ-46 imported mails (member + EEG).
// Symmetric on purpose so the templates stay parallel.
type importedTemplateData struct {
	Firstname       string
	Lastname        string
	MemberName      string // for the EEG-side header line
	ReferenceNumber string
	MemberNumber    string
	EEGName         string
	PDFFailed       bool
	IsB2B           bool
	// HasMandateAttachment (PROJ-48) — true wenn ein zweiter PDF-Anhang
	// (SEPA-Mandat) in der Mail steckt. Bei IsB2B immer true wenn das
	// B2B-Mandat erfolgreich generiert wurde; bei einzugsart=core nur
	// wenn EEG-Setting sepa_mandate_at_import=true ist (PROJ-48-Pfad).
	HasMandateAttachment bool
}

func buildImportedData(app *shared.Application, ep *shared.RegistrationEntrypoint, pdfFailed bool, hasMandate bool) importedTemplateData {
	return importedTemplateData{
		Firstname:            derefString(app.Firstname),
		Lastname:             derefString(app.Lastname),
		MemberName:           memberDisplayName(app),
		ReferenceNumber:      app.ReferenceNumber,
		MemberNumber:         derefString(app.MemberNumber),
		EEGName:              derefString(ep.EEGName),
		PDFFailed:            pdfFailed,
		IsB2B:                app.Einzugsart == "b2b",
		HasMandateAttachment: hasMandate,
	}
}

// SendImportedNotification (PROJ-46 Stage B + PROJ-47) sends two emails
// after a successful import: the Beitrittsbestätigungs-PDF goes to the
// member (with optional b2b-Bank-Hinweis), and a copy notification goes
// to the EEG contact (with optional b2b "warte auf Bank-Bestätigung"-
// Hinweis). For b2b einzugsart, an additional Firmenlastschrift-Mandat-
// PDF with eingedruckter Mandatsreferenz=Mitgliedsnummer is attached so
// the member can hand it to their bank. Both sends are best-effort:
// failures are logged + counted but do not roll back the import.
func (s *SMTPMailService) SendImportedNotification(app *shared.Application, ep *shared.RegistrationEntrypoint, pdfBytes []byte, pdfFailed bool, b2bMandatePDF []byte) error {
	data := buildImportedData(app, ep, pdfFailed, len(b2bMandatePDF) > 0)
	subject := fmt.Sprintf("Ihre Beitrittsbestätigung – Mitgliedsnummer %s", data.MemberNumber)
	filename := fmt.Sprintf("beitrittsbestaetigung-%s.pdf", app.ReferenceNumber)
	b2bFilename := fmt.Sprintf("sepa-firmenlastschrift-mandat-%s.pdf", data.MemberNumber)

	// Build the attachment list once; both member and EEG mail get the
	// same files (Beitrittsbestätigung + optional B2B-Mandat).
	attachments := []Attachment{}
	if len(pdfBytes) > 0 {
		attachments = append(attachments, Attachment{Name: filename, Data: pdfBytes})
	}
	if len(b2bMandatePDF) > 0 {
		attachments = append(attachments, Attachment{Name: b2bFilename, Data: b2bMandatePDF})
	}

	// Member mail.
	var memberBuf bytes.Buffer
	if err := s.importedMemberTpl.Execute(&memberBuf, data); err != nil {
		metrics.MailSentTotal.WithLabelValues("member_imported", "failed").Inc()
		return fmt.Errorf("render imported-member template: %w", err)
	}
	memberHTML := memberBuf.String()
	memberOpts := transactionalOpts(derefString(ep.ContactEmail))
	var memberSendErr error
	if len(attachments) > 0 {
		memberSendErr = s.sender.SendWithAttachments(memberOpts, app.Email, subject, memberHTML, htmlToText(memberHTML), attachments)
	} else {
		memberSendErr = s.sender.Send(memberOpts, app.Email, subject, memberHTML, htmlToText(memberHTML))
	}
	if memberSendErr != nil {
		metrics.MailSentTotal.WithLabelValues("member_imported", "failed").Inc()
		slog.Error("imported mail: member send failed", "application_id", app.ID, "error", memberSendErr)
	} else {
		metrics.MailSentTotal.WithLabelValues("member_imported", "success").Inc()
	}

	// EEG copy.
	if ep.ContactEmail == nil || *ep.ContactEmail == "" {
		return memberSendErr
	}
	var eegBuf bytes.Buffer
	if err := s.importedEEGTpl.Execute(&eegBuf, data); err != nil {
		metrics.MailSentTotal.WithLabelValues("eeg_imported", "failed").Inc()
		return fmt.Errorf("render imported-eeg template: %w", err)
	}
	eegHTML := eegBuf.String()
	eegSubject := fmt.Sprintf("Antrag importiert – %s (%s)", data.MemberName, app.ReferenceNumber)
	eegOpts := transactionalOpts(app.Email)
	var eegSendErr error
	if len(attachments) > 0 {
		eegSendErr = s.sender.SendWithAttachments(eegOpts, *ep.ContactEmail, eegSubject, eegHTML, htmlToText(eegHTML), attachments)
	} else {
		eegSendErr = s.sender.Send(eegOpts, *ep.ContactEmail, eegSubject, eegHTML, htmlToText(eegHTML))
	}
	if eegSendErr != nil {
		metrics.MailSentTotal.WithLabelValues("eeg_imported", "failed").Inc()
		slog.Error("imported mail: EEG send failed", "application_id", app.ID, "error", eegSendErr)
	} else {
		metrics.MailSentTotal.WithLabelValues("eeg_imported", "success").Inc()
	}
	if memberSendErr != nil {
		return memberSendErr
	}
	return eegSendErr
}

// activatedTemplateData is a minimal struct for the welcome-after-activation mail.
type activatedTemplateData struct {
	Firstname       string
	Lastname        string
	ReferenceNumber string
	MemberNumber    string
	EEGName         string
}

// SendActivatedNotification (PROJ-46 Stage B) sends a short welcome mail
// to the member when the admin (or activation-check) moves the application
// to 'activated'. Best-effort.
func (s *SMTPMailService) SendActivatedNotification(app *shared.Application, ep *shared.RegistrationEntrypoint) error {
	data := activatedTemplateData{
		Firstname:       derefString(app.Firstname),
		Lastname:        derefString(app.Lastname),
		ReferenceNumber: app.ReferenceNumber,
		MemberNumber:    derefString(app.MemberNumber),
		EEGName:         derefString(ep.EEGName),
	}
	var buf bytes.Buffer
	if err := s.activatedTpl.Execute(&buf, data); err != nil {
		metrics.MailSentTotal.WithLabelValues("member_activated", "failed").Inc()
		return fmt.Errorf("render activated template: %w", err)
	}
	subject := fmt.Sprintf("Willkommen bei %s – Ihre Mitgliedschaft ist aktiv",
		ifEmpty(data.EEGName, "Ihrer Energiegemeinschaft"))
	htmlBody := buf.String()
	opts := transactionalOpts(derefString(ep.ContactEmail))
	if err := s.sender.Send(opts, app.Email, subject, htmlBody, htmlToText(htmlBody)); err != nil {
		metrics.MailSentTotal.WithLabelValues("member_activated", "failed").Inc()
		return err
	}
	metrics.MailSentTotal.WithLabelValues("member_activated", "success").Inc()
	return nil
}

func ifEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// formatMeteringPointAddress returns "Straße Hausnummer, PLZ Ort" if the
// metering point has its own deviating address (PROJ-39), or "" if it uses
// the member's primary address.
func formatMeteringPointAddress(mp *shared.MeteringPoint) string {
	if !mp.HasDeviatingAddress() {
		return ""
	}
	street := derefString(mp.AddressStreet)
	streetNumber := derefString(mp.AddressStreetNumber)
	zip := derefString(mp.AddressZip)
	city := derefString(mp.AddressCity)
	return strings.TrimSpace(street+" "+streetNumber) + ", " + strings.TrimSpace(zip+" "+city)
}

// generationTypeLabels maps the internal generation_type token to the German
// label used on PDF, mail, and Excel.
var generationTypeLabels = map[string]string{
	"pv":      "PV",
	"hydro":   "Wasser",
	"wind":    "Wind",
	"biomass": "Biomasse",
}

// FormatGenerationLine returns a human-readable detail line for one metering
// point. Used as a sub-text under the Zählpunktnummer in mail templates and
// the approval PDF.
//
// PROJ-49: the line now carries both the generation info (PROJ-45) and the
// per-MP energy values (consumption / feed-in / PV-Leistung / Einspeiselimit).
// Examples:
//   PRODUCTION + pv: "PV 9,9 kWp, Prognose 6000 kWh/J, Speicher 10,5 kWh (Fronius), Einspeiselimit 7,0 kW"
//   PRODUCTION + wind: "Wind, Prognose 6000 kWh/J"
//   CONSUMPTION:    "Verbrauch Vorjahr 4200 kWh, Prognose 4000 kWh"
// Returns "" when no details are available for the row.
func FormatGenerationLine(mp *shared.MeteringPoint) string {
	if mp.Direction == shared.DirectionConsumption {
		var parts []string
		if mp.ConsumptionPreviousYear != nil {
			parts = append(parts, fmt.Sprintf("Verbrauch Vorjahr %d kWh", *mp.ConsumptionPreviousYear))
		}
		if mp.ConsumptionForecast != nil {
			parts = append(parts, fmt.Sprintf("Prognose %d kWh", *mp.ConsumptionForecast))
		}
		return strings.Join(parts, ", ")
	}
	if mp.Direction != shared.DirectionProduction || mp.GenerationType == nil {
		return ""
	}
	label, ok := generationTypeLabels[*mp.GenerationType]
	if !ok {
		label = *mp.GenerationType
	}
	var parts []string
	head := label
	if mp.PvPowerKwp != nil && *mp.GenerationType == "pv" {
		head = label + " " + formatKwh(*mp.PvPowerKwp) + " kWp"
	}
	parts = append(parts, head)
	if mp.FeedInForecast != nil {
		parts = append(parts, fmt.Sprintf("Prognose %d kWh/J", *mp.FeedInForecast))
	}
	// Battery / Wechselrichter sind PV-only (normalizeMeteringPointGeneration
	// im application package erzwingt das).
	if mp.BatterySizeKwh != nil {
		entry := "Speicher " + formatKwh(*mp.BatterySizeKwh) + " kWh"
		if mp.InverterManufacturer != nil && strings.TrimSpace(*mp.InverterManufacturer) != "" {
			entry += " (" + strings.TrimSpace(*mp.InverterManufacturer) + ")"
		}
		parts = append(parts, entry)
	} else if mp.InverterManufacturer != nil && strings.TrimSpace(*mp.InverterManufacturer) != "" {
		parts = append(parts, "Wechselrichter "+strings.TrimSpace(*mp.InverterManufacturer))
	}
	if mp.FeedInLimitPresent != nil && *mp.FeedInLimitPresent {
		if mp.FeedInLimitKw != nil {
			parts = append(parts, "Einspeiselimit "+formatKwh(*mp.FeedInLimitKw)+" kW")
		} else {
			parts = append(parts, "Einspeiselimit vorhanden")
		}
	}
	// PROJ-49 follow-up: Speichersteuerung-Antwort nur rendern, wenn das
	// Mitglied tatsächlich Stellung bezogen hat.
	if mp.BatteryControlAcceptable != nil {
		if *mp.BatteryControlAcceptable {
			parts = append(parts, "Speichersteuerung im Sinne der EEG: Ja")
		} else {
			parts = append(parts, "Speichersteuerung im Sinne der EEG: Nein")
		}
	}
	return strings.Join(parts, ", ")
}

// formatKwh rendert einen kWh-Wert mit deutschem Komma. 10.0 → "10", 10.5 → "10,5".
func formatKwh(v float64) string {
	s := strings.TrimRight(strings.TrimRight(strconv.FormatFloat(v, 'f', 2, 64), "0"), ".")
	return strings.ReplaceAll(s, ".", ",")
}

// transactionalOpts returns the per-message options every outgoing mail uses:
// - Auto-Submitted: auto-generated  (RFC 3834 — marks the mail as automated
//   so inbox providers like Gmail correctly classify it as transactional and
//   so auto-responders don't loop)
// - Reply-To routes "Reply" away from noreply@ and to a useful recipient:
//   the user's EEG for member confirmations, the applicant for EEG notices.
func transactionalOpts(replyTo string) Options {
	return Options{
		ReplyTo: replyTo,
		Headers: map[string]string{
			"Auto-Submitted": "auto-generated",
		},
	}
}

var (
	// Two-cell tables are our standard "Label: Value" layout. Rendering them
	// as "Label: Value" in the plain-text version dramatically reduces the
	// HTML-vs-Plain divergence that spam filters flag.
	reTwoCellRow = regexp.MustCompile(`(?is)<tr[^>]*>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*</tr>`)
	// A "spanning" row is a single TD with colspan — used by our templates
	// as section headers. Render as a standalone line.
	reSpanningRow = regexp.MustCompile(`(?is)<tr[^>]*>\s*<td[^>]*colspan=[^>]*>(.*?)</td>\s*</tr>`)
	reHead        = regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	reStyle       = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reScript      = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reAnchor      = regexp.MustCompile(`(?is)<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	reBlock       = regexp.MustCompile(`(?i)<(br\s*/?>|/?(p|h[1-6]|tr|li|div|blockquote|hr|table)[^>]*)>`)
	reListItem    = regexp.MustCompile(`(?i)<li[^>]*>`)
	reTag         = regexp.MustCompile(`<[^>]+>`)
	reSpaces      = regexp.MustCompile(`[ \t]+`)
	reNewlines    = regexp.MustCompile(`\n{3,}`)
)

// htmlToText converts an HTML email body to a plain-text alternative.
// Tuned for the templates in this package: two-cell tables become
// "Label: Value" lines, colspan rows become section headers, and links
// render as "text (url)" so the URL survives in the plain part (which
// helps Gmail's text/html similarity check). Drops <head>, <style>, and
// <script> so they don't leak into the visible plain text.
func htmlToText(h string) string {
	// 1. Strip head/style/script before tag-stripping touches their content.
	s := reHead.ReplaceAllString(h, "")
	s = reStyle.ReplaceAllString(s, "")
	s = reScript.ReplaceAllString(s, "")

	// 2. Tables: emit "label: value" for two-cell rows and a bare line for
	//    spanning section-header rows.
	s = reSpanningRow.ReplaceAllString(s, "\n\n$1\n")
	s = reTwoCellRow.ReplaceAllString(s, "\n$1: $2")

	// 3. Links → "text (url)" so the URL is visible in plain text too.
	s = reAnchor.ReplaceAllString(s, "$2 ($1)")

	// 4. Block-level elements turn into newlines; list items get a dash.
	s = reListItem.ReplaceAllString(s, "\n- ")
	s = reBlock.ReplaceAllString(s, "\n")

	// 5. Remove any remaining tags and decode entities.
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)

	// 6. Normalise whitespace per line, then collapse excessive blank lines.
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = reSpaces.ReplaceAllString(strings.TrimSpace(l), " ")
	}
	s = strings.Join(lines, "\n")
	s = reNewlines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
