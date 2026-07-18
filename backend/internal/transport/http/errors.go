// Package http provides REST API transport including middleware and error handling.
package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/usecase"
)

// ProblemJSON represents an RFC 7807 problem response.
type ProblemJSON struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

// writeProblem writes an RFC 7807 problem+json response.
func writeProblem(c echo.Context, status int, title, detail string) error {
	c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
	return c.JSON(status, ProblemJSON{
		Type:   "https://redpolitika.dev/errors/" + http.StatusText(status),
		Title:  title,
		Status: status,
		Detail: detail,
	})
}

// sanitizedDetail extracts a safe error message for RFC 7807 responses.
func sanitizedDetail(err error) string {
	var ucErr *usecase.Error
	if errors.As(err, &ucErr) {
		return ucErr.Message
	}
	var domErr *model.DomainError
	if errors.As(err, &domErr) {
		return domErr.Message
	}
	return "Internal Server Error"
}

// HTTPErrorHandler is a custom Echo error handler that returns RFC 7807 errors (A19/A32).
func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	if he, ok := err.(*echo.HTTPError); ok {
		var msg string
		switch m := he.Message.(type) {
		case string:
			msg = m
		case error:
			msg = m.Error()
		default:
			msg = fmt.Sprint(m)
		}
		writeProblem(c, he.Code, http.StatusText(he.Code), msg)
		return
	}

	writeProblem(c, http.StatusInternalServerError, "Internal Server Error", sanitizedDetail(err))
}
