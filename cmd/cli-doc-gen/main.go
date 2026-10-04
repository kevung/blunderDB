// Command cli-doc-gen captures every CLI subcommand's `--help` into
// CLI_USAGE.md between the generated markers; the hand-written prose above
// is untouched.
//
// Usage — from the repo root:
//
//	go run ./cmd/cli-doc-gen
//
// It walks cli.CommandNames() and the composite commands' sub-command tables,
// running each in-process with --help. Rerun it after changing a
// subcommand's flags and commit the CLI_USAGE.md diff.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/kevung/blunderdb/internal/cli"
)

const (
	beginMarker = "<!-- BEGIN GENERATED CLI REFERENCE (cmd/cli-doc-gen; do not edit by hand, run `go run ./cmd/cli-doc-gen`) -->"
	endMarker   = "<!-- END GENERATED CLI REFERENCE -->"
	usageFile   = "CLI_USAGE.md"

	listBegin = "<!-- BEGIN GENERATED COMMAND LIST (cmd/cli-doc-gen; do not edit by hand, run `go run ./cmd/cli-doc-gen`) -->"
	listEnd   = "<!-- END GENERATED COMMAND LIST -->"
)

// skip lists top-level commands with no flags of their own: capturing their
// --help would just repeat the general usage banner.
var skip = map[string]bool{"help": true, "version": true}

// composite returns the sub-command names of a command whose flags live one
// level down (`collection <sub>`, …).
func composite(name string) []string {
	switch name {
	case "collection":
		return cli.NewCLI().CollectionSubcommands()
	case "comment":
		return cli.NewCLI().CommentSubcommands()
	case "anki":
		return cli.NewCLI().AnkiSubcommands()
	case "lesson":
		return cli.NewCLI().LessonSubcommands()
	case "bearoff":
		return cli.NewCLI().BearoffSubcommands()
	case "tournament":
		return cli.NewCLI().TournamentSubcommands()
	case "stats":
		return cli.NewCLI().StatsSubcommands()
	case "training":
		return cli.NewCLI().TrainingSubcommands()
	default:
		return nil
	}
}

// captureHelp runs args in-process and returns what Usage() printed; --help
// is parsed before any required flag, so no database is needed. Both streams
// are captured: the banner goes to stdout, PrintDefaults to stderr.
func captureHelp(args []string) string {
	r, w, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cli-doc-gen: pipe: %v\n", err)
		os.Exit(1)
	}
	savedOut, savedErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r) //nolint:errcheck // best-effort capture of generated doc text
		done <- buf.String()
	}()

	// The error is expected (flag.ErrHelp); the text is already printed.
	_ = cli.NewCLI().Run(args)

	w.Close()
	os.Stdout, os.Stderr = savedOut, savedErr
	return <-done
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "cli-doc-gen: %v\n", err)
		os.Exit(1)
	}
}

// commandList renders the "Available Commands" list. The names come from
// cli.CommandNames (the dispatch table); each description is read from the
// banner `blunderdb help` prints, in the banner's order. A command with no
// line in the banner is an error: the list must never lag the table.
func commandList() (string, error) {
	desc := map[string]string{}
	var order []string
	for _, line := range strings.Split(captureHelp([]string{"help"}), "\n") {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			continue
		}
		name, d, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}
		if _, dup := desc[name]; !dup {
			order = append(order, name)
		}
		desc[name] = strings.TrimSpace(d)
	}
	var b strings.Builder
	b.WriteString(listBegin + "\n\n")
	seen := map[string]bool{}
	for _, name := range order {
		if !cli.IsCommand(name) {
			continue
		}
		seen[name] = true
		fmt.Fprintf(&b, "- `%s` - %s\n", name, desc[name])
	}
	for _, name := range cli.CommandNames() {
		if !seen[name] {
			return "", fmt.Errorf("command %q is dispatched but absent from the help banner", name)
		}
	}
	b.WriteString("\n" + listEnd)
	return b.String(), nil
}

func run() error {
	var out bytes.Buffer
	out.WriteString(beginMarker + "\n\n")
	out.WriteString("Captured verbatim from each subcommand's `--help`. Regenerate with " +
		"`go run ./cmd/cli-doc-gen` whenever a flag changes; the prose and\n" +
		"examples above are hand-written and this section never rewrites them.\n\n")

	names := cli.CommandNames()
	for _, name := range names {
		if skip[name] {
			continue
		}
		if subs := composite(name); subs != nil {
			for _, sub := range subs {
				writeSection(&out, fmt.Sprintf("%s %s", name, sub),
					captureHelp([]string{name, sub, "--help"}))
			}
			continue
		}
		writeSection(&out, name, captureHelp([]string{name, "--help"}))
	}
	out.WriteString(endMarker)

	generated := out.String()

	content, err := os.ReadFile(usageFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", usageFile, err)
	}
	re := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(beginMarker) + `.*` + regexp.QuoteMeta(endMarker))
	if !re.Match(content) {
		return fmt.Errorf("%s: markers not found (expected %q and %q)", usageFile, beginMarker, endMarker)
	}
	updated := re.ReplaceAllLiteral(content, []byte(generated))
	list, err := commandList()
	if err != nil {
		return err
	}
	lre := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(listBegin) + `.*` + regexp.QuoteMeta(listEnd))
	if !lre.Match(updated) {
		return fmt.Errorf("%s: markers not found (expected %q and %q)", usageFile, listBegin, listEnd)
	}
	updated = lre.ReplaceAllLiteral(updated, []byte(list))
	if err := os.WriteFile(usageFile, updated, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", usageFile, err)
	}
	fmt.Printf("cli-doc-gen: refreshed %d subcommand section(s) in %s\n", len(names), usageFile)
	return nil
}

func writeSection(out *bytes.Buffer, name, help string) {
	fmt.Fprintf(out, "### `blunderdb %s`\n\n```\n%s```\n\n", name, help)
}
