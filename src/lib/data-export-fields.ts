// PROJ-60: catalogue of available Excel-export fields, mirrored from
// internal/dataexport/excel/fields.go. Keep in sync — adding a field
// requires both: backend Extract + this frontend label/format map.
//
// Hardcoded here (not fetched) to avoid a network roundtrip per editor open;
// same pattern as CONFIGURABLE_FIELDS for PROJ-8.

export type ExcelFieldType = "text" | "date" | "bool" | "enum" | "number" | "multi";

export interface ExcelFieldDef {
  key: string;
  label: string;
  category: string;
  type: ExcelFieldType;
  // PROJ-60 DSGVO: marks fields that carry sensitive personal data so the
  // editor can render a warn-popover when admins add them to a config.
  sensitive?: boolean;
}

export const EXCEL_FIELD_CATALOG: ExcelFieldDef[] = [
  // Stammdaten
  { key: "member_type", label: "Mitgliedstyp", category: "Stammdaten", type: "enum" },
  { key: "titel", label: "Titel vor", category: "Stammdaten", type: "text" },
  { key: "firstname", label: "Vorname", category: "Stammdaten", type: "text" },
  { key: "lastname", label: "Nachname", category: "Stammdaten", type: "text" },
  { key: "titel_nach", label: "Titel nach", category: "Stammdaten", type: "text" },
  { key: "birth_date", label: "Geburtsdatum", category: "Stammdaten", type: "date", sensitive: true },
  { key: "company_name", label: "Firmenname", category: "Stammdaten", type: "text" },
  { key: "uid_number", label: "UID-Nummer", category: "Stammdaten", type: "text" },
  { key: "register_number", label: "Firmenbuch-Nr.", category: "Stammdaten", type: "text" },

  // Kontakt
  { key: "email", label: "E-Mail", category: "Kontakt", type: "text" },
  { key: "phone", label: "Telefon", category: "Kontakt", type: "text" },

  // Adresse
  { key: "resident_street", label: "Straße", category: "Adresse", type: "text" },
  { key: "resident_street_number", label: "Hausnummer", category: "Adresse", type: "text" },
  { key: "resident_zip", label: "PLZ", category: "Adresse", type: "text" },
  { key: "resident_city", label: "Ort", category: "Adresse", type: "text" },

  // Bank
  { key: "iban", label: "IBAN", category: "Bank", type: "text", sensitive: true },
  { key: "account_holder", label: "Kontoinhaber", category: "Bank", type: "text" },
  { key: "einzugsart", label: "Einzugsart", category: "Bank", type: "enum" },

  // EEG
  { key: "rc_number", label: "RC-Nummer", category: "EEG", type: "text" },
  { key: "reference_number", label: "Referenznummer", category: "EEG", type: "text" },
  { key: "member_number", label: "Mitgliedsnummer", category: "EEG", type: "text" },
  { key: "membership_start_date", label: "Beitrittsdatum", category: "EEG", type: "date" },
  { key: "status", label: "Status", category: "EEG", type: "enum" },
  { key: "imported_at", label: "Importiert am", category: "EEG", type: "date" },
  { key: "activated_at", label: "Aktiviert am", category: "EEG", type: "date" },

  // Zählpunkte
  { key: "meter_count", label: "Anzahl Zählpunkte", category: "Zählpunkte", type: "number" },
  { key: "meter_numbers", label: "Zählpunkte", category: "Zählpunkte", type: "multi" },

  // EEG-Stammdaten (aus registration_entrypoint, identisch für alle Anträge eines Exports)
  { key: "eeg_name", label: "EEG-Name", category: "EEG-Stammdaten", type: "text" },
  { key: "eeg_street", label: "EEG-Straße", category: "EEG-Stammdaten", type: "text" },
  { key: "eeg_street_number", label: "EEG-Hausnummer", category: "EEG-Stammdaten", type: "text" },
  { key: "eeg_zip", label: "EEG-PLZ", category: "EEG-Stammdaten", type: "text" },
  { key: "eeg_city", label: "EEG-Ort", category: "EEG-Stammdaten", type: "text" },
  { key: "eeg_id", label: "EEG-ID (Core)", category: "EEG-Stammdaten", type: "text" },
  { key: "eeg_creditor_id", label: "EEG-Creditor-ID (SEPA)", category: "EEG-Stammdaten", type: "text" },
  { key: "eeg_contact_email", label: "EEG-Kontakt-E-Mail", category: "EEG-Stammdaten", type: "text" },
];

export const EXCEL_FIELD_CATEGORIES = [
  "Stammdaten",
  "Kontakt",
  "Adresse",
  "Bank",
  "EEG",
  "Zählpunkte",
  "EEG-Stammdaten",
];

export interface ExcelFormatOption {
  value: string;
  label: string;
}

export function formatOptionsForType(type: ExcelFieldType): ExcelFormatOption[] {
  switch (type) {
    case "text":
      return [{ value: "string", label: "Text" }];
    case "date":
      return [
        { value: "date_dmy", label: "DD.MM.YYYY" },
        { value: "date_iso", label: "YYYY-MM-DD" },
        { value: "date_dmy_hm", label: "DD.MM.YYYY HH:MM" },
      ];
    case "bool":
      return [
        { value: "bool_yn", label: "Ja / Nein" },
        { value: "bool_tf", label: "true / false" },
        { value: "bool_10", label: "1 / 0" },
        { value: "bool_yn_short", label: "Y / N" },
      ];
    case "enum":
      return [
        { value: "enum_label", label: "Label (lesbar)" },
        { value: "enum_value", label: "Roh-Wert" },
      ];
    case "number":
      return [
        { value: "number_de", label: "DE-Format (1,23)" },
        { value: "number_iso", label: "ISO (1.23)" },
      ];
    case "multi":
      return [{ value: "comma_separated", label: "Komma-getrennt" }];
  }
}

export function defaultFormatForType(type: ExcelFieldType): string {
  return formatOptionsForType(type)[0].value;
}

export function findExcelField(key: string): ExcelFieldDef | undefined {
  return EXCEL_FIELD_CATALOG.find((f) => f.key === key);
}

// Excel-plugin column-config — matches the JSON shape ValidateConfig accepts
// in internal/dataexport/excel/plugin.go.
export interface ExcelColumnConfig {
  header: string;
  field: string;
  format: string;
}

export interface ExcelConfig {
  format: "xlsx" | "csv";
  columns: ExcelColumnConfig[];
}

export const EXCEL_MAX_COLUMNS = 50;
