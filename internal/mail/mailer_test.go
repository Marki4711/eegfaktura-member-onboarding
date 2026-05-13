package mail

import (
	"bytes"
	"strings"
	"testing"

	gomail "github.com/wneessen/go-mail"
)

// TestBuildMessage_MultipartStructureWithAttachment is a guard against the
// regression we found in the April 2026 production mail: a multipart/mixed
// body with only HTML + PDF and no text/plain alternative. The structure
// must be multipart/mixed { multipart/alternative { text/plain, text/html },
// application/pdf } so spam filters see both a HTML and a text body.
func TestBuildMessage_MultipartStructureWithAttachment(t *testing.T) {
	m := NewMailer("smtp.example.com", 587, "", "", "noreply@example.com", "Test")

	msg, err := m.buildMessage(Options{}, "rcpt@example.com", "Subject", "<p>HTML body</p>", "TEXT body")
	if err != nil {
		t.Fatalf("buildMessage failed: %v", err)
	}
	// Add a fake attachment so we exercise the mixed+alternative path.
	if err := msg.AttachReader("file.pdf", bytes.NewReader([]byte("%PDF-1.4 dummy"))); err != nil {
		t.Fatalf("AttachReader failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "multipart/mixed") {
		t.Errorf("expected outer multipart/mixed, got:\n%s", out)
	}
	if !strings.Contains(out, "multipart/alternative") {
		t.Errorf("expected inner multipart/alternative (body+alternative wrapping), got:\n%s", out)
	}
	if !strings.Contains(out, "text/plain") {
		t.Errorf("expected text/plain part, got:\n%s", out)
	}
	if !strings.Contains(out, "text/html") {
		t.Errorf("expected text/html part, got:\n%s", out)
	}
	if !strings.Contains(out, "application/pdf") {
		t.Errorf("expected application/pdf attachment, got:\n%s", out)
	}
}

// TestBuildMessage_HeadersForDeliverability locks in the deliverability
// headers we configured: From with display name, X-Mailer override, and
// custom headers from Options (Auto-Submitted, Reply-To).
func TestBuildMessage_HeadersForDeliverability(t *testing.T) {
	m := NewMailer("smtp.example.com", 587, "", "", "noreply@example.com", "eegFaktura Test")

	opts := Options{
		ReplyTo: "contact@eeg.example.com",
		Headers: map[string]string{
			"Auto-Submitted": "auto-generated",
		},
	}
	msg, err := m.buildMessage(opts, "rcpt@example.com", "Subject", "<p>HTML</p>", "TEXT")
	if err != nil {
		t.Fatalf("buildMessage failed: %v", err)
	}

	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	out := buf.String()

	want := []string{
		`From: "eegFaktura Test" <noreply@example.com>`,
		"Reply-To: <contact@eeg.example.com>",
		"Auto-Submitted: auto-generated",
		"X-Mailer: eegFaktura Member Onboarding",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("expected header %q in output, not found.\nFull output:\n%s", w, out)
		}
	}
}

// TestBuildMessage_UserAgentIsBranded ensures we do not leak the go-mail
// library identifier through the User-Agent header. Spam filters reward
// service-specific User-Agent strings over generic library defaults.
func TestBuildMessage_UserAgentIsBranded(t *testing.T) {
	m := NewMailer("smtp.example.com", 587, "", "", "noreply@example.com", "Test")
	msg, err := m.buildMessage(Options{}, "rcpt@example.com", "Subject", "<p>HTML</p>", "TEXT")
	if err != nil {
		t.Fatalf("buildMessage failed: %v", err)
	}
	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "User-Agent: go-mail") {
		t.Errorf("User-Agent leaks library identifier:\n%s", out)
	}
	if !strings.Contains(out, "User-Agent: eegFaktura Member Onboarding") {
		t.Errorf("expected branded User-Agent, got:\n%s", out)
	}
}

// TestBuildMessage_MessageIDDomain ensures the Message-ID uses the From
// address domain instead of the random kubernetes pod hostname.
func TestBuildMessage_MessageIDDomain(t *testing.T) {
	m := NewMailer("smtp.example.com", 587, "", "", "noreply@example.com", "Test")
	msg, err := m.buildMessage(Options{}, "rcpt@example.com", "Subject", "<p>HTML</p>", "TEXT")
	if err != nil {
		t.Fatalf("buildMessage failed: %v", err)
	}
	var buf bytes.Buffer
	if _, err := msg.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	out := buf.String()

	// Find the Message-ID line.
	lines := strings.Split(out, "\n")
	var msgID string
	for _, l := range lines {
		if strings.HasPrefix(l, "Message-ID:") {
			msgID = l
			break
		}
	}
	if msgID == "" {
		t.Fatalf("Message-ID header missing in:\n%s", out)
	}
	if !strings.Contains(msgID, "@example.com") {
		t.Errorf("Message-ID should end in @example.com (From-domain), got: %s", msgID)
	}
}

// Sanity check that the gomail constant matches what we expect.
var _ gomail.Header = gomail.HeaderMessageID
