package mail

import (
	"bytes"
	"fmt"
	"time"

	gomail "github.com/wneessen/go-mail"
)

// Sender is the low-level mail delivery contract used by SMTPMailService.
// Extracting it as an interface allows test doubles to be injected.
type Sender interface {
	Send(to, subject, htmlBody, plainBody string) error
	SendWithAttachment(to, subject, htmlBody, plainBody, attachmentName string, attachmentData []byte) error
}

// Mailer wraps SMTP credentials and implements Sender.
type Mailer struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

// NewMailer creates a Mailer from the given parameters.
func NewMailer(host string, port int, user, password, from string) *Mailer {
	return &Mailer{host: host, port: port, user: user, password: password, from: from}
}

// Send delivers a multipart/alternative email with both HTML and plain-text bodies.
func (m *Mailer) Send(to, subject, htmlBody, plainBody string) error {
	msg, err := m.buildMessage(to, subject, htmlBody, plainBody)
	if err != nil {
		return err
	}
	client, err := gomail.NewClient(m.host, m.clientOpts()...)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}
	if err := client.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send mail: %w", err)
	}
	return nil
}

// SendWithAttachment delivers a multipart/alternative email with a binary attachment.
func (m *Mailer) SendWithAttachment(to, subject, htmlBody, plainBody, attachmentName string, attachmentData []byte) error {
	msg, err := m.buildMessage(to, subject, htmlBody, plainBody)
	if err != nil {
		return err
	}
	if err := msg.AttachReader(attachmentName, bytes.NewReader(attachmentData)); err != nil {
		return fmt.Errorf("failed to attach file: %w", err)
	}
	client, err := gomail.NewClient(m.host, m.clientOpts()...)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}
	if err := client.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send mail with attachment: %w", err)
	}
	return nil
}

func (m *Mailer) buildMessage(to, subject, htmlBody, plainBody string) (*gomail.Msg, error) {
	msg := gomail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return nil, fmt.Errorf("invalid from address: %w", err)
	}
	if err := msg.To(to); err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}
	msg.Subject(subject)
	// Plain text first, HTML as alternative — mail clients prefer the last listed part.
	msg.SetBodyString(gomail.TypeTextPlain, plainBody)
	msg.AddAlternativeString(gomail.TypeTextHTML, htmlBody)
	return msg, nil
}

func (m *Mailer) clientOpts() []gomail.Option {
	opts := []gomail.Option{
		gomail.WithPort(m.port),
		gomail.WithTLSPolicy(gomail.TLSMandatory),
		gomail.WithTimeout(10 * time.Second),
	}
	// Only configure SMTP auth when credentials are provided.
	// Omitting auth allows open-relay / internal SMTP servers.
	if m.user != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(m.user),
			gomail.WithPassword(m.password),
		)
	}
	return opts
}
