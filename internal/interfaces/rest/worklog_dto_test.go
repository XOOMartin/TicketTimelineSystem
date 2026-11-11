package rest

import (
	"encoding/json"
	"testing"

	"crud_app/internal/domain/calendar"
)

// The options send BlackBox's steps in the shape the frontend's FlowStep
// takes. Either review asking for changes starts a review-back, at the step
// changesTo names.
func TestOptionsBlackBoxWorkflow(t *testing.T) {
	body, err := json.Marshal(toOptionsResponse())
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Workflows map[string]struct {
			First []map[string]any `json:"first"`
			Again []map[string]any `json:"again"`
			Redo  []map[string]any `json:"redo"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatal(err)
	}
	w, ok := res.Workflows[blackBoxWorkflowKey]
	if !ok {
		t.Fatalf("no %q workflow in %s", blackBoxWorkflowKey, body)
	}
	if _, ok := res.Workflows["implementation"]; !ok {
		t.Error("the SPinno types' workflows are gone")
	}
	step := func(key, label, who, kind, hint string, extra map[string]any) map[string]any {
		s := map[string]any{
			"key": key, "label": label, "who": who, "kind": kind, "phase": key, "hint": hint,
			"changesTo": "", "changesNewRound": false, "changesRestart": false, "approveLabel": "",
		}
		for k, v := range extra {
			s[k] = v
		}
		return s
	}
	want := []map[string]any{
		step("investigation", "Spec", "You", "work",
			"Read the spec, then reproduce and investigate the issue.", nil),
		step("confirmation", "Confirm", "SE / managers", "review",
			"Check the approach or spec with SE or managers, then record how long it took and their answer.",
			map[string]any{"changesTo": "investigation", "changesNewRound": true, "approveLabel": "No changes", "waitLabel": "Still waiting"}),
		step("planning", "Planning", "You", "work",
			"Plan the solution and any spec changes before you start coding.", nil),
		step("coding", "Coding", "You", "work", "Write the code.", nil),
		step("testing", "Testing", "You", "work", "Test your own implementation.", nil),
		step("review", "Review", "Reviewer", "review",
			"A manager reviews your work. Record how long it took and the result.",
			map[string]any{"changesTo": "planning", "changesNewRound": true}),
	}
	same := func(got, want []map[string]any) bool {
		if len(got) != len(want) {
			return false
		}
		a, _ := json.Marshal(got)
		b, _ := json.Marshal(want)
		return string(a) == string(b)
	}
	if !same(w.First, want) {
		t.Errorf("first = %v\nwant %v", w.First, want)
	}
	// A review-back round shows all six phases (it starts at the step the
	// review's changes go to)
	if !same(w.Again, want) {
		t.Errorf("again = %v\nwant %v", w.Again, want)
	}
	if w.Redo == nil || len(w.Redo) != 0 {
		t.Errorf("redo = %v, want []", w.Redo)
	}
}

// A download's name starts with the days it covers, as people say them: a
// week by its range, a day by itself.
func TestExportName(t *testing.T) {
	for _, c := range []struct{ first, last, what, want string }{
		{"2026-11-09", "2026-11-15", "Weekly Report", "Nov 9-15 2026 Weekly Report.xlsx"},
		{"2026-11-10", "2026-11-10", "Daily Report", "Nov 10 2026 Daily Report.xlsx"},
		{"2026-10-26", "2026-11-01", "Team Sheet", "Oct 26-Nov 1 2026 Team Sheet.xlsx"},
		{"2026-12-28", "2027-01-03", "Weekly Report", "Dec 28 2026-Jan 3 2027 Weekly Report.xlsx"},
		{"2026-01-05", "2026-01-11", fileSafe("Gerald Saberdo") + " Check-ins", "Jan 5-11 2026 Gerald Saberdo Check-ins.xlsx"},
		{"2026-03-02", "2026-03-08", fileSafe(`A/B "C"`) + " Check-ins", "Mar 2-8 2026 A_B _C_ Check-ins.xlsx"},
	} {
		first, err := calendar.ParseDay(c.first)
		if err != nil {
			t.Fatal(err)
		}
		last, err := calendar.ParseDay(c.last)
		if err != nil {
			t.Fatal(err)
		}
		period, err := calendar.NewPeriod(first, last)
		if err != nil {
			t.Fatal(err)
		}
		if got := exportName(period, c.what); got != c.want {
			t.Errorf("exportName(%s..%s, %q) = %q, want %q", c.first, c.last, c.what, got, c.want)
		}
	}
}
