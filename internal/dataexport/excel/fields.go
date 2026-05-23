package excel

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/dataexport"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// FieldType drives which Format options are valid for a field.
type FieldType string

const (
	FieldTypeText   FieldType = "text"
	FieldTypeDate   FieldType = "date"
	FieldTypeBool   FieldType = "bool"
	FieldTypeEnum   FieldType = "enum"
	FieldTypeNumber FieldType = "number"
	FieldTypeMulti  FieldType = "multi" // multi-value (e.g. meter list)
)

// FieldDefinition describes one selectable column source.
type FieldDefinition struct {
	Key         string
	Label       string
	Category    string // for UI grouping
	Type        FieldType
	Extract     func(app dataexport.ApplicationSnapshot) interface{}
	EnumLabels  map[string]string // for FieldTypeEnum: raw value → human-readable
}

// SupportsFormat reports whether the given format is valid for this
// field's type. Empty format counts as the default.
func (f FieldDefinition) SupportsFormat(format string) bool {
	if format == "" {
		return true
	}
	for _, valid := range formatsForType(f.Type) {
		if format == valid {
			return true
		}
	}
	return false
}

func formatsForType(t FieldType) []string {
	switch t {
	case FieldTypeText:
		return []string{"string"}
	case FieldTypeDate:
		return []string{"date_dmy", "date_iso", "date_dmy_hm"}
	case FieldTypeBool:
		return []string{"bool_yn", "bool_tf", "bool_10", "bool_yn_short"}
	case FieldTypeEnum:
		return []string{"enum_value", "enum_label"}
	case FieldTypeNumber:
		return []string{"number_de", "number_iso"}
	case FieldTypeMulti:
		return []string{"comma_separated"}
	}
	return nil
}

// formatValue applies the requested format to a raw value. Returns ""
// for nil values.
func formatValue(value interface{}, fieldType FieldType, format string, enumLabels map[string]string) string {
	if value == nil {
		return ""
	}
	if format == "" {
		format = defaultFormat(fieldType)
	}
	switch fieldType {
	case FieldTypeText:
		return fmt.Sprint(value)
	case FieldTypeDate:
		t, ok := asTime(value)
		if !ok {
			return ""
		}
		switch format {
		case "date_iso":
			return t.Format("2006-01-02")
		case "date_dmy_hm":
			return t.Format("02.01.2006 15:04")
		default: // date_dmy
			return t.Format("02.01.2006")
		}
	case FieldTypeBool:
		b, ok := asBool(value)
		if !ok {
			return ""
		}
		switch format {
		case "bool_tf":
			return strconv.FormatBool(b)
		case "bool_10":
			if b {
				return "1"
			}
			return "0"
		case "bool_yn_short":
			if b {
				return "Y"
			}
			return "N"
		default: // bool_yn
			if b {
				return "Ja"
			}
			return "Nein"
		}
	case FieldTypeEnum:
		raw := fmt.Sprint(value)
		if format == "enum_label" {
			if label, ok := enumLabels[raw]; ok {
				return label
			}
		}
		return raw
	case FieldTypeNumber:
		f64, ok := asFloat(value)
		if !ok {
			return fmt.Sprint(value)
		}
		switch format {
		case "number_iso":
			return strconv.FormatFloat(f64, 'f', -1, 64)
		default: // number_de
			s := strconv.FormatFloat(f64, 'f', -1, 64)
			return strings.Replace(s, ".", ",", 1)
		}
	case FieldTypeMulti:
		strs, ok := value.([]string)
		if !ok {
			return fmt.Sprint(value)
		}
		return strings.Join(strs, ", ")
	}
	return fmt.Sprint(value)
}

func defaultFormat(t FieldType) string {
	switch t {
	case FieldTypeDate:
		return "date_dmy"
	case FieldTypeBool:
		return "bool_yn"
	case FieldTypeEnum:
		return "enum_value"
	case FieldTypeNumber:
		return "number_de"
	case FieldTypeMulti:
		return "comma_separated"
	}
	return "string"
}

func asTime(v interface{}) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case *time.Time:
		if t == nil {
			return time.Time{}, false
		}
		return *t, true
	}
	return time.Time{}, false
}

func asBool(v interface{}) (bool, bool) {
	switch b := v.(type) {
	case bool:
		return b, true
	case *bool:
		if b == nil {
			return false, false
		}
		return *b, true
	}
	return false, false
}

func asFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case *float64:
		if n == nil {
			return 0, false
		}
		return *n, true
	case *int:
		if n == nil {
			return 0, false
		}
		return float64(*n), true
	case *int64:
		if n == nil {
			return 0, false
		}
		return float64(*n), true
	}
	return 0, false
}

// derefStr extracts a non-nil string value from a *string field.
func derefStr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// MemberTypeLabels maps MemberType constants to human-readable German labels
// for enum_label rendering.
var MemberTypeLabels = map[string]string{
	"private":         "Privatperson",
	"sole_proprietor": "Kleinunternehmer",
	"farmer":          "Pauschalierter Landwirt",
	"municipality":    "Gemeinde",
	"company":         "Unternehmen",
	"association":     "Verein",
}

// StatusLabels for ApplicationStatus enum.
var StatusLabels = map[string]string{
	"draft":                      "Entwurf",
	"submitted":                  "Eingereicht",
	"email_confirmed":            "E-Mail bestätigt",
	"under_review":               "In Prüfung",
	"needs_info":                 "Rückfragen",
	"approved":                   "Genehmigt",
	"rejected":                   "Abgelehnt",
	"imported":                   "Importiert",
	"import_failed":              "Import fehlgeschlagen",
	"awaiting_bank_confirmation": "Wartet auf Bank-Bestätigung",
	"ready_for_activation":       "Bereit zur Aktivierung",
	"activated":                  "Aktiviert",
}

