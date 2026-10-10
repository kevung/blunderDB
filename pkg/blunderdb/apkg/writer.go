package apkg

import (
	"archive/zip"
	"context"
	"crypto/sha1" // Anki defines its checksum column as SHA-1; no security rests on it
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // the "sqlite" driver the collection file is written with
)

// ModelID identifies blunderDB's note type in every package it writes. Anki
// matches an imported note to the one it holds by guid AND note type, so the id
// never changes: a re-export must land on the same note type to update notes.
const ModelID int64 = 1760000000627

// ModelName is the note type's name in Anki's browser.
const ModelName = "blunderDB"

// Fields are the note type's fields, in order. The first is the sort field
// Anki shows in its browser and must never be empty.
var Fields = []string{"Position", "Board", "Situation", "Answer", "Equity", "Played"}

// Note is one note of the package: one position, one card.
type Note struct {
	GUID   string   // stable across exports: what Anki matches a re-import on
	Fields []string // in the order of Fields
	Tags   []string
}

// Media is one file the cards reference by Name.
type Media struct {
	Name string
	Data []byte
}

// Package is everything Write puts in an .apkg.
type Package struct {
	DeckID          int64 // Anki matches a deck on its name and gives an unknown one a new id
	DeckName        string
	DeckDescription string
	Notes           []Note
	Media           []Media
	Modified        time.Time // the notes' modification time: newer updates them on re-import
}

// Write writes p as an .apkg: a zip holding collection.anki2 (schema 11), the
// "media" index and the media files named by their index.
func Write(ctx context.Context, w io.Writer, p Package) error {
	col, err := collection(ctx, p)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(w)
	if err := zipFile(zw, "collection.anki2", col); err != nil {
		return err
	}
	index := make(map[string]string, len(p.Media))
	for i, m := range p.Media {
		key := strconv.Itoa(i)
		index[key] = m.Name
		if err := zipFile(zw, key, m.Data); err != nil {
			return err
		}
	}
	mediaJSON, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if err := zipFile(zw, "media", mediaJSON); err != nil {
		return err
	}
	return zw.Close()
}

func zipFile(zw *zip.Writer, name string, data []byte) error {
	f, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("apkg: %s: %w", name, err)
	}
	_, err = f.Write(data)
	return err
}

// collection builds the SQLite file in a private temporary file: the driver
// writes to a path, and the bytes are all that leaves.
func collection(ctx context.Context, p Package) ([]byte, error) {
	dir, err := os.MkdirTemp("", "blunderdb-apkg-")
	if err != nil {
		return nil, fmt.Errorf("apkg: temp dir: %w", err)
	}
	defer os.RemoveAll(dir)
	path := dir + string(os.PathSeparator) + "collection.anki2"
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("apkg: open collection: %w", err)
	}
	if err := fill(ctx, db, p); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("apkg: close collection: %w", err)
	}
	return os.ReadFile(path)
}

// schema is Anki's collection schema 11, the one every Anki since 2.0 imports,
// AnkiDroid and AnkiMobile included.
const schema = `
CREATE TABLE col (
    id integer primary key, crt integer not null, mod integer not null,
    scm integer not null, ver integer not null, dty integer not null,
    usn integer not null, ls integer not null, conf text not null,
    models text not null, decks text not null, dconf text not null,
    tags text not null);
CREATE TABLE notes (
    id integer primary key, guid text not null, mid integer not null,
    mod integer not null, usn integer not null, tags text not null,
    flds text not null, sfld integer not null, csum integer not null,
    flags integer not null, data text not null);
CREATE TABLE cards (
    id integer primary key, nid integer not null, did integer not null,
    ord integer not null, mod integer not null, usn integer not null,
    type integer not null, queue integer not null, due integer not null,
    ivl integer not null, factor integer not null, reps integer not null,
    lapses integer not null, left integer not null, odue integer not null,
    odid integer not null, flags integer not null, data text not null);
CREATE TABLE revlog (
    id integer primary key, cid integer not null, usn integer not null,
    ease integer not null, ivl integer not null, lastIvl integer not null,
    factor integer not null, time integer not null, type integer not null);
CREATE TABLE graves (usn integer not null, oid integer not null, type integer not null);
CREATE INDEX ix_notes_usn on notes (usn);
CREATE INDEX ix_cards_usn on cards (usn);
CREATE INDEX ix_revlog_usn on revlog (usn);
CREATE INDEX ix_cards_nid on cards (nid);
CREATE INDEX ix_cards_sched on cards (did, queue, due);
CREATE INDEX ix_revlog_cid on revlog (cid);
CREATE INDEX ix_notes_csum on notes (csum);
`

