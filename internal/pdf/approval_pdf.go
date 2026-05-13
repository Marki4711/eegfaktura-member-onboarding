package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// ApprovalPDFData holds all data required for the Beitrittsbestätigung PDF.
type ApprovalPDFData struct {
	EEGName         string
	RCNumber        string
	ApprovedAt      time.Time
	ReferenceNumber string

	MemberType      string
	Titel           string
	Firstname       string
	Lastname        string
	BirthDate       *time.Time
	CompanyName     string
	UIDNumber       string
	RegisterNumber  string
	Email           string
	Phone           string

	ResidentStreet       string
	ResidentStreetNumber string
	ResidentZip          string
	ResidentCity         string

	IBAN            string
	AccountHolder   string
	SepaMandateType string

	MeteringPoints     []MeteringPointPDF
	Consents           []ConsentPDF
	StatusLog          []StatusLogPDF
	ConfigurableFields []ConfigurableFieldPDF

	// Statutory consents stored as booleans on the application record.
	PrivacyAccepted      bool
	PrivacyVersion       string
	AccuracyConfirmed    bool
	SepaMandateAccepted   bool
	SepaMandateAcceptedAt *time.Time
	SEPAMandateEnabled    bool // true = Checkbox im Formular, false = per E-Mail

	MemberNumber *int
}

// MeteringPointPDF holds metering point data for the approval PDF.
type MeteringPointPDF struct {
	MeteringPoint       string
	Direction           string
	ParticipationFactor int
}

// ConsentPDF holds a consent snapshot for the approval PDF.
type ConsentPDF struct {
	Title       string
	URL         string
	ConsentedAt time.Time
}

// StatusLogPDF holds one status log entry for the approval PDF.
type StatusLogPDF struct {
	FromStatus string
	ToStatus   string
	Timestamp  time.Time
	Reason     string
}

// ConfigurableFieldPDF holds a configurable field label/value for the approval PDF.
type ConfigurableFieldPDF struct {
	Label string
	Value string
}

var statusLabelsDE = map[string]string{
	"draft":         "Entwurf",
	"submitted":     "Eingereicht",
	"under_review":  "In Prüfung",
	"needs_info":    "Rückfrage",
	"approved":      "Genehmigt",
	"rejected":      "Abgelehnt",
	"imported":      "Importiert",
	"import_failed": "Import fehlgeschlagen",
}

func statusDE(s string) string {
	if label, ok := statusLabelsDE[s]; ok {
		return label
	}
	return s
}

// ApprovalPDFGenerator generates the Beitrittsbestätigung as a PDF byte slice.
type ApprovalPDFGenerator interface {
	GenerateApproval(data ApprovalPDFData) ([]byte, error)
}

// FPDFApprovalGenerator implements ApprovalPDFGenerator using go-pdf/fpdf.
type FPDFApprovalGenerator struct{}

// NewFPDFApprovalGenerator returns a new FPDFApprovalGenerator.
func NewFPDFApprovalGenerator() *FPDFApprovalGenerator {
	return &FPDFApprovalGenerator{}
}

