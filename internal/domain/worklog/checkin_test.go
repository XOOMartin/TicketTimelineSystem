package worklog

import (
	"math"
	"testing"
	"time"

	"crud_app/internal/domain/project"
)

func TestReviseKeepsWholeMinutes(t *testing.T) {
	entry := Entry{
		Ticket:   TicketKey{Project: project.BlackBox, Number: "3287"},
		Title:    "Ticket",
		Phase:    PhaseCoding,
		Status:   StatusOngoing,
		Hours:    3 + 20.0/60, // 3 h 20 m
		WorkedAt: time.Now(),
	}
	c, err := New("a", "ana", SourceApp, entry, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if minutes := c.Hours() * 60; math.Abs(minutes-200) > 1e-9 {
		t.Errorf("stored %v minutes, want 200", minutes)
	}

	// Less than half a minute rounds to nothing, which isn't time spent.
	entry.Hours = 0.005
	if _, err := New("b", "ana", SourceApp, entry, time.Now()); err == nil {
		t.Error("want an error for under a minute")
	}
}
