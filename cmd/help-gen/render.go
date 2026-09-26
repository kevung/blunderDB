package main

import (
	"fmt"
	"regexp"
	"strings"
)

// translator turns a French source string into the target language. A missing
// translation is an error, never a silent fallback to French.
type translator struct {
	lang string
	// cats is searched in order: the document's own catalogue first, then the
	// catalogues of the documents an internal reference points into.
	cats []catalogue
}

func (tr translator) tr(s string) (string, error) {
	if tr.lang == sourceLang || strings.TrimSpace(s) == "" {
		return s, nil
	}
	for _, c := range tr.cats {
		if v, ok := c[s]; ok {
			return v, nil
		}
	}
	return "", fmt.Errorf("no %s translation for %q", tr.lang, s)
}

// escapeHTML escapes the characters that can end a text node (fragments carry
// no attributes). The corpus is injected with {@html}, so nothing may skip it.
func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// boundary marks an RST escaped space (`\ `), used by Japanese next to CJK: a
// word boundary for the emphasis patterns, removed from the output.
const boundary = ""

var (
	literalRe = regexp.MustCompile("``([^`]+)``")
	// Default-role interpreted text, rendered as a literal.
	interpretedRe = regexp.MustCompile("`([^`]+)`")
	strongRe      = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	// Emphasis with docutils' delimiter rules, using Unicode punctuation
	// classes (openers Ps/Pi/Pf, closers Pe/Pi/Pf, Pd/Po either side) plus
	// the boundary rune, so `*TAB*-Taste` or `、*着手の誤り*` render. Slightly
	// wider than docutils; Sphinx reports what it would reject.
	emRe = regexp.MustCompile(`(^|[\s<\p{Ps}\p{Pi}\p{Pf}\p{Pd}\p{Po}\x{E000}])\*([^*\s][^*]*)\*($|[\s>\p{Pe}\p{Pi}\p{Pf}\p{Pd}\p{Po}\x{E000}])`)
	// :ref:`text <label>` and :ref:`label`, matched after escaping, so the
	// angle brackets are already `&lt;`/`&gt;`.
	refLabelRe = regexp.MustCompile(":ref:`([^`]+?)&lt;([^`]+?)&gt;`")
	refBareRe  = regexp.MustCompile(":ref:`([^`]+)`")
	roleRe     = regexp.MustCompile(":[a-z]+:`([^`]+)`")
	// A backslash escapes the next character in reStructuredText.
	escapeRe = regexp.MustCompile(`\\(.)`)
)

// rstUnescape applies reStructuredText's backslash escapes: `\ ` becomes the
// zero-width boundary above, and `\x` becomes `x` for anything else.
func rstUnescape(s string) string {
	return escapeRe.ReplaceAllStringFunc(s, func(m string) string {
		if r := []rune(m)[1]; r == ' ' {
			return boundary
		}
		return string([]rune(m)[1:])
	})
}

