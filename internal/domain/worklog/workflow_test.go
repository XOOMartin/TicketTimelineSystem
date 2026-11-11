package worklog

import (
	"errors"
	"testing"

	"crud_app/internal/domain/project"
)

func TestParseType(t *testing.T) {
	for _, c := range []struct {
		repo project.Repo
		in   string
		want TicketType
	}{
		{project.RepoSpec, "", TypeInvestigation},
		{project.RepoSpec, "integration_testing", TypeIntegration},
		{project.RepoBackend, "", TypeImplementation},
		{project.RepoWeb, "review", TypeReview},
	} {
		if got, err := ParseType(c.repo, c.in); err != nil || got != c.want {
			t.Errorf("ParseType(%s, %q) = %q, %v; want %q", c.repo, c.in, got, err, c.want)
		}
	}
	// A spec can't be an implementation, nor a web ticket an investigation
	for repo, in := range map[project.Repo]string{project.RepoSpec: "implementation", project.RepoWeb: "investigation"} {
		if _, err := ParseType(repo, in); !errors.Is(err, ErrInvalid) {
			t.Errorf("ParseType(%s, %q): got %v, want ErrInvalid", repo, in, err)
		}
	}
}

func TestAfterStep(t *testing.T) {
	key := TicketKey{Project: project.SPinno, Repo: project.RepoWeb, Number: "120"}
	next := func(tt TicketType, round int, step string) Progress {
		t.Helper()
		p, err := AfterStep("ana", key, tt, round, step)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	// Implementation and integration testing: the leads confirming the spec
	// leads to planning, whose time goes on Planning, then to the work
	for _, tt := range []TicketType{TypeImplementation, TypeIntegration} {
		if p := next(tt, 1, "lead_review"); p.FlowStep != "plan" || p.Step != PhasePlanning || p.Round != 1 {
			t.Errorf("%s: after lead_review = %+v, want plan", tt, p)
		}
	}
	if p := next(TypeImplementation, 1, "plan"); p.FlowStep != "code" || p.Step != PhaseCoding {
		t.Errorf("after plan = %+v, want code", p)
	}
	if p := next(TypeIntegration, 1, "plan"); p.FlowStep != "staging_test" || p.Step != PhaseTesting {
		t.Errorf("integration: after plan = %+v, want staging_test", p)
	}
	// Changes asked for in the final review start a review-back at planning
	for _, tt := range []TicketType{TypeImplementation, TypeIntegration} {
		if again := WorkflowOf(tt).Again; len(again) == 0 || again[0].Key != "plan" || again[0].Phase != PhasePlanning {
			t.Errorf("%s: review-back starts at %+v, want plan", tt, again)
		}
	}
	// No type files the ticket as a step of its own any more
	for _, tt := range Types() {
		if _, i := WorkflowOf(tt).Find(1, "file"); i >= 0 {
			t.Errorf("%s still has a File ticket step", tt)
		}
	}
	// A ticket saved on File ticket carries on from the step after it
	if p, ok := StepNow("ana", key, TypeIntegration, 1, "file"); !ok || p.FlowStep != "plan" {
		t.Errorf("saved on file = %+v, %v; want plan", p, ok)
	}
	if p, ok := StepNow("ana", key, TypeInvestigation, 1, "file"); !ok || p.FlowStep != "cho_review" {
		t.Errorf("investigation saved on file = %+v, %v; want cho_review", p, ok)
	}
	// The leads' and Cho-san's PR reviews are one step, its time on Review,
	// and approving it goes on to closing
	if s, i := WorkflowOf(TypeImplementation).Find(1, "pr_review"); i < 0 || s.Phase != PhaseReview {
		t.Errorf("pr_review = %+v at %d; want a step whose time goes on Review", s, i)
	}
	if p := next(TypeImplementation, 1, "push_pr"); p.FlowStep != "pr_review" {
		t.Errorf("after push_pr = %+v, want pr_review", p)
	}
	if p := next(TypeImplementation, 1, "pr_review"); p.FlowStep != "close" {
		t.Errorf("after pr_review = %+v, want close", p)
	}
	// A ticket saved on either old review carries on from the merged one
	for _, old := range []string{"lead_pr_review", "cho_review"} {
		if p, ok := StepNow("ana", key, TypeImplementation, 1, old); !ok || p.FlowStep != "pr_review" || p.Step != PhaseReview {
			t.Errorf("saved on %s = %+v, %v; want pr_review", old, p, ok)
		}
	}
	// A review-back still has Cho-san's review of its own
	if p, ok := StepNow("ana", key, TypeImplementation, 2, "cho_review"); !ok || p.FlowStep != "cho_review" {
		t.Errorf("review-back on cho_review = %+v, %v", p, ok)
	}
	// The leads asking for changes on the issue is a review-back too: the
	// next round redoes the original steps, from revising the issue
	for _, tt := range Types() {
		w := WorkflowOf(tt)
		if lead, _ := w.Find(1, "lead_review"); !lead.ChangesRestart {
			t.Errorf("%s: the leads' changes don't start a review-back", tt)
		}
		if len(w.Redo) != len(w.First) || w.Redo[0].Key != "redo_draft" || w.Redo[0].Label != "Revise issue" {
			t.Errorf("%s: redo round = %+v", tt, w.Redo)
		}
	}
	if p := next(TypeImplementation, 2, "redo_draft"); p.FlowStep != "redo_lead_review" || p.Round != 2 {
		t.Errorf("after redo_draft = %+v, want redo_lead_review", p)
	}
	if p := next(TypeImplementation, 2, "redo_lead_review"); p.FlowStep != "redo_plan" || p.Step != PhasePlanning {
		t.Errorf("after redo_lead_review = %+v, want redo_plan", p)
	}
	if p := next(TypeImplementation, 3, "redo_close"); !p.Finished {
		t.Errorf("after redo_close = %+v, want finished", p)
	}
	// A Cho-san review-back in round 2 still follows its own steps
	if p := next(TypeImplementation, 2, "revise_code"); p.FlowStep != "test" {
		t.Errorf("after revise_code = %+v, want test", p)
	}
	// Closing the PR and issue is the last step: done
	if p := next(TypeImplementation, 1, "close"); !p.Finished {
		t.Errorf("after close = %+v, want finished", p)
	}
	// A review-back round has its own steps: revising, then testing
	if p := next(TypeImplementation, 2, "revise_code"); p.FlowStep != "test" || p.Round != 2 {
		t.Errorf("after revise_code = %+v", p)
	}
	// Review type: Cho-san's review is the end
	if p := next(TypeReview, 1, "cho_review"); !p.Finished {
		t.Errorf("review after cho_review = %+v, want finished", p)
	}
	// Integration testing's first round has no revising step
	if _, err := AfterStep("ana", key, TypeIntegration, 1, "revise_issue"); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown step: got %v, want ErrInvalid", err)
	}
	// Every type's round of steps opens a review-back with a step of its own
	for _, tt := range Types() {
		w := WorkflowOf(tt)
		for _, s := range w.First {
			if s.ChangesNewRound && len(w.Again) == 0 {
				t.Errorf("%s: %s starts a review-back, but there are no review-back steps", tt, s.Key)
			}
			if s.ChangesTo != "" {
				if _, i := w.Find(1, s.ChangesTo); i < 0 {
					t.Errorf("%s: %s goes back to %q, which isn't a step", tt, s.Key, s.ChangesTo)
				}
			}
		}
	}
}

