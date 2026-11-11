package review

import (
	"fmt"
	"strings"
	"time"

	"crud_app/internal/domain/activity"
)

// Improvement is a value object: one prompting habit to fix, with a real
// example from the day and a rewritten version. It is copied by value, so
// changing a copy never changes the Review it came from.
type Improvement struct {
	Issue   string
	Example string
	Better  string
}

// Analysis is the raw result an Analyzer returns. New checks it before it
// becomes a Review, because AI output can't be trusted to follow the rules.
type Analysis struct {
	Summary      string
	Score        int
	Strengths    []string
	Improvements []Improvement
	Focus        string
}

// Review is the aggregate root: the AI's feedback on one day of prompts.
// There is at most one Review per day; generating again replaces it.
type Review struct {
	day          activity.Day
	summary      string
	score        int
	strengths    []string
	improvements []Improvement
	focus        string
	createdAt    time.Time
}

const (
	MinScore = 1
	MaxScore = 10
)

func New(day activity.Day, a Analysis, createdAt time.Time) (*Review, error) {
	summary := strings.TrimSpace(a.Summary)
	focus := strings.TrimSpace(a.Focus)
	if summary == "" {
		return nil, fmt.Errorf("%w: summary is required", ErrInvalid)
	}
	if focus == "" {
		return nil, fmt.Errorf("%w: focus is required", ErrInvalid)
	}
	if a.Score < MinScore || a.Score > MaxScore {
		return nil, fmt.Errorf("%w: score must be %d-%d, got %d", ErrInvalid, MinScore, MaxScore, a.Score)
	}
	for _, imp := range a.Improvements {
		if strings.TrimSpace(imp.Issue) == "" {
			return nil, fmt.Errorf("%w: every improvement needs an issue", ErrInvalid)
		}
	}

	return &Review{
		day:          day,
		summary:      summary,
		score:        a.Score,
		strengths:    append([]string{}, a.Strengths...),
		improvements: append([]Improvement{}, a.Improvements...),
		focus:        focus,
		createdAt:    createdAt,
	}, nil
}

func (r *Review) Day() activity.Day    { return r.day }
func (r *Review) Summary() string      { return r.summary }
func (r *Review) Score() int           { return r.score }
func (r *Review) Focus() string        { return r.focus }
func (r *Review) CreatedAt() time.Time { return r.createdAt }

// Strengths and Improvements return copies so callers can't change the Review.
func (r *Review) Strengths() []string         { return append([]string{}, r.strengths...) }
func (r *Review) Improvements() []Improvement { return append([]Improvement{}, r.improvements...) }
