package activity

import (
	"fmt"
	"strings"
	"time"
)

// ID uniquely identifies a Prompt. It comes from the transcript record, so
// importing the same transcript twice yields the same IDs.
type ID string

// Prompt is an entity: one message the user typed into Claude Code.
type Prompt struct {
	id        ID
	sessionID string
	project   string
	branch    string
	text      string
	sentAt    time.Time
}

func NewPrompt(id ID, sessionID, project, branch, text string, sentAt time.Time) (*Prompt, error) {
	text = strings.TrimSpace(text)
	if id == "" {
		return nil, fmt.Errorf("%w: prompt id is required", ErrInvalid)
	}
	if text == "" {
		return nil, fmt.Errorf("%w: prompt text is required", ErrInvalid)
	}
	if sentAt.IsZero() {
		return nil, fmt.Errorf("%w: prompt time is required", ErrInvalid)
	}

	return &Prompt{
		id:        id,
		sessionID: sessionID,
		project:   project,
		branch:    branch,
		text:      text,
		sentAt:    sentAt,
	}, nil
}

func (p *Prompt) ID() ID            { return p.id }
func (p *Prompt) SessionID() string { return p.sessionID }
func (p *Prompt) Project() string   { return p.project }
func (p *Prompt) Branch() string    { return p.branch }
func (p *Prompt) Text() string      { return p.text }
func (p *Prompt) SentAt() time.Time { return p.sentAt }
func (p *Prompt) Day() Day          { return DayOf(p.sentAt) }
