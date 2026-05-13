package shared

import (
	"time"
	// Embed the IANA timezone database directly into the binary so that
	// time.LoadLocation works even in minimal base images (Alpine without
	// the tzdata package, distroless, FROM scratch). Without this import
	// the LoadLocation call below silently fails and DisplayLocation falls
	// back to UTC — every "am HH:MM" line in PDFs and emails then renders
	// 1-2 hours off for Austrian users.
	_ "time/tzdata"
)

// DisplayLocation is the timezone used for every user-visible timestamp
// rendered by this service (PDFs, emails, anywhere that produces a human
// readable string). PostgreSQL stores timestamps as UTC; converting through
// this location keeps the visible output consistent and DST-aware for
// Austrian users.
var DisplayLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Vienna")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// FmtDateTime renders t as "02.01.2006 15:04" in the display timezone.
func FmtDateTime(t time.Time) string {
	return t.In(DisplayLocation).Format("02.01.2006 15:04")
}

// FmtDate renders t as "02.01.2006" in the display timezone. Use it only for
// timestamp columns; pure DATE columns have no time component and should be
// formatted directly to avoid spurious day-boundary shifts.
func FmtDate(t time.Time) string {
	return t.In(DisplayLocation).Format("02.01.2006")
}