func TestTypeOfParentLevel(t *testing.T) {
	spec := TicketKey{Project: project.SPinno, Repo: project.RepoSpec, Number: "45"}
	childSpec := TicketKey{Project: project.SPinno, Repo: project.RepoSpec, Number: "46"}
	web := TicketKey{Project: project.SPinno, Repo: project.RepoWeb, Number: "120"}
	newParent := TicketKey{Project: project.SPinno, Repo: project.RepoSpec, Number: "47"}
	loneChild := TicketKey{Project: project.SPinno, Repo: project.RepoSpec, Number: "48"}
	types := map[TicketKey]TicketType{spec: TypeIntegration, childSpec: TypeIntegration, loneChild: TypeInvestigation}
	parents := map[TicketKey]TicketKey{childSpec: spec}
	// A spec with tickets under it is a parent: no type, even one saved
	// before
	if got := TypeOf(types, parents, spec); got != "" {
		t.Errorf("parent spec type = %q, want none", got)
	}
	// So is one added as a parent (no type), under no spec
	if got := TypeOf(types, parents, newParent); got != "" {
		t.Errorf("new parent spec type = %q, want none", got)
	}
	// A child spec added before its parent spec is set keeps its type
	if got := TypeOf(types, parents, loneChild); got != TypeInvestigation {
		t.Errorf("child spec with no parent yet: type = %q, want %q", got, TypeInvestigation)
	}
	// A spec under a spec is a child: its type
	if got := TypeOf(types, parents, childSpec); got != TypeIntegration {
		t.Errorf("child spec type = %q, want %q", got, TypeIntegration)
	}
	// A backend or web ticket has a type whether or not it's under a spec
	if got := TypeOf(types, parents, web); got != TypeImplementation {
		t.Errorf("web ticket type = %q, want %q", got, TypeImplementation)
	}
}

// BlackBox's steps are its six phases, the moves RecordOutcome makes on them.
func TestBlackBoxWorkflow(t *testing.T) {
	w := BlackBoxWorkflow()
	if len(w.First) != len(Phases()) || len(w.Redo) != 0 {
		t.Fatalf("got %d first steps and %d redo steps", len(w.First), len(w.Redo))
	}
	for i, p := range Phases() {
		s := w.First[i]
		if s.Key != string(p) || s.Phase != p || s.Label != p.ShortLabel() || s.Hint != p.Hint() {
			t.Errorf("step %d = %+v, want phase %s", i, s, p)
		}
	}
	confirm, _ := w.Find(1, string(PhaseConfirmation))
	// Either review asking for changes starts a review-back: from the spec
	// after the confirmation, from planning after the final review
	if confirm.Kind != StepReview || !confirm.ChangesNewRound || confirm.ChangesTo != string(PhaseInvestigation) || confirm.WaitLabel == "" {
		t.Errorf("confirmation = %+v", confirm)
	}
	review, _ := w.Find(1, string(PhaseReview))
	if review.Kind != StepReview || !review.ChangesNewRound || review.ChangesTo != string(PhasePlanning) {
		t.Errorf("review = %+v", review)
	}
	// A review-back round shows all six phases
	if len(w.Again) != len(Phases()) {
		t.Errorf("a review-back has %d steps, want %d", len(w.Again), len(Phases()))
	}
}

func TestNextOrReview(t *testing.T) {
	for p, want := range map[Phase]Phase{
		PhaseInvestigation: PhaseConfirmation,
		PhaseTesting:       PhaseReview,
		PhaseReview:        PhaseReview,
	} {
		if got := p.NextOrReview(); got != want {
			t.Errorf("%s.NextOrReview() = %s, want %s", p, got, want)
		}
	}
}
