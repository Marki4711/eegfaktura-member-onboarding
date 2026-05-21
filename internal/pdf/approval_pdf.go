package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// Use the shared display-timezone helpers so PDF, mail and any future
// renderer stay consistent (Europe/Vienna).
func fmtDateTime(t time.Time) string { return shared.FmtDateTime(t) }
func fmtDate(t time.Time) string     { return shared.FmtDate(t) }

// formatMeteringPointNumber gruppiert einen 33-stelligen AT-Zählpunkt für
// die lesbare Anzeige auf der Netzbetreiber-Info-Seite (PROJ-56). Beispiel:
//
//	AT0090000000000000000000000351391
//	→ AT 003000 00000 0000000000 0000409072
//
// Format: AT (2) + 6 + 5 + 10 + 10 = 33 Zeichen. Bei abweichender Länge
// wird der Wert unformatiert zurückgegeben (defensiver Fallback).
func formatMeteringPointNumber(zp string) string {
	if len(zp) != 33 {
		return zp
	}
	return zp[0:2] + " " + zp[2:8] + " " + zp[8:13] + " " + zp[13:23] + " " + zp[23:33]
}

// ApprovalPDFData holds all data required for the Beitrittsbestätigung PDF.
type ApprovalPDFData struct {
	EEGName         string
	RCNumber        string
	ApprovedAt      time.Time
	ReferenceNumber string

	MemberType      string
	Titel           string
	TitelNach       string
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
	BankName        string
	SepaMandateType string

	MeteringPoints     []MeteringPointPDF
	Consents           []ConsentPDF
	ConfigurableFields []ConfigurableFieldPDF

	// Statutory consents stored as booleans on the application record.
	PrivacyAccepted       bool
	PrivacyVersion        string
	PrivacyAcceptedAt     *time.Time
	AccuracyConfirmed     bool
	AccuracyConfirmedAt   *time.Time // = submitted_at; accuracy has no dedicated column
	SepaMandateAccepted   bool
	SepaMandateAcceptedAt *time.Time
	SEPAMandateEnabled    bool // true = Mandat-PDF an Willkommensmail anhängen, false = Mandat wird separat per Mail übermittelt
	// PROJ-44: Netzbetreiber-Vollmacht. Wenn true, wird der volle
	// Vollmachtstext (shared.NetworkOperatorAuthText) als Zustimmung
	// im Block "ERTEILTE ZUSTIMMUNGEN" gerendert. AuthorizationAt ist
	// der Zeitpunkt der Erteilung (NOW() beim Submit, wenn die
	// Checkbox aktiv gesetzt wurde).
	NetworkOperatorAuthorization   bool
	NetworkOperatorAuthorizationAt *time.Time
	// PROJ-56: Daten für die zusätzliche Netzbetreiber-Info-Seite.
	// Werden nur ausgewertet, wenn NetworkOperatorAuthorization=true ist.
	// Customer-Number + Inventory-Number sind optional (Empty-String =
	// Label ohne Wert anzeigen). SubmittedAt ist der Vollmachts-
	// Zeitpunkt für die Seite ("Vollmacht erteilt am ...").
	NetworkOperatorCustomerNumber string
	MeterInventoryNumber          string
	SubmittedAt                   *time.Time

	MemberNumber *string

	// CooperativeSharesCount (PROJ-37) is the number the member subscribed.
	// CooperativeShareAmountCents is the price per share at PDF-render time
	// (NOT a submit-time snapshot — that's documented as discussion-worthy
	// in PROJ-37 spec § Out-of-Scope-for-V1). Both must be set together for
	// the section to render; missing one collapses the block silently.
	CooperativeSharesCount      *int
	CooperativeShareAmountCents *int64

	// LogoBytes is the EEG logo cached from the eegFaktura-billing service
	// (PROJ-33). Empty = no logo embedded; the PDF renders without it.
	LogoBytes []byte
	// LogoMIME is the Content-Type the billing service returned with the
	// bytes. Must be one of image/png, image/jpeg, image/gif; anything else
	// is silently skipped by embedLogoTopRight.
	LogoMIME string
}

