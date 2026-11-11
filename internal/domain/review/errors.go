package review

import "errors"

var (
	ErrNotFound       = errors.New("review not found")
	ErrInvalid        = errors.New("invalid review")
	ErrNoPrompts      = errors.New("no prompts on this day")
	ErrAnalysisFailed = errors.New("analysis failed")
)
