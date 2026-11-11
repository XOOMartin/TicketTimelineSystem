package worklog

import (
	"fmt"
	"strings"
)

// Phase is what kind of work a check-in was. Phases are listed in the order
// a ticket normally moves through them.
type Phase string

const (
	PhaseInvestigation Phase = "investigation"
	PhaseConfirmation  Phase = "confirmation"
	PhasePlanning      Phase = "planning"
	PhaseCoding        Phase = "coding"
	PhaseTesting       Phase = "testing"
	PhaseReview        Phase = "review"
)

// phaseLabels match the column headers of the team's progress sheets.
var phaseLabels = map[Phase]string{
	PhaseInvestigation: "Check Spec / Investigation",
	PhaseConfirmation:  "Confirmation / Review to SE/MGRs",
	PhasePlanning:      "Development / Spec Changes (Planning)",
	PhaseCoding:        "Coding (Implementation)",
	PhaseTesting:       "Testing",
	PhaseReview:        "Review After Implementation",
}

// shortPhaseLabels name the phases where space is tight, such as table
// headers and the progress steps.
var shortPhaseLabels = map[Phase]string{
	PhaseInvestigation: "Spec",
	PhaseConfirmation:  "Confirm",
	PhasePlanning:      "Planning",
	PhaseCoding:        "Coding",
	PhaseTesting:       "Testing",
	PhaseReview:        "Review",
}

// phaseHints say what each phase is in one plain sentence, shown under its
// name in the progress form.
var phaseHints = map[Phase]string{
	PhaseInvestigation: "Read the spec, then reproduce and investigate the issue.",
	PhaseConfirmation:  "Check the approach or spec with SE or managers, then record how long it took and their answer.",
	PhasePlanning:      "Plan the solution and any spec changes before you start coding.",
	PhaseCoding:        "Write the code.",
	PhaseTesting:       "Test your own implementation.",
	PhaseReview:        "A manager reviews your work. Record how long it took and the result.",
}

// oldPhaseLabels are earlier labels, still accepted so spreadsheets exported
// before the rename keep importing.
var oldPhaseLabels = map[Phase]string{
	PhaseConfirmation: "Confirmation to SE/MGRs",
	PhasePlanning:     "Planning / Spec changes",
	PhaseCoding:       "Coding",
	PhaseReview:       "Review",
}

// Phases lists every phase in workflow order.
func Phases() []Phase {
	return []Phase{PhaseInvestigation, PhaseConfirmation, PhasePlanning, PhaseCoding, PhaseTesting, PhaseReview}
}

// ParsePhase accepts the key ("coding") or the label ("Coding
// (Implementation)"), so values typed into a spreadsheet import cleanly.
func ParsePhase(s string) (Phase, error) {
	s = strings.TrimSpace(s)
	for _, p := range Phases() {
		if strings.EqualFold(s, string(p)) || strings.EqualFold(s, phaseLabels[p]) ||
			(oldPhaseLabels[p] != "" && strings.EqualFold(s, oldPhaseLabels[p])) {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w: unknown phase %q", ErrInvalid, s)
}

func (p Phase) Label() string { return phaseLabels[p] }

// ShortLabel is the phase's name where space is tight, e.g. "Confirm".
func (p Phase) ShortLabel() string { return shortPhaseLabels[p] }

// Hint says what the phase is.
func (p Phase) Hint() string { return phaseHints[p] }

func (p Phase) order() int {
	for i, q := range Phases() {
		if p == q {
			return i
		}
	}
	return -1
}

// MaxRound caps how many times a ticket can go round; well past any real
// ticket.
const MaxRound = 20

// RoundLabel names a round of work on a ticket: round 1 is the original
// work, and each review that sends the ticket back for changes starts the
// next, its own pass through the phases.
func RoundLabel(round int) string {
	if round <= 1 {
		return "Original"
	}
	return fmt.Sprintf("Revision %d", round-1)
}
