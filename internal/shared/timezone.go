package shared

import "time"

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
