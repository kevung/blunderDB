package cli

import (
	"strings"
	"testing"
)

// TestIsCommandCoversEveryHandler guards the seam between main.go's mode
// dispatch and this package: main.go asks IsCommand, so a subcommand wired into
// handlers() must be recognised there too. `vacuum` shipped once with a handler
// and no entry in main.go's hand-kept list, and `blunderdb vacuum` opened the
// GUI.
func TestIsCommandCoversEveryHandler(t *testing.T) {
	t.Parallel()
	for name := range (&CLI{}).handlers() {
		if !IsCommand(name) {
			t.Errorf("handlers() knows %q but IsCommand rejects it", name)
		}
		if !IsCommand(strings.ToUpper(name)) {
			t.Errorf("IsCommand(%q) must be case-insensitive", strings.ToUpper(name))
		}
	}
	for _, name := range []string{"", "serve", "call", "migrate", "nonsense"} {
		if IsCommand(name) {
			t.Errorf("IsCommand(%q) = true; it is not a CLI subcommand", name)
		}
	}
}

// TestUsageListsEveryCommand keeps `blunderdb help` honest: a subcommand the
// user cannot discover is as good as absent. Headless modes are dispatched in
// main.go, not here, so they are excluded from handlers() but still printed.
func TestUsageListsEveryCommand(t *testing.T) {
	usage := captureStdout(t, func() { (&CLI{}).printUsage() })
	for name := range (&CLI{}).handlers() {
		if !strings.Contains(usage, "  "+name+" ") {
			t.Errorf("printUsage does not list the %q command", name)
		}
	}
}
