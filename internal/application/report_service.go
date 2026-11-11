package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/report"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// WeeklyReportAt is when the weekly report is generated automatically:
// Friday at 17:00 local time.
var WeeklyReportAt = struct {
	Weekday time.Weekday
	Hour    int
}{time.Friday, 17}

// ReportService holds the use cases for compiling everyone's work.
type ReportService struct {
	reports  report.Repository
	checkIns worklog.Repository
	users    user.Repository
}

func NewReportService(reports report.Repository, checkIns worklog.Repository, users user.Repository) *ReportService {
	return &ReportService{reports: reports, checkIns: checkIns, users: users}
}

func (s *ReportService) List(ctx context.Context) ([]*report.Report, error) {
	return s.reports.FindAll(ctx)
}

func (s *ReportService) Get(ctx context.Context, id string) (*report.Report, error) {
	return s.reports.FindByID(ctx, report.ID(id))
}

// Delete removes a saved report. Its days can be generated again; the
// check-ins it was built from are untouched.
func (s *ReportService) Delete(ctx context.Context, id string) error {
	return s.reports.Delete(ctx, report.ID(id))
}

// Generate builds the daily or weekly report for the days from start to
// end, both included, and saves it, replacing an earlier report of the same
// kind and days. The admin picks the days, so a weekly report can be, say,
// Monday to Friday.
func (s *ReportService) Generate(ctx context.Context, kind, start, end, generatedBy string) (*report.Report, error) {
	k, period, err := ParseReportRange(kind, start, end)
	if err != nil {
		return nil, err
	}
	return s.generate(ctx, k, period, generatedBy, time.Now())
}

// PreviewRange builds the daily or weekly report for the days from start to
// end without saving it, so the Reports page can show it as soon as the
// days are picked; Generate saves it.
func (s *ReportService) PreviewRange(ctx context.Context, kind, start, end string) (*report.Report, error) {
	k, period, err := ParseReportRange(kind, start, end)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	data, err := s.build(ctx, period, now)
	if err != nil {
		return nil, err
	}
	return report.New("", k, period, data, "", now), nil
}

// ParseReportRange checks a report's kind and its first and last day.
func ParseReportRange(kind, start, end string) (report.Kind, calendar.Period, error) {
	k, err := report.ParseKind(kind)
	if err != nil {
		return "", calendar.Period{}, err
	}
	first, err := calendar.ParseDay(start)
	if err != nil {
		return "", calendar.Period{}, fmt.Errorf("%w: pick a start date", report.ErrInvalid)
	}
	last, err := calendar.ParseDay(end)
	if err != nil {
		return "", calendar.Period{}, fmt.Errorf("%w: pick an end date", report.ErrInvalid)
	}
	period, err := calendar.NewPeriod(first, last)
	if err != nil {
		return "", calendar.Period{}, fmt.Errorf("%w: the end date is before the start date", report.ErrInvalid)
	}
	return k, period, nil
}

// GenerateWeeklyIfDue makes this week's report, Monday to Friday, once it
// is Friday 17:00 or later, unless it already exists. It catches up on the
// weekend if the server was down on Friday.
func (s *ReportService) GenerateWeeklyIfDue(ctx context.Context, now time.Time) (*report.Report, error) {
	period := calendar.WorkWeekOf(calendar.DayOf(now))
	dueDay := period.First().AddDays((int(WeeklyReportAt.Weekday) + 6) % 7)
	due := dueDay.Start().Add(time.Duration(WeeklyReportAt.Hour) * time.Hour)
	if now.Before(due) {
		return nil, nil
	}
	_, err := s.reports.FindByPeriod(ctx, report.KindWeekly, period)
	if !errors.Is(err, report.ErrNotFound) {
		return nil, err // nil error: already generated
	}
	return s.generate(ctx, report.KindWeekly, period, report.GeneratedBySchedule, now)
}

// Preview builds the weekly report covering day without saving it, so the
// admin dashboard can show the week so far.
func (s *ReportService) Preview(ctx context.Context, day string) (*report.Report, error) {
	d, err := calendar.ParseDay(day)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	period := report.KindWeekly.PeriodFor(d)
	data, err := s.build(ctx, period, now)
	if err != nil {
		return nil, err
	}
	return report.New("", report.KindWeekly, period, data, "", now), nil
}

func (s *ReportService) build(ctx context.Context, period calendar.Period, now time.Time) (report.Data, error) {
	checkIns, err := s.checkIns.FindUntil(ctx, period.Last())
	if err != nil {
		return report.Data{}, err
	}
	users, err := s.users.FindAll(ctx)
	if err != nil {
		return report.Data{}, err
	}
	return report.Build(period, checkIns, users, calendar.DayOf(now)), nil
}

func (s *ReportService) generate(ctx context.Context, kind report.Kind, period calendar.Period, generatedBy string, now time.Time) (*report.Report, error) {
	data, err := s.build(ctx, period, now)
	if err != nil {
		return nil, err
	}
	id, err := s.reports.NextID(ctx)
	if err != nil {
		return nil, err
	}
	r := report.New(id, kind, period, data, generatedBy, now)
	if err := s.reports.Save(ctx, r); err != nil {
		return nil, err
	}
	// Saving over an existing period keeps the old id, so read it back.
	return s.reports.FindByPeriod(ctx, kind, period)
}
