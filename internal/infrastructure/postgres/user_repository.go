package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"

	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
)

// UserRepository is a PostgreSQL user.Repository.
type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, name, username, password_hash, team, role, created_at, first_name, last_name, deleted_at`

// The tables holding a user's things, removed with them on a hard delete
var userTables = []string{
	"sessions", "timers", "imports", "checkins", "planned_tickets",
	"ticket_dates", "ticket_progress", "ticket_parents", "ticket_types",
}

func (r *UserRepository) NextID(ctx context.Context) (user.ID, error) {
	return user.ID(uuid.NewString()), nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *UserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	return r.findMany(ctx, `SELECT `+userColumns+` FROM users WHERE deleted_at IS NULL ORDER BY team, name`)
}

func (r *UserRepository) FindDeleted(ctx context.Context) ([]*user.User, error) {
	return r.findMany(ctx, `SELECT `+userColumns+` FROM users WHERE deleted_at IS NOT NULL ORDER BY deleted_at DESC`)
}

// HardDelete removes the user and all that's theirs, all or nothing.
func (r *UserRepository) HardDelete(ctx context.Context, id user.ID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range userTables {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE user_id = $1`, string(id)); err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, string(id))
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return user.ErrNotFound
	}
	return tx.Commit()
}

func (r *UserRepository) findMany(ctx context.Context, query string) ([]*user.User, error) {
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) FindByID(ctx context.Context, id user.ID) (*user.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, string(id)))
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE username = $1`, user.NormalizeUsername(username)))
}

func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (`+userColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name,
			username = excluded.username,
			password_hash = excluded.password_hash,
			team = excluded.team,
			role = excluded.role,
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			deleted_at = excluded.deleted_at`,
		string(u.ID()), u.Name(), u.Username(), u.PasswordHash(), u.Team().String(), string(u.Role()),
		formatTime(u.CreatedAt()), u.FirstName(), u.LastName(), deletedAt(u),
	)
	if isUniqueViolation(err) {
		return user.ErrUsernameTaken
	}
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (*user.User, error) {
	var id, name, username, hash, team, role, createdAt, first, last string
	var deleted sql.NullString
	err := s.Scan(&id, &name, &username, &hash, &team, &role, &createdAt, &first, &last, &deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p, err := project.Parse(team)
	if err != nil {
		return nil, err
	}
	rl, err := user.ParseRole(role)
	if err != nil {
		return nil, err
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	u, err := user.New(user.ID(id), name, username, hash, p, rl, created)
	if err != nil {
		return nil, err
	}
	u.SetNames(first, last)
	if deleted.Valid {
		at, err := parseTime(deleted.String)
		if err != nil {
			return nil, err
		}
		u.Delete(at)
	}
	return u, nil
}

// deletedAt is the deleted_at column's value: NULL while the account is
// active.
func deletedAt(u *user.User) any {
	if at := u.DeletedAt(); at != nil {
		return formatTime(*at)
	}
	return nil
}

// SessionStore is a PostgreSQL user.SessionStore. Only a hash of each token is
// stored, so a leaked database can't be used to log in.
type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *SessionStore) Create(ctx context.Context, token string, id user.ID, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(token), string(id), formatTime(expiresAt),
	)
	return err
}

func (s *SessionStore) FindUser(ctx context.Context, token string) (user.ID, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > $2`,
		hashToken(token), formatTime(time.Now()),
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", user.ErrUnauthenticated
	}
	return user.ID(id), err
}

func (s *SessionStore) Delete(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(token))
	return err
}

func (s *SessionStore) DeleteForUser(ctx context.Context, id user.ID) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, string(id))
	return err
}
