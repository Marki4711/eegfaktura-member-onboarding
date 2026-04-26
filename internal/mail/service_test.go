package mail

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// TestNoOpMailService_Noop verifies the no-op implementation satisfies the interface
// and does nothing without panicking.
func TestNoOpMailService_Noop(t *testing.T) {
	var svc MailService = &NoOpMailService{}
	fn, ln := "Josef", "Muster"
	app := &shared.Application{
		ID:              uuid.New(),
		Firstname:       &fn,
		Lastname:        &ln,
		Email:           "max.mustermann@example.org",
		ReferenceNumber: "REF-2026-001",
	}
	ep := &shared.RegistrationEntrypoint{}
	svc.SendSubmissionEmails(app, nil, ep, nil, nil) // must not panic
}

// TestNewSMTPMailService_ParsesTemplates verifies that embedded HTML templates
// parse without error at startup time.
func TestNewSMTPMailService_ParsesTemplates(t *testing.T) {
	mailer := NewMailer("localhost", 587, "", "", "noreply@example.com")
	svc, err := NewSMTPMailService(mailer, "")
	if err != nil {
		t.Fatalf("NewSMTPMailService failed: %v", err)
	}
	if svc.memberTpl == nil {
		t.Error("member template is nil")
	}
	if svc.eegTpl == nil {
		t.Error("eeg template is nil")
	}
	if svc.approvalTpl == nil {
		t.Error("approval template is nil")
	}
}

// TestMemberTemplate_ContainsExpectedFields verifies the member confirmation
// email renders all required fields: firstname, lastname, reference number.
func TestMemberTemplate_ContainsExpectedFields(t *testing.T) {
	tpl, err := template.ParseFS(templateFS, "templates/application_submitted_member.html")
	if err != nil {
		t.Fatalf("failed to parse member template: %v", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, memberTemplateData{
		Firstname:       "Josef",
		Lastname:        "Muster",
		ReferenceNumber: "REF-2026-001",
	}); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}

	body := buf.String()
	checks := map[string]string{
		"firstname":        "Josef",
		"lastname":         "Muster",
		"reference number": "REF-2026-001",
	}
	for label, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("member template missing %s (%q)", label, want)
		}
	}
}

// TestMemberTemplate_IsGerman verifies the email body is in German.
func TestMemberTemplate_IsGerman(t *testing.T) {
	tpl, err := template.ParseFS(templateFS, "templates/application_submitted_member.html")
	if err != nil {
		t.Fatalf("failed to parse member template: %v", err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, memberTemplateData{Firstname: "X", Lastname: "Y", ReferenceNumber: "R"}); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Beitrittserklärung") {
		t.Error("member template does not appear to be in German")
	}
}

// TestEEGTemplate_ContainsExpectedFields verifies the EEG notification email
// renders applicant name, email, reference number, and metering points.
func TestEEGTemplate_ContainsExpectedFields(t *testing.T) {
	tpl, err := template.ParseFS(templateFS, "templates/application_submitted_eeg.html")
	if err != nil {
		t.Fatalf("failed to parse eeg template: %v", err)
	}

	points := []meteringPointView{
		{MeteringPoint: "AT0031000000000000000000990022105", Direction: "Verbrauch", ParticipationFactor: 100},
		{MeteringPoint: "AT0031000000000000000000990022106", Direction: "Einspeisung", ParticipationFactor: 50},
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, eegTemplateData{
		Firstname:       "Josef",
		Lastname:        "Muster",
		Email:           "max.mustermann@example.org",
		ReferenceNumber: "REF-2026-001",
		MeteringPoints:  points,
	}); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}

	body := buf.String()
	checks := map[string]string{
		"firstname":        "Josef",
		"lastname":         "Muster",
		"email":            "max.mustermann@example.org",
		"reference number": "REF-2026-001",
		"metering point 1": "AT0031000000000000000000990022105",
		"metering point 2": "AT0031000000000000000000990022106",
		"direction":        "Verbrauch",
	}
	for label, want := range checks {
		if !strings.Contains(body, want) {
			t.Errorf("eeg template missing %s (%q)", label, want)
		}
	}
}

// TestEEGTemplate_IsGerman verifies the EEG notification body is in German.
func TestEEGTemplate_IsGerman(t *testing.T) {
	tpl, err := template.ParseFS(templateFS, "templates/application_submitted_eeg.html")
	if err != nil {
		t.Fatalf("failed to parse eeg template: %v", err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, eegTemplateData{Firstname: "X", Lastname: "Y", Email: "x@y.com", ReferenceNumber: "R"}); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Beitrittsantrag") {
		t.Error("eeg template does not appear to be in German")
	}
}

// TestEEGTemplate_XSSEscaped verifies that malicious content in template data
// is HTML-escaped and not rendered as raw HTML.
func TestEEGTemplate_XSSEscaped(t *testing.T) {
	tpl, err := template.ParseFS(templateFS, "templates/application_submitted_eeg.html")
	if err != nil {
		t.Fatalf("failed to parse eeg template: %v", err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, eegTemplateData{
		Firstname:       "<script>alert(1)</script>",
		Lastname:        "Muster",
		Email:           "x@y.com",
		ReferenceNumber: "R",
	}); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}
	body := buf.String()
	if strings.Contains(body, "<script>") {
		t.Error("XSS: script tag not escaped in eeg template")
	}
}

