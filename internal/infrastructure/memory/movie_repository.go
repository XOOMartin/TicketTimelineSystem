package memory

import (
	"context"
	"strconv"
	"sync"

	"crud_app/internal/domain/movie"
)

// MovieRepository is an in-memory movie.Repository. It keeps insertion order
// and stores copies so callers can't mutate stored movies without Save.
type MovieRepository struct {
	mu     sync.RWMutex
	movies []*movie.Movie
	lastID int
}

func NewMovieRepository() *MovieRepository {
	return &MovieRepository{}
}

func (r *MovieRepository) NextID(ctx context.Context) (movie.ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastID++
	return movie.ID(strconv.Itoa(r.lastID)), nil
}

func (r *MovieRepository) FindAll(ctx context.Context) ([]*movie.Movie, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*movie.Movie, len(r.movies))
	for i, m := range r.movies {
		result[i] = clone(m)
	}
	return result, nil
}

func (r *MovieRepository) FindByID(ctx context.Context, id movie.ID) (*movie.Movie, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if i := r.indexOf(id); i >= 0 {
		return clone(r.movies[i]), nil
	}
	return nil, movie.ErrNotFound
}

func (r *MovieRepository) Save(ctx context.Context, m *movie.Movie) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i := r.indexOf(m.ID()); i >= 0 {
		r.movies[i] = clone(m)
	} else {
		r.movies = append(r.movies, clone(m))
	}
	return nil
}

func (r *MovieRepository) Delete(ctx context.Context, id movie.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := r.indexOf(id)
	if i < 0 {
		return movie.ErrNotFound
	}
	r.movies = append(r.movies[:i], r.movies[i+1:]...)
	return nil
}

// indexOf must be called with mu held.
func (r *MovieRepository) indexOf(id movie.ID) int {
	for i, m := range r.movies {
		if m.ID() == id {
			return i
		}
	}
	return -1
}

// clone is a shallow copy; Director is immutable so sharing it is safe.
func clone(m *movie.Movie) *movie.Movie {
	c := *m
	return &c
}
