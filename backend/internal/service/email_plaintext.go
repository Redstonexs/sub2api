package service

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// htmlToPlainText renders an HTML email body as its text/plain alternative.
//
// Every message this gateway sends used to go out as HTML only. That reads as a
// spam signal to most filters, and leaves nothing usable for clients that
// display the text part: notification previews, watch faces, screen readers and
// the plain-text views some corporate mail systems force. Deriving the text part
// from the HTML — rather than asking every call site to write one — keeps the
// two from drifting apart.
//
// Returns "" when the body yields no text, which buildSMTPMessage takes as the
// signal to send a single-part message rather than an empty alternative.
func htmlToPlainText(body string) string {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return ""
	}
	w := &plainTextWriter{}
	w.walk(doc)
	return collapsePlainText(w.b.String())
}

type plainTextWriter struct {
	b strings.Builder
	// pre tracks <pre> nesting: inside it, whitespace is content and collapsing
	// it would flatten every code block an announcement contains.
	pre int
}

// plainTextSkipped elements carry no reader-visible text.
var plainTextSkipped = map[atom.Atom]bool{
	atom.Head: true, atom.Style: true, atom.Script: true,
	atom.Noscript: true, atom.Title: true,
}

// plainTextBlocks are separated from their neighbours on both sides.
var plainTextBlocks = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Blockquote: true, atom.Pre: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
	atom.Ul: true, atom.Ol: true, atom.Table: true,
}

// plainTextRows only break *before* themselves. Closing them too would put a
// blank line between every list item and every table row once collapsePlainText
// reads the double break as a paragraph gap.
var plainTextRows = map[atom.Atom]bool{atom.Li: true, atom.Tr: true}

func (w *plainTextWriter) walk(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		if w.pre > 0 {
			_, _ = w.b.WriteString(n.Data)
		} else {
			_, _ = w.b.WriteString(collapseInlineSpace(n.Data))
		}
		return
	case html.CommentNode, html.DoctypeNode:
		return
	}

	if n.Type != html.ElementNode {
		w.children(n)
		return
	}

	if plainTextSkipped[n.DataAtom] || isHiddenForPlainText(n) {
		return
	}

	switch n.DataAtom {
	case atom.Br:
		_, _ = w.b.WriteString("\n")
		return
	case atom.Hr:
		_, _ = w.b.WriteString("\n--------\n")
		return
	case atom.Img:
		// The image itself cannot survive, but its alt text is the only description
		// a text-only reader gets of an announcement's screenshots.
		if alt := strings.TrimSpace(nodeAttr(n, "alt")); alt != "" {
			_, _ = w.b.WriteString("[")
			_, _ = w.b.WriteString(alt)
			_, _ = w.b.WriteString("]")
		}
		return
	case atom.A:
		w.link(n)
		return
	case atom.Td, atom.Th:
		w.children(n)
		if nextCell(n) != nil {
			_, _ = w.b.WriteString(" | ")
		}
		return
	}

	if plainTextBlocks[n.DataAtom] || plainTextRows[n.DataAtom] {
		_, _ = w.b.WriteString("\n")
	}
	if n.DataAtom == atom.Li {
		_, _ = w.b.WriteString("- ")
	}
	if n.DataAtom == atom.Pre {
		w.pre++
	}
	w.children(n)
	if n.DataAtom == atom.Pre {
		w.pre--
	}
	if plainTextBlocks[n.DataAtom] {
		_, _ = w.b.WriteString("\n")
	}
}

func (w *plainTextWriter) children(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.walk(c)
	}
}

// link renders an anchor as "label (url)" so the destination survives, which
// matters most for the one-click actions: reset password, recharge, unsubscribe.
func (w *plainTextWriter) link(n *html.Node) {
	inner := &plainTextWriter{pre: w.pre}
	inner.children(n)
	label := strings.TrimSpace(collapseInlineSpace(inner.b.String()))
	href := strings.TrimSpace(nodeAttr(n, "href"))

	switch {
	case href == "":
		_, _ = w.b.WriteString(label)
	case label == "" || label == href:
		_, _ = w.b.WriteString(href)
	default:
		_, _ = w.b.WriteString(label)
		_, _ = w.b.WriteString(" (")
		_, _ = w.b.WriteString(href)
		_, _ = w.b.WriteString(")")
	}
}

// isHiddenForPlainText matches the inline display:none the shell uses for its
// preheader — text meant for the inbox preview, not for the body.
func isHiddenForPlainText(n *html.Node) bool {
	style := strings.ToLower(nodeAttr(n, "style"))
	return strings.Contains(strings.ReplaceAll(style, " ", ""), "display:none")
}

func nodeAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// nextCell returns the following cell in the same row, if any.
func nextCell(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode && (s.DataAtom == atom.Td || s.DataAtom == atom.Th) {
			return s
		}
	}
	return nil
}

// collapseInlineSpace folds every whitespace run to one space, the way an HTML
// renderer would, so the source's own indentation does not reach the reader.
//
// A leading run collapses to a space rather than vanishing: text nodes are
// visited one at a time, and dropping it would weld the gap out of markup like
// "<strong>15</strong> minutes". collapsePlainText trims whatever ends up at the
// start of a line.
func collapseInlineSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\f', '\v', ' ':
			space = true
		default:
			if space {
				_ = b.WriteByte(' ')
			}
			space = false
			_, _ = b.WriteRune(r)
		}
	}
	if space {
		_ = b.WriteByte(' ')
	}
	return b.String()
}

// collapsePlainText trims trailing spaces per line and caps blank runs at one,
// turning the block markers emitted above into ordinary paragraph spacing.
func collapsePlainText(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			blank++
			if blank > 1 || len(out) == 0 {
				continue
			}
			out = append(out, "")
			continue
		}
		blank = 0
		out = append(out, strings.TrimLeft(line, " "))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
