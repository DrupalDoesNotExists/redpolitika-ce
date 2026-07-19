package model

// Page — static content page from plugin (static.pages EP, Level 1).
// Slug is the URL path segment, Title is human-readable,
// ContentMarkdown is the Markdown body.
type Page struct {
	Slug            string
	Title           string
	ContentMarkdown string
	Description     string
	Language        string
}
