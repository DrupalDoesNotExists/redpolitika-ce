package plugin

import (
	"context"
	"fmt"

	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/model"
	"github.com/drupaldoesnotexists/redpolitika/ce/internal/domain/ports"
	pagespb "github.com/drupaldoesnotexists/redpolitika/ce/proto/pages"
)

// PagesAdapter implements ports.StaticPagesProvider via plugin gRPC.
type PagesAdapter struct {
	registry *Registry
}

// NewPagesAdapter creates a PagesAdapter.
// Plugin lookup deferred to call time — plugins load during OnStart, after construction.
func NewPagesAdapter(registry *Registry) ports.StaticPagesProvider {
	return &PagesAdapter{registry: registry}
}

func (a *PagesAdapter) ListPages(ctx context.Context) ([]model.Page, error) {
	plugins := a.registry.FindByCapability(CapPages)
	if len(plugins) == 0 {
		return nil, nil
	}
	client := pagespb.NewPagesServiceClient(plugins[0].Conn)
	resp, err := client.ListPages(ctx, &pagespb.ListPagesRequest{})
	if err != nil {
		return nil, fmt.Errorf("pages adapter: list: %w", err)
	}
	pages := make([]model.Page, 0, len(resp.Pages))
	for _, p := range resp.Pages {
		pages = append(pages, model.Page{
			Slug:            p.Slug,
			Title:           p.Title,
			ContentMarkdown: p.ContentMarkdown,
			Description:     p.Description,
			Language:        p.Language,
		})
	}
	return pages, nil
}

func (a *PagesAdapter) GetPage(ctx context.Context, slug string) (*model.Page, error) {
	plugins := a.registry.FindByCapability(CapPages)
	if len(plugins) == 0 {
		return nil, nil
	}
	client := pagespb.NewPagesServiceClient(plugins[0].Conn)
	resp, err := client.GetPage(ctx, &pagespb.GetPageRequest{Slug: slug})
	if err != nil {
		return nil, fmt.Errorf("pages adapter: get: %w", err)
	}
	if resp.Page == nil {
		return nil, nil
	}
	return &model.Page{
		Slug:            resp.Page.Slug,
		Title:           resp.Page.Title,
		ContentMarkdown: resp.Page.ContentMarkdown,
		Description:     resp.Page.Description,
		Language:        resp.Page.Language,
	}, nil
}

// Compile-time check.
var _ ports.StaticPagesProvider = (*PagesAdapter)(nil)
