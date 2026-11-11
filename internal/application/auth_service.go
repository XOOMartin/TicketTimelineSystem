package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
)

const (
	SessionDuration = 7 * 24 * time.Hour
	// bcrypt ignores everything past 72 bytes.
	maxPasswordLength = 72
)

// SignupCommand: the first and last name are the name shown in the app and
// in the team sheet; the username is for logging in.
type SignupCommand struct {
	FirstName string
	LastName  string
	Username  string
	Password  string
	Team      string
}

// AuthService holds the use cases for accounts and sessions.
type AuthService struct {
	users    user.Repository
	sessions user.SessionStore
	hasher   user.PasswordHasher
}

func NewAuthService(users user.Repository, sessions user.SessionStore, hasher user.PasswordHasher) *AuthService {
	return &AuthService{users: users, sessions: sessions, hasher: hasher}
}

// Signup creates an account and logs it in. The very first account becomes
// an admin, so a fresh install always has someone who can manage it.
func (s *AuthService) Signup(ctx context.Context, cmd SignupCommand) (*user.User, string, error) {
	if err := checkPassword(cmd.Password); err != nil {
		return nil, "", err
	}
	first, last := strings.TrimSpace(cmd.FirstName), strings.TrimSpace(cmd.LastName)
	if first == "" || last == "" {
		return nil, "", fmt.Errorf("%w: enter your first and last name", user.ErrInvalid)
	}
	team, err := project.Parse(cmd.Team)
	if err != nil {
		return nil, "", fmt.Errorf("%w: choose a team", user.ErrInvalid)
	}
	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, "", err
	}
	role := user.RoleMember
	if count == 0 {
		role = user.RoleAdmin
	}

	id, err := s.users.NextID(ctx)
	if err != nil {
		return nil, "", err
	}
	hash, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, "", err
	}
	u, err := user.New(id, first+" "+last, cmd.Username, hash, team, role, time.Now())
	if err != nil {
		return nil, "", err
	}
	u.SetNames(first, last)
	if err := s.users.Save(ctx, u); err != nil {
		return nil, "", err
	}
	token, err := s.startSession(ctx, u.ID())
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*user.User, string, error) {
	u, err := s.users.FindByUsername(ctx, username)
	if errors.Is(err, user.ErrNotFound) {
		return nil, "", user.ErrBadCredentials
	}
	if err != nil {
		return nil, "", err
	}
	if !s.hasher.Matches(u.PasswordHash(), password) {
		return nil, "", user.ErrBadCredentials
	}
	// A deleted account can't log in (told only with the right password)
	if u.IsDeleted() {
		return nil, "", user.ErrDeactivated
	}
	token, err := s.startSession(ctx, u.ID())
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

// Authenticate returns the user a session token belongs to.
func (s *AuthService) Authenticate(ctx context.Context, token string) (*user.User, error) {
	if token == "" {
		return nil, user.ErrUnauthenticated
	}
	id, err := s.sessions.FindUser(ctx, token)
	if err != nil {
		return nil, err
	}
	u, err := s.users.FindByID(ctx, id)
	if errors.Is(err, user.ErrNotFound) {
		return nil, user.ErrUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	// Deleting an account ends its sessions; this covers any left over
	if u.IsDeleted() {
		return nil, user.ErrUnauthenticated
	}
	return u, nil
}

func (s *AuthService) startSession(ctx context.Context, id user.ID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	if err := s.sessions.Create(ctx, token, id, time.Now().Add(SessionDuration)); err != nil {
		return "", err
	}
	return token, nil
}
