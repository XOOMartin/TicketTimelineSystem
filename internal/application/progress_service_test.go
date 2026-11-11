package application

import (
	"testing"

	"crud_app/internal/domain/report"
	"crud_app/internal/domain/worklog"
)

func TestWorkedOutStep(t *testing.T) {
	rounds := []report.RoundRow{{Round: 1}, {Round: 2}}
	for _, tc := range []struct {
		name      string
		row       report.TicketRow
		wantStep  worklog.Phase
		wantRound int
	}{
		{"after the spec comes confirmation", report.TicketRow{Phase: worklog.PhaseInvestigation, Status: worklog.StatusOngoing}, worklog.PhaseConfirmation, 1},
		{"a confirmation still waiting stays", report.TicketRow{Phase: worklog.PhaseConfirmation, Status: worklog.StatusWaiting}, worklog.PhaseConfirmation, 1},
		{"a confirmed one moves on", report.TicketRow{Phase: worklog.PhaseConfirmation, Status: worklog.StatusOngoing}, worklog.PhasePlanning, 1},
		{"a review waits for its outcome", report.TicketRow{Phase: worklog.PhaseReview, Status: worklog.StatusWaiting}, worklog.PhaseReview, 1},
		{"done is finished, in the latest round", report.TicketRow{Phase: worklog.PhaseTesting, Status: worklog.StatusDone, Rounds: rounds}, "", 2},
	} {
		step, round := workedOutStep(tc.row)
		if step != tc.wantStep || round != tc.wantRound {
			t.Errorf("%s: got %q in round %d, want %q in round %d", tc.name, step, round, tc.wantStep, tc.wantRound)
		}
	}
}

func TestWithStartedRound(t *testing.T) {
	row := report.TicketRow{Rounds: []report.RoundRow{{Round: 1, Hours: 3}}}
	got := withStartedRound(row, 2)
	if got.ReviewBacks != 1 || len(got.Rounds) != 2 || got.Rounds[1].Round != 2 || got.Rounds[1].Hours != 0 {
		t.Errorf("changes requested: got %d review-backs, rounds %+v", got.ReviewBacks, got.Rounds)
	}
	if len(row.Rounds) != 1 {
		t.Errorf("the original row changed: %+v", row.Rounds)
	}
	if again := withStartedRound(got, 2); len(again.Rounds) != 2 {
		t.Errorf("a round with time is added again: %+v", again.Rounds)
	}
	if same := withStartedRound(row, 1); same.ReviewBacks != 0 || len(same.Rounds) != 1 {
		t.Errorf("round 1 added a review-back: %+v", same)
	}
}

func TestAddNote(t *testing.T) {
	if got := addNote("", "No changes"); got != "No changes" {
		t.Errorf("first note = %q", got)
	}
	if got := addNote("Changes requested: use v2", "No changes"); got != "Changes requested: use v2 · No changes" {
		t.Errorf("second note = %q", got)
	}
	if got := addNote("No changes", "No changes"); got != "No changes" {
		t.Errorf("repeat = %q", got)
	}
}
