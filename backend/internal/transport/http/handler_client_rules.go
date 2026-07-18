package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/detect"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/fix"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
)

// ClientRulesHandler serves client-side rules for frontend.
type ClientRulesHandler struct {
	ruleRepo ports.RuleRepository
}

// NewClientRulesHandler creates a ClientRulesHandler.
func NewClientRulesHandler(ruleRepo ports.RuleRepository) *ClientRulesHandler {
	return &ClientRulesHandler{ruleRepo: ruleRepo}
}

// Handle responds with client-side rules only (regex/wordlist/expr per A29).
func (h *ClientRulesHandler) Handle(c echo.Context) error {
	ruleset, err := h.ruleRepo.LoadAll(c.Request().Context())
	if err != nil {
		return err
	}

		type ruleDTO struct {
			ID           string          `json:"id"`
			Severity     int             `json:"severity"`
			Category     string          `json:"category"`
			Method       string          `json:"method"`
			Pattern      string          `json:"pattern,omitempty"`
			Words        []string        `json:"words,omitempty"`
			CaseSensitive bool           `json:"case_sensitive,omitempty"`
			Suggestion   string          `json:"suggestion,omitempty"`
			AutoFix      *string         `json:"auto_fix,omitempty"`
			Engine       string          `json:"engine,omitempty"`
			Name         string          `json:"name,omitempty"`
			URL          string          `json:"url,omitempty"`
			Examples     model.Examples  `json:"examples,omitempty"`
			Related      []model.Related `json:"related,omitempty"`
		}

	var out []ruleDTO
	for _, r := range ruleset.ClientRules() {
		dto := ruleDTO{
			ID:           r.ID().Value(),
			Severity:     r.Severity().Value(),
			Category:     r.Category().Value(),
			Method:       detectMethodName(r),
			Suggestion:   r.Suggestion().Value(),
			Name:         r.Name(),
			URL:          r.URL(),
			Examples:     r.Examples(),
			Related:      r.Related(),
		}
		// Extract flat fields from detect tree for client-side rules
		switch n := r.DetectNode().(type) {
		case *detect.RegexNode:
			dto.Pattern = n.Pattern.String()
		case *detect.WordlistNode:
			dto.Words = n.Words
			dto.CaseSensitive = n.CaseSensitive
		}
		// Extract autofix from fix tree
		if s := fixString(r.FixNode()); s != nil {
			dto.AutoFix = s
		}
		// Все regex-правила валидированы как RE2 (A31).
		// Клиент обязан использовать RE2-совместимый движок для идентичных результатов.
		if dto.Method == "regex" {
			dto.Engine = "re2"
		}
		out = append(out, dto)
	}

	return c.JSON(http.StatusOK, out)
}

// detectMethodName returns the method string from detect node type.
func detectMethodName(r *model.Rule) string {
	dm := r.DetectMethod().Value()
	if dm != "" {
		return dm
	}
	switch r.DetectNode().(type) {
	case *detect.RegexNode:
		return "regex"
	case *detect.WordlistNode:
		return "wordlist"
	default:
		return ""
	}
}

// fixString extracts the replacement string from a simple ReplaceNode fix tree.
func fixString(n fix.Node) *string {
	if n == nil {
		return nil
	}
	if rn, ok := n.(*fix.ReplaceNode); ok {
		return &rn.With
	}
	return nil
}
