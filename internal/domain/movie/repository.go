package movie

import "context"

// Repository is defined by the domain and implemented by infrastructure.
type Repository interface {
	NextID(ctx context.Context) (ID, error)
	FindAll(ctx context.Context) ([]*Movie, error)
	// FindByID returns ErrNotFound when no movie has the given id.
	FindByID(ctx context.Context, id ID) (*Movie, error)
	// Save inserts the movie, or replaces it if one with the same id exists.
	Save(ctx context.Context, m *Movie) error
	// Delete returns ErrNotFound when no movie has the given id.
	Delete(ctx context.Context, id ID) error
}
