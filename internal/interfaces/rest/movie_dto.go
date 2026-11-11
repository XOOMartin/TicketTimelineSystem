package rest

import (
	"crud_app/internal/application"
	"crud_app/internal/domain/movie"
)

// JSON field names match the original API so the frontend is unaffected.

type directorDTO struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}

type movieRequest struct {
	Isbn     string       `json:"isbn"`
	Title    string       `json:"title"`
	Director *directorDTO `json:"director"`
}

type movieResponse struct {
	ID       string       `json:"id"`
	Isbn     string       `json:"isbn"`
	Title    string       `json:"title"`
	Director *directorDTO `json:"director"`
}

func (req movieRequest) toCommand() application.MovieCommand {
	cmd := application.MovieCommand{ISBN: req.Isbn, Title: req.Title}
	if req.Director != nil {
		cmd.Director = &application.DirectorInput{
			FirstName: req.Director.Firstname,
			LastName:  req.Director.Lastname,
		}
	}
	return cmd
}

func toMovieResponse(m *movie.Movie) movieResponse {
	res := movieResponse{ID: string(m.ID()), Isbn: m.ISBN(), Title: m.Title()}
	if d := m.Director(); d != nil {
		res.Director = &directorDTO{Firstname: d.FirstName(), Lastname: d.LastName()}
	}
	return res
}

func toMovieResponses(movies []*movie.Movie) []movieResponse {
	res := make([]movieResponse, len(movies))
	for i, m := range movies {
		res[i] = toMovieResponse(m)
	}
	return res
}