// TestMemberTemplate_XSSEscaped verifies HTML escaping in the member template.
func TestMemberTemplate_XSSEscaped(t *testing.T) {
	tpl, err := template.ParseFS(templateFS, "templates/application_submitted_member.html")
	if err != nil {
		t.Fatalf("failed to parse member template: %v", err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, memberTemplateData{
		Firstname:       "<script>alert(1)</script>",
		Lastname:        "Y",
		ReferenceNumber: "R",
	}); err != nil {
		t.Fatalf("template execution failed: %v", err)
	}
	if strings.Contains(buf.String(), "<script>") {
		t.Error("XSS: script tag not escaped in member template")
	}
}

// --- Sender interface tests (enabled after BUG-3 fix) ---

// spySender captures Send calls for assertion in tests.
type spySender struct {
	calls []spyCall
}

type spyCall struct {
	to      string
	subject string
	body    string
}

func (s *spySender) Send(to, subject, htmlBody string) error {
	s.calls = append(s.calls, spyCall{to: to, subject: subject, body: htmlBody})
	return nil
}

func (s *spySender) SendWithAttachment(to, subject, htmlBody, _ string, _ []byte) error {
	s.calls = append(s.calls, spyCall{to: to, subject: subject, body: htmlBody})
	return nil
}

func newTestService(t *testing.T, spy *spySender) *SMTPMailService {
	t.Helper()
	svc, err := NewSMTPMailService(spy, "")
	if err != nil {
		t.Fatalf("NewSMTPMailService: %v", err)
	}
	return svc
}

func testApp() *shared.Application {
	fn, ln := "Josef", "Muster"
	return &shared.Application{
		ID:              uuid.New(),
		RCNumber:        "RC123456",
		MemberType:      shared.MemberTypePrivate,
		Firstname:       &fn,
		Lastname:        &ln,
		Email:           "max.mustermann@example.org",
		ReferenceNumber: "REF-2026-001",
	}
}

func testMeteringPoints() []shared.MeteringPoint {
	return []shared.MeteringPoint{
		{MeteringPoint: "AT0031000000000000000000990022105", Direction: "CONSUMPTION"},
	}
}

// TestSendSubmissionEmails_MemberAlwaysSent verifies the member email is always sent.
func TestSendSubmissionEmails_MemberAlwaysSent(t *testing.T) {
	spy := &spySender{}
	svc := newTestService(t, spy)
	ep := &shared.RegistrationEntrypoint{}

	svc.SendSubmissionEmails(testApp(), testMeteringPoints(), ep, nil, nil)

	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 send call, got %d", len(spy.calls))
	}
	if spy.calls[0].to != "max.mustermann@example.org" {
		t.Errorf("member email sent to wrong address: %s", spy.calls[0].to)
	}
	if !strings.Contains(spy.calls[0].subject, "REF-2026-001") {
		t.Errorf("member email subject missing reference number: %s", spy.calls[0].subject)
	}
}

// TestSendSubmissionEmails_EEGSentWhenContactEmailSet verifies the EEG email is
// sent in addition to the member email when contact_email is configured.
func TestSendSubmissionEmails_EEGSentWhenContactEmailSet(t *testing.T) {
	spy := &spySender{}
	svc := newTestService(t, spy)
	contactEmail := "eeg@example.at"
	ep := &shared.RegistrationEntrypoint{ContactEmail: &contactEmail}

	svc.SendSubmissionEmails(testApp(), testMeteringPoints(), ep, nil, nil)

	if len(spy.calls) != 2 {
		t.Fatalf("expected 2 send calls, got %d", len(spy.calls))
	}
	if spy.calls[1].to != contactEmail {
		t.Errorf("eeg email sent to wrong address: %s", spy.calls[1].to)
	}
	if !strings.Contains(spy.calls[1].body, "AT0031000000000000000000990022105") {
		t.Error("eeg email body missing metering point")
	}
}

// TestSendSubmissionEmails_EEGSkippedWhenNoContactEmail verifies that no EEG
// email is sent when contact_email is nil.
func TestSendSubmissionEmails_EEGSkippedWhenNoContactEmail(t *testing.T) {
	spy := &spySender{}
	svc := newTestService(t, spy)
	ep := &shared.RegistrationEntrypoint{ContactEmail: nil}

	svc.SendSubmissionEmails(testApp(), testMeteringPoints(), ep, nil, nil)

	if len(spy.calls) != 1 {
		t.Errorf("expected only 1 send call (member), got %d", len(spy.calls))
	}
}

// TestSendSubmissionEmails_EEGSkippedWhenContactEmailEmpty verifies that an
// empty-string contact_email is treated the same as nil.
func TestSendSubmissionEmails_EEGSkippedWhenContactEmailEmpty(t *testing.T) {
	spy := &spySender{}
	svc := newTestService(t, spy)
	empty := ""
	ep := &shared.RegistrationEntrypoint{ContactEmail: &empty}

	svc.SendSubmissionEmails(testApp(), testMeteringPoints(), ep, nil, nil)

	if len(spy.calls) != 1 {
		t.Errorf("expected only 1 send call (member), got %d", len(spy.calls))
	}
}
