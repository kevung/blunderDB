package lessonmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseSplitsTitleDescriptionAndSteps(t *testing.T) {
	docs, err := Parse("# Bear-off\n\nBasics.\n\nTwo lines.\n\n## Look\n\nfirst\n\n## Words\n\nsecond\n")
	if err != nil {
		t.Fatal(err)
	}
	want := []Doc{{Name: "Bear-off", Description: "Basics.\n\nTwo lines.", Steps: []Step{{"Look", "first"}, {"Words", "second"}}}}
	if !reflect.DeepEqual(docs, want) {
		t.Errorf("got %+v, want %+v", docs, want)
	}
}

func TestAnArticleWithoutSectionsIsALessonWithoutSteps(t *testing.T) {
	docs, err := Parse("# Why the gammon matters\n\nBody only.\n")
	if err != nil || len(docs) != 1 || docs[0].Description != "Body only." || len(docs[0].Steps) != 0 {
		t.Fatalf("got %+v, %v", docs, err)
	}
}

func TestSeveralArticlesInOneFile(t *testing.T) {
	docs, err := Parse("# One\n\na\n\n# Two\n\nb\n")
	if err != nil || len(docs) != 2 || docs[1].Name != "Two" {
		t.Fatalf("got %+v, %v", docs, err)
	}
}

func TestAHeadingInsideACodeFenceIsText(t *testing.T) {
	src := "# T\n\n## S\n\n```\n## not a step\n# not a title\n```\n"
	docs, err := Parse(src)
	if err != nil || len(docs) != 1 || len(docs[0].Steps) != 1 || !strings.Contains(docs[0].Steps[0].Text, "## not a step") {
		t.Fatalf("got %+v, %v", docs, err)
	}
}

func TestRefusesTextBeforeTheFirstTitle(t *testing.T) {
	for _, src := range []string{"", "\n\n", "no title\n# T\n"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("%q parsed", src)
		}
	}
}

// What Render writes, Parse reads back unchanged, including body lines that
// look like headings and an empty step title.
func TestRenderThenParseIsTheIdentity(t *testing.T) {
	in := Doc{
		Name:        "Title with # and `code`",
		Description: "intro\n\n# looks like a title\n\\# already escaped\n\n- a list\n- b",
		Steps: []Step{
			{"First", "text\n\n## looks like a step\n\n```\n## inside a fence\n```"},
			{"", "untitled step"},
			{"Empty", ""},
		},
	}
	docs, err := Parse(Render(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || !reflect.DeepEqual(docs[0], in) {
		t.Errorf("round trip changed it:\n got %+v\nwant %+v", docs, in)
	}
}

// A fence never closed would swallow every later step and lesson as text.
func TestRefusesAFenceNeverClosed(t *testing.T) {
	if _, err := Parse("# T\n\n## S\n\n```\ncode\n\n## Next\n\ntext\n\n# Other\n"); err == nil {
		t.Error("an open fence ran to the end of the file without a word")
	}
}

// What Render writes always parses back to the same number of documents and
// steps, even from a body that opens a fence and does not close it.
func TestRenderClosesAFenceItOpened(t *testing.T) {
	in := Doc{Name: "A", Description: "```\nopen", Steps: []Step{{"S", "next"}}}
	docs, err := Parse(Render(in) + "\n" + Render(Doc{Name: "B"}))
	if err != nil || len(docs) != 2 || len(docs[0].Steps) != 1 || docs[0].Steps[0].Text != "next" {
		t.Fatalf("got %+v, %v", docs, err)
	}
}
