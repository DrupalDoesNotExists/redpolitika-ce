package usecase

import (
	"context"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
)

// PageDTO — JSON-safe DTO for static pages.
type PageDTO struct {
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	Description     string `json:"description,omitempty"`
	Language        string `json:"language,omitempty"`
	Weight          int    `json:"weight,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
	IsIndex         bool   `json:"is_index,omitempty"`
}

// PagesUseCase provides static page operations.
type PagesUseCase struct {
	provider ports.StaticPagesProvider
}

// NewPagesUseCase creates a PagesUseCase.
func NewPagesUseCase(provider ports.StaticPagesProvider) *PagesUseCase {
	return &PagesUseCase{provider: provider}
}

// ListPages returns all static pages.
func (uc *PagesUseCase) ListPages(ctx context.Context) ([]PageDTO, error) {
	pages, err := uc.provider.ListPages(ctx)
	if err != nil {
		return nil, &Error{Op: "ListPages", Message: "list pages", Err: err}
	}
	if pages == nil {
		return []PageDTO{}, nil
	}
	return pagesToDTOs(pages), nil
}

// GetPage returns a single page by slug.
func (uc *PagesUseCase) GetPage(ctx context.Context, slug string) (*PageDTO, error) {
	page, err := uc.provider.GetPage(ctx, slug)
	if err != nil {
		return nil, &Error{Op: "GetPage", Message: "get page", Err: err}
	}
	if page == nil {
		return nil, nil
	}
	return pageToDTO(page), nil
}

func pageToDTO(p *model.Page) *PageDTO {
	return &PageDTO{
		Slug:            p.Slug,
		Title:           p.Title,
		ContentMarkdown: p.ContentMarkdown,
		Description:     p.Description,
		Language:        p.Language,
		Weight:          p.Weight,
		UpdatedAt:       p.UpdatedAt,
		IsIndex:         p.IsIndex,
	}
}

func pagesToDTOs(pages []model.Page) []PageDTO {
	dtos := make([]PageDTO, 0, len(pages))
	for i := range pages {
		dtos = append(dtos, *pageToDTO(&pages[i]))
	}
	return dtos
}
