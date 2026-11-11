package activity

import "context"

// DaySummary is a read model: how many prompts were sent on one day.
type DaySummary struct {
	Day         Day
	PromptCount int
}

// Repository is defined by the domain and implemented by infrastructure.
type Repository interface {
	// SaveAll stores the prompts, skipping ones that are already stored.
	// It returns how many were new.
	SaveAll(ctx context.Context, prompts []*Prompt) (int, error)
	// FindByDay returns the day's prompts, oldest first.
	FindByDay(ctx context.Context, day Day) ([]*Prompt, error)
	// Summaries returns one entry per day that has prompts, newest first.
	Summaries(ctx context.Context) ([]DaySummary, error)
}

// Source is where prompts come from (e.g. Claude Code's transcript files).
// Like Repository, the domain defines it and infrastructure implements it.
type Source interface {
	Prompts(ctx context.Context) ([]*Prompt, error)
}
