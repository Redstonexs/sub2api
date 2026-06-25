// Package mdhtml converts admin/user-authored announcement text into a safe HTML
// fragment for embedding in notification emails.
//
// The current implementation HTML-escapes all input (so it is injection-safe even
// for untrusted content) and preserves the author's line structure: blank-line
// separated blocks become <p> paragraphs and single newlines within a block become
// <br>. It intentionally does NOT interpret Markdown syntax (headings, lists, links,
// emphasis) — doing that safely requires a Markdown parser plus an HTML sanitizer.
//
// This package is the single place to plug in a real Markdown renderer (e.g.
// goldmark + bluemonday) later: swap the body of ToSafeHTML and every caller keeps
// working unchanged.
package mdhtml

import (
	"html"
	"strings"
)

// ToSafeHTML returns an HTML fragment (a sequence of <p>…</p> blocks) that is safe
// to interpolate into an email body. The returned string is already HTML-escaped.
// An empty or whitespace-only input yields an empty string.
func ToSafeHTML(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return ""
	}

	var b strings.Builder
	for _, block := range strings.Split(normalized, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		escaped := html.EscapeString(block)
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		b.WriteString("<p>")
		b.WriteString(escaped)
		b.WriteString("</p>")
	}
	return b.String()
}
