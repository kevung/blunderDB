// Package lessonmd reads and writes a Lesson as Markdown, the plain text a
// player keeps in any editor. One file holds one or more documents:
//
//	# Name              the Lesson's name
//	text                its description (until the first section)
//	## Step title       one Step
//	step text
//
// An article, a Markdown text with no sections, is a Lesson with a
// description and no Steps. The Markdown carries text only: the collection or
// board a Step shows is not named in it, so an imported Step shows none.
// A heading inside a code fence is text; a body line that would read as a
// heading, or open a fence nothing closes, is written with a leading backslash
// and read back without it.
package lessonmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// Step is one section of a Doc.
type Step struct {
	Title string
	Text  string
}

// Doc is one Lesson (or article) in Markdown form.
type Doc struct {
	Name        string
	Description string
	Steps       []Step
}

func isTitle(l string) bool { return strings.HasPrefix(l, "# ") && strings.TrimSpace(l[2:]) != "" }
func isSection(l string) bool {
	return l == "##" || strings.HasPrefix(l, "## ")
}

// escapable reports a body line a backslash protects, once leading backslashes
// are set aside: one the parser would take for a heading or a fence.
func escapable(l string) bool {
	t := strings.TrimLeft(l, `\`)
	if t == "#" || t == "##" || strings.HasPrefix(t, "# ") || strings.HasPrefix(t, "## ") {
		return true
	}
	_, ok := openFence(t)
	return ok
}

// fence is an open code fence, as CommonMark reads one.
type fence struct {
	mark byte
	size int
}

// fenceRun splits a line that may be a fence into its run of marks and what
// follows. Four spaces of indent make it an indented code line instead.
func fenceRun(l string) (mark byte, size int, rest string) {
	t := strings.TrimLeft(l, " ")
	if len(l)-len(t) > 3 || t == "" || (t[0] != '`' && t[0] != '~') {
		return 0, 0, ""
	}
	mark = t[0]
	for size < len(t) && t[size] == mark {
		size++
	}
	return mark, size, t[size:]
}

// openFence reads a line that opens a fence. A backtick further on makes the
// line inline code: "```x``` text" opens nothing.
func openFence(l string) (fence, bool) {
	mark, size, rest := fenceRun(l)
	if size < 3 || (mark == '`' && strings.Contains(rest, "`")) {
		return fence{}, false
	}
	return fence{mark, size}, true
}

// closes reports a line that closes f: the same mark, at least as many, and
// nothing after them.
func (f fence) closes(l string) bool {
	mark, size, rest := fenceRun(l)
	return mark == f.mark && size >= f.size && strings.TrimRight(rest, " \t") == ""
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// Render writes the document.
func Render(d Doc) string {
	var b strings.Builder
	b.WriteString("# " + oneLine(d.Name) + "\n")
	body := func(s string) {
		if s == "" {
			return
		}
		b.WriteString("\n")
		ls := strings.Split(s, "\n")
		for i := 0; i < len(ls); i++ {
			l := ls[i]
			if f, ok := openFence(l); ok {
				end := i + 1
				for end < len(ls) && !f.closes(ls[end]) {
					end++
				}
				if end < len(ls) {
					b.WriteString(strings.Join(ls[i:end+1], "\n") + "\n")
					i = end
					continue
				}
				// Nothing closes it: as a fence it would take the next step for
				// code, and a closing line added here would come back as text.
			}
			if escapable(l) {
				l = `\` + l
			}
			b.WriteString(l + "\n")
		}
	}
	body(d.Description)
	for _, st := range d.Steps {
		b.WriteString("\n##")
		if t := oneLine(st.Title); t != "" {
			b.WriteString(" " + t)
		}
		b.WriteString("\n")
		body(st.Text)
	}
	return b.String()
}

// Parse reads every document of src. Only blank lines may precede the first
// title; a file with no title is ErrInvalid.
func Parse(src string) ([]Doc, error) {
	var docs []Doc
	var cur *Doc
	var step *Step
	var lines []string
	flush := func() {
		text := strings.Trim(strings.Join(lines, "\n"), "\n")
		lines = nil
		switch {
		case step != nil:
			step.Text = text
		case cur != nil:
			cur.Description = text
		}
	}
	closeStep := func() {
		flush()
		if step != nil {
			cur.Steps = append(cur.Steps, *step)
			step = nil
		}
	}
	closeDoc := func() {
		closeStep()
		if cur != nil {
			docs = append(docs, *cur)
			cur = nil
		}
	}
	var open *fence
	swallowed := false
	for _, l := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		l = strings.TrimRight(l, "\r")
		if open == nil {
			switch {
			case isTitle(l):
				closeDoc()
				cur = &Doc{Name: strings.TrimSpace(l[2:])}
				continue
			case cur != nil && isSection(l):
				closeStep()
				step = &Step{Title: strings.TrimSpace(strings.TrimPrefix(l, "##"))}
				continue
			}
		}
		if cur == nil {
			if strings.TrimSpace(l) != "" {
				return nil, fmt.Errorf("lessonmd: text before the first \"# title\" line: %w", storage.ErrInvalid)
			}
			continue
		}
		switch f, opens := openFence(l); {
		case open != nil:
			if open.closes(l) {
				open = nil
				swallowed = false
			} else if isTitle(l) || isSection(l) {
				swallowed = true
			}
		case opens:
			open = &f
		case strings.HasPrefix(l, `\`) && escapable(l):
			l = l[1:]
		}
		lines = append(lines, l)
	}
	if swallowed {
		// An open fence ends with the file, as in CommonMark; refused only when
		// it has taken what reads as a later step or lesson.
		return nil, fmt.Errorf("lessonmd: a code fence never closed takes the headings after it: %w", storage.ErrInvalid)
	}
	closeDoc()
	if len(docs) == 0 {
		return nil, fmt.Errorf("lessonmd: no \"# title\" line: %w", storage.ErrInvalid)
	}
	return docs, nil
}

// FromLesson is the document of a Lesson read from the store.
func FromLesson(l *domain.Lesson) Doc {
	d := Doc{Name: l.Name, Description: l.Description}
	for _, st := range l.Steps {
		d.Steps = append(d.Steps, Step{Title: st.Title, Text: st.Text})
	}
	return d
}

// Export writes the Lessons ids name, one document each, in that order.
func Export(ctx context.Context, s storage.Stores, scope string, ids ...int64) (string, error) {
	var parts []string
	for _, id := range ids {
		l, err := s.Lessons().Get(ctx, scope, id)
		if err != nil {
			return "", err
		}
		parts = append(parts, Render(FromLesson(l)))
	}
	return strings.Join(parts, "\n"), nil
}

// Import creates one Lesson per document of src and returns their ids, in one
// transaction: a document that fails to be written, or a connection lost
// half-way, leaves none of src's Lessons behind.
func Import(ctx context.Context, s storage.Stores, scope, src string) ([]int64, error) {
	docs, err := Parse(src)
	if err != nil {
		return nil, err
	}
	var ids []int64
	err = trash.InTx(ctx, s, scope, func(s storage.Stores) error {
		ids = nil
		for _, d := range docs {
			id, err := s.Lessons().Create(ctx, scope, d.Name, d.Description)
			if err != nil {
				return err
			}
			ids = append(ids, id)
			for _, st := range d.Steps {
				if _, err := s.Lessons().AddStep(ctx, scope, id, domain.LessonStep{Title: st.Title, Text: st.Text}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}