// SchemaVersion is the col.ver the file declares.
const SchemaVersion = 11

func fill(ctx context.Context, db *sql.DB, p Package) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("apkg: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apkg: schema: %w", err)
	}
	mod := p.Modified.Unix()
	modMS := p.Modified.UnixMilli()
	models, decks, dconf, conf, err := colJSON(p, mod)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO col VALUES (1, ?, ?, ?, ?, 0, 0, 0, ?, ?, ?, ?, '{}')`,
		mod, modMS, modMS, SchemaVersion, conf, models, decks, dconf); err != nil {
		return fmt.Errorf("apkg: col: %w", err)
	}
	for i, n := range p.Notes {
		if len(n.Fields) != len(Fields) {
			return fmt.Errorf("apkg: note %s has %d fields, the note type %d", n.GUID, len(n.Fields), len(Fields))
		}
		sfld := stripHTML(n.Fields[0])
		tags := ""
		if len(n.Tags) > 0 {
			tags = " " + strings.Join(n.Tags, " ") + " "
		}
		// Ids are creation times in milliseconds to Anki and need only be
		// unique in the package: the importer matches a note on its guid and
		// renumbers what collides with its own ids.
		id := modMS + int64(i)
		if _, err := tx.ExecContext(ctx, `INSERT INTO notes VALUES (?, ?, ?, ?, -1, ?, ?, ?, ?, 0, '')`,
			id, n.GUID, ModelID, mod, tags, strings.Join(n.Fields, "\x1f"), sfld, checksum(sfld)); err != nil {
			return fmt.Errorf("apkg: note %s: %w", n.GUID, err)
		}
		// A new card (type 0, queue 0) whose due is its rank: Anki shows new
		// cards in the deck's order.
		if _, err := tx.ExecContext(ctx, `INSERT INTO cards VALUES (?, ?, ?, 0, ?, -1, 0, 0, ?, 0, 0, 0, 0, 0, 0, 0, 0, '')`,
			id, id, p.DeckID, mod, i+1); err != nil {
			return fmt.Errorf("apkg: card %s: %w", n.GUID, err)
		}
	}
	return tx.Commit()
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

// stripHTML is the text Anki sorts and checksums a note on.
func stripHTML(s string) string {
	return strings.TrimSpace(tagRE.ReplaceAllString(s, ""))
}

// checksum is Anki's csum: the first 8 hex digits of the SHA-1 of the sort
// field, as an integer.
func checksum(s string) int64 {
	sum := sha1.Sum([]byte(s))
	v, _ := strconv.ParseInt(hex.EncodeToString(sum[:])[:8], 16, 64)
	return v
}

func colJSON(p Package, mod int64) (models, decks, dconf, conf string, err error) {
	flds := make([]map[string]any, len(Fields))
	for i, name := range Fields {
		flds[i] = map[string]any{"name": name, "ord": i, "font": "Arial", "size": 20, "rtl": false, "sticky": false, "media": []any{}}
	}
	model := map[string]any{
		"id": ModelID, "name": ModelName, "type": 0, "mod": mod, "usn": -1,
		"sortf": 0, "did": p.DeckID, "tags": []any{}, "vers": []any{},
		"flds": flds,
		"tmpls": []map[string]any{{
			"name": "Card 1", "ord": 0, "qfmt": frontTemplate, "afmt": backTemplate,
			"bqfmt": "", "bafmt": "", "did": nil,
		}},
		"css":       css,
		"latexPre":  "\\documentclass[12pt]{article}\n\\special{papersize=3in,5in}\n\\usepackage[utf8]{inputenc}\n\\usepackage{amssymb,amsmath}\n\\pagestyle{empty}\n\\setlength{\\parindent}{0in}\n\\begin{document}\n",
		"latexPost": "\\end{document}",
		// The card exists when the Board field is filled.
		"req": []any{[]any{0, "any", []int{1}}},
	}
	deck := func(id int64, name, desc string) map[string]any {
		return map[string]any{
			"id": id, "name": name, "desc": desc, "mod": mod, "usn": -1, "conf": 1, "dyn": 0,
			"collapsed": false, "extendNew": 10, "extendRev": 50,
			"newToday": []int{0, 0}, "revToday": []int{0, 0}, "lrnToday": []int{0, 0}, "timeToday": []int{0, 0},
		}
	}
	dc := map[string]any{
		"id": 1, "name": "Default", "mod": 0, "usn": 0, "maxTaken": 60, "autoplay": true, "timer": 0, "replayq": true,
		"new":   map[string]any{"bury": true, "delays": []int{1, 10}, "initialFactor": 2500, "ints": []int{1, 4, 7}, "order": 1, "perDay": 20, "separate": true},
		"lapse": map[string]any{"delays": []int{10}, "leechAction": 0, "leechFails": 8, "minInt": 1, "mult": 0},
		"rev":   map[string]any{"bury": true, "ease4": 1.3, "fuzz": 0.05, "ivlFct": 1, "maxIvl": 36500, "minSpace": 1, "perDay": 100},
	}
	cf := map[string]any{
		"activeDecks": []int64{p.DeckID}, "curDeck": p.DeckID, "curModel": strconv.FormatInt(ModelID, 10),
		"newSpread": 0, "collapseTime": 1200, "timeLim": 0, "estTimes": true, "dueCounts": true,
		"nextPos": len(p.Notes) + 1, "sortType": "noteFld", "sortBackwards": false, "addToCur": true,
	}
	enc := func(v any) string {
		if err != nil {
			return ""
		}
		var b []byte
		b, err = json.Marshal(v)
		return string(b)
	}
	models = enc(map[string]any{strconv.FormatInt(ModelID, 10): model})
	decks = enc(map[string]any{
		"1":                             deck(1, "Default", ""),
		strconv.FormatInt(p.DeckID, 10): deck(p.DeckID, p.DeckName, p.DeckDescription),
	})
	dconf = enc(map[string]any{"1": dc})
	conf = enc(cf)
	return models, decks, dconf, conf, err
}

const frontTemplate = `<div class="board">{{Board}}</div>
<div class="situation">{{Situation}}</div>`

const backTemplate = `{{FrontSide}}
<hr id="answer">
<div class="answer">{{Answer}}</div>
{{#Equity}}<div class="equity">{{Equity}}</div>{{/Equity}}
{{#Played}}<div class="played">{{Played}}</div>{{/Played}}
<div class="position">{{Position}}</div>`

const css = `.card { font-family: sans-serif; font-size: 18px; text-align: center; color: #222; background: #fafafa; }
.nightMode .card, .card.nightMode { color: #eee; background: #2a2a2a; }
.board img { max-width: 100%; height: auto; }
.situation { margin-top: 0.4em; }
.answer { font-weight: bold; font-size: 1.15em; }
.equity table { margin: 0.4em auto; border-collapse: collapse; }
.equity td { padding: 0 0.6em; }
.played { margin-top: 0.4em; }
.position { margin-top: 0.8em; font-size: 0.7em; opacity: 0.6; font-family: monospace; }`
