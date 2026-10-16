package cli

import (
	"strings"
	"testing"
)

const testOpeningBook = "pkg/blunderdb/openingbook/testdata/book.jsonl"

func TestOpeningBook_EmptyEmbeddedBookSaysSo(t *testing.T) {
	cli := NewCLI()
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"openingbook", "question"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "empty") {
		t.Fatalf("an empty book must say so, got %q", out)
	}
}

func TestOpeningBook_QuestionAndAnswer(t *testing.T) {
	cli := NewCLI()
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"openingbook", "question", "--book", testOpeningBook, "--score", "2-4"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "play the opening 31, then the reply 21") || !strings.Contains(out, "XGID=") {
		t.Fatalf("question: %q", out)
	}
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"openingbook", "answer", "--book", testOpeningBook,
			"--opening", "31", "--reply", "21", "--score", "4-2", "13/10 6/5"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Wrong: 13/10 6/5 is 0.060 behind") || strings.Count(out, "tied") != 2 ||
		!strings.Contains(out, "after the book's best opening play at this score, 8/5 6/5") {
		t.Fatalf("opening answer: %q", out)
	}
	out = captureStdout(t, func() {
		if err := cli.Run([]string{"openingbook", "answer", "--book", testOpeningBook,
			"--opening", "31", "--reply", "21", "--score", "4-2", "--after", "8/5 6/5", "24/23 13/11"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Right: 24/23 13/11 is tied with the best") || strings.Contains(out, "Now reply") {
		t.Fatalf("reply answer: %q", out)
	}
	if err := cli.Run([]string{"openingbook", "answer", "--book", testOpeningBook,
		"--opening", "31", "--reply", "11", "--score", "4-2", "8/7"}); err == nil {
		t.Fatal("a line the book lacks must be refused")
	}
}
