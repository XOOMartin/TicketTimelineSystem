package user

import "errors"

var (
	ErrNotFound        = errors.New("user not found")
	ErrInvalid         = errors.New("invalid user")
	ErrUsernameTaken   = errors.New("that username is taken")
	ErrBadCredentials  = errors.New("username or password is incorrect")
	ErrUnauthenticated = errors.New("please log in")
	ErrForbidden       = errors.New("admins only")
	ErrDeactivated     = errors.New("this account has been deleted: ask an admin to restore it")
)
