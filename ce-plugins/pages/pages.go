package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pagespb "github.com/drupaldoesnotexists/redpolitika/ce/plugins/pages/proto/pages"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

// frontmatter holds optional YAML frontmatter fields from .md files.
type frontmatter struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Language    string `yaml:"language"`
}

// parseFrontmatter extracts YAML frontmatter (between --- and ---) from content.
// Returns the frontmatter struct and the remaining body (after the closing ---).
// If no frontmatter is found, returns empty frontmatter and original content.
func parseFrontmatter(content string) (fm frontmatter, body string) {
	const delimiter = "---\n"

	if !strings.HasPrefix(content, delimiter) {
		return fm, content
	}

	rest := content[len(delimiter):]
	endIdx := strings.Index(rest, delimiter)
	if endIdx == -1 {
		// Opening --- but no closing ---; treat everything as content.
		return fm, content
	}

	yamlBlock := rest[:endIdx]
	body = rest[endIdx+len(delimiter):]

	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		// Invalid YAML — ignore frontmatter and return original content.
		return fm, content
	}

	return fm, body
}

// pagesService implements PagesServiceServer by reading .md files from disk.
type pagesService struct {
	pagespb.UnimplementedPagesServiceServer
}

// ListPages scans pagesDir recursively for *.md files and returns their slug + metadata.
func (s *pagesService) ListPages(_ context.Context, _ *pagespb.ListPagesRequest) (*pagespb.ListPagesResponse, error) {
	var pages []*pagespb.Page

	err := filepath.WalkDir(pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // skip dirs, recurse into them
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		// Compute relative slug from pagesDir: "docs/page-1" from "pagesDir/docs/page-1.md"
		rel, err := filepath.Rel(pagesDir, path)
		if err != nil {
			return err
		}
		slug := strings.TrimSuffix(rel, ".md")
		title, desc, lang := pageMetadata(path, slug)
		pages = append(pages, &pagespb.Page{
			Slug:        slug,
			Title:       title,
			Description: desc,
			Language:    lang,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk pages dir %q: %w", pagesDir, err)
	}

	if pages == nil {
		pages = []*pagespb.Page{}
	}

	return &pagespb.ListPagesResponse{Pages: pages}, nil
}

// GetPage reads slug.md from pagesDir and returns full content.
func (s *pagesService) GetPage(_ context.Context, req *pagespb.GetPageRequest) (*pagespb.GetPageResponse, error) {
	// Sanitize slug — deny path separators and traversal.
	if strings.Contains(req.Slug, "..") {
		return nil, fmt.Errorf("invalid slug: %q", req.Slug)
	}
	slug := req.Slug

	path := filepath.Join(pagesDir, slug) + ".md"

	// Resolve to absolute paths to verify the result stays under pagesDir.
	absPagesDir, err := filepath.Abs(pagesDir)
	if err != nil {
		return nil, fmt.Errorf("resolve pages dir: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve page path: %w", err)
	}
	if !strings.HasPrefix(absPath, absPagesDir) {
		return nil, fmt.Errorf("access denied: %q", req.Slug)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("page not found: %q", req.Slug))
		}
		return nil, fmt.Errorf("read page %q: %w", req.Slug, err)
	}

	content := string(data)
	fm, body := parseFrontmatter(content)
	title := fm.Title
	if title == "" {
		title = fallbackTitle(body, slug)
	}

	return &pagespb.GetPageResponse{
		Page: &pagespb.Page{
			Slug:            slug,
			Title:           title,
			Description:     fm.Description,
			Language:        fm.Language,
			ContentMarkdown: body,
		},
	}, nil
}

// pageMetadata reads a .md file and returns its title, description, and language.
// Title priority: frontmatter > first # heading > slug.
func pageMetadata(path, slug string) (title, description, language string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return slug, "", ""
	}

	fm, body := parseFrontmatter(string(data))
	title = fm.Title
	if title == "" {
		title = fallbackTitle(body, slug)
	}
	return title, fm.Description, fm.Language
}

// fallbackTitle returns the first `# ` heading from content, or fallback.
func fallbackTitle(content, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimPrefix(trimmed, "# ")
		}
	}
	return fallback
}
