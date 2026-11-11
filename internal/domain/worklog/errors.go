package worklog

import "errors"

var (
	ErrNotFound  = errors.New("check-in not found")
	ErrInvalid   = errors.New("invalid check-in")
	ErrDuplicate = errors.New("you already logged this phase of this ticket, in this round, at this time")
	ErrNoTimer   = errors.New("no timer is running")
)
