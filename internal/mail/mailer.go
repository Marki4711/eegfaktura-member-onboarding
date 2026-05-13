package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	gomail "github.com/wneessen/go-mail"
)

// Options bundles per-message overrides so the service layer can attach a
// Reply-To address or custom headers (Auto-Submitted, In-Reply-To, …)
// without the mailer needing to know per-mail-type semantics.
type Options struct {
	// ReplyTo, when set, populates the Reply-To header. Improves deliverability:
	// mail clients use it for "Reply", and inbox providers count a working
	// reply path as a positive engagement signal.
	ReplyTo string
	// Headers carry extra raw headers (e.g. "Auto-Submitted":"auto-generated").
	// Standard headers like Message-ID, Date, MIME-Version, From, To, Subject
	// are already managed by go-mail.
	Headers map[string]string
}

// Sender is the low-level mail delivery contract used by SMTPMailService.
// Extracting it as an interface allows test doubles to be injected.
type Sender interface {
	Send(opts Options, to, subject, htmlBody, plainBody string) error
	SendWithAttachment(opts Options, to, subject, htmlBody, plainBody, attachmentName string, attachmentData []byte) error
}

// Mailer wraps SMTP credentials and implements Sender.
type Mailer struct {
	host     string
	port     int
	user     string
	password string
	fromAddr string
	fromName string
}

// NewMailer creates a Mailer from the given parameters. `fromName` is the
// display name that will appear before the address in mail clients (improves
// recognition + slightly improves spam scoring). Empty fromName falls back to
// just the address.
func NewMailer(host string, port int, user, password, fromAddr, fromName string) *Mailer {
	return &Mailer{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		fromAddr: fromAddr,
		fromName: fromName,
	}
}

// Send delivers a multipart/alternative email with both HTML and plain-text bodies.
func (m *Mailer) Send(opts Options, to, subject, htmlBody, plainBody string) error {
	msg, err := m.buildMessage(opts, to, subject, htmlBody, plainBody)
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
func (m *Mailer) SendWithAttachment(opts Options, to, subject, htmlBody, plainBody, attachmentName string, attachmentData []byte) error {
	msg, err := m.buildMessage(opts, to, subject, htmlBody, plainBody)
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

func (m *Mailer) buildMessage(opts Options, to, subject, htmlBody, plainBody string) (*gomail.Msg, error) {
	msg := gomail.NewMsg()
	// Use FromFormat when a display name is configured so the header reads
	// `"eegFaktura …" <noreply@…>` instead of the bare address.
	if m.fromName != "" {
		if err := msg.FromFormat(m.fromName, m.fromAddr); err != nil {
			return nil, fmt.Errorf("invalid from address: %w", err)
		}
	} else {
		if err := msg.From(m.fromAddr); err != nil {
			return nil, fmt.Errorf("invalid from address: %w", err)
		}
	}
	if err := msg.To(to); err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}
	if opts.ReplyTo != "" {
		if err := msg.ReplyTo(opts.ReplyTo); err != nil {
			return nil, fmt.Errorf("invalid reply-to address: %w", err)
		}
	}
	msg.Subject(subject)

	// SetUserAgent overwrites BOTH User-Agent and X-Mailer with our brand,
	// instead of leaking the go-mail library identifier
	// ("go-mail v0.7.2 // https://github.com/wneessen/go-mail") that some
	// spam filters specifically flag.
	msg.SetUserAgent("eegFaktura Member Onboarding")

	// Override the Message-ID so it uses the From-address domain instead of
	// gomail's default `os.Hostname()` — in our Kubernetes deployment that
	// hostname is a random pod-hash like `backend-9df68fbc9-wlsq4`, which
	// looks suspicious to spam filters that expect a real FQDN.
	msg.SetMessageIDWithValue(generateMessageID(m.fromAddr))

	for k, v := range opts.Headers {
		msg.SetGenHeader(gomail.Header(k), v)
	}

	// Plain text first, HTML as alternative — mail clients prefer the last listed part.
	msg.SetBodyString(gomail.TypeTextPlain, plainBody)
	msg.AddAlternativeString(gomail.TypeTextHTML, htmlBody)
	return msg, nil
}

// generateMessageID builds a "<random>@<domain>" Message-ID using the From
// address domain. Falls back to `localhost.invalid` if the From address has
// no @ (which would be a misconfiguration the rest of the pipeline catches).
func generateMessageID(fromAddr string) string {
	domain := "localhost.invalid"
	if i := strings.LastIndex(fromAddr, "@"); i >= 0 && i < len(fromAddr)-1 {
		domain = fromAddr[i+1:]
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// Fall back to time-based; collision risk is acceptable for a logger.
		return fmt.Sprintf("%d@%s", time.Now().UnixNano(), domain)
	}
	return fmt.Sprintf("%s@%s", hex.EncodeToString(buf), domain)
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
