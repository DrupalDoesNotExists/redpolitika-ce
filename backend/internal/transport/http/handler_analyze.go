package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/transport/dto"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/usecase"
)

// AnalyzeHandler handles text analysis requests.
type AnalyzeHandler struct {
	analyzeUC *usecase.AnalyzeTextUseCase
	health    *HealthHandler
}

// NewAnalyzeHandler creates an AnalyzeHandler.
func NewAnalyzeHandler(analyzeUC *usecase.AnalyzeTextUseCase, health *HealthHandler) *AnalyzeHandler {
	return &AnalyzeHandler{analyzeUC: analyzeUC, health: health}
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

	start := time.Now()

	result, err := h.analyzeUC.Execute(c.Request().Context(), usecase.AnalyzeRequest{
		Text: req.Text,
	})
	if err != nil {
		return err
	}

	flagCount := 0
	if result.Analysis != nil {
		flagCount = len(result.Analysis.Flags())
	}
	if h.health != nil {
		h.health.ObserveAnalyze(time.Since(start), flagCount)
	}

	resp := dto.NewAnalysisResponse(result.Analysis, "")
	return c.JSON(http.StatusOK, resp)
}
