package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
)

// RulesHandler lists available rules metadata.
type RulesHandler struct {
	ruleRepo ports.RuleRepository
}

// NewRulesHandler creates a RulesHandler.
func NewRulesHandler(ruleRepo ports.RuleRepository) *RulesHandler {
	return &RulesHandler{ruleRepo: ruleRepo}
}

type ruleMetaDTO struct {
	ID           string          `json:"id"`
	Severity     int             `json:"severity"`
	Category     string          `json:"category"`
	DetectMethod string          `json:"detect_method"`
	Suggestion   string          `json:"suggestion,omitempty"`
	AutoFix      *string         `json:"auto_fix,omitempty"`
	ClientSide   bool            `json:"client_side"`
	Name         string          `json:"name,omitempty"`
	URL          string          `json:"url,omitempty"`
	Examples     model.Examples  `json:"examples,omitempty"`
	Related      []model.Related `json:"related,omitempty"`
}

// Handle returns all loaded rules.
func (h *RulesHandler) Handle(c echo.Context) error {
	ruleset, err := h.ruleRepo.LoadAll(c.Request().Context())
	if err != nil {
		return err
	}

	out := make([]ruleMetaDTO, 0, len(ruleset.Rules()))
	for _, r := range ruleset.Rules() {
		out = append(out, ruleMetaDTO{
			ID:           r.ID().Value(),
			Severity:     r.Severity().Value(),
			Category:     r.Category().Value(),
			DetectMethod: detectMethodName(r),
			Suggestion:   r.Suggestion().Value(),
			AutoFix:      fixString(r.FixNode()),
			ClientSide:   r.IsClientSide(),
			Name:         r.Name(),
			URL:          r.URL(),
			Examples:     r.Examples(),
			Related:      r.Related(),
		})
	}
	return c.JSON(http.StatusOK, out)
}
