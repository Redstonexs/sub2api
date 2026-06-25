// Package mdhtml converts announcement Markdown into safe HTML for emails.
package mdhtml

import (
	"bytes"
	"html"
	"io"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

var (
	markdownRenderer = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(gmhtml.WithHardWraps()),
	)
	safeHTMLPolicy = newSafeHTMLPolicy()
	convertMarkdown = func(source []byte, output io.Writer) error {
		return markdownRenderer.Convert(source, output)
	}
)

// ToSafeHTML returns a sanitized HTML fragment suitable for email bodies.
func ToSafeHTML(content string) string {
	normalized := normalizeContent(content)
	if normalized == "" {
		return ""
	}

	var rendered bytes.Buffer
	if err := convertMarkdown([]byte(normalized), &rendered); err != nil {
		return fallbackHTML(normalized)
	}

	return safeHTMLPolicy.Sanitize(rendered.String())

}

func normalizeContent(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.TrimSpace(normalized)
}

func fallbackHTML(content string) string {
	escaped := html.EscapeString(content)
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")
	if escaped == "" {
		return ""
	}
	return "<p>" + escaped + "</p>"
}

func newSafeHTMLPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	policy.AllowElements("p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "strong", "em", "s", "code", "pre", "blockquote", "ul", "ol", "li", "table", "thead", "tbody", "tr", "th", "td", "hr", "a")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowStandardURLs()
	return policy
}