// refuseEscapedBackslash rejects `\\` outside an inline literal: docutils
// prints a backslash, never wanted in prose. An error, not a repair: the fix
// belongs in the catalogue.
func refuseEscapedBackslash(s string) error {
	for _, m := range escapeRe.FindAllStringSubmatch(literalRe.ReplaceAllString(s, ""), -1) {
		if m[1] == `\` {
			return fmt.Errorf("escaped backslash renders as a visible \"\\\" in %q: an escaped space is written `\\\\ ` in a .po, not `\\\\\\\\ `", s)
		}
	}
	return nil
}

// inline renders RST inline markup as the help modal's HTML. The text is
// escaped first, so only the fixed tags written here can appear.
func (g *generator) inline(text string, tr translator) (string, error) {
	s := escapeHTML(text)

	// Cross-references resolve to the title of the section they point at, in
	// the target language: the help modal has nowhere to link to.
	var refErr error
	s = refLabelRe.ReplaceAllStringFunc(s, func(m string) string {
		return strings.TrimSpace(refLabelRe.FindStringSubmatch(m)[1])
	})
	s = refBareRe.ReplaceAllStringFunc(s, func(m string) string {
		label := refBareRe.FindStringSubmatch(m)[1]
		title, err := g.refTitle(label, tr)
		if err != nil {
			refErr = err
			return m
		}
		return escapeHTML(title)
	})
	if refErr != nil {
		return "", refErr
	}

	// Any interpreted-text role other than :ref: (:kbd:, :menuselection:, …)
	// renders as its own text rather than as unreadable markup.
	s = roleRe.ReplaceAllString(s, "$1")

	if err := refuseEscapedBackslash(s); err != nil {
		return "", err
	}
	s = rstUnescape(s)
	s = literalRe.ReplaceAllString(s, "<code>$1</code>")
	s = strongRe.ReplaceAllString(s, "<strong>$1</strong>")
	// Twice: the pattern consumes the delimiter that follows the closing `*`, so
	// two emphases separated by a single character (`*A*-*B*`) need a second pass.
	s = emRe.ReplaceAllString(s, "$1<em>$2</em>$3")
	s = emRe.ReplaceAllString(s, "$1<em>$2</em>$3")
	s = interpretedRe.ReplaceAllString(s, "<code>$1</code>")
	return strings.ReplaceAll(s, boundary, ""), nil
}

// renderTab renders one parsed document as the inner HTML of a help tab.
func (g *generator) renderTab(doc *document, tr translator) (string, error) {
	return g.renderBlocks(doc.blocks, tr)
}

// renderBlocks renders a run of blocks; a nested body re-enters it.
func (g *generator) renderBlocks(blocks []block, tr translator) (string, error) {
	var b strings.Builder
	write := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	for _, blk := range blocks {
		switch v := blk.(type) {
		case section:
			// Level 1 is the document title; the tab header already names it.
			if v.level == 1 {
				continue
			}
			title, err := tr.tr(v.title)
			if err != nil {
				return "", err
			}
			// h3, h4, h5 for the manual's three levels, floored at h5.
			level := min(v.level+1, 5)
			write("<h%d>%s</h%d>\n", level, escapeHTML(title), level)
		case paragraph:
			html, err := g.trInline(v.text, tr)
			if err != nil {
				return "", err
			}
			write("<p>%s</p>\n", html)
		case admonition:
			inner, err := g.renderBlocks(v.blocks, tr)
			if err != nil {
				return "", err
			}
			write("<div class=\"admonition %s\">\n%s</div>\n", v.kind, inner)
		case bulletList:
			write("<ul>\n")
			for _, it := range v.items {
				html, err := g.trInline(it, tr)
				if err != nil {
					return "", err
				}
				write("<li>%s</li>\n", html)
			}
			write("</ul>\n")
		case table:
			write("<table>\n<thead>\n<tr>\n")
			for _, h := range v.header {
				html, err := g.trInline(h, tr)
				if err != nil {
					return "", err
				}
				write("<th>%s</th>\n", html)
			}
			write("</tr>\n</thead>\n<tbody>\n")
			for _, row := range v.rows {
				write("<tr>\n")
				for _, cell := range row {
					html, err := g.trInline(cell, tr)
					if err != nil {
						return "", err
					}
					write("<td>%s</td>\n", html)
				}
				write("</tr>\n")
			}
			write("</tbody>\n</table>\n")
		case literal:
			text, err := tr.tr(v.text)
			if err != nil {
				return "", err
			}
			// Encode braces: `{i}` would read as an interpolation
			// placeholder, which help.safety.test.js refuses.
			braces := strings.NewReplacer("{", "&#123;", "}", "&#125;")
			write("<pre class=\"%s\">%s</pre>\n", v.class, braces.Replace(escapeHTML(text)))
		case blockquote:
			inner, err := g.renderBlocks(v.blocks, tr)
			if err != nil {
				return "", err
			}
			write("<blockquote>\n%s</blockquote>\n", inner)
		}
	}
	return b.String(), nil
}

// trInline translates a source string, then renders its inline markup.
func (g *generator) trInline(text string, tr translator) (string, error) {
	translated, err := tr.tr(text)
	if err != nil {
		return "", err
	}
	return g.inline(translated, tr)
}
