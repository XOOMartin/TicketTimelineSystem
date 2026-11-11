package sqlite

import (
	"context"
	"database/sql"
	"time"

	"crud_app/internal/domain/activity"
)

// PromptRepository is a SQLite activity.Repository.
type PromptRepository struct {
	db *sql.DB
}

func NewPromptRepository(db *sql.DB) *PromptRepository {
	return &PromptRepository{db: db}
}

func (r *PromptRepository) SaveAll(ctx context.Context, prompts []*activity.Prompt) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// The id comes from the transcript, so OR IGNORE skips prompts imported before.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO prompts (id, session_id, project, branch, text, sent_at, day)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	added := 0
	for _, p := range prompts {
		res, err := stmt.ExecContext(ctx,
			string(p.ID()), p.SessionID(), p.Project(), p.Branch(), p.Text(),
			p.SentAt().UTC().Format(timeLayout), p.Day().String(),
		)
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		added += int(n)
	}
	return added, tx.Commit()
}

func (r *PromptRepository) FindByDay(ctx context.Context, day activity.Day) ([]*activity.Prompt, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, session_id, project, branch, text, sent_at
		FROM prompts WHERE day = ? ORDER BY sent_at`, day.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []*activity.Prompt
	for rows.Next() {
		var id, sessionID, project, branch, text, sentAt string
		if err := rows.Scan(&id, &sessionID, &project, &branch, &text, &sentAt); err != nil {
			return nil, err
		}
		t, err := time.Parse(timeLayout, sentAt)
		if err != nil {
			return nil, err
		}
		// Rebuild through the constructor, so stored rows obey the same rules.
		p, err := activity.NewPrompt(activity.ID(id), sessionID, project, branch, text, t)
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	return prompts, rows.Err()
}

func (r *PromptRepository) Summaries(ctx context.Context) ([]activity.DaySummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT day, COUNT(*) FROM prompts GROUP BY day ORDER BY day DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []activity.DaySummary
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		d, err := activity.ParseDay(day)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, activity.DaySummary{Day: d, PromptCount: count})
	}
	return summaries, rows.Err()
}
