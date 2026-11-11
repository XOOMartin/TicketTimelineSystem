package worklog

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
)

const (
	// The most one entry of time can hold: a working day
	MaxHours       = 8
	maxTitleLength = 200
	maxNoteLength  = 1000
)

var ticketNumberPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,20}$`)

// ID uniquely identifies a CheckIn.
type ID string

// Source records how a check-in got into the system.
type Source string

const (
	SourceApp    Source = "app"
	SourceImport Source = "import"
)

// TicketKey identifies a ticket across check-ins, e.g. BlackBox#3221.
type TicketKey struct {
	Project project.Project
	// Repo is set for SPinno tickets, each repo numbering its own; "" for
	// BlackBox and SPinno tickets from before repos
	Repo   project.Repo
	Number string
}

// String is e.g. "BlackBox#3221", or "SPinno/web#120" for a repo's ticket.
func (k TicketKey) String() string {
	if k.Repo != "" {
		return fmt.Sprintf("%s/%s#%s", k.Project, k.Repo, k.Number)
	}
	return fmt.Sprintf("%s#%s", k.Project, k.Number)
}

// ParseTicketKey reads a ticket from its project, repo ("" for none) and
// number, as typed or sent.
func ParseTicketKey(projectName, repo, ticket string) (TicketKey, error) {
	p, err := project.Parse(projectName)
	if err != nil {
		return TicketKey{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	r, err := p.ParseRepo(repo)
	if err != nil {
		// "SPinno's repo must be …", without project.ErrInvalid's own prefix
		return TicketKey{}, fmt.Errorf("%w: %s", ErrInvalid, strings.TrimPrefix(err.Error(), project.ErrInvalid.Error()+": "))
	}
	number, err := ParseTicketNumber(ticket)
	if err != nil {
		return TicketKey{}, err
	}
	return TicketKey{Project: p, Repo: r, Number: number}, nil
}

// ParseTicketNumber accepts "3221", "#3221" or "BlackBox#3221".
func ParseTicketNumber(s string) (string, error) {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "#"); i >= 0 {
		s = s[i+1:]
	}
	if !ticketNumberPattern.MatchString(s) {
		return "", fmt.Errorf("%w: ticket must be a number like 3221", ErrInvalid)
	}
	return s, nil
}

// Entry is the part of a check-in the user types in.
type Entry struct {
	Ticket TicketKey
	Title  string
	// Round is 1 for the original work, 2 after the first review sent the
	// ticket back, and so on. Zero means 1.
	Round    int
	Phase    Phase
	Status   Status
	Hours    float64
	Note     string
	WorkedAt time.Time
}

// CheckIn is an aggregate root: one block of work a user reports on a
// ticket. A ticket's history is the sequence of its check-ins.
type CheckIn struct {
	id        ID
	userID    user.ID
	source    Source
	entry     Entry
	createdAt time.Time
}

func New(id ID, userID user.ID, source Source, entry Entry, createdAt time.Time) (*CheckIn, error) {
	if id == "" || userID == "" {
		return nil, fmt.Errorf("%w: id and user are required", ErrInvalid)
	}
	c := &CheckIn{id: id, userID: userID, source: source, createdAt: createdAt}
	if err := c.Revise(entry); err != nil {
		return nil, err
	}
	return c, nil
}

// Restore rebuilds a check-in that was already saved, as it was saved. The
// rules for new entries (e.g. MaxHours) aren't applied again: an entry
// saved under an earlier limit still loads, and only a change to it has to
// meet today's rules.
func Restore(id ID, userID user.ID, source Source, entry Entry, createdAt time.Time) *CheckIn {
	return &CheckIn{id: id, userID: userID, source: source, entry: entry, createdAt: createdAt}
}

// Revise replaces what the user reported.
func (c *CheckIn) Revise(e Entry) error {
	number, err := ParseTicketNumber(e.Ticket.Number)
	if err != nil {
		return err
	}
	e.Ticket.Number = number
	e.Title = strings.TrimSpace(e.Title)
	e.Note = strings.TrimSpace(e.Note)
	if e.Round == 0 {
		e.Round = 1
	}
	// Round to the minute so the same entry always stores the same time.
	e.WorkedAt = e.WorkedAt.Truncate(time.Minute)
	// Time spent is entered in hours and minutes, so keep whole minutes:
	// 20 minutes stays 1/3 h rather than 0.33 h, and sums add up exactly.
	e.Hours = math.Round(e.Hours*60) / 60

	switch {
	case e.Ticket.Project == "":
		return fmt.Errorf("%w: project is required", ErrInvalid)
	case e.Title == "":
		return fmt.Errorf("%w: title is required", ErrInvalid)
	case len([]rune(e.Title)) > maxTitleLength:
		return fmt.Errorf("%w: title is longer than %d characters", ErrInvalid, maxTitleLength)
	case e.Round < 1 || e.Round > MaxRound:
		return fmt.Errorf("%w: round must be between 1 and %d", ErrInvalid, MaxRound)
	case e.Phase.order() < 0:
		return fmt.Errorf("%w: phase is required", ErrInvalid)
	case e.Status.Label() == "":
		return fmt.Errorf("%w: status is required", ErrInvalid)
	case math.IsNaN(e.Hours) || e.Hours <= 0 || e.Hours > MaxHours:
		return fmt.Errorf("%w: time spent must be at least 1 minute and at most %d hours", ErrInvalid, MaxHours)
	case e.Status == StatusBlocked && e.Note == "":
		return fmt.Errorf("%w: say what is blocking the ticket in the note", ErrInvalid)
	case len([]rune(e.Note)) > maxNoteLength:
		return fmt.Errorf("%w: note is longer than %d characters", ErrInvalid, maxNoteLength)
	case e.WorkedAt.IsZero():
		return fmt.Errorf("%w: date and time are required", ErrInvalid)
	}

	c.entry = e
	return nil
}

func (c *CheckIn) ID() ID                  { return c.id }
func (c *CheckIn) UserID() user.ID         { return c.userID }
func (c *CheckIn) Source() Source          { return c.source }
func (c *CheckIn) Ticket() TicketKey       { return c.entry.Ticket }
func (c *CheckIn) Title() string           { return c.entry.Title }
func (c *CheckIn) Round() int              { return c.entry.Round }
func (c *CheckIn) Phase() Phase            { return c.entry.Phase }
func (c *CheckIn) Status() Status          { return c.entry.Status }
func (c *CheckIn) Hours() float64          { return c.entry.Hours }
func (c *CheckIn) Note() string            { return c.entry.Note }
func (c *CheckIn) WorkedAt() time.Time     { return c.entry.WorkedAt }
func (c *CheckIn) Day() calendar.Day       { return calendar.DayOf(c.entry.WorkedAt) }
func (c *CheckIn) CreatedAt() time.Time    { return c.createdAt }
func (c *CheckIn) Entry() Entry            { return c.entry }
func (c *CheckIn) OwnedBy(id user.ID) bool { return c.userID == id }

// Before reports whether c comes before other in a ticket's history: by
// time, then for entries logged at the same time, an earlier round first and
// within a round, workflow order.
func (c *CheckIn) Before(other *CheckIn) bool {
	if !c.entry.WorkedAt.Equal(other.entry.WorkedAt) {
		return c.entry.WorkedAt.Before(other.entry.WorkedAt)
	}
	if c.entry.Round != other.entry.Round {
		return c.entry.Round < other.entry.Round
	}
	return c.entry.Phase.order() < other.entry.Phase.order()
}