// MeteringPointPDF holds metering point data for the approval PDF.
type MeteringPointPDF struct {
	MeteringPoint       string
	Direction           string
	ParticipationFactor int
	// PROJ-39: pre-formatted address line ("Straße Nr, PLZ Ort"). Empty
	// when the metering point uses the member's primary address.
	AddressLine string
	// PROJ-56: zwei separat befüllte Adress-Zeilen für die Netzbetreiber-
	// Info-Seite (Tabellen-Layout zeigt Straße + PLZ Ort in zwei Cell-
	// Zeilen). Werden aus PROJ-39-Daten oder dem Mitglieder-Hauptsitz
	// abgeleitet. Beispiel: "Frau Elster-Straße 1" / "1234 Märchenwald".
	AddressStreetLine string
	AddressCityLine   string
	// PROJ-45: pre-formatted "Erzeugung"-Zeile für PRODUCTION-Zählpunkte.
	// Beispiele: "PV", "PV, Speicher 10,5 kWh (Fronius)". Leer für
	// CONSUMPTION oder wenn nichts darzustellen ist.
	GenerationLine string
}

// ConsentPDF holds a consent snapshot for the approval PDF.
type ConsentPDF struct {
	Title       string
	URL         string
	ConsentedAt time.Time
	// Informational (PROJ-36) is true when the consent was recorded as
	// a passive acknowledgement of a non-required info document rather
	// than as an active checkbox tick. Affects only the label text in
	// the rendered PDF — the timestamp + URL are shown either way.
	Informational bool
}

