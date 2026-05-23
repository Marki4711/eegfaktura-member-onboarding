package excel

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/dataexport"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// =====================================================================
// ValidateConfig
// =====================================================================

func TestValidateConfig_AcceptsValid(t *testing.T) {
	p := &Plugin{}
	err := p.ValidateConfig(map[string]interface{}{
		"format": "xlsx",
		"columns": []interface{}{
			map[string]interface{}{"header": "Vorname", "field": "firstname", "format": "string"},
			map[string]interface{}{"header": "E-Mail", "field": "email", "format": "string"},
		},
	})
	if err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateConfig_RejectsUnknownFormat(t *testing.T) {
	p := &Plugin{}
	err := p.ValidateConfig(map[string]interface{}{
		"format": "pdf",
		"columns": []interface{}{
			map[string]interface{}{"header": "Vorname", "field": "firstname", "format": "string"},
		},
	})
	verr, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, ok := verr.Fields["format"]; !ok {
		t.Errorf("expected 'format' field error, got %v", verr.Fields)
	}
}

func TestValidateConfig_RejectsEmptyColumns(t *testing.T) {
	p := &Plugin{}
	err := p.ValidateConfig(map[string]interface{}{
		"format":  "xlsx",
		"columns": []interface{}{},
	})
	verr, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, ok := verr.Fields["columns"]; !ok {
		t.Errorf("expected 'columns' field error, got %v", verr.Fields)
	}
}

func TestValidateConfig_RejectsTooManyColumns(t *testing.T) {
	p := &Plugin{}
	cols := make([]interface{}, maxColumns+1)
	for i := range cols {
		cols[i] = map[string]interface{}{
			"header": "h" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)),
			"field":  "firstname",
			"format": "string",
		}
	}
	err := p.ValidateConfig(map[string]interface{}{
		"format":  "xlsx",
		"columns": cols,
	})
	verr, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if _, ok := verr.Fields["columns"]; !ok {
		t.Errorf("expected 'columns' field error, got %v", verr.Fields)
	}
}

func TestValidateConfig_RejectsUnknownField(t *testing.T) {
	p := &Plugin{}
	err := p.ValidateConfig(map[string]interface{}{
		"format": "xlsx",
		"columns": []interface{}{
			map[string]interface{}{"header": "X", "field": "no_such_field", "format": "string"},
		},
	})
	verr, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	found := false
	for k := range verr.Fields {
		if strings.Contains(k, ".field") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a .field error in %v", verr.Fields)
	}
}

func TestValidateConfig_RejectsInvalidFormatForField(t *testing.T) {
	p := &Plugin{}
	err := p.ValidateConfig(map[string]interface{}{
		"format": "xlsx",
		"columns": []interface{}{
			// birth_date is a date field — `string` is not a valid format for it.
			map[string]interface{}{"header": "Geburt", "field": "birth_date", "format": "string"},
		},
	})
	verr, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	found := false
	for k := range verr.Fields {
		if strings.Contains(k, ".format") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a .format error in %v", verr.Fields)
	}
}

func TestValidateConfig_RejectsEmptyHeader(t *testing.T) {
	p := &Plugin{}
	err := p.ValidateConfig(map[string]interface{}{
		"format": "xlsx",
		"columns": []interface{}{
			map[string]interface{}{"header": "", "field": "firstname", "format": "string"},
		},
	})
	verr, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	found := false
	for k := range verr.Fields {
		if strings.Contains(k, ".header") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a .header error in %v", verr.Fields)
	}
}

