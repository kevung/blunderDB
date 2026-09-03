package cli

import (
	"strings"
	"testing"
)

func TestCLI_CompletionBash(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"completion", "bash"}); err != nil {
			t.Fatalf("completion bash: %v", err)
		}
	})
	if !strings.Contains(out, "complete -F _blunderdb_completions blunderdb") {
		t.Errorf("bash completion script looks wrong:\n%s", out)
	}
	if !strings.Contains(out, " import ") {
		t.Errorf("bash completion script does not list the import command:\n%s", out)
	}
}

func TestCLI_CompletionZsh(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"completion", "zsh"}); err != nil {
			t.Fatalf("completion zsh: %v", err)
		}
	})
	if !strings.Contains(out, "#compdef blunderdb blunderDB") {
		t.Errorf("zsh completion script looks wrong:\n%s", out)
	}
}

func TestCLI_CompletionFish(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	out := captureStdout(t, func() {
		if err := cli.Run([]string{"completion", "fish"}); err != nil {
			t.Fatalf("completion fish: %v", err)
		}
	})
	if !strings.Contains(out, "complete -c blunderdb") {
		t.Errorf("fish completion script looks wrong:\n%s", out)
	}
}

func TestCLI_CompletionUnknownShell(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	if err := cli.Run([]string{"completion", "powershell"}); err == nil {
		t.Fatal("expected an error for an unsupported shell")
	}
}

func TestCLI_CompletionRequiresOneArg(t *testing.T) {
	cli := &CLI{db: NewDatabase()}
	if err := cli.Run([]string{"completion"}); err == nil {
		t.Fatal("expected an error when no shell name is given")
	}
	if err := cli.Run([]string{"completion", "bash", "zsh"}); err == nil {
		t.Fatal("expected an error when more than one shell name is given")
	}
}

// commandNames must stay in sync with handlers() by construction (it reads
// it directly), so every registered command, including completion itself,
// must show up in every generated script.
func TestCommandNames_IncludesEveryHandler(t *testing.T) {
	cli := &CLI{}
	names := cli.commandNames()
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	for name := range cli.handlers() {
		if !set[name] {
			t.Errorf("commandNames() is missing handler %q", name)
		}
	}
	for _, headless := range []string{"serve", "call", "migrate"} {
		if !set[headless] {
			t.Errorf("commandNames() is missing headless command %q", headless)
		}
	}
}
