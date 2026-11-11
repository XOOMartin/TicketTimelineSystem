package application

import (
	"context"

	"crud_app/internal/domain/movie"
)

type DirectorInput struct {
	FirstName string
	LastName  string
}

// MovieCommand carries the user-supplied fields for creating or updating a movie.
type MovieCommand struct {
	ISBN     string
	Title    string
	Director *DirectorInput
}

func (cmd MovieCommand) director() *movie.Director {
	if cmd.Director == nil {
		return nil
	}
	d := movie.NewDirector(cmd.Director.FirstName, cmd.Director.LastName)
	return &d
}

// MovieService holds the movie use cases. It orchestrates the domain and the
// repository; the business rules themselves live in the movie package.
type MovieService struct {
	repo movie.Repository
}

func NewMovieService(repo movie.Repository) *MovieService {
	return &MovieService{repo: repo}
}

func (s *MovieService) List(ctx context.Context) ([]*movie.Movie, error) {
	return s.repo.FindAll(ctx)
}

func (s *MovieService) Get(ctx context.Context, id string) (*movie.Movie, error) {
	return s.repo.FindByID(ctx, movie.ID(id))
}

func (s *MovieService) Create(ctx context.Context, cmd MovieCommand) (*movie.Movie, error) {
	id, err := s.repo.NextID(ctx)
	if err != nil {
		return nil, err
	}
	m, err := movie.New(id, cmd.ISBN, cmd.Title, cmd.director())
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MovieService) Update(ctx context.Context, id string, cmd MovieCommand) (*movie.Movie, error) {
	m, err := s.repo.FindByID(ctx, movie.ID(id))
	if err != nil {
		return nil, err
	}
	if err := m.Revise(cmd.ISBN, cmd.Title, cmd.director()); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MovieService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, movie.ID(id))
}
