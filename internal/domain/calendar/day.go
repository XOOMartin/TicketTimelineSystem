package calendar

import (
	"errors"
	"fmt"
	"time"
)

// DayLayout is how a day is written, e.g. 2026-10-28.
const DayLayout = "2006-01-02"

var ErrInvalid = errors.New("invalid date")

// Day is a value object for one calendar date in local time, e.g. 2026-10-28.
// The only ways to get one are ParseDay and DayOf, so a Day is always valid.
type Day struct {
	value string
}

func ParseDay(s string) (Day, error) {
	t, err := time.ParseInLocation(DayLayout, s, time.Local)
	if err != nil {
		return Day{}, fmt.Errorf("%w: day must look like 2026-10-28", ErrInvalid)
	}
	return Day{value: t.Format(DayLayout)}, nil
}

// DayOf returns the local calendar day that t falls on.
func DayOf(t time.Time) Day {
	return Day{value: t.Local().Format(DayLayout)}
}

func (d Day) String() string { return d.value }

// MarshalText and UnmarshalText let a Day be stored as "2026-10-28" in JSON.
func (d Day) MarshalText() ([]byte, error) { return []byte(d.value), nil }

func (d *Day) UnmarshalText(b []byte) error {
	parsed, err := ParseDay(string(b))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Start is local midnight at the beginning of the day.
func (d Day) Start() time.Time {
	t, _ := time.ParseInLocation(DayLayout, d.value, time.Local)
	return t
}

func (d Day) AddDays(n int) Day {
	return DayOf(d.Start().AddDate(0, 0, n))
}

func (d Day) Weekday() time.Weekday { return d.Start().Weekday() }

func (d Day) IsWeekend() bool {
	w := d.Weekday()
	return w == time.Saturday || w == time.Sunday
}

func (d Day) Before(other Day) bool { return d.value < other.value }

// Period is an inclusive range of days.
type Period struct {
	first Day
	last  Day
}

func NewPeriod(first, last Day) (Period, error) {
	if last.Before(first) {
		return Period{}, fmt.Errorf("%w: period ends before it starts", ErrInvalid)
	}
	return Period{first: first, last: last}, nil
}

// SingleDay is the period covering just d.
func SingleDay(d Day) Period { return Period{first: d, last: d} }

// WeekOf is the Monday–Sunday week that d falls in.
func WeekOf(d Day) Period {
	offset := (int(d.Weekday()) + 6) % 7 // days since Monday
	monday := d.AddDays(-offset)
	return Period{first: monday, last: monday.AddDays(6)}
}

func (p Period) First() Day { return p.first }
func (p Period) Last() Day  { return p.last }

func (p Period) Contains(d Day) bool {
	return !d.Before(p.first) && !p.last.Before(d)
}

// Days lists every day in the period, in order.
func (p Period) Days() []Day {
	var days []Day
	for d := p.first; !p.last.Before(d); d = d.AddDays(1) {
		days = append(days, d)
	}
	return days
}
