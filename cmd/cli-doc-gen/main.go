// Command cli-doc-gen captures the `--help` text of every CLI subcommand
// (internal/cli, driven straight off each command's flag.FlagSet) and writes
// it into CLI_USAGE.md between the generated markers, as a mechanical
// skeleton that cannot drift from the flags a command actually accepts — the
// hand-written prose and examples in the sections above it are untouched.
//
// Usage — from the repo root:
//
//	go run ./cmd/cli-doc-gen
//
// It walks cli.CommandNames() (the exported, sorted view of the same
// handlers() table main.go's mode dispatch trusts), plus the two composite
// commands' own sub-command tables (cli.CollectionSubcommands(),
// cli.AnkiSubcommands()), invoking each in-process with a trailing --help
// and capturing what its Usage() prints to stdout. `help` and `version` are
// skipped: neither takes flags, so there is nothing here for them to drift
// on.
//
// Run it after adding, renaming or re-flagging a CLI subcommand, review the
// diff in CLI_USAGE.md, and commit both — the same discipline as `go
// generate`, just not wired to it because the output lands inside a
// hand-maintained file rather than a dedicated one.
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

// composite maps a top-level command to the accessor for its own
// sub-command names, for the two commands whose flags live one level down
// (`collection <sub>`, `anki <sub>`) rather than on the top-level command
// itself.
func composite(name string) []string {
	switch name {
	case "collection":
		return cli.NewCLI().CollectionSubcommands()
	case "anki":
		return cli.NewCLI().AnkiSubcommands()
	default:
		return nil
	}
}

// captureHelp runs args through a fresh CLI in-process and returns whatever
// its Usage() printed. Every subcommand's Parse(args) sees --help before any
// required flag is checked, so this never needs a database. Both stdout and
// stderr are captured: the hand-written banner goes through fmt.Println
// (stdout), but flag.FlagSet.PrintDefaults() writes to the FlagSet's default
// Output(), which is os.Stderr unless a command explicitly redialed it —
// none do, so the flag list itself only ever shows up on stderr.
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

	// Errors are expected here: --help makes every Parse return flag.ErrHelp
	// (or a missing --db error after printing Usage()); the text already
	// reached stdout/stderr by the time Run returns.
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
