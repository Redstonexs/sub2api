// Package mdhtml converts announcement Markdown into safe HTML for emails.
package mdhtml

import (
	"bytes"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var (
	markdownRenderer = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithASTTransformers(
			util.Prioritized(externalImagesOnly{}, 100),
		)),
		goldmark.WithRendererOptions(gmhtml.WithHardWraps()),
	)
	safeHTMLPolicy  = newSafeHTMLPolicy()
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

// absoluteHTTPURL matches only absolute http(s) URLs. Announcement images must be
// externally hosted: this gateway has no image upload endpoint, and a relative or
// protocol-relative src resolves against the *mail client's* base, not ours.
var absoluteHTTPURL = regexp.MustCompile(`^https?://`)

// responsiveImageStyle is stamped on every surviving image.
//
// Externally hosted images arrive at their natural size, so a 1600px screenshot
// used to render 1600px wide and run off the side of the email card — visibly
// cut off on every phone. This has to be an *inline* style: the shell's
// stylesheet rule covers most clients, but the ones that strip <style> are
// exactly the ones where a cropped image is never recoverable.
//
// It is a fixed literal so newSafeHTMLPolicy can allow this exact value and
// nothing else, which keeps `style` from becoming a general-purpose CSS
// injection point on announcement content.
const responsiveImageStyle = "max-width:100%;height:auto"

// responsiveImageStylePattern must stay anchored: bluemonday matches attribute
// policies with an unanchored MatchString, so an unanchored pattern here would
// admit any style string that merely contained the literal.
var responsiveImageStylePattern = regexp.MustCompile(`^` + regexp.QuoteMeta(responsiveImageStyle) + `$`)

// externalImagesOnly drops image nodes whose destination is not an absolute
// http(s) URL and makes the survivors fluid, before rendering.
//
// Doing the drop in the parser rather than leaving it to bluemonday matters: the
// sanitizer can only strip the offending src attribute, and an element that still
// carries an allowed attribute (alt) survives as a src-less <img>, which mail
// clients draw as a broken-image placeholder. Removing the node emits nothing at
// all. bluemonday's src regex stays as the defense-in-depth backstop.
type externalImagesOnly struct{}

func (externalImagesOnly) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	var doomed []ast.Node
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}
		if !absoluteHTTPURL.Match(img.Destination) {
			doomed = append(doomed, n)
			return ast.WalkSkipChildren, nil
		}
		// goldmark renders image attributes through ImageAttributeFilter, which
		// extends GlobalAttributeFilter — "style" is in the latter, so this lands.
		img.SetAttributeString("style", []byte(responsiveImageStyle))
		return ast.WalkContinue, nil
	})
	// Mutating during the walk would invalidate the sibling links it follows.
	for _, n := range doomed {
		if parent := n.Parent(); parent != nil {
			parent.RemoveChild(parent, n)
		}
	}
}

// newSafeHTMLPolicy builds the sanitizer for announcement HTML.
//
// The element and attribute allowlist below is mirrored in the frontend at
// frontend/src/utils/markdown.ts so the admin preview and the delivered email
// agree; change both together.
func newSafeHTMLPolicy() *bluemonday.Policy {
	policy := bluemonday.NewPolicy()
	// "del" matters: goldmark's GFM strikethrough renders ~~text~~ as <del>, not
	// <s>, so without it strikethrough was silently flattened to plain text in every
	// broadcast email.
	policy.AllowElements("p", "br", "h1", "h2", "h3", "h4", "h5", "h6", "strong", "em", "s", "del", "code", "pre", "blockquote", "ul", "ol", "li", "table", "thead", "tbody", "tr", "th", "td", "hr", "a", "img")
	policy.AllowAttrs("href").OnElements("a")

	// Deliberately not bluemonday.AllowImages(): that registers a bare
	// AllowAttrs("src"), and AllowStandardURLs() below turns on AllowRelativeURLs,
	// so together they would admit <img src="/anything"> and <img src="//evil">.
	// The explicit regex keeps images to absolute http(s) only.
	policy.AllowAttrs("src").Matching(absoluteHTTPURL).OnElements("img")
	policy.AllowAttrs("alt").Matching(bluemonday.Paragraph).OnElements("img")
	policy.AllowAttrs("width", "height").Matching(bluemonday.NumberOrPercent).OnElements("img")
	// Only the one literal externalImagesOnly writes; see responsiveImageStyle.
	// Deliberately not policy.AllowStyles(): registering any style policy flips
	// bluemonday to its CSS-property sanitizer for *every* element, which is a far
	// wider surface than the single fixed declaration we actually need.
	policy.AllowAttrs("style").Matching(responsiveImageStylePattern).OnElements("img")

	policy.AllowStandardURLs()
	return policy
}
