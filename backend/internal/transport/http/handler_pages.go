package http

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/usecase"
)

// PagesHandler serves static pages via usecase.
type PagesHandler struct {
	pagesUC *usecase.PagesUseCase
	logger  *zap.Logger
}

// NewPagesHandler creates a PagesHandler.
func NewPagesHandler(pagesUC *usecase.PagesUseCase, logger *zap.Logger) *PagesHandler {
	return &PagesHandler{pagesUC: pagesUC, logger: logger}
}

// ListPages returns all static pages as JSON.
func (h *PagesHandler) ListPages(c echo.Context) error {
	pages, err := h.pagesUC.ListPages(c.Request().Context())
	if err != nil {
		h.logger.Error("list pages", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list pages")
	}
	return c.JSON(http.StatusOK, pages)
}

// GetPage returns a single page as JSON.
func (h *PagesHandler) GetPage(c echo.Context) error {
	slug := c.Param("*slug")
	slug = strings.TrimPrefix(slug, "/")
	slug = strings.TrimSuffix(slug, "/")
	if slug == "" {
		return echo.NewHTTPError(http.StatusNotFound, "page not found")
	}
	page, err := h.pagesUC.GetPage(c.Request().Context(), slug)
	if err != nil {
		h.logger.Error("get page", zap.String("slug", slug), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get page")
	}
	if page == nil {
		return echo.NewHTTPError(http.StatusNotFound, "page not found")
	}
	return c.JSON(http.StatusOK, page)
}
