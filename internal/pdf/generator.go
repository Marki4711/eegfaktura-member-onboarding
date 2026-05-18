package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"golang.org/x/text/encoding/charmap"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// SEPAMandateData holds all data required to fill a SEPA direct debit mandate.
type SEPAMandateData struct {
	EEGName            string
	EEGStreet          string
	EEGStreetNumber    string
	EEGZip             string
	EEGCity            string
	CreditorID         string
	MemberName         string
	MemberStreet       string
	MemberStreetNumber string
	MemberZip          string
	MemberCity         string
	IBAN               string
	// MandateReference (PROJ-47) is the Mandatsreferenz printed below the
	// title. Empty means "to be filled in by EEG" — used at submission time
	// when the member number hasn't been assigned yet. Non-empty means the
	// reference is already known (e.g. the Mitgliedsnummer after import for
	// B2B-SEPA-Firmenlastschrift).
	MandateReference string
	// MandateDate ist der Tag, an dem das Mandat-PDF erzeugt und an das
	// Mitglied übermittelt wird (PROJ-52 Mini-Lücke 3). Wird im Unter-
	// schriftsfeld als „Datum" vorbefüllt. Default in der Service-Layer
	// auf time.Now() — bleibt im PDF leer, wenn hier Zero-Time übergeben
	// wird (defensiv für Legacy-Tests).
	MandateDate time.Time
	// LogoBytes is the EEG logo cached from the eegFaktura-billing service
	// (PROJ-33). Empty = no logo embedded; the PDF renders without it.
	LogoBytes []byte
	// LogoMIME is the Content-Type the billing service returned with the
	// bytes. Must be one of image/png, image/jpeg, image/gif; anything else
	// is silently skipped by embedLogoTopRight.
	LogoMIME string
}

// SEPAMandateGenerator generates a SEPA direct debit mandate as a PDF byte slice.
type SEPAMandateGenerator interface {
	Generate(data SEPAMandateData) ([]byte, error)
	GenerateCompany(data SEPAMandateData) ([]byte, error)
}

// FPDFGenerator implements SEPAMandateGenerator using go-pdf/fpdf.
type FPDFGenerator struct{}

// NewFPDFGenerator returns a new FPDFGenerator.
func NewFPDFGenerator() *FPDFGenerator {
	return &FPDFGenerator{}
}

// enc converts UTF-8 strings to Windows-1252 for use with fpdf core fonts.
var enc = charmap.Windows1252.NewEncoder()

func w1252(s string) string {
	out, err := enc.String(s)
	if err != nil {
		return s
	}
	return out
}

