package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// timeLayout is fixed-width so stored times sort correctly as text.
// Times are kept as text (as they were in SQLite) so moved data reads back
// exactly as it was written.
const timeLayout = "2006-01-02T15:04:05.000Z07:00"

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	name          TEXT NOT NULL,
	username      TEXT NOT NULL UNIQUE, -- lower case; an email for accounts from before usernames
	first_name    TEXT NOT NULL DEFAULT '',
	last_name     TEXT NOT NULL DEFAULT '',
	deleted_at    TEXT, -- set when an admin deleted the account (soft delete)
	password_hash TEXT NOT NULL,
	team          TEXT NOT NULL,
	role          TEXT NOT NULL,
	created_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	token_hash TEXT PRIMARY KEY, -- sha256 of the cookie value
	user_id    TEXT NOT NULL REFERENCES users (id),
	expires_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS checkins (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users (id),
	source     TEXT NOT NULL,
	project    TEXT NOT NULL,
	repo       TEXT NOT NULL DEFAULT '', -- SPinno: spec, backend or web; '' for none
	ticket     TEXT NOT NULL,
	title      TEXT NOT NULL,
	round      INTEGER NOT NULL DEFAULT 1, -- 1 = original work, 2 = first review-back, ...
	phase      TEXT NOT NULL,
	status     TEXT NOT NULL,
	hours      DOUBLE PRECISION NOT NULL,
	note       TEXT NOT NULL,
	worked_at  TEXT NOT NULL, -- UTC, timeLayout
	day        TEXT NOT NULL, -- local date, YYYY-MM-DD
	created_at TEXT NOT NULL,
	-- Re-importing a spreadsheet must not double the hours.
	UNIQUE (user_id, project, repo, ticket, round, phase, worked_at)
);
CREATE INDEX IF NOT EXISTS checkins_user_day ON checkins (user_id, day);
CREATE INDEX IF NOT EXISTS checkins_day ON checkins (day, worked_at);

-- Start and end dates a user set on a ticket; without a row they are worked
-- out from the check-ins.
CREATE TABLE IF NOT EXISTS ticket_dates (
	user_id   TEXT NOT NULL REFERENCES users (id),
	project   TEXT NOT NULL,
	repo      TEXT NOT NULL DEFAULT '',
	ticket    TEXT NOT NULL,
	start_day TEXT, -- YYYY-MM-DD, NULL: first day logged
	end_day   TEXT, -- YYYY-MM-DD, NULL: not finished
	PRIMARY KEY (user_id, project, repo, ticket)
);

-- Tickets a user added to their timeline before logging any time on them.
-- Once time is logged, the check-ins hold the ticket's title and status.
CREATE TABLE IF NOT EXISTS planned_tickets (
	user_id   TEXT NOT NULL REFERENCES users (id),
	project   TEXT NOT NULL,
	repo      TEXT NOT NULL DEFAULT '',
	ticket    TEXT NOT NULL,
	title     TEXT NOT NULL,
	status    TEXT NOT NULL,
	note      TEXT NOT NULL,
	added_day TEXT NOT NULL, -- local date, YYYY-MM-DD
	added_at  TEXT NOT NULL DEFAULT '', -- UTC, timeLayout; '' in rows from before it was kept
	PRIMARY KEY (user_id, project, repo, ticket)
);

-- The step each user is on in a ticket, from the progress steps.
CREATE TABLE IF NOT EXISTS ticket_progress (
	user_id   TEXT NOT NULL REFERENCES users (id),
	project   TEXT NOT NULL,
	repo      TEXT NOT NULL DEFAULT '',
	ticket    TEXT NOT NULL,
	round     INTEGER NOT NULL,
	step      TEXT NOT NULL, -- a phase; '' once finished
	finished  BOOLEAN NOT NULL DEFAULT FALSE,
	flow_step TEXT NOT NULL DEFAULT '', -- a typed ticket's workflow step; '' for none
	PRIMARY KEY (user_id, project, repo, ticket)
);

-- The type chosen for each SPinno ticket, per user (its steps follow it);
-- no row: the repo's default.
CREATE TABLE IF NOT EXISTS ticket_types (
	user_id     TEXT NOT NULL REFERENCES users (id),
	project     TEXT NOT NULL,
	repo        TEXT NOT NULL,
	ticket      TEXT NOT NULL,
	type        TEXT NOT NULL,
	made_parent BOOLEAN NOT NULL DEFAULT FALSE, -- a child spec made a parent by dragging
	PRIMARY KEY (user_id, project, repo, ticket)
);

-- The spec ticket each SPinno backend or web ticket sits under, per user;
-- no row: none chosen.
CREATE TABLE IF NOT EXISTS ticket_parents (
	user_id TEXT NOT NULL REFERENCES users (id),
	project TEXT NOT NULL,
	repo    TEXT NOT NULL,
	ticket  TEXT NOT NULL,
	parent  TEXT NOT NULL, -- the spec ticket's number
	PRIMARY KEY (user_id, project, repo, ticket)
);

-- The timer each user has running, if any: one phase of one ticket.
CREATE TABLE IF NOT EXISTS timers (
	user_id        TEXT PRIMARY KEY REFERENCES users (id),
	project        TEXT NOT NULL,
	repo           TEXT NOT NULL DEFAULT '',
	ticket         TEXT NOT NULL,
	title          TEXT NOT NULL,
	round          INTEGER NOT NULL,
	phase          TEXT NOT NULL,
	started_at     TEXT NOT NULL, -- UTC, timeLayout
	paused_at      TEXT,          -- when the current pause began; NULL while counting
	paused_seconds INTEGER NOT NULL DEFAULT 0, -- earlier pauses, left out of the time
	closing_at     TEXT,          -- when the site was closed; NULL while it's open
	flow_step      TEXT NOT NULL DEFAULT '' -- a typed ticket's workflow step being timed
);

CREATE TABLE IF NOT EXISTS imports (
	user_id     TEXT NOT NULL REFERENCES users (id),
	week        TEXT NOT NULL, -- the Monday, YYYY-MM-DD
	filename    TEXT NOT NULL,
	added       INTEGER NOT NULL,
	skipped     INTEGER NOT NULL,
	imported_by TEXT NOT NULL REFERENCES users (id),
	imported_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS imports_week ON imports (week, imported_at);

CREATE TABLE IF NOT EXISTS reports (
	id           TEXT PRIMARY KEY,
	kind         TEXT NOT NULL,
	first_day    TEXT NOT NULL,
	last_day     TEXT NOT NULL,
	data         TEXT NOT NULL, -- JSON of report.Data
	generated_by TEXT NOT NULL,
	generated_at TEXT NOT NULL,
	UNIQUE (kind, first_day)
);
`

// Tables lists every table, parents before the tables that reference them.
var Tables = []string{
	"users", "sessions", "checkins", "ticket_dates", "planned_tickets",
	"ticket_progress", "ticket_types", "ticket_parents", "timers", "imports", "reports",
}

// Open connects to the database at dsn (a postgres:// URL or key=value
// string) and makes sure the tables exist.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}
	return db, nil
}

// isUniqueViolation reports whether err came from a UNIQUE constraint.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) (time.Time, error) { return time.Parse(timeLayout, s) }