// EinzugsartLabels for the Einzugsart enum.
var EinzugsartLabels = map[string]string{
	"basis": "SEPA-Basismandat",
	"b2b":   "SEPA-Firmenmandat",
}

// AvailableFields is the central catalogue of fields admins can pick in
// the column mapping. Keyed by the persistent field-key stored in the
// column-config.
var AvailableFields = map[string]FieldDefinition{
	// Stammdaten
	"member_type": {
		Key: "member_type", Label: "Mitgliedstyp", Category: "Stammdaten",
		Type:       FieldTypeEnum,
		EnumLabels: MemberTypeLabels,
		Extract:    func(a dataexport.ApplicationSnapshot) interface{} { return string(a.Application.MemberType) },
	},
	"titel": {
		Key: "titel", Label: "Titel vor", Category: "Stammdaten",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.Titel) },
	},
	"firstname": {
		Key: "firstname", Label: "Vorname", Category: "Stammdaten",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.Firstname) },
	},
	"lastname": {
		Key: "lastname", Label: "Nachname", Category: "Stammdaten",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.Lastname) },
	},
	"titel_nach": {
		Key: "titel_nach", Label: "Titel nach", Category: "Stammdaten",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.TitelNach) },
	},
	"birth_date": {
		Key: "birth_date", Label: "Geburtsdatum", Category: "Stammdaten",
		Type: FieldTypeDate, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.BirthDate },
	},
	"company_name": {
		Key: "company_name", Label: "Firmenname", Category: "Stammdaten",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.CompanyName) },
	},
	"uid_number": {
		Key: "uid_number", Label: "UID-Nummer", Category: "Stammdaten",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.UIDNumber) },
	},
	"register_number": {
		Key: "register_number", Label: "Firmenbuch-Nr.", Category: "Stammdaten",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.RegisterNumber) },
	},

	// Kontakt
	"email": {
		Key: "email", Label: "E-Mail", Category: "Kontakt",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.Email },
	},
	"phone": {
		Key: "phone", Label: "Telefon", Category: "Kontakt",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.Phone) },
	},

	// Adresse
	"resident_street": {
		Key: "resident_street", Label: "Straße", Category: "Adresse",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.ResidentStreet },
	},
	"resident_street_number": {
		Key: "resident_street_number", Label: "Hausnummer", Category: "Adresse",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.ResidentStreetNumber },
	},
	"resident_zip": {
		Key: "resident_zip", Label: "PLZ", Category: "Adresse",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.ResidentZip },
	},
	"resident_city": {
		Key: "resident_city", Label: "Ort", Category: "Adresse",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.ResidentCity },
	},

	// Bank
	"iban": {
		Key: "iban", Label: "IBAN", Category: "Bank",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.IBAN) },
	},
	"account_holder": {
		Key: "account_holder", Label: "Kontoinhaber", Category: "Bank",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.AccountHolder) },
	},
	"einzugsart": {
		Key: "einzugsart", Label: "Einzugsart", Category: "Bank",
		Type:       FieldTypeEnum,
		EnumLabels: EinzugsartLabels,
		Extract:    func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.Einzugsart },
	},

	// EEG
	"rc_number": {
		Key: "rc_number", Label: "RC-Nummer", Category: "EEG",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.RCNumber },
	},
	"reference_number": {
		Key: "reference_number", Label: "Referenznummer", Category: "EEG",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.ReferenceNumber },
	},
	"member_number": {
		Key: "member_number", Label: "Mitgliedsnummer", Category: "EEG",
		Type: FieldTypeText, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return derefStr(a.Application.MemberNumber) },
	},
	"membership_start_date": {
		Key: "membership_start_date", Label: "Beitrittsdatum", Category: "EEG",
		Type: FieldTypeDate, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.MembershipStartDate },
	},
	"status": {
		Key: "status", Label: "Status", Category: "EEG",
		Type:       FieldTypeEnum,
		EnumLabels: StatusLabels,
		Extract:    func(a dataexport.ApplicationSnapshot) interface{} { return string(a.Application.Status) },
	},
	"imported_at": {
		Key: "imported_at", Label: "Importiert am", Category: "EEG",
		Type: FieldTypeDate, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.ImportedAt },
	},
	"activated_at": {
		Key: "activated_at", Label: "Aktiviert am", Category: "EEG",
		Type: FieldTypeDate, Extract: func(a dataexport.ApplicationSnapshot) interface{} { return a.Application.ActivatedAt },
	},

	// Zählpunkte
	"meter_count": {
		Key: "meter_count", Label: "Anzahl Zählpunkte", Category: "Zählpunkte",
		Type:    FieldTypeNumber,
		Extract: func(a dataexport.ApplicationSnapshot) interface{} { return len(a.MeteringPoints) },
	},
	"meter_numbers": {
		Key: "meter_numbers", Label: "Zählpunkte", Category: "Zählpunkte",
		Type: FieldTypeMulti,
		Extract: func(a dataexport.ApplicationSnapshot) interface{} {
			out := make([]string, len(a.MeteringPoints))
			for i, mp := range a.MeteringPoints {
				out[i] = mp.MeteringPoint
			}
			return out
		},
	},
}

// suppress unused import warnings for shared package referenced via types
var _ = shared.MemberTypePrivate