// GenerateApproval produces a DIN-A4 Beitrittsbestätigung PDF.
func (g *FPDFApprovalGenerator) GenerateApproval(data ApprovalPDFData) ([]byte, error) {
	f := fpdf.New("P", "mm", "A4", "")
	f.SetMargins(15, 15, 15)
	f.SetAutoPageBreak(true, 15)
	f.AddPage()

	lm, _, rm, _ := f.GetMargins()
	pageW, _ := f.GetPageSize()
	cw := pageW - lm - rm

	setFont := func(style string, size float64) {
		f.SetFont("Helvetica", style, size)
	}

	sectionHeader := func(title string) {
		f.Ln(3)
		setFont("B", 9)
		f.SetFillColor(230, 230, 230)
		f.CellFormat(cw, 6, w1252(title), "1", 1, "L", true, 0, "")
		setFont("", 9)
	}

	dataRow := func(label, value string) {
		if value == "" {
			return
		}
		setFont("B", 9)
		f.CellFormat(55, 5, w1252(label), "0", 0, "L", false, 0, "")
		setFont("", 9)
		f.CellFormat(cw-55, 5, w1252(value), "0", 1, "L", false, 0, "")
	}

	// ── Title ────────────────────────────────────────────────────────────────
	setFont("B", 14)
	f.CellFormat(cw, 10, w1252("Beitrittsbestätigung"), "0", 1, "L", false, 0, "")
	f.Ln(1)

	// ── Header info ──────────────────────────────────────────────────────────
	eegName := data.EEGName
	if eegName == "" {
		eegName = data.RCNumber
	}
	setFont("", 9)
	f.CellFormat(cw/3*2, 5, w1252("EEG: "+eegName), "0", 0, "L", false, 0, "")
	f.CellFormat(cw/3, 5, w1252("Datum: "+data.ApprovedAt.Format("02.01.2006")), "0", 1, "R", false, 0, "")
	f.CellFormat(cw/3*2, 5, w1252("RC-Nummer: "+data.RCNumber), "0", 0, "L", false, 0, "")
	f.CellFormat(cw/3, 5, w1252("Antrag: "+data.ReferenceNumber), "0", 1, "R", false, 0, "")
	f.Ln(2)
	f.Line(lm, f.GetY(), lm+cw, f.GetY())

	// ── MITGLIEDSDATEN ───────────────────────────────────────────────────────
	sectionHeader("MITGLIEDSDATEN")
	if data.MemberNumber != nil {
		dataRow("Mitgliedsnummer:", fmt.Sprintf("%d", *data.MemberNumber))
	}
	dataRow("Mitgliedstyp:", data.MemberType)
	if data.Firstname != "" || data.Lastname != "" {
		name := strings.TrimSpace(data.Titel + " " + data.Firstname + " " + data.Lastname)
		dataRow("Name:", name)
	}
	if data.CompanyName != "" {
		dataRow("Firmenname:", data.CompanyName)
	}
	if data.UIDNumber != "" {
		dataRow("UID-Nummer:", data.UIDNumber)
	}
	if data.RegisterNumber != "" {
		dataRow("Firmenbuchnummer:", data.RegisterNumber)
	}
	if data.BirthDate != nil {
		dataRow("Geburtsdatum:", data.BirthDate.Format("02.01.2006"))
	}
	dataRow("E-Mail:", data.Email)
	if data.Phone != "" {
		dataRow("Telefon:", data.Phone)
	}
	addr := strings.TrimSpace(data.ResidentStreet+" "+data.ResidentStreetNumber) +
		", " + data.ResidentZip + " " + data.ResidentCity
	dataRow("Adresse:", addr)

	// ── BANKVERBINDUNG ───────────────────────────────────────────────────────
	if data.IBAN != "" {
		sectionHeader("BANKVERBINDUNG")
		dataRow("IBAN:", data.IBAN)
		if data.AccountHolder != "" {
			dataRow("Kontoinhaber:", data.AccountHolder)
		}
		dataRow("SEPA-Ermächtigung:", data.SepaMandateType)
	}

	// ── ZÄHLPUNKTE ───────────────────────────────────────────────────────────
	sectionHeader("ZÄHLPUNKTE")
	setFont("B", 9)
	col1 := cw * 0.55
	col2 := cw * 0.25
	col3 := cw - col1 - col2
	f.CellFormat(col1, 6, w1252("Zählpunktnummer"), "B", 0, "L", false, 0, "")
	f.CellFormat(col2, 6, w1252("Richtung"), "B", 0, "L", false, 0, "")
	f.CellFormat(col3, 6, w1252("Teilnahmefaktor"), "B", 1, "R", false, 0, "")
	setFont("", 9)
	for _, mp := range data.MeteringPoints {
		f.CellFormat(col1, 5, w1252(mp.MeteringPoint), "0", 0, "L", false, 0, "")
		f.CellFormat(col2, 5, w1252(mp.Direction), "0", 0, "L", false, 0, "")
		f.CellFormat(col3, 5, w1252(fmt.Sprintf("%d %%", mp.ParticipationFactor)), "0", 1, "R", false, 0, "")
	}

	// ── ERTEILTE ZUSTIMMUNGEN ─────────────────────────────────────────────────
	sectionHeader("ERTEILTE ZUSTIMMUNGEN")
	setFont("", 9)
	if data.PrivacyAccepted {
		line := "- Datenschutzerklärung akzeptiert"
		if data.PrivacyVersion != "" {
			line += fmt.Sprintf(" (Version %s)", data.PrivacyVersion)
		}
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
	}
	if data.AccuracyConfirmed {
		f.MultiCell(cw, 5, w1252("- Richtigkeit der Angaben bestätigt"), "0", "L", false)
	}
	if data.SEPAMandateEnabled && data.SepaMandateAccepted {
		line := "- SEPA-Lastschriftmandat erteilt"
		if data.SepaMandateAcceptedAt != nil {
			line += " am " + data.SepaMandateAcceptedAt.Format("02.01.2006 15:04")
		}
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
	} else if !data.SEPAMandateEnabled {
		line := "- SEPA-Lastschriftmandat per E-Mail übermittelt"
		if data.SepaMandateAcceptedAt != nil {
			line += " am " + data.SepaMandateAcceptedAt.Format("02.01.2006 15:04")
		}
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
	}
	for _, c := range data.Consents {
		line := fmt.Sprintf("- %s — Zugestimmt am %s", c.Title, c.ConsentedAt.Format("02.01.2006"))
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
		if c.URL != "" {
			setFont("", 8)
			f.MultiCell(cw, 4, w1252("  "+c.URL), "0", "L", false)
			setFont("", 9)
		}
	}
	sepaShown := (data.SEPAMandateEnabled && data.SepaMandateAccepted) || !data.SEPAMandateEnabled
	if !data.PrivacyAccepted && !data.AccuracyConfirmed && !sepaShown && len(data.Consents) == 0 {
		f.MultiCell(cw, 5, w1252("Keine Zustimmungen erfasst."), "0", "L", false)
	}

	// ── STATUSVERLAUF ─────────────────────────────────────────────────────────
	sectionHeader("STATUSVERLAUF")
	setFont("B", 9)
	sc1 := cw * 0.27
	sc2 := cw * 0.27
	sc3 := cw * 0.25
	sc4 := cw - sc1 - sc2 - sc3
	f.CellFormat(sc1, 6, w1252("Von"), "B", 0, "L", false, 0, "")
	f.CellFormat(sc2, 6, w1252("Nach"), "B", 0, "L", false, 0, "")
	f.CellFormat(sc3, 6, w1252("Zeitpunkt"), "B", 0, "L", false, 0, "")
	f.CellFormat(sc4, 6, w1252("Kommentar"), "B", 1, "L", false, 0, "")
	setFont("", 9)
	for _, sl := range data.StatusLog {
		from := statusDE(sl.FromStatus)
		if from == "" {
			from = "—"
		}
		ts := sl.Timestamp.Format("02.01.2006 15:04")
		f.CellFormat(sc1, 5, w1252(from), "0", 0, "L", false, 0, "")
		f.CellFormat(sc2, 5, w1252(statusDE(sl.ToStatus)), "0", 0, "L", false, 0, "")
		f.CellFormat(sc3, 5, w1252(ts), "0", 0, "L", false, 0, "")
		f.CellFormat(sc4, 5, w1252(sl.Reason), "0", 1, "L", false, 0, "")
	}

	// ── WEITERE ANGABEN ───────────────────────────────────────────────────────
	if len(data.ConfigurableFields) > 0 {
		sectionHeader("WEITERE ANGABEN")
		for _, cf := range data.ConfigurableFields {
			dataRow(cf.Label+":", cf.Value)
		}
	}

	if f.Error() != nil {
		return nil, fmt.Errorf("pdf rendering error: %w", f.Error())
	}

	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output failed: %w", err)
	}
	return buf.Bytes(), nil
}
