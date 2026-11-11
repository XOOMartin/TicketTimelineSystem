package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// timeLayout is fixed-width so stored times sort correctly as text.
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
	hours      REAL NOT NULL,
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
	user_id  TEXT NOT NULL REFERENCES users (id),
	project  TEXT NOT NULL,
	repo     TEXT NOT NULL DEFAULT '',
	ticket   TEXT NOT NULL,
	round    INTEGER NOT NULL,
	step     TEXT NOT NULL, -- a phase; '' once finished
	finished INTEGER NOT NULL DEFAULT 0,
	flow_step TEXT NOT NULL DEFAULT '', -- a typed ticket's workflow step; '' for none
	PRIMARY KEY (user_id, project, repo, ticket)
);

-- The spec ticket each SPinno backend or web ticket sits under, per user;
-- no row: none chosen.
-- The type chosen for each SPinno ticket, per user (its steps follow it);
-- no row: the repo's default.
CREATE TABLE IF NOT EXISTS ticket_types (
	user_id TEXT NOT NULL REFERENCES users (id),
	project TEXT NOT NULL,
	repo    TEXT NOT NULL,
	ticket  TEXT NOT NULL,
	type    TEXT NOT NULL,
	PRIMARY KEY (user_id, project, repo, ticket)
);

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
	user_id    TEXT PRIMARY KEY REFERENCES users (id),
	project    TEXT NOT NULL,
	repo       TEXT NOT NULL DEFAULT '',
	ticket     TEXT NOT NULL,
	title      TEXT NOT NULL,
	round      INTEGER NOT NULL,
	phase      TEXT NOT NULL,
	started_at TEXT NOT NULL, -- UTC, timeLayout
	paused_at  TEXT,          -- when the current pause began; NULL while counting
	paused_seconds INTEGER NOT NULL DEFAULT 0, -- earlier pauses, left out of the time
	closing_at TEXT,          -- when the site was closed; NULL while it's open
	flow_step  TEXT NOT NULL DEFAULT '' -- a typed ticket's workflow step being timed
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

// Open opens (or creates) the database file and makes sure the tables exist.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("update tables: %w", err)
	}
	return db, nil
}

// migrate brings databases created by older versions up to the schema above.
// CREATE TABLE IF NOT EXISTS leaves existing tables as they were.
func migrate(db *sql.DB) error {
	// Accounts logged in with an email; now a username, the email kept as it
	if err := renameColumn(db, "users", "email", "username"); err != nil {
		return err
	}
	// Check-ins before rounds existed were all original work.
	if err := addColumn(db, "checkins", "round", "INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	// Tickets added without time before the exact time was kept
	if err := addColumn(db, "planned_tickets", "added_at", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	// Timers made before they could be paused
	// Accounts from before signup asked for them have no first or last name
	if err := addColumn(db, "users", "first_name", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumn(db, "users", "last_name", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	// Accounts from before soft delete are all active
	if err := addColumn(db, "users", "deleted_at", "TEXT"); err != nil {
		return err
	}
	if err := addColumn(db, "timers", "paused_at", "TEXT"); err != nil {
		return err
	}
	if err := addColumn(db, "timers", "paused_seconds", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	// Timers made before closing the site ended them
	if err := addColumn(db, "timers", "closing_at", "TEXT"); err != nil {
		return err
	}
	// Made before SPinno tickets had types, each with its own steps
	if err := addColumn(db, "timers", "flow_step", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumn(db, "ticket_progress", "flow_step", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	// A child spec made a parent by dragging keeps its type for when it's
	// dragged back to being a child
	if err := addColumn(db, "ticket_types", "made_parent", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addRoundToCheckInKey(db); err != nil {
		return err
	}
	// Timers made before SPinno tickets had repos
	if err := addColumn(db, "timers", "repo", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addRepoToTicketKeys(db); err != nil {
		return err
	}
	return reviewBacksFromPlanning(db)
}

// reviewBacksFromPlanning moves review-backs started before they began at
// planning (they began at coding) back to planning, if no time has been
// logged in them yet. It runs once, recorded in the database's user_version,
// so a later choice to skip to coding is kept.
func reviewBacksFromPlanning(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version >= 1 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	steps := []string{
		`UPDATE ticket_progress SET step = 'planning'
		WHERE step = 'coding' AND round > 1 AND NOT EXISTS (
			SELECT 1 FROM checkins c
			WHERE c.user_id = ticket_progress.user_id AND c.project = ticket_progress.project
				AND c.repo = ticket_progress.repo AND c.ticket = ticket_progress.ticket
				AND c.round = ticket_progress.round
		)`,
		`PRAGMA user_version = 1`,
	}
	for _, step := range steps {
		if _, err := tx.Exec(step); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// addRoundToCheckInKey lets the same phase be logged at the same time in
// different rounds. SQLite can't change a table's UNIQUE constraint, so an
// old table is rebuilt from the schema and its rows copied across.
func addRoundToCheckInKey(db *sql.DB) error {
	var current string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'checkins'`).Scan(&current); err != nil {
		return err
	}
	if strings.Contains(current, "ticket, round, phase") {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	const columns = `id, user_id, source, project, ticket, title, round, phase, status, hours, note, worked_at, day, created_at`
	steps := []string{
		`ALTER TABLE checkins RENAME TO checkins_old`,
		schema, // the new checkins table
		`INSERT INTO checkins (` + columns + `) SELECT ` + columns + ` FROM checkins_old`,
		`DROP TABLE checkins_old`,
		schema, // its indexes, whose names the old table held until now
	}
	for _, step := range steps {
		if _, err := tx.Exec(step); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// addRepoToTicketKeys makes each table that names a ticket name its repo too,
// in its key, so a SPinno repo's ticket numbers are its own. SQLite can't
// change a key, so each older table is rebuilt from the schema and its rows
// copied across, with no repo.
func addRepoToTicketKeys(db *sql.DB) error {
	for _, table := range []string{"checkins", "ticket_dates", "planned_tickets", "ticket_progress"} {
		var current string
		err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&current)
		if err != nil {
			return err
		}
		if strings.Contains(current, "repo") {
			continue
		}
		if err := rebuildWithRepo(db, table); err != nil {
			return fmt.Errorf("adding repo to %s: %w", table, err)
		}
	}
	return nil
}

func rebuildWithRepo(db *sql.DB, table string) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return err
	}
	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		columns = append(columns, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	list := strings.Join(columns, ", ")

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	steps := []string{
		`ALTER TABLE ` + table + ` RENAME TO ` + table + `_old`,
		schema, // the new table
		`INSERT INTO ` + table + ` (` + list + `) SELECT ` + list + ` FROM ` + table + `_old`,
		`DROP TABLE ` + table + `_old`,
		schema, // its indexes, whose names the old table held until now
	}
	for _, step := range steps {
		if _, err := tx.Exec(step); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// addColumn adds the column unless the table already has it.
func addColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, definition))
	return err
}

// renameColumn renames a column if the table still has it by its old name.
func renameColumn(db *sql.DB, table, from, to string) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, from).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to))
	return err
}

// isUniqueViolation reports whether err came from a UNIQUE constraint.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) (time.Time, error) { return time.Parse(timeLayout, s) }
