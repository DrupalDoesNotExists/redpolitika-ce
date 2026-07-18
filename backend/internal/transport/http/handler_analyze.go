package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/transport/dto"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/usecase"
)

// AnalyzeHandler handles text analysis requests.
type AnalyzeHandler struct {
	analyzeUC *usecase.AnalyzeTextUseCase
}

// NewAnalyzeHandler creates an AnalyzeHandler.
func NewAnalyzeHandler(analyzeUC *usecase.AnalyzeTextUseCase) *AnalyzeHandler {
	return &AnalyzeHandler{analyzeUC: analyzeUC}
}

type analyzeRequest struct {
	Text string `json:"text"`
}

// Handle processes a text analysis request.
func (h *AnalyzeHandler) Handle(c echo.Context) error {
	var req analyzeRequest
	if err := c.Bind(&req); err != nil {
		return writeProblem(c, http.StatusBadRequest, "Bad Request", "invalid JSON body")
	}

	full := c.QueryParam("full") == "true"

	result, err := h.analyzeUC.Execute(c.Request().Context(), usecase.AnalyzeRequest{
		Text: req.Text,
		Full: full,
	})
	if err != nil {
		return err
	}

	resp := dto.NewAnalysisResponse(result.Analysis, "")
	return c.JSON(http.StatusOK, resp)
}
