package worklog

import (
	"errors"
	"testing"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
)

func TestNewTicketDates(t *testing.T) {
	day := func(s string) *calendar.Day {
		d, err := calendar.ParseDay(s)
		if err != nil {
			t.Fatal(err)
		}
		return &d
	}
	key := TicketKey{Project: project.BlackBox, Number: "3287"}

	if _, err := NewTicketDates("ana", key, day("2026-11-03"), day("2026-11-06")); err != nil {
		t.Errorf("start before end: %v", err)
	}
	// Not finished yet, or no start given: both fine
	if _, err := NewTicketDates("ana", key, day("2026-11-03"), nil); err != nil {
		t.Errorf("no end: %v", err)
	}
	if _, err := NewTicketDates("ana", key, nil, day("2026-11-06")); err != nil {
		t.Errorf("no start: %v", err)
	}
	// The sheet had a ticket "ending" before it started; that's a typo
	if _, err := NewTicketDates("ana", key, day("2026-11-06"), day("2026-11-03")); !errors.Is(err, ErrInvalid) {
		t.Errorf("end before start: got %v, want ErrInvalid", err)
	}
}