// Generate produces a DIN-A4 SEPA mandate PDF pre-filled with the given data.
func (g *FPDFGenerator) Generate(data SEPAMandateData) ([]byte, error) {
	f := fpdf.New("P", "mm", "A4", "")
	f.SetMargins(15, 15, 15)
	f.SetAutoPageBreak(false, 0)
	f.AddPage()
	embedLogoTopRight(f, data.LogoBytes, data.LogoMIME)

	lm, _, rm, _ := f.GetMargins()
	pageW, _ := f.GetPageSize()
	cw := pageW - lm - rm // usable content width: ~180mm

	setFont := func(style string, size float64) {
		f.SetFont("Helvetica", style, size)
	}

	// helper: bordered cell row with label + value
	fieldRow := func(label, value string, labelW, h float64) {
		setFont("B", 9)
		f.CellFormat(labelW, h, w1252(label), "0", 0, "L", false, 0, "")
		setFont("", 9)
		f.CellFormat(cw-labelW, h, w1252(value), "B", 1, "L", false, 0, "")
	}

	// ── Title ─────────────────────────────────────────────────────────────
	setFont("B", 13)
	f.CellFormat(cw, 8, w1252("SEPA-Lastschriftmandat"), "0", 1, "L", false, 0, "")
	f.Ln(2)

	// ── Mandatsreferenz ────────────────────────────────────────────────────
	// PROJ-47: when MandateReference is set, print it inline; otherwise
	// fall back to the "wird von … ausgefüllt"-Platzhalter for the
	// submission-time PDF where the Mitgliedsnummer is not yet known.
	setFont("", 9)
	if data.MandateReference != "" {
		setFont("B", 9)
		f.CellFormat(35, 6, w1252("Mandatsreferenz:"), "0", 0, "L", false, 0, "")
		setFont("", 9)
		f.CellFormat(cw-35, 6, w1252(data.MandateReference), "B", 1, "L", false, 0, "")
		f.Ln(4)
	} else {
		mandatsRef := fmt.Sprintf("Mandatsreferenz (wird von %s ausgefüllt):", data.EEGName)
		f.CellFormat(cw, 6, w1252(mandatsRef), "0", 1, "L", false, 0, "")
		f.Ln(1)
		f.CellFormat(cw, 0.3, "", "0", 1, "L", false, 0, "") // horizontal line placeholder
		f.Line(lm, f.GetY(), lm+80, f.GetY())
		f.Ln(4)
	}

	// ── ZAHLUNGSEMPFÄNGER ──────────────────────────────────────────────────
	// outer border start
	boxTop := f.GetY()
	setFont("B", 9)
	f.SetFillColor(230, 230, 230)
	f.CellFormat(cw, 6, w1252("ZAHLUNGSEMPFÄNGER"), "LRT", 1, "L", true, 0, "")
	f.Ln(1)
	setFont("", 9)
	fieldRow("Name:", data.EEGName, 30, 6)
	fieldRow("Anschrift (Straße, PLZ, Ort, Land):", fmt.Sprintf("%s %s, %s %s, Österreich",
		data.EEGStreet, data.EEGStreetNumber, data.EEGZip, data.EEGCity), 70, 6)
	f.Ln(1)
	fieldRow("Creditor-ID:", data.CreditorID, 30, 6)
	// close box
	boxBot := f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(3)

	// ── Ermächtigungstext ──────────────────────────────────────────────────
	ermText := fmt.Sprintf(
		"Ich ermächtige / Wir ermächtigen %s, Zahlungen von meinem / unserem Konto mittels SEPA-Lastschriften einzuziehen. "+
			"Zugleich weise ich mein / weisen wir unser Kreditinstitut an, die von %s auf mein / unser Konto gezogenen SEPA-Lastschriften einzulösen.\n\n"+
			"Hinweis: Ich kann / Wir können innerhalb von acht Wochen, beginnend mit dem Belastungsdatum, die Erstattung des belasteten Betrages verlangen. "+
			"Es gelten dabei die mit meinem / unserem Kreditinstitut vereinbarten Bedingungen.",
		data.EEGName, data.EEGName,
	)
	boxTop = f.GetY()
	setFont("", 9)
	f.SetFillColor(255, 255, 255)
	f.MultiCell(cw, 5, w1252(ermText), "LR", "L", false)
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(3)

	// ── Zahlungsart ────────────────────────────────────────────────────────
	boxTop = f.GetY()
	setFont("", 9)
	f.CellFormat(25, 7, w1252("Zahlungsart:"), "0", 0, "L", false, 0, "")
	// einmalig (unchecked) — draw an empty rectangle
	einmX := f.GetX() + 3
	einmY := f.GetY()
	f.Rect(einmX, einmY+1.5, 4, 4, "D")
	f.SetX(einmX + 7)
	f.CellFormat(22, 7, w1252("einmalig"), "0", 0, "L", false, 0, "")
	// wiederkehrend (checked) — white box with black X (two diagonal lines)
	checkX := f.GetX() + 3
	checkY := f.GetY()
	f.Rect(checkX, checkY+1.5, 4, 4, "D")
	f.SetLineWidth(0.4)
	f.Line(checkX+0.5, checkY+2, checkX+3.5, checkY+5)
	f.Line(checkX+3.5, checkY+2, checkX+0.5, checkY+5)
	f.SetLineWidth(0.2)
	f.SetX(checkX + 7)
	setFont("", 9)
	f.CellFormat(40, 7, w1252("wiederkehrend"), "0", 1, "L", false, 0, "")
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(3)

	// ── ZAHLUNGSPFLICHTIGER ────────────────────────────────────────────────
	boxTop = f.GetY()
	setFont("B", 9)
	f.SetFillColor(230, 230, 230)
	f.CellFormat(cw, 6, w1252("ZAHLUNGSPFLICHTIGER"), "LRT", 1, "L", true, 0, "")
	f.Ln(1)
	setFont("", 9)
	fieldRow("Name:", data.MemberName, 30, 6)
	fieldRow("Anschrift (Straße, PLZ, Ort, Land):", fmt.Sprintf("%s %s, %s %s",
		data.MemberStreet, data.MemberStreetNumber, data.MemberZip, data.MemberCity), 70, 6)
	f.Ln(1)
	// IBAN + BIC on same row
	setFont("B", 9)
	f.CellFormat(12, 6, w1252("IBAN:"), "0", 0, "L", false, 0, "")
	setFont("", 9)
	f.CellFormat(75, 6, w1252(data.IBAN), "B", 0, "L", false, 0, "")
	f.CellFormat(8, 6, "", "0", 0, "L", false, 0, "") // gap
	setFont("B", 9)
	f.CellFormat(12, 6, w1252("BIC:*"), "0", 0, "L", false, 0, "")
	setFont("", 9)
	f.CellFormat(cw-107, 6, "", "B", 1, "L", false, 0, "")
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(8)

	// ── Unterschriftsfeld ─────────────────────────────────────────────────
	boxTop = f.GetY()
	f.Ln(15) // space for signature
	sigY := f.GetY()
	// PROJ-52 Mini-Lücke 3: Datum wird oberhalb der Linie vorbefüllt
	// (Tag der Übermittlung). Mitglied trägt nur noch Ort + Unterschrift
	// ein. Bei Zero-Time bleibt die Zeile leer (defensiv für Legacy-Tests).
	if !data.MandateDate.IsZero() {
		setFont("", 9)
		f.SetXY(lm, sigY-5)
		f.CellFormat(70, 5, w1252(shared.FmtDate(data.MandateDate)), "0", 0, "L", false, 0, "")
	}
	// Datum/Ort line
	f.Line(lm, sigY, lm+70, sigY)
	setFont("", 8)
	f.SetXY(lm, sigY+1)
	f.CellFormat(70, 5, w1252("Datum, Ort"), "0", 0, "L", false, 0, "")
	// Unterschrift line
	f.Line(lm+90, sigY, lm+cw, sigY)
	f.SetXY(lm+90, sigY+1)
	f.CellFormat(cw-90, 5, w1252("Unterschrift"), "0", 1, "L", false, 0, "")
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop+2, "D")
	f.Ln(4)

	// ── BIC-Fußnote ────────────────────────────────────────────────────────
	setFont("", 7)
	bic := "*Ab 01.02.2014 kann die Angabe des BIC entfallen, wenn es sich um nationale Lastschriften handelt. " +
		"Ab 01.02.2016 ist der BIC auch für grenzüberschreitende Lastschriften innerhalb der EU/EWR nicht mehr erforderlich."
	f.MultiCell(cw, 4, w1252(bic), "1", "L", false)

	if f.Error() != nil {
		return nil, fmt.Errorf("pdf rendering error: %w", f.Error())
	}

	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output failed: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateCompany produces a DIN-A4 SEPA-Firmenlastschrift-Mandat (B2B) PDF.
