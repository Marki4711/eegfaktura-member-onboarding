package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

//go:embed templates/*.html
var templateFS embed.FS

// MailService defines the contract for sending submission notification emails.
type MailService interface {
	SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, attachment []byte)
	SendMemberConfirmation(app *shared.Application) error
}

// NoOpMailService silently drops all mail calls. Used when SMTP is not configured.
type NoOpMailService struct{}

func (n *NoOpMailService) SendSubmissionEmails(_ *shared.Application, _ []shared.MeteringPoint, _ *shared.RegistrationEntrypoint, _ []byte) {
}
func (n *NoOpMailService) SendMemberConfirmation(_ *shared.Application) error { return nil }

// SMTPMailService sends HTML emails via SMTP.
type SMTPMailService struct {
	sender    Sender
	memberTpl *template.Template
	eegTpl    *template.Template
}

// NewSMTPMailService parses the embedded templates and returns a ready service.
func NewSMTPMailService(sender Sender) (*SMTPMailService, error) {
	memberTpl, err := template.ParseFS(templateFS, "templates/application_submitted_member.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse member template: %w", err)
	}
	eegTpl, err := template.ParseFS(templateFS, "templates/application_submitted_eeg.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse eeg template: %w", err)
	}
	return &SMTPMailService{sender: sender, memberTpl: memberTpl, eegTpl: eegTpl}, nil
}

type memberTemplateData struct {
	Firstname       string
	Lastname        string
	ReferenceNumber string
	HasSEPAMandate  bool
}

type eegTemplateData struct {
	Firstname       string
	Lastname        string
	Email           string
	ReferenceNumber string
	MeteringPoints  []shared.MeteringPoint
}

// SendSubmissionEmails sends the member confirmation and, if a contact email is
// configured, the EEG notification. Errors are logged but never propagate to the caller.
// If attachment is non-nil it is appended to the member confirmation email as sepa-lastschriftmandat.pdf.
func (s *SMTPMailService) SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint, attachment []byte) {
	slog.Info("mail: sending submission emails", "application_id", app.ID, "ref", app.ReferenceNumber, "to", app.Email)

	// Member confirmation
	var memberBuf bytes.Buffer
	if err := s.memberTpl.Execute(&memberBuf, memberTemplateData{
		Firstname:       derefString(app.Firstname),
		Lastname:        derefString(app.Lastname),
		ReferenceNumber: app.ReferenceNumber,
		HasSEPAMandate:  len(attachment) > 0,
	}); err != nil {
		slog.Error("mail: failed to render member template", "application_id", app.ID, "error", err)
	} else {
		subject := fmt.Sprintf("Ihre Beitrittserklärung wurde eingereicht (%s)", app.ReferenceNumber)
		var sendErr error
		if len(attachment) > 0 {
			sendErr = s.sender.SendWithAttachment(app.Email, subject, memberBuf.String(), "sepa-lastschriftmandat.pdf", attachment)
		} else {
			sendErr = s.sender.Send(app.Email, subject, memberBuf.String())
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

	firstname := derefString(app.Firstname)
	lastname := derefString(app.Lastname)

	var eegBuf bytes.Buffer
	if err := s.eegTpl.Execute(&eegBuf, eegTemplateData{
		Firstname:       firstname,
		Lastname:        lastname,
		Email:           app.Email,
		ReferenceNumber: app.ReferenceNumber,
		MeteringPoints:  meteringPoints,
	}); err != nil {
		slog.Error("mail: failed to render EEG template", "application_id", app.ID, "error", err)
		return
	}

	subject := fmt.Sprintf("Neuer Beitrittsantrag: %s %s (%s)", firstname, lastname, app.ReferenceNumber)
	if err := s.sender.Send(*entrypoint.ContactEmail, subject, eegBuf.String()); err != nil {
		slog.Error("mail: failed to send EEG notification", "application_id", app.ID, "to", *entrypoint.ContactEmail, "error", err)
	} else {
		slog.Info("mail: EEG notification sent", "application_id", app.ID, "to", *entrypoint.ContactEmail)
	}
}

// SendMemberConfirmation sends only the member confirmation email and returns any error.
func (s *SMTPMailService) SendMemberConfirmation(app *shared.Application) error {
	var buf bytes.Buffer
	if err := s.memberTpl.Execute(&buf, memberTemplateData{
		Firstname:       derefString(app.Firstname),
		Lastname:        derefString(app.Lastname),
		ReferenceNumber: app.ReferenceNumber,
	}); err != nil {
		return fmt.Errorf("render member template: %w", err)
	}
	subject := fmt.Sprintf("Ihre Beitrittserklärung wurde eingereicht (%s)", app.ReferenceNumber)
	return s.sender.Send(app.Email, subject, buf.String())
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
