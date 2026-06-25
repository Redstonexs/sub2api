package mdhtml

import (
	"strings"
	"testing"
)

func TestToSafeHTMLEscapesAndPreservesStructure(t *testing.T) {
	got := ToSafeHTML("First line\nSecond line\n\nNew paragraph <script>")

	if strings.Contains(got, "<script>") {
		t.Fatalf("expected raw HTML to be escaped, got %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("expected escaped script tag, got %q", got)
	}
	if !strings.Contains(got, "First line<br>Second line") {
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
	if !strings.Contains(got, "a<br>b") {
		t.Fatalf("expected CRLF normalized to <br>, got %q", got)
	}
}
