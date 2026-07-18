// Package dto provides shared transport DTOs for handling rule flags.
// Both HTTP and WebSocket handlers use these for consistent JSON responses.
package dto

import "github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"

// AnchorDTO is the nested anchor object for a flag.
type AnchorDTO struct {
	ParagraphIndex int    `json:"paragraph_index"`
	Occurrence     int    `json:"occurrence"`
	MatchText      string `json:"match_text"`
}

// FlagDTO is the JSON-safe representation of a rule flag.
type FlagDTO struct {
	ID         string          `json:"id"`
	RuleID     string          `json:"rule_id"`
	Category   string          `json:"category"`
	Severity   int             `json:"severity"`
	Message    string          `json:"message"`
	Suggestion string          `json:"suggestion,omitempty"`
	AutoFix    *string         `json:"auto_fix,omitempty"`
	Anchor     AnchorDTO       `json:"anchor"`
	State      string          `json:"state,omitempty"`
	RuleName   string          `json:"rule_name,omitempty"`
	RuleURL    string          `json:"rule_url,omitempty"`
	Examples   model.Examples  `json:"examples,omitempty"`
	Related    []model.Related `json:"related,omitempty"`
}

// AnalysisResponse is the JSON response for an analysis run.
type AnalysisResponse struct {
	Flags            []FlagDTO `json:"flags"`
	CleanlinessScore float64   `json:"cleanliness_score"`
	ReadabilityScore float64   `json:"readability_score"`
	FlagCount        int       `json:"flag_count"`
	SessionID        string    `json:"session_id,omitempty"`
}

// FlagToDTO converts a domain Flag to a transport DTO.
func FlagToDTO(f *model.Flag) FlagDTO {
	return FlagDTO{
		ID:         f.ID().String(),
		RuleID:     f.RuleID().Value(),
		Category:   f.Category().Value(),
		Severity:   f.Severity().Value(),
		Message:    f.Message(),
		Suggestion: f.Suggestion().Value(),
		AutoFix:    f.Autofix(),
		Anchor: AnchorDTO{
			ParagraphIndex: f.ParagraphIndex().Value(),
			Occurrence:     f.Occurrence().Value(),
			MatchText:      f.MatchText().Value(),
		},
		State:    f.State().String(),
		RuleName: f.RuleName(),
		RuleURL:  f.RuleURL(),
		Examples: f.Examples(),
		Related:  f.Related(),
	}
}

// FlagsToDTOs converts a slice of domain Flags to transport DTOs.
func FlagsToDTOs(flags []*model.Flag) []FlagDTO {
	dtos := make([]FlagDTO, 0, len(flags))
	for _, f := range flags {
		dtos = append(dtos, FlagToDTO(f))
	}
	return dtos
}

// NewAnalysisResponse builds a response from an analysis result.
func NewAnalysisResponse(analysis *model.Analysis, sessionID string) AnalysisResponse {
	return AnalysisResponse{
		Flags:            FlagsToDTOs(analysis.Flags()),
		CleanlinessScore: analysis.CleanlinessScore().Value(),
		ReadabilityScore: analysis.ReadabilityScore().Value(),
		FlagCount:        len(analysis.Flags()),
		SessionID:        sessionID,
	}
}
