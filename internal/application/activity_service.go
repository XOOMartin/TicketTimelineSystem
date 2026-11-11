package application

import (
	"context"

	"crud_app/internal/domain/activity"
)

type ImportResult struct {
	Found int
	Added int
}

// ActivityService holds the use cases for recording what the user did.
type ActivityService struct {
	source activity.Source
	repo   activity.Repository
}

func NewActivityService(source activity.Source, repo activity.Repository) *ActivityService {
	return &ActivityService{source: source, repo: repo}
}

// Import copies prompts from the source into the repository. It is safe to
// run repeatedly: prompts already stored are skipped.
func (s *ActivityService) Import(ctx context.Context) (ImportResult, error) {
	prompts, err := s.source.Prompts(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	added, err := s.repo.SaveAll(ctx, prompts)
	if err != nil {
		return ImportResult{}, err
	}
	return ImportResult{Found: len(prompts), Added: added}, nil
}

func (s *ActivityService) Days(ctx context.Context) ([]activity.DaySummary, error) {
	return s.repo.Summaries(ctx)
}

func (s *ActivityService) Prompts(ctx context.Context, day string) ([]*activity.Prompt, error) {
	d, err := activity.ParseDay(day)
	if err != nil {
		return nil, err
	}
	return s.repo.FindByDay(ctx, d)
}
