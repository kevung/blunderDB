package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// A deliberately small reader for the subset of reStructuredText the two
// reference documents (raccourcis.rst, cmd_mode.rst) are written in: sections,
// paragraphs, bullet lists, csv-tables and the three admonitions. It is not a
// docutils replacement and does not try to be one — anything it does not
// recognise is skipped, and every string it does recognise is looked up in the
// gettext catalogues, so a construct that silently changed shape shows up as a
// missing translation rather than as a wrong rendering.

type block interface{ isBlock() }

// section is a titled division. level 1 is the document title (dropped: the
// help tab already carries the name), level 2 becomes an <h3>.
type section struct {
	level int
	title string
}

type paragraph struct{ text string }

// admonition is a .. note:: / .. tip:: / .. warning:: / .. important:: /
// .. caution:: with its body, parsed like any other: manuel.rst puts a
// formula and a bullet list inside one.
type admonition struct {
	kind   string
	blocks []block
}

// literal is a body reproduced verbatim — the one .. math:: in the
// documentation. The help modal has no formula renderer, so the source is
// shown as it is written rather than dropped along with the sentence that
// introduces it.
type literal struct {
	class string
	text  string
}

type bulletList struct{ items []string }

// blockquote is a body indented under the paragraph that introduces it — the
// shape manuel.rst uses for a term and its definition, and the only place a
// directive appears anywhere but column zero. Its content is parsed by the
// same reader, one level in.
type blockquote struct{ blocks []block }

type table struct {
	header []string
	rows   [][]string
}

func (section) isBlock()    {}
func (paragraph) isBlock()  {}
func (admonition) isBlock() {}
func (bulletList) isBlock() {}
func (blockquote) isBlock() {}
func (literal) isBlock()    {}
func (table) isBlock()      {}

// document is one parsed .rst file. Internal link targets are collected
// separately by scanTargets, which reads every documentation file — including
// the ones this reader cannot parse.
type document struct {
	blocks []block
}

var (
	targetRe    = regexp.MustCompile(`^\.\. _([A-Za-z0-9_.-]+):\s*$`)
	directiveRe = regexp.MustCompile(`^\.\. ([a-z-]+)::\s*(.*)$`)
	// A section underline: four or more copies of one punctuation character.
	underlineChars = `=-~^"'+*#`
)

func parseRST(path string) (*document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")

	// Section underline characters in order of first appearance, so the nesting
	// level of a title is the index of its underline character. The slice is
	// shared with nested calls: a level is a property of the document, not of
	// the indentation depth the reader happens to be at.
	var levels []string
	blocks, err := parseBlocks(lines, path, &levels)
	if err != nil {
		return nil, err
	}
	return &document{blocks: blocks}, nil
}

// parseBlocks reads a run of lines whose own content starts at column zero;
// collectIndented has already removed the common indent of a nested body.
func parseBlocks(lines []string, path string, levels *[]string) ([]block, error) {
	var blocks []block
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Hyperlink targets carry no rendered content; scanTargets reads them.
		if targetRe.MatchString(line) {
			continue
		}

		// A section title is a line whose successor is an underline of at least
		// the same length made of a single punctuation character.
		if !isIndented(line) && i+1 < len(lines) && isUnderline(lines[i+1], len([]rune(trimmed))) {
			char := string(lines[i+1][0])
			level := indexOrAppend(levels, char)
			blocks = append(blocks, section{level: level + 1, title: trimmed})
			i++
			continue
		}

		// An indented run opens a nested body, parsed by the same reader. It
		// is what carries manuel.rst's definitions — and, inside them, the
		// only csv-table and admonitions that do not start at column zero.
		if isIndented(line) {
			body, next := collectIndented(lines, i)
			sub, err := parseBlocks(body, path, levels)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, blockquote{blocks: sub})
			i = next - 1
			continue
		}

		if m := directiveRe.FindStringSubmatch(line); m != nil {
			name, inline := m[1], m[2]
			body, next := collectIndented(lines, i+1)
			switch name {
			case "csv-table":
				t, err := parseCSVTable(body)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: %w", path, i+1, err)
				}
				blocks = append(blocks, t)
			case "note", "warning", "tip", "important", "caution":
				// `.. note:: text` continues into the indented body, so the
				// inline part is read as that body's first line.
				if inline != "" {
					body = append([]string{inline}, body...)
				}
				sub, err := parseBlocks(body, path, levels)
				if err != nil {
					return nil, fmt.Errorf("%s:%d: %w", path, i+1, err)
				}
				blocks = append(blocks, admonition{kind: name, blocks: sub})
			case "math":
				blocks = append(blocks, literal{class: "math", text: strings.Join(trimBlank(body), "\n")})
			default:
				// Unknown directive (toctree, figure, …): skipped whole.
			}
			i = next - 1
			continue
		}

		// A bullet list runs until the first line that is neither a bullet nor
		// its continuation.
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			items, next := collectBullets(lines, i)
			blocks = append(blocks, bulletList{items: items})
			i = next - 1
			continue
		}

		// Anything else at column zero is an ordinary paragraph, running to the
		// next blank line.
		var para []string
		for ; i < len(lines) && strings.TrimSpace(lines[i]) != ""; i++ {
			// A paragraph never continues into a deeper indent: that is a
			// nested body, and gettext makes it a msgid of its own.
			if len(para) > 0 && isIndented(lines[i]) {
				break
			}
			para = append(para, strings.TrimSpace(lines[i]))
		}
		i--
		blocks = append(blocks, paragraph{text: strings.Join(para, " ")})
	}
	return blocks, nil
}