func TestValidateConfig_RejectsDuplicateHeader(t *testing.T) {
	p := &Plugin{}
	err := p.ValidateConfig(map[string]interface{}{
		"format": "xlsx",
		"columns": []interface{}{
			map[string]interface{}{"header": "Name", "field": "firstname", "format": "string"},
			map[string]interface{}{"header": "Name", "field": "lastname", "format": "string"},
		},
	})
	verr, ok := err.(shared.ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(verr.Fields) == 0 {
		t.Errorf("expected duplicate-header error, got none")
	}
}

// =====================================================================
// Format transformations
// =====================================================================

func TestFormatValue_Date(t *testing.T) {
	d := time.Date(2026, 5, 23, 14, 30, 0, 0, time.UTC)
	cases := map[string]string{
		"date_dmy":    "23.05.2026",
		"date_iso":    "2026-05-23",
		"date_dmy_hm": "23.05.2026 14:30",
		"":            "23.05.2026", // default = date_dmy
	}
	for fmt, expected := range cases {
		got := formatValue(d, FieldTypeDate, fmt, nil)
		if got != expected {
			t.Errorf("format %q: expected %q, got %q", fmt, expected, got)
		}
	}
}

func TestFormatValue_Bool(t *testing.T) {
	cases := []struct {
		val      bool
		format   string
		expected string
	}{
		{true, "", "Ja"},
		{false, "", "Nein"},
		{true, "bool_tf", "true"},
		{false, "bool_10", "0"},
		{true, "bool_yn_short", "Y"},
	}
	for _, c := range cases {
		got := formatValue(c.val, FieldTypeBool, c.format, nil)
		if got != c.expected {
			t.Errorf("bool(%v, %q): expected %q, got %q", c.val, c.format, c.expected, got)
		}
	}
}

func TestFormatValue_Enum_UsesLabelWhenAvailable(t *testing.T) {
	labels := map[string]string{"private": "Privatperson"}
	if got := formatValue("private", FieldTypeEnum, "enum_label", labels); got != "Privatperson" {
		t.Errorf("expected label, got %q", got)
	}
	if got := formatValue("private", FieldTypeEnum, "enum_value", labels); got != "private" {
		t.Errorf("expected raw value, got %q", got)
	}
	// Unknown raw value falls back to raw.
	if got := formatValue("xyz", FieldTypeEnum, "enum_label", labels); got != "xyz" {
		t.Errorf("expected fallback to raw, got %q", got)
	}
}

func TestFormatValue_Number_GermanDefault(t *testing.T) {
	if got := formatValue(1.5, FieldTypeNumber, "", nil); got != "1,5" {
		t.Errorf("default DE: expected '1,5', got %q", got)
	}
	if got := formatValue(1.5, FieldTypeNumber, "number_iso", nil); got != "1.5" {
		t.Errorf("ISO: expected '1.5', got %q", got)
	}
}

func TestFormatValue_NilReturnsEmpty(t *testing.T) {
	if got := formatValue(nil, FieldTypeText, "string", nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
	if got := formatValue((*time.Time)(nil), FieldTypeDate, "date_dmy", nil); got != "" {
		t.Errorf("expected empty for nil date, got %q", got)
	}
}

func TestFormatValue_Multi(t *testing.T) {
	got := formatValue([]string{"AT00001", "AT00002"}, FieldTypeMulti, "comma_separated", nil)
	if got != "AT00001, AT00002" {
		t.Errorf("expected comma-joined, got %q", got)
	}
}

// =====================================================================
// Renderer end-to-end
// =====================================================================

func makeAppSnapshot(firstname, lastname, email string) dataexport.ApplicationSnapshot {
	first := firstname
	last := lastname
	return dataexport.ApplicationSnapshot{
		Application: &shared.Application{
			ID:        uuid.New(),
			RCNumber:  "AT00001",
			Firstname: &first,
			Lastname:  &last,
			Email:     email,
		},
		MeteringPoints: nil,
	}
}

func TestRenderCSV_StructureAndBOM(t *testing.T) {
	cfg := excelConfig{
		Format: FormatCSV,
		Columns: []columnConfig{
			{Header: "Vorname", Field: "firstname", Format: "string"},
			{Header: "Nachname", Field: "lastname", Format: "string"},
			{Header: "E-Mail", Field: "email", Format: "string"},
		},
	}
	apps := []dataexport.ApplicationSnapshot{
		makeAppSnapshot("Max", "Mustermann", "max@example.com"),
		makeAppSnapshot("Erika", "Musterfrau", "erika@example.com"),
	}
	out, err := renderCSV(cfg, apps, nil)
	if err != nil {
		t.Fatalf("renderCSV: %v", err)
	}
	// BOM
	if len(out) < 3 || out[0] != 0xEF || out[1] != 0xBB || out[2] != 0xBF {
		t.Errorf("expected UTF-8 BOM, got %x", out[:min(3, len(out))])
	}
	// Parse as CSV with semicolon and verify rows.
	r := csv.NewReader(bytes.NewReader(out[3:])) // skip BOM
	r.Comma = ';'
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("expected 3 records (header + 2 rows), got %d", len(records))
	}
	if records[0][0] != "Vorname" {
		t.Errorf("header[0]: %q", records[0][0])
	}
	if records[1][0] != "Max" || records[1][1] != "Mustermann" || records[1][2] != "max@example.com" {
		t.Errorf("row 1: %v", records[1])
	}
}

func TestRenderXLSX_ProducesValidArchive(t *testing.T) {
	cfg := excelConfig{
		Format: FormatXLSX,
		Columns: []columnConfig{
			{Header: "Vorname", Field: "firstname", Format: "string"},
			{Header: "E-Mail", Field: "email", Format: "string"},
		},
	}
	apps := []dataexport.ApplicationSnapshot{
		makeAppSnapshot("Max", "Mustermann", "max@example.com"),
	}
	out, err := renderXLSX(cfg, apps, nil)
	if err != nil {
		t.Fatalf("renderXLSX: %v", err)
	}
	// XLSX is a ZIP archive — magic bytes "PK".
	if len(out) < 2 || out[0] != 'P' || out[1] != 'K' {
		t.Errorf("expected XLSX (ZIP) magic 'PK', got %x", out[:min(2, len(out))])
	}
}

func TestRenderCSV_ProgressCalled(t *testing.T) {
	cfg := excelConfig{
		Format: FormatCSV,
		Columns: []columnConfig{
			{Header: "Vorname", Field: "firstname", Format: "string"},
		},
	}
	apps := make([]dataexport.ApplicationSnapshot, 120)
	for i := range apps {
		apps[i] = makeAppSnapshot("X", "Y", "z@z.at")
	}
	calls := []int{}
	progress := func(n int) { calls = append(calls, n) }
	_, err := renderCSV(cfg, apps, progress)
	if err != nil {
		t.Fatalf("renderCSV: %v", err)
	}
	if len(calls) == 0 {
		t.Errorf("expected progress callback to be invoked")
	}
	if calls[len(calls)-1] != 120 {
		t.Errorf("final progress should be total (120), got %d", calls[len(calls)-1])
	}
}

// =====================================================================
// Process: end-to-end through Plugin.Process
// =====================================================================

func TestProcess_Csv(t *testing.T) {
	p := &Plugin{}
	res, err := p.Process(context.Background(),
		map[string]interface{}{
			"format": "csv",
			"columns": []interface{}{
				map[string]interface{}{"header": "E-Mail", "field": "email", "format": "string"},
			},
		},
		[]dataexport.ApplicationSnapshot{makeAppSnapshot("M", "M", "m@example.com")},
		nil,
	)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	dl, ok := res.(dataexport.DownloadResult)
	if !ok {
		t.Fatalf("expected DownloadResult, got %T", res)
	}
	if dl.MimeType != "text/csv; charset=utf-8" {
		t.Errorf("mime: %q", dl.MimeType)
	}
	if !strings.HasSuffix(dl.FileName, ".csv") {
		t.Errorf("filename should end .csv: %q", dl.FileName)
	}
	if !strings.Contains(string(dl.Bytes), "m@example.com") {
		t.Errorf("output should contain the email")
	}
}

func TestProcess_Xlsx(t *testing.T) {
	p := &Plugin{}
	res, err := p.Process(context.Background(),
		map[string]interface{}{
			"format": "xlsx",
			"columns": []interface{}{
				map[string]interface{}{"header": "E-Mail", "field": "email", "format": "string"},
			},
		},
		[]dataexport.ApplicationSnapshot{makeAppSnapshot("M", "M", "m@example.com")},
		nil,
	)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	dl, ok := res.(dataexport.DownloadResult)
	if !ok {
		t.Fatalf("expected DownloadResult, got %T", res)
	}
	if dl.MimeType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("mime: %q", dl.MimeType)
	}
	if !strings.HasSuffix(dl.FileName, ".xlsx") {
		t.Errorf("filename should end .xlsx: %q", dl.FileName)
	}
}

