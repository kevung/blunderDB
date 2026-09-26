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

	"github.com/kevung/blunderdb/internal/cli"
)

const (
	beginMarker = "<!-- BEGIN GENERATED CLI REFERENCE (cmd/cli-doc-gen; do not edit by hand, run `go run ./cmd/cli-doc-gen`) -->"
	endMarker   = "<!-- END GENERATED CLI REFERENCE -->"
	usageFile   = "CLI_USAGE.md"
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
	case "anki":
		return cli.NewCLI().AnkiSubcommands()
	case "bearoff":
		return cli.NewCLI().BearoffSubcommands()
	case "tournament":
		return cli.NewCLI().TournamentSubcommands()
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
	if err := os.WriteFile(usageFile, updated, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", usageFile, err)
	}
	fmt.Printf("cli-doc-gen: refreshed %d subcommand section(s) in %s\n", len(names), usageFile)
	return nil
}

func writeSection(out *bytes.Buffer, name, help string) {
	fmt.Fprintf(out, "### `blunderdb %s`\n\n```\n%s```\n\n", name, help)
}