// trimBlank drops leading and trailing blank lines.
func trimBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// isIndented reports whether a line's content starts past column zero.
func isIndented(line string) bool {
	return strings.TrimSpace(line) != "" && line[0] == ' '
}

func isUnderline(line string, minLen int) bool {
	s := strings.TrimRight(line, " \t")
	if len([]rune(s)) < 4 || len([]rune(s)) < minLen {
		return false
	}
	if !strings.ContainsAny(s[:1], underlineChars) {
		return false
	}
	return strings.Count(s, s[:1]) == len(s)
}

func indexOrAppend(levels *[]string, char string) int {
	for i, c := range *levels {
		if c == char {
			return i
		}
	}
	*levels = append(*levels, char)
	return len(*levels) - 1
}

// collectIndented returns the dedented body of the directive starting at line
// `from`, and the index of the first line after it.
func collectIndented(lines []string, from int) ([]string, int) {
	i := from
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return nil, i
	}
	indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
	if indent == 0 {
		return nil, from
	}
	var body []string
	for ; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			body = append(body, "")
			continue
		}
		if len(lines[i])-len(strings.TrimLeft(lines[i], " ")) < indent {
			break
		}
		body = append(body, lines[i][indent:])
	}
	// Trailing blank lines belong to the document, not to the directive.
	for len(body) > 0 && body[len(body)-1] == "" {
		body = body[:len(body)-1]
		i--
	}
	return body, i
}

func collectBullets(lines []string, from int) ([]string, int) {
	var items []string
	var cur []string
	i := from
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		switch {
		case t == "":
			if i+1 < len(lines) && isBullet(lines[i+1]) {
				continue
			}
			goto done
		case isBullet(lines[i]):
			if len(cur) > 0 {
				items = append(items, strings.Join(cur, " "))
			}
			cur = []string{strings.TrimSpace(t[2:])}
		default:
			cur = append(cur, t)
		}
	}
done:
	if len(cur) > 0 {
		items = append(items, strings.Join(cur, " "))
	}
	return items, i
}

func isBullet(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ")
}

func splitParagraphs(body []string) []string {
	var out []string
	var cur []string
	for _, l := range body {
		if strings.TrimSpace(l) == "" {
			if len(cur) > 0 {
				out = append(out, strings.Join(cur, " "))
				cur = nil
			}
			continue
		}
		cur = append(cur, strings.TrimSpace(l))
	}
	if len(cur) > 0 {
		out = append(out, strings.Join(cur, " "))
	}
	return out
}

// parseCSVTable reads the `:header:` option and the data rows of a
// `.. csv-table::` directive body, the same way docutils does: each is a CSV
// record, so a comma inside a quoted cell is data.
func parseCSVTable(body []string) (table, error) {
	var t table
	i := 0
	for ; i < len(body); i++ {
		l := strings.TrimSpace(body[i])
		if l == "" {
			continue
		}
		if !strings.HasPrefix(l, ":") {
			break
		}
		if strings.HasPrefix(l, ":header:") {
			cells, err := readCSV(strings.TrimSpace(strings.TrimPrefix(l, ":header:")))
			if err != nil {
				return t, err
			}
			if len(cells) > 0 {
				t.header = cells[0]
			}
		}
	}
	rows, err := readCSV(strings.Join(body[i:], "\n"))
	if err != nil {
		return t, err
	}
	t.rows = rows
	if len(t.rows) == 0 {
		return t, fmt.Errorf("csv-table has no data rows")
	}
	return t, nil
}

func readCSV(s string) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(s))
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var out [][]string
	for _, rec := range recs {
		if len(rec) == 1 && strings.TrimSpace(rec[0]) == "" {
			continue
		}
		for i := range rec {
			rec[i] = strings.TrimSpace(rec[i])
		}
		out = append(out, rec)
	}
	return out, nil
}

// scanTargets collects the `.. _label:` → section-title map of a document
// without parsing its body. Cross-references may point into any of the sixteen
// documentation files, most of which use constructs this reader deliberately
// does not understand; resolving a reference only needs the titles.
func scanTargets(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	targets := map[string]string{}
	var pending []string
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if m := targetRe.FindStringSubmatch(lines[i]); m != nil {
			pending = append(pending, m[1])
			continue
		}
		// An overlined title (`====` / title / `====`) is the shape the
		// standalone chapters use for their own name — mode_headless.rst
		// among them, which manuel.rst cross-references. The overline is
		// not a title; step over it and read the line it introduces.
		if isUnderline(lines[i], 0) && i+2 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next != "" && isUnderline(lines[i+2], len([]rune(next))) {
				for _, label := range pending {
					targets[label] = next
				}
				pending = nil
				i += 2
				continue
			}
		}
		if i+1 < len(lines) && isUnderline(lines[i+1], len([]rune(trimmed))) {
			for _, label := range pending {
				targets[label] = trimmed
			}
			i++
		}
		pending = nil
	}
	return targets, nil
}
