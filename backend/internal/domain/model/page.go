package model

// Page — static content page from plugin (static.pages EP, Level 1).
// Slug is the URL path segment, Title is human-readable,
// ContentMarkdown is the Markdown body.
type Page struct {
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	Description     string `json:"description,omitempty"`
	Language        string `json:"language,omitempty"`
}