// =====================================================================
// CSV/Excel-Injection defence (Medium-Finding fix)
// =====================================================================

func TestSanitiseSpreadsheetValue_DefangesFormulaPrefixes(t *testing.T) {
	cases := map[string]string{
		"":                                  "",
		"Max":                               "Max",                                  // safe untouched
		"=HYPERLINK(\"http://evil/\",\"x\")": "'=HYPERLINK(\"http://evil/\",\"x\")", // formula
		"+1234":                             "'+1234",                               // formula
		"-7":                                "'-7",                                  // formula
		"@SUM(A1:A2)":                       "'@SUM(A1:A2)",                         // formula
		"\tinjected":                        "'\tinjected",                          // tab (DDE)
		"\rinjected":                        "'\rinjected",                          // CR
	}
	for in, expected := range cases {
		got := sanitiseSpreadsheetValue(in)
		if got != expected {
			t.Errorf("sanitise(%q): expected %q, got %q", in, expected, got)
		}
	}
}

// LibreOffice/Excel-Importmodi trimmen führende Whitespace vor Formel-
// Erkennung — auch NBSP (U+00A0) und BOM (U+FEFF) zählen dazu. Der
// sanitiser muss diese Bypass-Pfade abdecken.
func TestSanitiseSpreadsheetValue_DefangesLeadingWhitespaceBypass(t *testing.T) {
	cases := map[string]string{
		" =SUM(A1:A99)":      "' =SUM(A1:A99)",
		"  +1234":            "'  +1234",
		"\t=HYPERLINK(\"x\")": "'\t=HYPERLINK(\"x\")",
		"\u00a0=evil":        "'\u00a0=evil", // NBSP
		"\ufeff=evil":        "'\ufeff=evil", // BOM
		" Max Mustermann":    " Max Mustermann", // harmlos, untouched
		"   ":                "   ",             // all-whitespace untouched
	}
	for in, expected := range cases {
		got := sanitiseSpreadsheetValue(in)
		if got != expected {
			t.Errorf("sanitise(%q): expected %q, got %q", in, expected, got)
		}
	}
}

