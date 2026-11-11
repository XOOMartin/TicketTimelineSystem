package movie

import "errors"

var (
	ErrNotFound = errors.New("movie not found")
	ErrInvalid  = errors.New("invalid movie")
)
