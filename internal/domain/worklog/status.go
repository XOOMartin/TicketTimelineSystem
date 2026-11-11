package worklog

import (
	"fmt"
	"strings"
)

// Status is where the ticket stands after a check-in.
type Status string

const (
	StatusNotStarted Status = "not_started"
	StatusOngoing    Status = "ongoing"
	StatusWaiting    Status = "waiting"
	StatusBlocked    Status = "blocked"
	StatusDone       Status = "done"
)

var statusLabels = map[Status]string{
	StatusNotStarted: "Not started",
	StatusOngoing:    "Ongoing",
	StatusWaiting:    "Waiting",
	StatusBlocked:    "Blocked",
	StatusDone:       "Done",
}

func Statuses() []Status {
	return []Status{StatusNotStarted, StatusOngoing, StatusWaiting, StatusBlocked, StatusDone}
}

// ParseStatus accepts the key ("not_started") or the label ("Not started").
func ParseStatus(s string) (Status, error) {
	s = strings.TrimSpace(s)
	for _, st := range Statuses() {
		if strings.EqualFold(s, string(st)) || strings.EqualFold(s, statusLabels[st]) {
			return st, nil
		}
	}
	return "", fmt.Errorf("%w: unknown status %q", ErrInvalid, s)
}

func (s Status) Label() string { return statusLabels[s] }

// FamilyStatus is a parent ticket's status, from its tickets' (it has no
// time of its own): done once all are done; ongoing while any is ongoing,
// whatever the others; waiting or blocked when every unfinished one is;
// not started while none has started; and ongoing for any other mix (some
// done and the rest not started, or some waiting and some blocked). With no
// tickets it has none to give: "".
func FamilyStatus(children []Status) Status {
	if len(children) == 0 {
		return ""
	}
	count := map[Status]int{}
	for _, s := range children {
		count[s]++
	}
	open := len(children) - count[StatusDone]
	switch {
	case open == 0:
		return StatusDone
	case count[StatusOngoing] > 0:
		return StatusOngoing
	case count[StatusWaiting] == open:
		return StatusWaiting
	case count[StatusBlocked] == open:
		return StatusBlocked
	case count[StatusNotStarted] == len(children):
		return StatusNotStarted
	}
	return StatusOngoing
}
