package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// catalogue is one gettext catalogue: msgid → msgstr, for a single document in
// a single language. Only the two forms Sphinx emits are read (single-line and
// continuation-line strings); plurals and contexts do not occur in the
// documentation catalogues.
type catalogue map[string]string

// loadCatalogue reads a .po file. An entry whose msgstr is empty is dropped:
// gettext's own semantics for "untranslated", and the caller must then fail
// loudly rather than emit the French string under a foreign language.
func loadCatalogue(path string) (catalogue, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	cat := catalogue{}
	var id, str strings.Builder
	inStr := false
	flush := func() {
		if id.Len() > 0 && str.Len() > 0 {
			cat[id.String()] = str.String()
		}
		id.Reset()
		str.Reset()
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		switch {
		case strings.HasPrefix(line, "msgid "):
			flush()
			inStr = false
			s, err := unquotePO(line[len("msgid "):])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			id.WriteString(s)
		case strings.HasPrefix(line, "msgstr "):
			inStr = true
			s, err := unquotePO(line[len("msgstr "):])
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			str.WriteString(s)
		case strings.HasPrefix(line, `"`):
			s, err := unquotePO(line)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			if inStr {
				str.WriteString(s)
			} else {
				id.WriteString(s)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	flush()
	return cat, nil
}

// unquotePO decodes one C-style quoted string as written in a .po file.
func unquotePO(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, `"`) || !strings.HasSuffix(s, `"`) || len(s) < 2 {
		return "", fmt.Errorf("malformed .po string: %q", s)
	}
	return strconv.Unquote(s)
}