// ConfigurableFieldPDF holds a configurable field label/value for the approval PDF.
type ConfigurableFieldPDF struct {
	Label string
	Value string
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
	// PROJ-33 follow-up: the logo embed is deferred until after the title
	// + 4 header lines + separator are drawn, so we can vertically center
	// it in the resulting band. See call below.

	lm, topMargin, rm, _ := f.GetMargins()
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
	// PROJ-33 follow-up: header info stacks vertically on the left so the
	// top-right logo (max 50 mm wide) has clear space. Used to be two rows
	// with right-aligned Datum/Antrag, which collided with the logo. Robust
	// against long PROJ-35 reference numbers (`RC105720-2026-0001`).
	setFont("", 9)
	f.CellFormat(cw, 5, w1252("EEG: "+eegName), "0", 1, "L", false, 0, "")
	f.CellFormat(cw, 5, w1252("RC-Nummer: "+data.RCNumber), "0", 1, "L", false, 0, "")
	f.CellFormat(cw, 5, w1252("Datum: "+fmtDate(data.ApprovedAt)), "0", 1, "L", false, 0, "")
	f.CellFormat(cw, 5, w1252("Antrag: "+data.ReferenceNumber), "0", 1, "L", false, 0, "")
	f.Ln(2)
	separatorY := f.GetY()
	f.Line(lm, separatorY, lm+cw, separatorY)

	// Drop the logo into the right side of the header band, vertically
	// centered between the top margin and the separator line. Cursor is
	// preserved by the helper; the next section (MITGLIEDSDATEN) renders
	// in its natural position below.
	embedLogoCenteredRight(f, data.LogoBytes, data.LogoMIME, topMargin, separatorY)

	// ── MITGLIEDSDATEN ───────────────────────────────────────────────────────
	sectionHeader("MITGLIEDSDATEN")
	if data.MemberNumber != nil && *data.MemberNumber != "" {
		dataRow("Mitgliedsnummer:", *data.MemberNumber)
	}
	dataRow("Mitgliedstyp:", data.MemberType)
	if data.Firstname != "" || data.Lastname != "" {
		name := strings.TrimSpace(data.Titel + " " + data.Firstname + " " + data.Lastname)
		if data.TitelNach != "" {
			name = strings.TrimSpace(name + ", " + data.TitelNach)
		}
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
		if data.BankName != "" {
			dataRow("Bankname:", data.BankName)
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
		// PROJ-52: Zählpunkte werden in der offiziellen E-Control-Gruppierung
		// 2-6-5-20 ausgegeben, damit Mitglieder die Nummer auf der
		// Beitrittsbestätigung leichter mit Stromrechnung/Netzbetreiber-
		// Portal abgleichen können.
		f.CellFormat(col1, 5, w1252(shared.FormatMeteringPoint(mp.MeteringPoint)), "0", 0, "L", false, 0, "")
		f.CellFormat(col2, 5, w1252(mp.Direction), "0", 0, "L", false, 0, "")
		f.CellFormat(col3, 5, w1252(fmt.Sprintf("%d %%", mp.ParticipationFactor)), "0", 1, "R", false, 0, "")
		// PROJ-39: zusätzliche Zeile bei abweichender Adresse, eingerückt
		// unter der Zählpunktnummer damit die Zuordnung visuell klar bleibt.
		if mp.AddressLine != "" {
			setFont("I", 8)
			f.CellFormat(cw, 5, w1252("    Adresse: "+mp.AddressLine), "0", 1, "L", false, 0, "")
			setFont("", 9)
		}
		// PROJ-45: Erzeugungs-Details (Form, Batterie, Wechselrichter).
		if mp.GenerationLine != "" {
			setFont("I", 8)
			f.CellFormat(cw, 5, w1252("    Erzeugung: "+mp.GenerationLine), "0", 1, "L", false, 0, "")
			setFont("", 9)
		}
	}

	// ── ERTEILTE ZUSTIMMUNGEN ─────────────────────────────────────────────────
	sectionHeader("ERTEILTE ZUSTIMMUNGEN")
	setFont("", 9)
	if data.PrivacyAccepted {
		line := "- Datenschutzerklärung akzeptiert"
		if data.PrivacyVersion != "" {
			line += fmt.Sprintf(" (Version %s)", data.PrivacyVersion)
		}
		if data.PrivacyAcceptedAt != nil {
			line += " am " + fmtDateTime(*data.PrivacyAcceptedAt)
		}
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
	}
	if data.AccuracyConfirmed {
		line := "- Richtigkeit der Angaben bestätigt"
		if data.AccuracyConfirmedAt != nil {
			line += " am " + fmtDateTime(*data.AccuracyConfirmedAt)
		}
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
	}
	if data.SEPAMandateEnabled && data.SepaMandateAccepted {
		line := "- SEPA-Lastschriftmandat per E-Mail übermittelt"
		if data.SepaMandateAcceptedAt != nil {
			line += " am " + fmtDateTime(*data.SepaMandateAcceptedAt)
		}
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
	} else if !data.SEPAMandateEnabled {
		line := "- SEPA-Lastschriftmandat erteilt"
		if data.SepaMandateAcceptedAt != nil {
			line += " am " + fmtDateTime(*data.SepaMandateAcceptedAt)
		}
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
	}
	// PROJ-44: Netzbetreiber-Vollmacht als eigene Zustimmung mit
	// vollständigem Vertragstext + Zeitstempel. Wird nur gerendert,
	// wenn das Mitglied die Vollmacht aktiv erteilt hat.
	if data.NetworkOperatorAuthorization {
		header := "- Netzbetreiber-Vollmacht erteilt"
		if data.NetworkOperatorAuthorizationAt != nil {
			header += " am " + fmtDateTime(*data.NetworkOperatorAuthorizationAt)
		}
		f.MultiCell(cw, 5, w1252(header), "0", "L", false)
		setFont("I", 8)
		f.MultiCell(cw, 4, w1252("  „"+shared.NetworkOperatorAuthText+"\""), "0", "L", false)
		setFont("", 9)
	}
	// PROJ-36: render the two consent kinds as separate blocks so the audit
	// trail clearly distinguishes active acceptance from informational
	// acknowledgement. Order matches the form order: explicit first.
	for _, c := range data.Consents {
		if c.Informational {
			continue
		}
		line := fmt.Sprintf("- %s zugestimmt am %s", c.Title, fmtDateTime(c.ConsentedAt))
		f.MultiCell(cw, 5, w1252(line), "0", "L", false)
		if c.URL != "" {
			setFont("", 8)
			f.MultiCell(cw, 4, w1252("  "+c.URL), "0", "L", false)
			setFont("", 9)
		}
	}
	var informational []ConsentPDF
	for _, c := range data.Consents {
		if c.Informational {
			informational = append(informational, c)
		}
	}
	if len(informational) > 0 {
		f.Ln(2)
		setFont("B", 9)
		f.MultiCell(cw, 5, w1252("Zur Kenntnis genommene Dokumente:"), "0", "L", false)
		setFont("", 9)
		for _, c := range informational {
			line := fmt.Sprintf("- %s — Kenntnis genommen am %s", c.Title, fmtDateTime(c.ConsentedAt))
			f.MultiCell(cw, 5, w1252(line), "0", "L", false)
			if c.URL != "" {
				setFont("", 8)
				f.MultiCell(cw, 4, w1252("  "+c.URL), "0", "L", false)
				setFont("", 9)
			}
		}
	}
	sepaShown := (data.SEPAMandateEnabled && data.SepaMandateAccepted) || !data.SEPAMandateEnabled
	if !data.PrivacyAccepted && !data.AccuracyConfirmed && !sepaShown && len(data.Consents) == 0 {
		f.MultiCell(cw, 5, w1252("Keine Zustimmungen erfasst."), "0", "L", false)
	}

	// ── GENOSSENSCHAFTSANTEILE (PROJ-37) ─────────────────────────────────────
	if data.CooperativeSharesCount != nil && data.CooperativeShareAmountCents != nil {
		count := *data.CooperativeSharesCount
		amount := *data.CooperativeShareAmountCents
		total := int64(count) * amount
		// Inline EUR formatter — Cents → "1.234,56 €" with de-AT thousand
		// separator. Local to this section because no other PDF code
		// renders currency yet.
		formatEur := func(cents int64) string {
			euros := cents / 100
			rem := cents % 100
			if rem < 0 {
				rem = -rem
			}
			// thousand-separator on euro part
			s := fmt.Sprintf("%d", euros)
			var withDots []byte
			for i, c := range []byte(s) {
				if i > 0 && (len(s)-i)%3 == 0 {
					withDots = append(withDots, '.')
				}
				withDots = append(withDots, c)
			}
			return fmt.Sprintf("%s,%02d €", string(withDots), rem)
		}
		sectionHeader("GENOSSENSCHAFTSANTEILE")
		dataRow("Anzahl gezeichneter Anteile:", fmt.Sprintf("%d", count))
		dataRow("Wert je Anteil:", formatEur(amount))
		dataRow("Gesamtbetrag:", formatEur(total))
	}

	// ── WEITERE ANGABEN ───────────────────────────────────────────────────────
	if len(data.ConfigurableFields) > 0 {
		sectionHeader("WEITERE ANGABEN")
		for _, cf := range data.ConfigurableFields {
			dataRow(cf.Label+":", cf.Value)
		}
	}

	// ── PROJ-56: NETZBETREIBER-INFO-SEITE ────────────────────────────────────
	// Eigene Seite, gerendert nur wenn die Vollmacht erteilt wurde.
	// Enthält: Kundennummer + Inventarnummer + Vollmachts-Block + Zählpunkt-
	// Tabelle mit Adresse / Typ / Teilnahmefaktor — als Hilfe für die
	// Netzbetreiber-Korrespondenz der EEG-Verwaltung.
	if data.NetworkOperatorAuthorization {
		f.AddPage()

		// Überschrift
		setFont("B", 14)
		f.CellFormat(cw, 10, w1252("Informationen für den Netzbetreiber"), "0", 1, "L", false, 0, "")
		// Trennlinie unter dem Titel
		curY := f.GetY()
		f.Line(f.GetX(), curY, f.GetX()+cw, curY)
		f.Ln(4)

		// Zwei-Spalten-Header: Kundennummer + Inventarnummer
		setFont("B", 9)
		f.CellFormat(cw/2, 5, w1252("Netzbetreiber Kundennummer:"), "0", 0, "L", false, 0, "")
		f.CellFormat(cw/2, 5, w1252("Inventarnummer eines Zählers"), "0", 1, "L", false, 0, "")
		setFont("", 9)
		f.CellFormat(cw/2, 5, w1252(data.NetworkOperatorCustomerNumber), "0", 0, "L", false, 0, "")
		f.CellFormat(cw/2, 5, w1252(data.MeterInventoryNumber), "0", 1, "L", false, 0, "")
		f.Ln(4)

		// Vollmachtsblock: [X]-Box + Volltext + Timestamp
		// Box vor dem Text — manuell als Linien gezeichnet, plus "X" drin.
		boxX := f.GetX()
		boxY := f.GetY()
		boxSize := 4.0
		f.Rect(boxX, boxY, boxSize, boxSize, "D")
		// X-Symbol mittig
		setFont("B", 9)
		f.SetXY(boxX, boxY-0.5)
		f.CellFormat(boxSize, boxSize+1, w1252("X"), "0", 0, "C", false, 0, "")
		setFont("", 9)
		// Vollmachtstext rechts neben der Box, eingerückt
		textX := boxX + boxSize + 2
		f.SetXY(textX, boxY)
		f.MultiCell(cw-(boxSize+2), 4, w1252(shared.NetworkOperatorAuthText), "0", "L", false)
		f.Ln(1)
		setFont("B", 9)
		// Timestamp eingerückt unter dem Text
		f.SetX(textX)
		grantedAt := ""
		if data.SubmittedAt != nil {
			grantedAt = fmtDateTime(*data.SubmittedAt)
		} else if data.NetworkOperatorAuthorizationAt != nil {
			grantedAt = fmtDateTime(*data.NetworkOperatorAuthorizationAt)
		}
		f.CellFormat(cw-(boxSize+2), 5, w1252("Vollmacht erteilt am "+grantedAt), "0", 1, "L", false, 0, "")
		setFont("", 9)
		f.Ln(4)

		// Zählpunkt-Tabelle: 4 Spalten (ZP / Adresse / Typ / TF)
		// Spaltenbreiten in mm, Summe = cw
		colZP := cw * 0.50
		colAddr := cw * 0.30
		colTyp := cw * 0.10
		colTF := cw - (colZP + colAddr + colTyp)

		// Tabellen-Header mit dunklem Background
		setFont("B", 9)
		f.SetFillColor(220, 230, 240)
		f.CellFormat(colZP, 6, w1252("Zählpunktnummer"), "1", 0, "L", true, 0, "")
		f.CellFormat(colAddr, 6, w1252("Adresse"), "1", 0, "L", true, 0, "")
		f.CellFormat(colTyp, 6, w1252("Typ"), "1", 0, "C", true, 0, "")
		f.CellFormat(colTF, 6, w1252("TF"), "1", 1, "C", true, 0, "")
		setFont("", 8)
		f.SetFillColor(245, 248, 252)

		// Datenzeilen — jede ZP-Row 10mm hoch (2× 5mm).
		// ZP, Typ, TF werden auf der ersten 5mm-Zeile gerendert mit
		// Border-Top+Left+Right; auf der zweiten 5mm-Zeile mit
		// Border-Bottom+Left+Right (= ein Block aus Sicht des Lesers).
		// Adresse trägt auf jeder 5mm-Zeile eine Adressezeile.
		for _, mp := range data.MeteringPoints {
			typeCode := "CNSM"
			if mp.Direction == "Einspeisung" {
				typeCode = "GNRT"
			}
			zpFormatted := formatMeteringPointNumber(mp.MeteringPoint)
			tfStr := fmt.Sprintf("%d%%", mp.ParticipationFactor)

			// Erste 5mm-Zeile
			f.CellFormat(colZP, 5, w1252(zpFormatted), "LTR", 0, "L", true, 0, "")
			f.CellFormat(colAddr, 5, w1252(mp.AddressStreetLine), "LTR", 0, "L", true, 0, "")
			f.CellFormat(colTyp, 5, w1252(typeCode), "LTR", 0, "C", true, 0, "")
			f.CellFormat(colTF, 5, w1252(tfStr), "LTR", 1, "C", true, 0, "")
			// Zweite 5mm-Zeile (Adresse Line 2, sonst leer)
			f.CellFormat(colZP, 5, w1252(""), "LBR", 0, "L", true, 0, "")
			f.CellFormat(colAddr, 5, w1252(mp.AddressCityLine), "LBR", 0, "L", true, 0, "")
			f.CellFormat(colTyp, 5, w1252(""), "LBR", 0, "C", true, 0, "")
			f.CellFormat(colTF, 5, w1252(""), "LBR", 1, "C", true, 0, "")
		}
		setFont("", 9)
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
