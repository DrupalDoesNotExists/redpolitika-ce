package model

// Page — static content page from plugin (static.pages EP, Level 1).
// Slug is the URL path segment, Title is human-readable,
// ContentMarkdown is the Markdown body.
// Domain model — no JSON tags (use usecase.PageDTO for serialization).
type Page struct {
	Slug            string
	Title           string
	ContentMarkdown string
	Description     string
	Language        string
}
