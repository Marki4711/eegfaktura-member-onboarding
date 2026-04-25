package excel

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

const (
	markerText   = "[### Leerzeile für Importer ###]"
	sheetName    = "Sheet1"
	headerRow    = 7
	dataStartRow = 10
)

var columnHeaders = []string{
	"Netzbetreiber", "Gemeinschafts-ID", "Ortsgebiet", "PLZ", "Ort",
	"Straße", "Hausnummer", "Stiege", "Stock", "Tür", "Adresszusatz",
	"Zählpunkt", "Energierichtung", "EquipmentNr", "ObjektName",
	"Überschusseinspeisung", "Energiequelle", "Verteilungsmodell",
	"Zugeteilte Menge in Prozent", "TitelVor",
	"Name 1", "Name 2", "TitelNach", "BusinessRole",
	"Mitglied seit", "IBAN", "Kontoinhaber", "Bankname",
	"Email", "TelefonNr", "SteuerNr", "UmsatzsteuerNr",
	"MitgliedsNr", "Zählpunktstatus", "registriert seit", "Meter Codes",
}

// GenerateExcel produces an xlsx file matching the eegFaktura import template.
// eegID is written to column B (Gemeinschafts-ID); pass empty string if not configured.
// Template structure:
//   - Rows 1–6:  importer marker rows (with metadata in rows 2–5)
//   - Row 7:     column headers
//   - Rows 8–9:  importer marker rows
//   - Row 10+:   one data row per metering point
func GenerateExcel(app *shared.Application, meteringPoints []shared.MeteringPoint, eegID string) ([]byte, error) {
	if len(meteringPoints) == 0 {
		return nil, fmt.Errorf("application has no metering points")
	}

	f := excelize.NewFile()
	defer f.Close()

	if err := writeTemplateHeader(f); err != nil {
		return nil, err
	}

	for i, mp := range meteringPoints {
		if err := writeDataRow(f, dataStartRow+i, app, &mp, eegID); err != nil {
			return nil, fmt.Errorf("failed to write row %d: %w", dataStartRow+i, err)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to serialize excel: %w", err)
	}
	return buf.Bytes(), nil
}

// writeTemplateHeader writes the 9-row template header required by the eegFaktura importer.
func writeTemplateHeader(f *excelize.File) error {
	set := func(cell, val string) error {
		return f.SetCellValue(sheetName, cell, val)
	}

	// Row 1: marker only
	if err := set("A1", markerText); err != nil {
		return err
	}

	// Row 2: marker + EDA label
	if err := set("A2", markerText); err != nil {
		return err
	}
	if err := set("E2", "….Felder aus EDA"); err != nil {
		return err
	}

	// Row 3: marker + Faktura label
	if err := set("A3", markerText); err != nil {
		return err
	}
	if err := set("E3", "….Felder für Faktura"); err != nil {
		return err
	}

	// Row 4: marker + "Erforderlich" annotations
	if err := set("A4", markerText); err != nil {
		return err
	}
	for _, col := range []string{"B", "C", "D", "E", "F", "G", "L", "M", "U", "V", "AH"} {
		if err := set(col+"4", "Erforderlich"); err != nil {
			return err
		}
	}

	// Row 5: marker + field descriptions
	if err := set("A5", markerText); err != nil {
		return err
	}
	if err := set("M5", "CONSUMPTION oder GENERATION"); err != nil {
		return err
	}
	if err := set("U5", "Vorname (privat) od. Firmenname (business)"); err != nil {
		return err
	}
	if err := set("V5", "Nachname"); err != nil {
		return err
	}
	if err := set("X5", "privat oder business"); err != nil {
		return err
	}
	if err := set("Y5", "Default: Today\nString: \n<tag>.<monat>.<jahr>\nBsp:\n1.1.2023"); err != nil {
		return err
	}
	if err := set("AH5", "ACTIVE, ACTIVATED oder REGISTERED werden als aktivierte Zählpunkte übernommen. Zählpunkte mit Status NEW werden als neue, nicht aktivierte Zählpunkte übernommen."); err != nil {
		return err
	}
	if err := set("AI5", "Zählpunkt registriert seit. Default 1. Jan des aktuellen Jahres\nString: \n<tag>.<monat>.<jahr>\nBsp:\n1.1.2023"); err != nil {
		return err
	}

	// Row 6: marker only
	if err := set("A6", markerText); err != nil {
		return err
	}

	// Row 7: column headers
	for i, h := range columnHeaders {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return fmt.Errorf("invalid column index %d: %w", i+1, err)
		}
		if err := f.SetCellValue(sheetName, col+"7", h); err != nil {
			return fmt.Errorf("failed to set header cell: %w", err)
		}
	}

	// Rows 8–9: post-header marker rows
	if err := set("A8", markerText); err != nil {
		return err
	}
	if err := set("A9", markerText); err != nil {
		return err
	}

	return nil
}

