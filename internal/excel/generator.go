package excel

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
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
// One data row is written per metering point; member data is repeated for each row.
func GenerateExcel(app *shared.Application, meteringPoints []shared.MeteringPoint) ([]byte, error) {
	if len(meteringPoints) == 0 {
		return nil, fmt.Errorf("application has no metering points")
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"

	// Row 1: column headers
	for i, h := range columnHeaders {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return nil, fmt.Errorf("invalid column index %d: %w", i+1, err)
		}
		if err := f.SetCellValue(sheet, col+"1", h); err != nil {
			return nil, fmt.Errorf("failed to set header cell: %w", err)
		}
	}

	// Row 2: importer marker (required by eegFaktura import logic)
	if err := f.SetCellValue(sheet, "A2", "[### Leerzeile für Importer ###]"); err != nil {
		return nil, fmt.Errorf("failed to set marker row: %w", err)
	}

	// Rows 3+: one row per metering point
	for i, mp := range meteringPoints {
		if err := writeDataRow(f, sheet, i+3, app, &mp); err != nil {
			return nil, fmt.Errorf("failed to write row %d: %w", i+3, err)
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to serialize excel: %w", err)
	}
	return buf.Bytes(), nil
}

func writeDataRow(f *excelize.File, sheet string, rowNum int, app *shared.Application, mp *shared.MeteringPoint) error {
	r := strconv.Itoa(rowNum)
	set := func(col, val string) error {
		return f.SetCellValue(sheet, col+r, val)
	}
	setInt := func(col string, val int) error {
		return f.SetCellInt(sheet, col+r, int64(val))
	}

	// A: Netzbetreiber — empty (V1)
	// B: Gemeinschafts-ID
	if err := set("B", app.RCNumber); err != nil {
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
	// H-K: Stiege, Stock, Tür, Adresszusatz — empty
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
	// P-R: empty
	// S: Zugeteilte Menge in Prozent
	if err := setInt("S", mp.ParticipationFactor); err != nil {
		return err
	}
	// T: TitelVor — empty
	// U: Name 1 — company name (business) or last name (private)
	name1 := ""
	if app.CompanyName != nil && *app.CompanyName != "" {
		name1 = *app.CompanyName
	} else if app.Lastname != nil {
		name1 = *app.Lastname
	}
	if err := set("U", name1); err != nil {
		return err
	}
	// V: Name 2 — first name for private only
	name2 := ""
	if (app.CompanyName == nil || *app.CompanyName == "") && app.Firstname != nil {
		name2 = *app.Firstname
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
	// AG: MitgliedsNr
	if err := set("AG", app.ReferenceNumber); err != nil {
		return err
	}
	// AH: Zählpunktstatus — default ACTIVATED
	if err := set("AH", "ACTIVATED"); err != nil {
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
