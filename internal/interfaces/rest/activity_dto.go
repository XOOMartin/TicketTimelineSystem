package rest

import (
	"time"

	"crud_app/internal/application"
	"crud_app/internal/domain/activity"
	"crud_app/internal/domain/review"
)

type importResponse struct {
	Found int `json:"found"`
	Added int `json:"added"`
}

type daySummaryResponse struct {
	Day         string `json:"day"`
	PromptCount int    `json:"promptCount"`
}

type promptResponse struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	Project   string    `json:"project"`
	Branch    string    `json:"branch"`
	Text      string    `json:"text"`
	SentAt    time.Time `json:"sentAt"`
}

type improvementResponse struct {
	Issue   string `json:"issue"`
	Example string `json:"example"`
	Better  string `json:"better"`
}

type reviewResponse struct {
	Day          string                `json:"day"`
	Summary      string                `json:"summary"`
	Score        int                   `json:"score"`
	Strengths    []string              `json:"strengths"`
	Improvements []improvementResponse `json:"improvements"`
	Focus        string                `json:"focus"`
	CreatedAt    time.Time             `json:"createdAt"`
}

func toImportResponse(res application.ImportResult) importResponse {
	return importResponse{Found: res.Found, Added: res.Added}
}

func toDaySummaryResponses(summaries []activity.DaySummary) []daySummaryResponse {
	res := make([]daySummaryResponse, len(summaries))
	for i, s := range summaries {
		res[i] = daySummaryResponse{Day: s.Day.String(), PromptCount: s.PromptCount}
	}
	return res
}

func toPromptResponses(prompts []*activity.Prompt) []promptResponse {
	res := make([]promptResponse, len(prompts))
	for i, p := range prompts {
		res[i] = promptResponse{
			ID:        string(p.ID()),
			SessionID: p.SessionID(),
			Project:   p.Project(),
			Branch:    p.Branch(),
			Text:      p.Text(),
			SentAt:    p.SentAt(),
		}
	}
	return res
}

func toReviewResponse(r *review.Review) reviewResponse {
	imps := r.Improvements()
	improvements := make([]improvementResponse, len(imps))
	for i, imp := range imps {
		improvements[i] = improvementResponse{Issue: imp.Issue, Example: imp.Example, Better: imp.Better}
	}
	return reviewResponse{
		Day:          r.Day().String(),
		Summary:      r.Summary(),
		Score:        r.Score(),
		Strengths:    r.Strengths(),
		Improvements: improvements,
		Focus:        r.Focus(),
		CreatedAt:    r.CreatedAt(),
	}
}
