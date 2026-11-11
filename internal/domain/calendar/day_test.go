package calendar

import "testing"

// A weekly report covers Monday to Friday; a weekend day belongs to the week
// that just ended.
func TestWorkWeekOf(t *testing.T) {
	for _, day := range []string{"2026-11-09", "2026-11-11", "2026-11-13", "2026-11-14", "2026-11-15"} {
		d, err := ParseDay(day)
		if err != nil {
			t.Fatal(err)
		}
		got := WorkWeekOf(d)
		if got.First().String() != "2026-11-09" || got.Last().String() != "2026-11-13" {
			t.Errorf("WorkWeekOf(%s) = %s..%s, want 2026-11-09..2026-11-13", day, got.First(), got.Last())
		}
	}
}