func TestRenderCSV_DefangsFormulaInjection(t *testing.T) {
	// Member's lastname contains a hostile formula payload.
	cfg := excelConfig{
		Format: FormatCSV,
		Columns: []columnConfig{
			{Header: "Vorname", Field: "firstname", Format: "string"},
			{Header: "Nachname", Field: "lastname", Format: "string"},
		},
	}
	apps := []dataexport.ApplicationSnapshot{
		makeAppSnapshot("Max", `=HYPERLINK("http://evil/?","klick")`, "x@y.at"),
	}
	out, err := renderCSV(cfg, apps, nil)
	if err != nil {
		t.Fatalf("renderCSV: %v", err)
	}
	r := csv.NewReader(bytes.NewReader(out[3:])) // skip BOM
	r.Comma = ';'
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v", err)
	}
	// data row 1, column "Nachname" — first character must be the leading apostrophe.
	if !strings.HasPrefix(records[1][1], "'=HYPERLINK") {
		t.Errorf("expected lastname to be defanged with leading apostrophe, got %q", records[1][1])
	}
}

// =====================================================================
// StandardConfigs
// =====================================================================

func TestStandardConfigs_AllPassValidateConfig(t *testing.T) {
	p := &Plugin{}
	configs := p.StandardConfigs()
	if len(configs) == 0 {
		t.Fatal("expected at least one standard config")
	}
	for _, sc := range configs {
		if err := p.ValidateConfig(sc.Config); err != nil {
			t.Errorf("standard config %q failed validation: %v", sc.Name, err)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
