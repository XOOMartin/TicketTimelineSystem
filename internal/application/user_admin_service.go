package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
)

// UserCommand is an account as an admin adds or edits it. Password may be
// left empty when editing, to keep the current one.
type UserCommand struct {
	FirstName string
	LastName  string
	Username  string
	Password  string
	Team      string
	Role      string
}

// UserAdminService holds the admin use cases for managing accounts: add,
// edit, delete (soft, so it can be restored), restore, and delete for good.
type UserAdminService struct {
	users    user.Repository
	sessions user.SessionStore
	hasher   user.PasswordHasher
}

func NewUserAdminService(users user.Repository, sessions user.SessionStore, hasher user.PasswordHasher) *UserAdminService {
	return &UserAdminService{users: users, sessions: sessions, hasher: hasher}
}

// List returns the active accounts, then the deleted ones.
func (s *UserAdminService) List(ctx context.Context) ([]*user.User, []*user.User, error) {
	active, err := s.users.FindAll(ctx)
	if err != nil {
		return nil, nil, err
	}
	deleted, err := s.users.FindDeleted(ctx)
	if err != nil {
		return nil, nil, err
	}
	return active, deleted, nil
}

// Create adds an account, as signup does but with any team and role.
func (s *UserAdminService) Create(ctx context.Context, cmd UserCommand) (*user.User, error) {
	if err := checkPassword(cmd.Password); err != nil {
		return nil, err
	}
	team, role, err := parseTeamRole(cmd.Team, cmd.Role)
	if err != nil {
		return nil, err
	}
	hash, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, err
	}
	id, err := s.users.NextID(ctx)
	if err != nil {
		return nil, err
	}
	first, last := strings.TrimSpace(cmd.FirstName), strings.TrimSpace(cmd.LastName)
	u, err := user.New(id, strings.TrimSpace(first+" "+last), cmd.Username, hash, team, role, time.Now())
	if err != nil {
		return nil, err
	}
	if err := u.Edit(first, last, cmd.Username); err != nil {
		return nil, err
	}
	if err := s.users.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Update changes an account's names, username, team and role, and its
// password when one is given. An admin can't take away their own admin role,
// nor the last admin's.
func (s *UserAdminService) Update(ctx context.Context, admin *user.User, id string, cmd UserCommand) (*user.User, error) {
	team, role, err := parseTeamRole(cmd.Team, cmd.Role)
	if err != nil {
		return nil, err
	}
	u, err := s.users.FindByID(ctx, user.ID(id))
	if err != nil {
		return nil, err
	}
	if err := s.canSetRole(ctx, admin, u, role); err != nil {
		return nil, err
	}
	if err := u.Edit(cmd.FirstName, cmd.LastName, cmd.Username); err != nil {
		return nil, err
	}
	u.Assign(team, role)
	if cmd.Password != "" {
		if err := checkPassword(cmd.Password); err != nil {
			return nil, err
		}
		hash, err := s.hasher.Hash(cmd.Password)
		if err != nil {
			return nil, err
		}
		if err := u.SetPassword(hash); err != nil {
			return nil, err
		}
	}
	if err := s.users.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// Assign changes only an account's team and role (the Members page), with
// the same safeguards as Update.
func (s *UserAdminService) Assign(ctx context.Context, admin *user.User, id, team, role string) (*user.User, error) {
	t, r, err := parseTeamRole(team, role)
	if err != nil {
		return nil, err
	}
	u, err := s.users.FindByID(ctx, user.ID(id))
	if err != nil {
		return nil, err
	}
	if err := s.canSetRole(ctx, admin, u, r); err != nil {
		return nil, err
	}
	u.Assign(t, r)
	if err := s.users.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// canSetRole: an admin can't take away their own admin role, nor the last
// admin's, so the team always has one.
func (s *UserAdminService) canSetRole(ctx context.Context, admin, u *user.User, role user.Role) error {
	if !u.IsAdmin() || role == user.RoleAdmin {
		return nil
	}
	if u.ID() == admin.ID() {
		return fmt.Errorf("%w: you can't remove your own admin role", user.ErrInvalid)
	}
	return s.notLastAdmin(ctx, u)
}

// Delete soft-deletes an account: it's logged out and can't log in, and it's
// left out of member lists, reports and exports, but its data is kept and
// it can be restored. Not the admin's own account, nor the last admin.
func (s *UserAdminService) Delete(ctx context.Context, admin *user.User, id string) error {
	if user.ID(id) == admin.ID() {
		return fmt.Errorf("%w: you can't delete your own account", user.ErrInvalid)
	}
	u, err := s.users.FindByID(ctx, user.ID(id))
	if err != nil {
		return err
	}
	if u.IsDeleted() {
		return nil
	}
	if u.IsAdmin() {
		if err := s.notLastAdmin(ctx, u); err != nil {
			return err
		}
	}
	u.Delete(time.Now())
	if err := s.users.Save(ctx, u); err != nil {
		return err
	}
	return s.sessions.DeleteForUser(ctx, u.ID())
}

// Restore brings back a soft-deleted account, as it was.
func (s *UserAdminService) Restore(ctx context.Context, id string) (*user.User, error) {
	u, err := s.users.FindByID(ctx, user.ID(id))
	if err != nil {
		return nil, err
	}
	u.Restore()
	if err := s.users.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// HardDelete removes an account and all of its data for good. Only an
// account already deleted (soft) can be, so it always takes two steps.
func (s *UserAdminService) HardDelete(ctx context.Context, admin *user.User, id string) error {
	if user.ID(id) == admin.ID() {
		return fmt.Errorf("%w: you can't delete your own account", user.ErrInvalid)
	}
	u, err := s.users.FindByID(ctx, user.ID(id))
	if err != nil {
		return err
	}
	if !u.IsDeleted() {
		return fmt.Errorf("%w: delete the account first; it can be deleted permanently once it's in Deleted accounts", user.ErrInvalid)
	}
	return s.users.HardDelete(ctx, u.ID())
}

// notLastAdmin refuses to leave the team with no active admin.
func (s *UserAdminService) notLastAdmin(ctx context.Context, u *user.User) error {
	active, err := s.users.FindAll(ctx)
	if err != nil {
		return err
	}
	for _, other := range active {
		if other.IsAdmin() && other.ID() != u.ID() {
			return nil
		}
	}
	return fmt.Errorf("%w: %s is the only admin; make someone else an admin first", user.ErrInvalid, u.Name())
}

func checkPassword(password string) error {
	if n := len(password); n < user.MinPasswordLength || n > maxPasswordLength {
		return fmt.Errorf("%w: password must be %d to %d characters", user.ErrInvalid, user.MinPasswordLength, maxPasswordLength)
	}
	return nil
}

func parseTeamRole(team, role string) (project.Project, user.Role, error) {
	t, err := project.Parse(team)
	if err != nil {
		return "", "", fmt.Errorf("%w: choose a team", user.ErrInvalid)
	}
	r, err := user.ParseRole(role)
	if err != nil {
		return "", "", err
	}
	return t, r, nil
}
