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
// heading is written with a leading backslash and read back without it.
package lessonmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

// headingLike reports a body line the parser would take for a heading, once
// leading backslashes are set aside.
func headingLike(l string) bool {
	t := strings.TrimLeft(l, `\`)
	return t == "#" || t == "##" || strings.HasPrefix(t, "# ") || strings.HasPrefix(t, "## ")
}

func fenceToggles(l string) bool {
	t := strings.TrimLeft(l, " ")
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
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
		fenced := false
		for _, l := range strings.Split(s, "\n") {
			if fenceToggles(l) {
				fenced = !fenced
			} else if !fenced && headingLike(l) {
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
	fenced := false
	for _, l := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		l = strings.TrimRight(l, "\r")
		if !fenced {
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
		if fenceToggles(l) {
			fenced = !fenced
		} else if !fenced && strings.HasPrefix(l, `\`) && headingLike(l) {
			l = l[1:]
		}
		lines = append(lines, l)
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

// Import creates one Lesson per document of src and returns their ids. A
// document that fails to be written leaves none of src's Lessons behind.
func Import(ctx context.Context, s storage.Stores, scope, src string) (ids []int64, err error) {
	docs, err := Parse(src)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			return
		}
		for _, id := range ids {
			err = errors.Join(err, s.Lessons().Delete(ctx, scope, id))
		}
		ids = nil
	}()
	for _, d := range docs {
		id, err := s.Lessons().Create(ctx, scope, d.Name, d.Description)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
		for _, st := range d.Steps {
			if _, err := s.Lessons().AddStep(ctx, scope, id, domain.LessonStep{Title: st.Title, Text: st.Text}); err != nil {
				return ids, err
			}
		}
	}
	return ids, nil
}
