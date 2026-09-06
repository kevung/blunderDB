package main

import (
	"fmt"
	"regexp"
	"strings"
)

// translator turns a French source string into the target language. It is the
// only place the gettext catalogues are consulted; missing is an error, never a
// silent fallback to French (that fallback is exactly the failure mode this
// generator exists to remove).
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

// escapeHTML escapes the three characters that can end a text node. The
// generated fragments carry no attributes, so quote escaping is not needed —
// and the help corpus is injected with {@html}, which is why nothing reaches
// the renderer unescaped (see the safety test in frontend/src/__tests__).
func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// boundary marks where a reStructuredText escaped space (`\ `) stood. Japanese
// translations use it to separate inline markup from adjacent CJK text, where a
// real space would be wrong; it must still act as a word boundary while the
// emphasis patterns run, and disappear from the output afterwards.
const boundary = ""

var (
	literalRe = regexp.MustCompile("``([^`]+)``")
	// Interpreted text with the default role, used in the documentation the way
	// a literal is; the help modal has no title-reference styling to offer.
	interpretedRe = regexp.MustCompile("`([^`]+)`")
	strongRe      = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	// Emphasis, with docutils' own delimiter rules: the opening `*` must follow
	// the start of the text, whitespace or an opening punctuation mark, and the
	// closing `*` must precede the end, whitespace or a closing one — which is
	// what makes the German `*TAB*-Taste` and the Finnish `*Historique*-välilehti`
	// emphasis rather than literal asterisks. The private-use boundary rune
	// (an escaped space, see above) counts on both sides.
	emRe = regexp.MustCompile(`(^|[\s\-:/'"<(\[{\x{00AB}\x{FF08}\x{2018}\x{201C}\x{E000}])\*([^*\s][^*]*)\*($|[\s\-.,;:!?\\/'")\]}>\x{00BB}\x{FF09}\x{3001}\x{3002}\x{2019}\x{201D}\x{E000}])`)
	// :ref:`text <label>` and :ref:`label`. inline() escapes before it
	// substitutes, so by the time these run the angle brackets of the
	// explicit-title form are already `&lt;`/`&gt;` — matching the raw `<`
	// here silently sent `text &lt;label&gt;` to refBareRe as if it were a
	// label, and every explicit-title reference in manuel.rst failed to
	// resolve.
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

// inline renders reStructuredText inline markup as the small HTML vocabulary
// the help modal styles. The text is escaped FIRST and the markup applied
// afterwards, so no fragment of the source can inject an element: escaping
// introduces none of the characters the patterns below look for, and the
// patterns only ever emit the fixed tags written here.
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
			write("<h3>%s</h3>\n", escapeHTML(title))
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
			// LaTeX is made of braces, and `{i}` reads exactly like the
			// `{appVersion}` placeholder the About tab interpolates. Encode
			// them: the rendering is identical, and help.safety.test.js can
			// keep refusing every brace outside that one tab.
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
