package user

import (
	"context"
	"time"
)

// Repository is defined by the domain and implemented by infrastructure.
type Repository interface {
	NextID(ctx context.Context) (ID, error)
	Count(ctx context.Context) (int, error)
	// FindAll returns every active user (not deleted), ordered by team then
	// name.
	FindAll(ctx context.Context) ([]*User, error)
	// FindDeleted returns the soft-deleted users, most recently deleted first.
	FindDeleted(ctx context.Context) ([]*User, error)
	// HardDelete removes the user and everything that is theirs: sessions,
	// check-ins, tickets, timers, imports. It can't be undone.
	HardDelete(ctx context.Context, id ID) error
	// FindByID and FindByUsername return ErrNotFound when there is no match.
	FindByID(ctx context.Context, id ID) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	// Save inserts or updates the user. It returns ErrUsernameTaken when
	// another user already has the username.
	Save(ctx context.Context, u *User) error
}

// SessionStore keeps logged-in sessions. Tokens are opaque random strings.
type SessionStore interface {
	Create(ctx context.Context, token string, id ID, expiresAt time.Time) error
	// FindUser returns ErrUnauthenticated for unknown or expired tokens.
	FindUser(ctx context.Context, token string) (ID, error)
	Delete(ctx context.Context, token string) error
	// DeleteForUser ends every session of the user (logs them out).
	DeleteForUser(ctx context.Context, id ID) error
}

// PasswordHasher hashes passwords one way.
type PasswordHasher interface {
	Hash(password string) (string, error)
	// Matches reports whether password produced hash.
	Matches(hash, password string) bool
}
