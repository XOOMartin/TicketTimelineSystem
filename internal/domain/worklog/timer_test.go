package worklog

import (
	"errors"
	"testing"
	"time"

	"crud_app/internal/domain/project"
)

func TestNewTimer(t *testing.T) {
	key := TicketKey{Project: project.BlackBox, Number: "#3287"}
	now := time.Now()

	timer, err := NewTimer("ana", key, " Status-only interceptor ", 0, PhaseCoding, now)
	if err != nil {
		t.Fatal(err)
	}
	// "#3287" is read as 3287, a missing round as the original work
	if timer.Ticket.Number != "3287" || timer.Round != 1 || timer.Title != "Status-only interceptor" {
		t.Errorf("timer = %+v", timer)
	}

	for name, start := range map[string]func() (Timer, error){
		"no title":  func() (Timer, error) { return NewTimer("ana", key, " ", 1, PhaseCoding, now) },
		"no phase":  func() (Timer, error) { return NewTimer("ana", key, "Title", 1, "", now) },
		"bad round": func() (Timer, error) { return NewTimer("ana", key, "Title", MaxRound+1, PhaseCoding, now) },
	} {
		if _, err := start(); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: got %v, want ErrInvalid", name, err)
		}
	}
}

func TestTimerPause(t *testing.T) {
	start := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }
	timer, err := NewTimer("ana", TicketKey{Project: project.BlackBox, Number: "1"}, "Title", 1, PhaseCoding, start)
	if err != nil {
		t.Fatal(err)
	}

	timer.Pause(at(60)) // an hour's work, then lunch
	if !timer.IsPaused() || timer.Elapsed(at(80)) != time.Hour {
		t.Errorf("paused: elapsed %v, want 1h frozen", timer.Elapsed(at(80)))
	}
	timer.Pause(at(70)) // pausing again changes nothing
	timer.Resume(at(90))
	if timer.IsPaused() || timer.Elapsed(at(110)) != 80*time.Minute {
		t.Errorf("resumed: elapsed %v, want 1h20m", timer.Elapsed(at(110)))
	}
}

func TestTimerClosing(t *testing.T) {
	start := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return start.Add(time.Duration(minutes) * time.Minute) }
	timer, err := NewTimer("ana", TicketKey{Project: project.BlackBox, Number: "1"}, "Title", 1, PhaseCoding, start)
	if err != nil {
		t.Fatal(err)
	}

	// Closed at 9:30; a reload comes back within the grace, so it carries on
	timer.Closing(at(30))
	if timer.IsClosed(at(30).Add(CloseGrace - time.Second)) {
		t.Error("closed before the grace ran out")
	}
	timer.Reopen()
	if timer.ClosingAt != nil || timer.IsClosed(at(600)) {
		t.Error("still closing after reopening")
	}

	// Closed at 10:00 and a second close at 10:05 keeps the first; nobody
	// comes back, so it ended, with the hour up to the close
	timer.Closing(at(60))
	timer.Closing(at(65))
	if !timer.IsClosed(at(60).Add(CloseGrace)) {
		t.Error("not closed after the grace")
	}
	if got := timer.Elapsed(*timer.ClosingAt); got != time.Hour {
		t.Errorf("time up to the close = %v, want 1h", got)
	}
}
