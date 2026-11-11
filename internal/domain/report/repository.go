package report

import (
	"context"

	"crud_app/internal/domain/calendar"
)

// Repository is defined by the domain and implemented by infrastructure.
type Repository interface {
	NextID(ctx context.Context) (ID, error)
	// Save stores the report, replacing any report of the same kind and
	// period.
	Save(ctx context.Context, r *Report) error
	// FindAll returns every report, newest period first.
	FindAll(ctx context.Context) ([]*Report, error)
	// FindByID returns ErrNotFound when no report has the id.
	FindByID(ctx context.Context, id ID) (*Report, error)
	// FindByPeriod returns ErrNotFound when no report of the kind starts on
	// the day.
	FindByPeriod(ctx context.Context, kind Kind, first calendar.Day) (*Report, error)
}
