package mdhtml

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestToSafeHTMLRendersMarkdown(t *testing.T) {
	got := ToSafeHTML("# Heading\n\n**bold** and [link](https://example.com)\n\n> quote\n\nline one\nline two")

	if !strings.Contains(got, "<h1>") {
		t.Fatalf("expected heading markdown to render as <h1>, got %q", got)
	}
	if !strings.Contains(got, "<strong>") {
		t.Fatalf("expected bold markdown to render as <strong>, got %q", got)
	}
	if !strings.Contains(got, `<a href="https://example.com"`) {
		t.Fatalf("expected link markdown to render as <a href=\"https://example.com\">, got %q", got)
	}
	if !strings.Contains(got, "<blockquote>") {
		t.Fatalf("expected blockquote markdown to render as <blockquote>, got %q", got)
	}
	if !strings.Contains(got, "line one<br>") || !strings.Contains(got, "line two") {
		t.Fatalf("expected single newline to render as <br> inside paragraphs, got %q", got)
	}
	if !strings.Contains(got, "<p>") {
		t.Fatalf("expected paragraph output, got %q", got)
	}
}

func TestToSafeHTMLSanitizesDangerousMarkdownHTML(t *testing.T) {
	got := ToSafeHTML("<script>alert(1)</script>\n\n<a href=\"javascript:alert(1)\" onclick=\"alert(2)\">x</a>")

	if strings.Contains(got, "<script>") {
		t.Fatalf("expected script tags to be removed, got %q", got)
	}
	if strings.Contains(got, "javascript:") {
		t.Fatalf("expected javascript: URLs to be removed, got %q", got)
	}
	if strings.Contains(got, "onclick") {
		t.Fatalf("expected event handler attributes to be removed, got %q", got)
	}
}

func TestToSafeHTMLAllowsExternalImages(t *testing.T) {
	got := ToSafeHTML(`![logo](https://cdn.example.com/logo.png "title")`)

	if !strings.Contains(got, `<img src="https://cdn.example.com/logo.png"`) {
		t.Fatalf("expected an absolute https image to survive sanitization, got %q", got)
	}
	if !strings.Contains(got, `alt="logo"`) {
		t.Fatalf("expected alt text to be preserved, got %q", got)
	}
}

func TestToSafeHTMLRejectsNonHTTPImageSources(t *testing.T) {
	cases := map[string]string{
		"data URI":          `![x](data:image/png;base64,iVBORw0KGgo=)`,
		"relative path":     `![x](/uploads/local.png)`,
		"protocol relative": `![x](//evil.example.com/tracker.gif)`,
		"javascript":        `![x](javascript:alert(1))`,
	}
	for name, in := range cases {
		got := ToSafeHTML(in)
		if strings.Contains(got, "<img") {
			t.Fatalf("%s: expected image to be dropped, got %q", name, got)
		}
	}
}

// TestToSafeHTMLDropsLiteralImageHTML documents a deliberate asymmetry: goldmark is
// built without WithUnsafe, so raw HTML never reaches bluemonday. Markdown image
// syntax renders; a hand-written <img> tag does not, even with an allowed src.
// frontend/src/utils/markdown.ts models this with a no-op raw-HTML renderer so the
// admin's email preview shows the same thing.
func TestToSafeHTMLDropsLiteralImageHTML(t *testing.T) {
	got := ToSafeHTML(`<img src="https://cdn.example.com/logo.png" alt="logo">`)

	if strings.Contains(got, "<img") {
		t.Fatalf("expected literal HTML <img> to be dropped by goldmark, got %q", got)
	}
}

func TestToSafeHTMLEscapesAndPreservesStructure(t *testing.T) {
	got := ToSafeHTML("First line\nSecond line\n\nNew paragraph <script>")

	if strings.Contains(got, "<script>") {
		t.Fatalf("expected raw HTML to be escaped, got %q", got)
	}
	if !strings.Contains(got, "First line<br>\nSecond line") {
		t.Fatalf("expected single newline to become <br>, got %q", got)
	}
	if strings.Count(got, "<p>") != 2 {
		t.Fatalf("expected 2 paragraphs, got %q", got)
	}
}

func TestToSafeHTMLEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\n", "\r\n  \r\n"} {
		if got := ToSafeHTML(in); got != "" {
			t.Fatalf("expected empty output for %q, got %q", in, got)
		}
	}
}

func TestToSafeHTMLNormalizesCRLF(t *testing.T) {
	got := ToSafeHTML("a\r\nb")
	if !strings.Contains(got, "a<br>") || !strings.Contains(got, "b") {
		t.Fatalf("expected CRLF normalized to <br>, got %q", got)
	}
}

func TestToSafeHTMLAllowsRelativeAndMailtoLinks(t *testing.T) {
	got := ToSafeHTML("[home](/docs) [mail](mailto:test@example.com)")

	if !strings.Contains(got, `<a href="/docs"`) {
		t.Fatalf("expected relative links to be allowed, got %q", got)
	}
	if !strings.Contains(got, `<a href="mailto:test@example.com"`) {
		t.Fatalf("expected mailto links to be allowed, got %q", got)
	}
}

func TestToSafeHTMLEscapesRawHTMLWhenMarkdownFails(t *testing.T) {
	previous := convertMarkdown
	convertMarkdown = func([]byte, io.Writer) error {
		return errors.New("boom")
	}
	t.Cleanup(func() {
		convertMarkdown = previous
	})

	got := ToSafeHTML("<script>alert(1)</script>")

	if got != "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>" {
		t.Fatalf("expected escaped raw html fallback, got %q", got)
	}
	if strings.Contains(got, "<script>") {
		t.Fatalf("expected escaped raw html fallback, got %q", got)
	}
}

func TestToSafeHTMLKeepsStrikethrough(t *testing.T) {
	// goldmark's GFM strikethrough renders <del>, not <s>; the policy allowlist must
	// carry both or ~~text~~ silently flattens to plain text in emails.
	got := ToSafeHTML("~~gone~~")

	if !strings.Contains(got, "<del>gone</del>") {
		t.Fatalf("expected strikethrough to survive sanitization, got %q", got)
	}
}
