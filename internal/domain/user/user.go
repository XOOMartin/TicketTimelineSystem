package user

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"crud_app/internal/domain/project"
)

const MinPasswordLength = 8

// A username: 3 to 64 letters, digits or . _ - @ (accounts from before
// usernames have their email as theirs). Matched without regard to case.
var usernamePattern = regexp.MustCompile(`^[a-z0-9._@-]{3,64}$`)

// NormalizeUsername is the username as it's stored and looked up.
func NormalizeUsername(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// parseUsername normalizes a username and checks it, for every way one is set.
func parseUsername(s string) (string, error) {
	username := NormalizeUsername(s)
	if !usernamePattern.MatchString(username) {
		return "", fmt.Errorf("%w: a username is 3 to 64 letters, digits, dots, dashes or underscores", ErrInvalid)
	}
	return username, nil
}

// ID uniquely identifies a User.
type ID string

type Role string

const (
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
)

func ParseRole(s string) (Role, error) {
	switch r := Role(strings.TrimSpace(s)); r {
	case RoleMember, RoleAdmin:
		return r, nil
	}
	return "", fmt.Errorf("%w: role must be member or admin", ErrInvalid)
}

// User is an aggregate root: one person who logs check-ins. The password is
// only ever held as a hash.
type User struct {
	id           ID
	name         string
	firstName    string
	lastName     string
	username     string
	passwordHash string
	team         project.Project
	role         Role
	createdAt    time.Time
	// deletedAt is set when an admin deleted the account (soft): it can't
	// log in and is left out of lists and exports, but can be restored
	deletedAt *time.Time
}

func New(id ID, name, username, passwordHash string, team project.Project, role Role, createdAt time.Time) (*User, error) {
	name = strings.TrimSpace(name)
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", ErrInvalid)
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	username, err := parseUsername(username)
	if err != nil {
		return nil, err
	}
	if passwordHash == "" {
		return nil, fmt.Errorf("%w: password is required", ErrInvalid)
	}

	return &User{
		id:           id,
		name:         name,
		username:     username,
		passwordHash: passwordHash,
		team:         team,
		role:         role,
		createdAt:    createdAt,
	}, nil
}

// SetNames gives the user a first and last name, as asked at signup.
// Accounts from before don't have them.
func (u *User) SetNames(first, last string) {
	u.firstName = strings.TrimSpace(first)
	u.lastName = strings.TrimSpace(last)
}

// Edit changes the user's names and username, as an admin does. The name
// shown in the app is the first and last name.
func (u *User) Edit(first, last, username string) error {
	first, last = strings.TrimSpace(first), strings.TrimSpace(last)
	if first == "" || last == "" {
		return fmt.Errorf("%w: first and last name are required", ErrInvalid)
	}
	username, err := parseUsername(username)
	if err != nil {
		return err
	}
	u.firstName, u.lastName, u.username = first, last, username
	u.name = first + " " + last
	return nil
}

// SetPassword replaces the password (its hash).
func (u *User) SetPassword(hash string) error {
	if hash == "" {
		return fmt.Errorf("%w: password is required", ErrInvalid)
	}
	u.passwordHash = hash
	return nil
}

// Delete marks the account deleted (soft delete); Restore undoes it.
func (u *User) Delete(at time.Time) { u.deletedAt = &at }
func (u *User) Restore()            { u.deletedAt = nil }

// DeletedAt is when the account was deleted, nil while it's active.
func (u *User) DeletedAt() *time.Time { return u.deletedAt }
func (u *User) IsDeleted() bool       { return u.deletedAt != nil }

// Assign moves the user to another team and/or role. Only admins do this.
func (u *User) Assign(team project.Project, role Role) {
	u.team = team
	u.role = role
}

func (u *User) ID() ID            { return u.id }
func (u *User) Name() string      { return u.name }
func (u *User) Username() string  { return u.username }
func (u *User) FirstName() string { return u.firstName }
func (u *User) LastName() string  { return u.lastName }

// ShortName is the first name, or the name for an account without one.
func (u *User) ShortName() string {
	if u.firstName != "" {
		return u.firstName
	}
	return u.name
}

// FullName is the first and last name, or the name for an account without.
func (u *User) FullName() string {
	if full := strings.TrimSpace(u.firstName + " " + u.lastName); full != "" {
		return full
	}
	return u.name
}
func (u *User) PasswordHash() string  { return u.passwordHash }
func (u *User) Team() project.Project { return u.team }
func (u *User) Role() Role            { return u.role }
func (u *User) IsAdmin() bool         { return u.role == RoleAdmin }
func (u *User) CreatedAt() time.Time  { return u.createdAt }
