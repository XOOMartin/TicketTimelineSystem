package application

import (
	"context"
	"fmt"
	"time"

	"crud_app/internal/domain/activity"
	"crud_app/internal/domain/review"
)

// ReviewService holds the use cases for AI feedback on a day's prompts.
type ReviewService struct {
	prompts  activity.Repository
	reviews  review.Repository
	analyzer review.Analyzer
}

func NewReviewService(prompts activity.Repository, reviews review.Repository, analyzer review.Analyzer) *ReviewService {
	return &ReviewService{prompts: prompts, reviews: reviews, analyzer: analyzer}
}

func (s *ReviewService) Get(ctx context.Context, day string) (*review.Review, error) {
	d, err := activity.ParseDay(day)
	if err != nil {
		return nil, err
	}
	return s.reviews.FindByDay(ctx, d)
}

// Generate asks the analyzer to review the day's prompts and saves the result,
// replacing any earlier review of that day.
func (s *ReviewService) Generate(ctx context.Context, day string) (*review.Review, error) {
	d, err := activity.ParseDay(day)
	if err != nil {
		return nil, err
	}
	prompts, err := s.prompts.FindByDay(ctx, d)
	if err != nil {
		return nil, err
	}
	if len(prompts) == 0 {
		return nil, review.ErrNoPrompts
	}

	analysis, err := s.analyzer.Analyze(ctx, d, prompts)
	if err != nil {
		return nil, err
	}
	r, err := review.New(d, analysis, time.Now())
	if err != nil {
		// The AI returned something that breaks the review rules.
		return nil, fmt.Errorf("%w: %v", review.ErrAnalysisFailed, err)
	}
	if err := s.reviews.Save(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}
