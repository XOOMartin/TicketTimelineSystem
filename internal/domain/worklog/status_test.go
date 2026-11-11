package worklog

import "testing"

func TestFamilyStatus(t *testing.T) {
	const (
		todo  = StatusNotStarted
		doing = StatusOngoing
		wait  = StatusWaiting
		block = StatusBlocked
		done  = StatusDone
	)
	for _, c := range []struct {
		name     string
		children []Status
		want     Status
	}{
		{"no tickets: nothing to say", nil, ""},
		{"all done", []Status{done, done}, done},
		{"one ongoing makes it ongoing", []Status{done, doing, todo}, doing},
		{"ongoing beats waiting", []Status{doing, wait}, doing},
		{"ongoing beats blocked", []Status{block, doing}, doing},
		{"every unfinished one waiting", []Status{done, wait, wait}, wait},
		{"every unfinished one blocked", []Status{block, done}, block},
		{"none started", []Status{todo, todo}, todo},
		{"some done, the rest not started", []Status{done, todo}, doing},
		{"waiting and blocked", []Status{wait, block}, doing},
		{"waiting and not started", []Status{wait, todo}, doing},
	} {
		if got := FamilyStatus(c.children); got != c.want {
			t.Errorf("%s: FamilyStatus(%v) = %q, want %q", c.name, c.children, got, c.want)
		}
	}
}
