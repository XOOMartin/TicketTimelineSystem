package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"crud_app/internal/domain/activity"
	"crud_app/internal/domain/review"
)

// ReviewRepository is a SQLite review.Repository.
type ReviewRepository struct {
	db *sql.DB
}

func NewReviewRepository(db *sql.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

// improvementRow is how an Improvement is stored inside the JSON column.
type improvementRow struct {
	Issue   string `json:"issue"`
	Example string `json:"example"`
	Better  string `json:"better"`
}

func (r *ReviewRepository) FindByDay(ctx context.Context, day activity.Day) (*review.Review, error) {
	var summary, strengthsJSON, improvementsJSON, focus, createdAt string
	var score int
	err := r.db.QueryRowContext(ctx, `
		SELECT summary, score, strengths, improvements, focus, created_at
		FROM reviews WHERE day = ?`, day.String(),
	).Scan(&summary, &score, &strengthsJSON, &improvementsJSON, &focus, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, review.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var strengths []string
	if err := json.Unmarshal([]byte(strengthsJSON), &strengths); err != nil {
		return nil, err
	}
	var rows []improvementRow
	if err := json.Unmarshal([]byte(improvementsJSON), &rows); err != nil {
		return nil, err
	}
	improvements := make([]review.Improvement, len(rows))
	for i, row := range rows {
		improvements[i] = review.Improvement{Issue: row.Issue, Example: row.Example, Better: row.Better}
	}
	created, err := time.Parse(timeLayout, createdAt)
	if err != nil {
		return nil, err
	}

	return review.New(day, review.Analysis{
		Summary:      summary,
		Score:        score,
		Strengths:    strengths,
		Improvements: improvements,
		Focus:        focus,
	}, created)
}

func (r *ReviewRepository) Save(ctx context.Context, rv *review.Review) error {
	strengths, err := json.Marshal(rv.Strengths())
	if err != nil {
		return err
	}
	imps := rv.Improvements()
	rows := make([]improvementRow, len(imps))
	for i, imp := range imps {
		rows[i] = improvementRow{Issue: imp.Issue, Example: imp.Example, Better: imp.Better}
	}
	improvements, err := json.Marshal(rows)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO reviews (day, summary, score, strengths, improvements, focus, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (day) DO UPDATE SET
			summary = excluded.summary,
			score = excluded.score,
			strengths = excluded.strengths,
			improvements = excluded.improvements,
			focus = excluded.focus,
			created_at = excluded.created_at`,
		rv.Day().String(), rv.Summary(), rv.Score(), string(strengths), string(improvements),
		rv.Focus(), rv.CreatedAt().UTC().Format(timeLayout),
	)
	return err
}