func (g *FPDFGenerator) GenerateCompany(data SEPAMandateData) ([]byte, error) {
	f := fpdf.New("P", "mm", "A4", "")
	f.SetMargins(15, 15, 15)
	f.SetAutoPageBreak(false, 0)
	f.AddPage()
	embedLogoTopRight(f, data.LogoBytes, data.LogoMIME)

	lm, _, rm, _ := f.GetMargins()
	pageW, _ := f.GetPageSize()
	cw := pageW - lm - rm

	setFont := func(style string, size float64) { f.SetFont("Helvetica", style, size) }
	fieldRow := func(label, value string, labelW, h float64) {
		setFont("B", 9)
		f.CellFormat(labelW, h, w1252(label), "0", 0, "L", false, 0, "")
		setFont("", 9)
		f.CellFormat(cw-labelW, h, w1252(value), "B", 1, "L", false, 0, "")
	}

	// ── Title ─────────────────────────────────────────────────────────────────
	setFont("B", 13)
	f.CellFormat(cw, 8, w1252("SEPA-Firmenlastschrift-Mandat"), "0", 1, "L", false, 0, "")
	f.Ln(2)

	// ── Mandatsreferenz ────────────────────────────────────────────────────────
	// PROJ-47: print Mandatsreferenz inline when set (post-import B2B path
	// passes the Mitgliedsnummer). Fallback to the "wird von … ausgefüllt"-
	// Platzhalter for submission-time PDFs where the number isn't known yet.
	setFont("", 9)
	if data.MandateReference != "" {
		setFont("B", 9)
		f.CellFormat(35, 6, w1252("Mandatsreferenz:"), "0", 0, "L", false, 0, "")
		setFont("", 9)
		f.CellFormat(cw-35, 6, w1252(data.MandateReference), "B", 1, "L", false, 0, "")
		f.Ln(4)
	} else {
		mandatsRef := fmt.Sprintf("Mandatsreferenz (wird von %s ausgefüllt):", data.EEGName)
		f.CellFormat(cw, 6, w1252(mandatsRef), "0", 1, "L", false, 0, "")
		f.Ln(1)
		f.Line(lm, f.GetY(), lm+80, f.GetY())
		f.Ln(4)
	}

	// ── ZAHLUNGSEMPFÄNGER ──────────────────────────────────────────────────────
	boxTop := f.GetY()
	setFont("B", 9)
	f.SetFillColor(230, 230, 230)
	f.CellFormat(cw, 6, w1252("ZAHLUNGSEMPFÄNGER"), "LRT", 1, "L", true, 0, "")
	f.Ln(1)
	setFont("", 9)
	fieldRow("Creditor CD:", data.CreditorID, 30, 6)
	fieldRow("Name:", data.EEGName, 30, 6)
	fieldRow("Anschrift (Straße, Ort, Land):", fmt.Sprintf("%s %s, %s %s, Österreich",
		data.EEGStreet, data.EEGStreetNumber, data.EEGZip, data.EEGCity), 60, 6)
	boxBot := f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(3)

	// ── Ermächtigungstext (B2B-Wortlaut) ──────────────────────────────────────
	ermText := fmt.Sprintf(
		"Ich ermächtige/Wir ermächtigen %s, Zahlungen von meinem/unserem Konto mittels SEPA-Firmenlastschriften einzuziehen. "+
			"Zugleich weise ich mein/weisen wir unser Kreditinstitut an, die von %s auf mein/unser Konto gezogenen SEPA-Firmenlastschriften einzulösen.\n\n"+
			"Hinweis: Dieses SEPA-Firmenlastschrift-Mandat dient nur dem Einzug von SEPA-Firmenlastschriften, die auf Konten von Unternehmen gezogen sind. "+
			"Ich bin/Wir sind nicht berechtigt, nach der erfolgten Einlösung eine Erstattung des belasteten Betrages zu verlangen. "+
			"Ich bin/Wir sind berechtigt, mein/unser Kreditinstitut bis zum Fälligkeitstag anzuweisen, SEPA-Firmenlastschriften nicht einzulösen.",
		data.EEGName, data.EEGName,
	)
	boxTop = f.GetY()
	setFont("", 9)
	f.SetFillColor(255, 255, 255)
	f.MultiCell(cw, 5, w1252(ermText), "LR", "L", false)
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(3)

	// ── Zahlungsart ────────────────────────────────────────────────────────────
	boxTop = f.GetY()
	setFont("", 9)
	f.CellFormat(25, 7, w1252("Zahlungsart:"), "0", 0, "L", false, 0, "")
	// einmalig (unchecked box)
	einmX := f.GetX() + 3
	einmY := f.GetY()
	f.Rect(einmX, einmY+1.5, 4, 4, "D")
	f.SetX(einmX + 7)
	f.CellFormat(22, 7, w1252("einmalig"), "0", 0, "L", false, 0, "")
	// wiederkehrend (checked) — white box with black X (two diagonal lines)
	checkX := f.GetX() + 3
	checkY := f.GetY()
	f.Rect(checkX, checkY+1.5, 4, 4, "D")
	f.SetLineWidth(0.4)
	f.Line(checkX+0.5, checkY+2, checkX+3.5, checkY+5)
	f.Line(checkX+3.5, checkY+2, checkX+0.5, checkY+5)
	f.SetLineWidth(0.2)
	f.SetX(checkX + 7)
	setFont("", 9)
	f.CellFormat(40, 7, w1252("wiederkehrend"), "0", 1, "L", false, 0, "")
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(3)

	// ── ZAHLUNGSPFLICHTIGER ────────────────────────────────────────────────────
	boxTop = f.GetY()
	setFont("B", 9)
	f.SetFillColor(230, 230, 230)
	f.CellFormat(cw, 6, w1252("ZAHLUNGSPFLICHTIGER"), "LRT", 1, "L", true, 0, "")
	f.Ln(1)
	setFont("", 9)
	fieldRow("Name:", data.MemberName, 30, 6)
	fieldRow("Anschrift (Straße, Ort, Land):", fmt.Sprintf("%s %s, %s %s",
		data.MemberStreet, data.MemberStreetNumber, data.MemberZip, data.MemberCity), 60, 6)
	f.Ln(1)
	setFont("B", 9)
	f.CellFormat(12, 6, w1252("IBAN:"), "0", 0, "L", false, 0, "")
	setFont("", 9)
	f.CellFormat(75, 6, w1252(data.IBAN), "B", 0, "L", false, 0, "")
	f.CellFormat(8, 6, "", "0", 0, "L", false, 0, "")
	setFont("B", 9)
	f.CellFormat(12, 6, w1252("BIC:*"), "0", 0, "L", false, 0, "")
	setFont("", 9)
	f.CellFormat(cw-107, 6, "", "B", 1, "L", false, 0, "")
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop, "D")
	f.Ln(8)

	// ── Unterschriftsfeld ─────────────────────────────────────────────────────
	boxTop = f.GetY()
	f.Ln(15)
	sigY := f.GetY()
	// PROJ-52 Mini-Lücke 3: Datum wird vorbefüllt, Mitglied trägt nur noch
	// Ort + Unterschrift ein.
	if !data.MandateDate.IsZero() {
		setFont("", 9)
		f.SetXY(lm, sigY-5)
		f.CellFormat(70, 5, w1252(shared.FmtDate(data.MandateDate)), "0", 0, "L", false, 0, "")
	}
	f.Line(lm, sigY, lm+70, sigY)
	setFont("", 8)
	f.SetXY(lm, sigY+1)
	f.CellFormat(70, 5, w1252("Ort, Datum, Unterschrift"), "0", 0, "L", false, 0, "")
	boxBot = f.GetY()
	f.Rect(lm, boxTop, cw, boxBot-boxTop+2, "D")
	f.Ln(4)

	// ── BIC-Fußnote ────────────────────────────────────────────────────────────
	setFont("", 7)
	f.MultiCell(cw, 4, w1252(
		"* Seit 01.06.2016 kann die Angabe des BIC bei nationalen und grenzüberschreitenden Lastschriften entfallen.",
	), "1", "L", false)

	if f.Error() != nil {
		return nil, fmt.Errorf("pdf rendering error: %w", f.Error())
	}
	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output failed: %w", err)
	}
	return buf.Bytes(), nil
}
