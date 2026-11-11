package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/report"
)

// ReportRepository is a SQLite report.Repository. The report's tables are
// stored as one JSON column, since a report is only ever read whole.
type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

const reportColumns = `id, kind, first_day, last_day, data, generated_by, generated_at`

func (r *ReportRepository) NextID(ctx context.Context) (report.ID, error) {
	return report.ID(uuid.NewString()), nil
}

func (r *ReportRepository) Save(ctx context.Context, rp *report.Report) error {
	data, err := json.Marshal(rp.Data())
	if err != nil {
		return err
	}
	// Regenerating a period keeps its row but replaces the content.
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO reports (`+reportColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (kind, first_day) DO UPDATE SET
			last_day = excluded.last_day,
			data = excluded.data,
			generated_by = excluded.generated_by,
			generated_at = excluded.generated_at`,
		string(rp.ID()), string(rp.Kind()), rp.Period().First().String(), rp.Period().Last().String(),
		string(data), rp.GeneratedBy(), formatTime(rp.GeneratedAt()),
	)
	return err
}

func (r *ReportRepository) FindAll(ctx context.Context) ([]*report.Report, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+reportColumns+` FROM reports ORDER BY first_day DESC, kind DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		rp, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, rp)
	}
	return reports, rows.Err()
}

func (r *ReportRepository) FindByID(ctx context.Context, id report.ID) (*report.Report, error) {
	rp, err := scanReport(r.db.QueryRowContext(ctx, `SELECT `+reportColumns+` FROM reports WHERE id = ?`, string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, report.ErrNotFound
	}
	return rp, err
}

func (r *ReportRepository) FindByPeriod(ctx context.Context, kind report.Kind, first calendar.Day) (*report.Report, error) {
	rp, err := scanReport(r.db.QueryRowContext(ctx,
		`SELECT `+reportColumns+` FROM reports WHERE kind = ? AND first_day = ?`, string(kind), first.String(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, report.ErrNotFound
	}
	return rp, err
}

func scanReport(s scanner) (*report.Report, error) {
	var id, kind, firstDay, lastDay, data, generatedBy, generatedAt string
	if err := s.Scan(&id, &kind, &firstDay, &lastDay, &data, &generatedBy, &generatedAt); err != nil {
		return nil, err
	}
	k, err := report.ParseKind(kind)
	if err != nil {
		return nil, err
	}
	first, err := calendar.ParseDay(firstDay)
	if err != nil {
		return nil, err
	}
	last, err := calendar.ParseDay(lastDay)
	if err != nil {
		return nil, err
	}
	period, err := calendar.NewPeriod(first, last)
	if err != nil {
		return nil, err
	}
	var d report.Data
	if err := json.Unmarshal([]byte(data), &d); err != nil {
		return nil, err
	}
	at, err := parseTime(generatedAt)
	if err != nil {
		return nil, err
	}
	return report.New(report.ID(id), k, period, d, generatedBy, at), nil
}
