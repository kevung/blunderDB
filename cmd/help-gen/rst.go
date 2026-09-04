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

// admonition is a .. note:: / .. warning:: / .. tip:: with its paragraphs.
type admonition struct {
	kind  string
	paras []string
}

type bulletList struct{ items []string }

type table struct {
	header []string
	rows   [][]string
}

func (section) isBlock()    {}
func (paragraph) isBlock()  {}
func (admonition) isBlock() {}
func (bulletList) isBlock() {}
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

	doc := &document{}
	// Section underline characters in order of first appearance, so the nesting
	// level of a title is the index of its underline character.
	var levels []string

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
		if i+1 < len(lines) && isUnderline(lines[i+1], len([]rune(trimmed))) {
			char := string(lines[i+1][0])
			level := indexOrAppend(&levels, char)
			doc.blocks = append(doc.blocks, section{level: level + 1, title: trimmed})
			i++
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
				doc.blocks = append(doc.blocks, t)
			case "note", "warning", "tip":
				paras := splitParagraphs(body)
				if inline != "" {
					// `.. note:: text` continues into the indented body.
					if len(paras) > 0 {
						paras[0] = inline + " " + paras[0]
					} else {
						paras = []string{inline}
					}
				}
				doc.blocks = append(doc.blocks, admonition{kind: name, paras: paras})
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
			doc.blocks = append(doc.blocks, bulletList{items: items})
			i = next - 1
			continue
		}

		// Anything else at column zero is an ordinary paragraph, running to the
		// next blank line.
		var para []string
		for ; i < len(lines) && strings.TrimSpace(lines[i]) != ""; i++ {
			para = append(para, strings.TrimSpace(lines[i]))
		}
		doc.blocks = append(doc.blocks, paragraph{text: strings.Join(para, " ")})
	}
	return doc, nil
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
