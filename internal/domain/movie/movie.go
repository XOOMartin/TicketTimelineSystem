package movie

import (
	"fmt"
	"strings"
)

// ID uniquely identifies a Movie.
type ID string

// Director is a value object: it has no identity and is never mutated.
type Director struct {
	firstName string
	lastName  string
}

func NewDirector(firstName, lastName string) Director {
	return Director{
		firstName: strings.TrimSpace(firstName),
		lastName:  strings.TrimSpace(lastName),
	}
}

func (d Director) FirstName() string { return d.firstName }
func (d Director) LastName() string  { return d.lastName }

// Movie is the aggregate root. Its fields are unexported so every change
// goes through New or Revise, which enforce the invariants.
type Movie struct {
	id       ID
	isbn     string
	title    string
	director *Director
}

func New(id ID, isbn, title string, director *Director) (*Movie, error) {
	m := &Movie{id: id}
	if err := m.Revise(isbn, title, director); err != nil {
		return nil, err
	}
	return m, nil
}

// Revise replaces the movie's details. director may be nil.
func (m *Movie) Revise(isbn, title string, director *Director) error {
	isbn = strings.TrimSpace(isbn)
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalid)
	}
	if isbn == "" {
		return fmt.Errorf("%w: isbn is required", ErrInvalid)
	}

	m.isbn = isbn
	m.title = title
	m.director = director
	return nil
}

func (m *Movie) ID() ID              { return m.id }
func (m *Movie) ISBN() string        { return m.isbn }
func (m *Movie) Title() string       { return m.title }
func (m *Movie) Director() *Director { return m.director }