func writeDataRow(f *excelize.File, rowNum int, app *shared.Application, mp *shared.MeteringPoint, eegID string) error {
	r := strconv.Itoa(rowNum)
	set := func(col, val string) error {
		return f.SetCellValue(sheetName, col+r, val)
	}
	setInt := func(col string, val int) error {
		return f.SetCellInt(sheetName, col+r, int64(val))
	}

	// A: Netzbetreiber — first 8 chars of metering point number (grid operator code)
	netzbetreiber := ""
	if len(mp.MeteringPoint) >= 8 {
		netzbetreiber = mp.MeteringPoint[:8]
	}
	if err := set("A", netzbetreiber); err != nil {
		return err
	}
	// B: Gemeinschafts-ID — eeg_id from registration_entrypoint
	if err := set("B", eegID); err != nil {
		return err
	}
	// C: Ortsgebiet — default LOKAL
	if err := set("C", "LOKAL"); err != nil {
		return err
	}
	// D: PLZ
	if err := set("D", app.ResidentZip); err != nil {
		return err
	}
	// E: Ort
	if err := set("E", app.ResidentCity); err != nil {
		return err
	}
	// F: Straße
	if err := set("F", app.ResidentStreet); err != nil {
		return err
	}
	// G: Hausnummer
	if err := set("G", app.ResidentStreetNumber); err != nil {
		return err
	}
	// H–K: Stiege, Stock, Tür, Adresszusatz — empty
	// L: Zählpunkt
	if err := set("L", mp.MeteringPoint); err != nil {
		return err
	}
	// M: Energierichtung — CONSUMPTION or GENERATION
	direction := string(mp.Direction)
	if mp.Direction == shared.DirectionProduction {
		direction = "GENERATION"
	}
	if err := set("M", direction); err != nil {
		return err
	}
	// N: EquipmentNr
	if mp.Transformer != nil {
		if err := set("N", *mp.Transformer); err != nil {
			return err
		}
	}
	// O: ObjektName
	if mp.InstallationName != nil {
		if err := set("O", *mp.InstallationName); err != nil {
			return err
		}
	}
	// P: Überschusseinspeisung — empty
	// Q: Energiequelle — empty
	// R: Verteilungsmodell — empty
	// S: Zugeteilte Menge in Prozent
	if err := setInt("S", mp.ParticipationFactor); err != nil {
		return err
	}
	// T: TitelVor — empty
	// U: Name 1 — first name (private) or company name (business)
	name1 := ""
	if app.CompanyName != nil && *app.CompanyName != "" {
		name1 = *app.CompanyName
	} else if app.Firstname != nil {
		name1 = *app.Firstname
	}
	if err := set("U", name1); err != nil {
		return err
	}
	// V: Name 2 — last name for private only, empty for business
	name2 := ""
	if (app.CompanyName == nil || *app.CompanyName == "") && app.Lastname != nil {
		name2 = *app.Lastname
	}
	if err := set("V", name2); err != nil {
		return err
	}
	// W: TitelNach — empty
	// X: BusinessRole
	if err := set("X", mapBusinessRole(app.MemberType)); err != nil {
		return err
	}
	// Y: Mitglied seit
	if app.MembershipStartDate != nil {
		if err := set("Y", formatDate(*app.MembershipStartDate)); err != nil {
			return err
		}
	}
	// Z: IBAN
	if app.IBAN != nil {
		if err := set("Z", *app.IBAN); err != nil {
			return err
		}
	}
	// AA: Kontoinhaber
	if app.AccountHolder != nil {
		if err := set("AA", *app.AccountHolder); err != nil {
			return err
		}
	}
	// AB: Bankname — empty
	// AC: Email
	if err := set("AC", app.Email); err != nil {
		return err
	}
	// AD: TelefonNr
	if app.Phone != nil {
		if err := set("AD", *app.Phone); err != nil {
			return err
		}
	}
	// AE: SteuerNr — empty
	// AF: UmsatzsteuerNr
	if app.UIDNumber != nil {
		if err := set("AF", *app.UIDNumber); err != nil {
			return err
		}
	}
	// AG: MitgliedsNr — leer lassen (wird in eegFaktura vergeben)
	// AH: Zählpunktstatus — NEW (Zählpunkt wird neu angelegt)
	if err := set("AH", "NEW"); err != nil {
		return err
	}
	// AI: registriert seit
	if err := set("AI", formatDate(app.CreatedAt)); err != nil {
		return err
	}
	// AJ: Meter Codes — empty

	return nil
}

func mapBusinessRole(mt shared.MemberType) string {
	switch mt {
	case shared.MemberTypePrivate, shared.MemberTypeFarmer:
		return "privat"
	default:
		return "business"
	}
}

// formatDate formats a time as D.M.YYYY (e.g. 1.4.2026) per eegFaktura import spec.
func formatDate(t time.Time) string {
	return fmt.Sprintf("%d.%d.%d", t.Day(), int(t.Month()), t.Year())
}
