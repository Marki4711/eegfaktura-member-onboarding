package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

//go:embed templates/*.html
var templateFS embed.FS

// MailService defines the contract for sending submission notification emails.
type MailService interface {
	SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint)
}

// NoOpMailService silently drops all mail calls. Used when SMTP is not configured.
type NoOpMailService struct{}

func (n *NoOpMailService) SendSubmissionEmails(_ *shared.Application, _ []shared.MeteringPoint, _ *shared.RegistrationEntrypoint) {
}

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
func (s *SMTPMailService) SendSubmissionEmails(app *shared.Application, meteringPoints []shared.MeteringPoint, entrypoint *shared.RegistrationEntrypoint) {
	// Member confirmation
	var memberBuf bytes.Buffer
	if err := s.memberTpl.Execute(&memberBuf, memberTemplateData{
		Firstname:       app.Firstname,
		Lastname:        app.Lastname,
		ReferenceNumber: app.ReferenceNumber,
	}); err != nil {
		log.Printf("mail: failed to render member template for application %s: %v", app.ID, err)
	} else {
		subject := fmt.Sprintf("Ihre Beitrittserklärung wurde eingereicht (%s)", app.ReferenceNumber)
		if err := s.sender.Send(app.Email, subject, memberBuf.String()); err != nil {
			log.Printf("mail: failed to send member confirmation for application %s: %v", app.ID, err)
		}
	}

	// EEG notification — only when contact_email is set
	if entrypoint.ContactEmail == nil || *entrypoint.ContactEmail == "" {
		return
	}

	var eegBuf bytes.Buffer
	if err := s.eegTpl.Execute(&eegBuf, eegTemplateData{
		Firstname:       app.Firstname,
		Lastname:        app.Lastname,
		Email:           app.Email,
		ReferenceNumber: app.ReferenceNumber,
		MeteringPoints:  meteringPoints,
	}); err != nil {
		log.Printf("mail: failed to render eeg template for application %s: %v", app.ID, err)
		return
	}

	subject := fmt.Sprintf("Neuer Beitrittsantrag: %s %s (%s)", app.Firstname, app.Lastname, app.ReferenceNumber)
	if err := s.sender.Send(*entrypoint.ContactEmail, subject, eegBuf.String()); err != nil {
		log.Printf("mail: failed to send eeg notification for application %s: %v", app.ID, err)
	}
}
