package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var shortcutCell = regexp.MustCompile(`(?m)^\s+"([^"]+)",\s*"`)

// keyNamesOf reads the `keyNames` section of a locale: the single place where a key is spelled
// for a language, shared by the toolbar tooltips and, here, by the shortcuts help.
func keyNamesOf(t *testing.T, root, lang string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "frontend", "src", "i18n", "locales", lang+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		KeyNames map[string]string `json:"keyNames"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.KeyNames
}

// TestShortcutKeysUseTheKeyNamesOfTheLocale checks that every key named in the first column of
// raccourcis.rst is spelled, in each language, as `keyNames` of that language spells it, so the
// help never says "PageUp" where the toolbar says "Page préc.".
func TestShortcutKeysUseTheKeyNamesOfTheLocale(t *testing.T) {
	root := repoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "doc", "source", "raccourcis.rst"))
	if err != nil {
		t.Fatal(err)
	}
	var cells []string
	for _, m := range shortcutCell.FindAllStringSubmatch(string(src), -1) {
		cells = append(cells, m[1])
	}
	if len(cells) < 50 {
		t.Fatalf("only %d shortcut cells found", len(cells))
	}

	fr := keyNamesOf(t, root, sourceLang)
	var ids []string
	for id, name := range fr {
		if id != "ctrl" && id != "alt" && id != "tab" && name != "" {
			ids = append(ids, id)
		}
	}
	// Longest names first: "Page préc." must be consumed before "Haut" or "Bas" can match in it.
	sort.Slice(ids, func(i, j int) bool { return len(fr[ids[i]]) > len(fr[ids[j]]) })
	boundary := func(name string) *regexp.Regexp {
		return regexp.MustCompile(`(?i)(^|[^\p{L}])` + regexp.QuoteMeta(name) + `([^\p{L}]|$)`)
	}
	// The French source must not carry the English spellings.
	english := regexp.MustCompile(`(?i)\b(PageUp|PageDown|Home|End|Shift|Delete|Escape|Enter)\b`)

	for _, cell := range cells {
		if english.MatchString(cell) {
			t.Errorf("fr: %q spells a key in English; keyNames.fr names it", cell)
		}
		var used []string
		rest := cell
		for _, id := range ids {
			re := boundary(fr[id])
			if re.MatchString(rest) {
				used = append(used, id)
				rest = re.ReplaceAllString(rest, "$1 $2")
			}
		}
		if len(used) == 0 {
			continue
		}
		for _, lang := range []string{"en", "de", "el", "es", "fi", "it", "ja", "ru"} {
			cat, err := loadCatalogue(filepath.Join(root, "doc", "source", "locale", lang, "LC_MESSAGES", "raccourcis.po"))
			if err != nil {
				t.Fatal(err)
			}
			tr, ok := cat[cell]
			if !ok {
				t.Errorf("%s: no translation of %q", lang, cell)
				continue
			}
			names := keyNamesOf(t, root, lang)
			for _, id := range used {
				if !strings.Contains(strings.ToLower(tr), strings.ToLower(names[id])) {
					t.Errorf("%s: %q is %q, which lacks keyNames.%s = %q", lang, cell, tr, id, names[id])
				}
			}
		}
	}
}
