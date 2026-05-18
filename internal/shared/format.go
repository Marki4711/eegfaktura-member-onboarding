package shared

// FormatMeteringPoint renders an Austrian Zählpunkt-Nummer in the
// official E-Control / MeteringCode grouping `2-6-5-20`:
//
//	"AT0031000000000000000000990022105"
//	→ "AT 003100 00000 0000000099002210" + last digit
//
// Used for human-readable rendering in PDFs and member-facing mails
// (PROJ-52). Storage and the public-form mask remain the canonical
// 33-char form without spaces — only the display is grouped.
//
// Inputs that are not exactly 33 characters or do not start with "AT"
// are returned unchanged, so callers can safely apply this to legacy
// or partial values without breaking the layout.
func FormatMeteringPoint(s string) string {
	if len(s) != 33 || s[:2] != "AT" {
		return s
	}
	return s[:2] + " " + s[2:8] + " " + s[8:13] + " " + s[13:]
}
