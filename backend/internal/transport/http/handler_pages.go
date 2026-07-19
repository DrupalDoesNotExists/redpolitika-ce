package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
)

// PagesHandler serves static pages from a StaticPagesProvider.
type PagesHandler struct {
	provider ports.StaticPagesProvider
	logger   *zap.Logger
}

// NewPagesHandler creates a PagesHandler.
func NewPagesHandler(provider ports.StaticPagesProvider, logger *zap.Logger) *PagesHandler {
	return &PagesHandler{provider: provider, logger: logger}
}

// ListPages returns all static pages as JSON.
func (h *PagesHandler) ListPages(c echo.Context) error {
	if h.provider == nil {
		return c.JSON(http.StatusOK, []model.Page{})
	}
	pages, err := h.provider.ListPages(c.Request().Context())
	if err != nil {
		h.logger.Error("list pages", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list pages")
	}
	if pages == nil {
		pages = []model.Page{}
	}
	return c.JSON(http.StatusOK, pages)
}

// GetPage returns a single page as JSON.
func (h *PagesHandler) GetPage(c echo.Context) error {
	if h.provider == nil {
		return echo.NewHTTPError(http.StatusNotFound, "page not found")
	}
	slug := c.Param("slug")
	page, err := h.provider.GetPage(c.Request().Context(), slug)
	if err != nil {
		h.logger.Error("get page", zap.String("slug", slug), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get page")
	}
	if page == nil {
		return echo.NewHTTPError(http.StatusNotFound, "page not found")
	}
	return c.JSON(http.StatusOK, page)
}
