package review

import (
	"context"

	"crud_app/internal/domain/activity"
)

// Repository is defined by the domain and implemented by infrastructure.
type Repository interface {
	// FindByDay returns ErrNotFound when the day has no review yet.
	FindByDay(ctx context.Context, day activity.Day) (*Review, error)
	// Save inserts the review, or replaces the existing one for that day.
	Save(ctx context.Context, r *Review) error
}

// Analyzer reads a day's prompts and gives feedback on them. The domain only
// says what it needs; infrastructure decides which AI does the work.
// Implementations wrap their failures in ErrAnalysisFailed.
type Analyzer interface {
	Analyze(ctx context.Context, day activity.Day, prompts []*activity.Prompt) (Analysis, error)
}
