package activity

import (
	"fmt"

	"crud_app/internal/domain/calendar"
)

// Day is one calendar date in local time, the same value object the
// worklog uses.
type Day = calendar.Day

// DayOf returns the local calendar day that t falls on.
var DayOf = calendar.DayOf

// ParseDay reads a day such as 2026-10-28. Its error is activity's own
// ErrInvalid, as it was before Day was shared with the calendar.
func ParseDay(s string) (Day, error) {
	d, err := calendar.ParseDay(s)
	if err != nil {
		return Day{}, fmt.Errorf("%w: day must look like 2026-10-28", ErrInvalid)
	}
	return d, nil
}
