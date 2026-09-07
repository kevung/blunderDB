package database

import (
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The directory (issue #391, fonctionnel.md §4.2).
//
// A club director runs the same thirty people every month. Retyping their names at every
// tournament is the first place the software gets abandoned.
//
// The directory is EVERY Participant of EVERY Direction of this database, seen as one list
// de-duplicated by name. It is a DERIVED VIEW and never a table: two spellings stay two lines,
// and blunderDB keeps having no notion of a person — the identity CONTEXT.md has always
// refused. Deleting a Direction takes its Participants out of the directory, because there was
// never anywhere else they were written.

// DirectoryEntry is one line of the directory: a name, and what the LAST entry under that name
// said about the club and the rating.
type DirectoryEntry struct {
	Name   string  `json:"name"`
	Club   string  `json:"club,omitempty"`
	Rating float64 `json:"rating,omitempty"`
	// Entries counts the Directions this name was entered in — what tells a club's regulars
	// from a one-off visitor.
	Entries int `json:"entries"`
	// LastTournament names the most recent tournament they entered, so the director can say
	// "the ones from last month" rather than reading thirty names.
	LastTournamentID int64  `json:"lastTournamentId,omitempty"`
	LastTournament   string `json:"lastTournament,omitempty"`
	LastDate         string `json:"lastDate,omitempty"`
}

// DirectorySource is a directed tournament whose entrants can be taken again wholesale.
type DirectorySource struct {
	TournamentID int64  `json:"tournamentId"`
	Name         string `json:"name"`
	Date         string `json:"date"`
	Entrants     int    `json:"entrants"`
}

// Directory rebuilds the directory. Nothing is stored; every call replays the journals.
func (d *Database) Directory() ([]DirectoryEntry, error) {
	ctx := context.Background()
	recs, err := d.DirectionStore().ListDirections(ctx)
	if err != nil {
		return nil, err
	}
	names, err := d.tournamentNames()
	if err != nil {
		return nil, err
	}
	byName := map[string]*DirectoryEntry{}
	var order []string
	// Oldest first, so the LAST entry under a name is the one that wins.
	sort.Slice(recs, func(i, j int) bool { return recs[i].UpdatedAt.Before(recs[j].UpdatedAt) })
	for _, rec := range recs {
		players, err := d.entrantsOf(ctx, rec.TournamentID)
		if err != nil {
			return nil, err
		}
		t := names[rec.TournamentID]
		for _, p := range players {
			key := directoryKey(p.Name)
			if key == "" {
				continue
			}
			e, ok := byName[key]
			if !ok {
				e = &DirectoryEntry{}
				byName[key] = e
				order = append(order, key)
			}
			// The last entry wins: a player who changed club is listed under the new one.
			e.Name = p.Name
			e.Club = p.Club
			e.Rating = p.Rating
			e.Entries++
			e.LastTournamentID = rec.TournamentID
			e.LastTournament = t.Name
			e.LastDate = t.Date
		}
	}
	out := make([]DirectoryEntry, 0, len(order))
	for _, k := range order {
		out = append(out, *byName[k])
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// DirectorySources lists the directed tournaments whose entrants can be taken again.
func (d *Database) DirectorySources() ([]DirectorySource, error) {
	ctx := context.Background()
	recs, err := d.DirectionStore().ListDirections(ctx)
	if err != nil {
		return nil, err
	}
	names, err := d.tournamentNames()
	if err != nil {
		return nil, err
	}
	out := make([]DirectorySource, 0, len(recs))
	for _, rec := range recs {
		players, err := d.entrantsOf(ctx, rec.TournamentID)
		if err != nil {
			return nil, err
		}
		t := names[rec.TournamentID]
		out = append(out, DirectorySource{
			TournamentID: rec.TournamentID, Name: t.Name, Date: t.Date, Entrants: len(players),
		})
	}
	// Most recent first: "the ones from last month" is at the top.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date > out[j].Date
		}
		return out[i].TournamentID > out[j].TournamentID
	})
	return out, nil
}

// DirectoryEntrants gives one Direction's entrants, ready to be entered into another.
func (d *Database) DirectoryEntrants(tournamentID int64) ([]DirectoryEntry, error) {
	players, err := d.entrantsOf(context.Background(), tournamentID)
	if err != nil {
		return nil, err
	}
	out := make([]DirectoryEntry, 0, len(players))
	for _, p := range players {
		out = append(out, DirectoryEntry{Name: p.Name, Club: p.Club, Rating: p.Rating, Entries: 1})
	}
	return out, nil
}

// entrantsOf replays one Direction and returns its Participants in entry order.
func (d *Database) entrantsOf(ctx context.Context, tournamentID int64) ([]tournoi.Player, error) {
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	out := make([]tournoi.Player, 0, len(st.Order))
	for _, id := range st.Order {
		if p := st.Players[id]; p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}

// tournamentNames indexes the tournaments by id, so a directory line can say where it came from.
func (d *Database) tournamentNames() (map[int64]Tournament, error) {
	all, err := d.GetAllTournaments()
	if err != nil {
		return nil, err
	}
	out := make(map[int64]Tournament, len(all))
	for _, t := range all {
		out[t.ID] = t
	}
	return out, nil
}

// directoryKey is what makes two lines one. Case and surrounding space only: two SPELLINGS
// stay two lines, deliberately — deciding that "J. Dupont" and "Jean Dupont" are one person is
// exactly the inference blunderDB does not make.
func directoryKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// DirectoryCSV renders the directory as a CSV the director keeps between seasons.
func (d *Database) DirectoryCSV() (string, error) {
	entries, err := d.Directory()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"name", "club", "rating"}); err != nil {
		return "", err
	}
	for _, e := range entries {
		rating := ""
		if e.Rating != 0 {
			rating = strconv.FormatFloat(e.Rating, 'f', -1, 64)
		}
		if err := w.Write([]string{e.Name, e.Club, rating}); err != nil {
			return "", err
		}
	}
	w.Flush()
	return b.String(), w.Error()
}

// DirectoryCSVError is one line a CSV could not give: the line number and a CODE, never a
// sentence — the frontend says it in the user's language.
type DirectoryCSVError struct {
	Line int    `json:"line"`
	Code string `json:"code"`
	Text string `json:"text,omitempty"`
}

// DirectoryImport is what a pasted CSV would give, WITHOUT writing anything.
//
// Parsing and entering are two gestures on purpose: the director sees the lines that are wrong
// and the ones that are fine, and nothing is entered until they say so.
type DirectoryImport struct {
	Rows   []DirectoryEntry    `json:"rows"`
	Errors []DirectoryCSVError `json:"errors"`
	// Skipped is what was read as a header rather than as data. It is reported and not
	// silent: a line that disappears without a word is a player who does not show up on the
	// day, and nobody knows why.
	Skipped []DirectoryCSVError `json:"skipped"`
}

// ParseDirectoryCSV reads a CSV of entries. It writes nothing, ever.
//
// The header is recognised by a RULE and not by a list of words: a first line whose rating
// column is present and is not a number cannot be data, whatever language it is written in.
// blunderDB speaks nine, and a closed list of accepted headers would be the same closed list of
// synonyms that was taken out of the `ask` command. What is skipped is reported.
//
// A semicolon-separated file is read too — it is what a French spreadsheet produces — and a
// decimal comma with it. A line with no name is an error, not an empty entry.
func (d *Database) ParseDirectoryCSV(body string) (*DirectoryImport, error) {
	out := &DirectoryImport{Rows: []DirectoryEntry{}, Errors: []DirectoryCSVError{}, Skipped: []DirectoryCSVError{}}
	body = strings.TrimPrefix(body, "\ufeff")
	if strings.TrimSpace(body) == "" {
		return out, nil
	}
	r := csv.NewReader(strings.NewReader(body))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	if separatorIsSemicolon(body) {
		r.Comma = ';'
	}
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("directory: %w", err)
	}
	for i, rec := range records {
		line := i + 1
		if len(rec) == 0 || strings.TrimSpace(strings.Join(rec, "")) == "" {
			continue
		}
		name := strings.TrimSpace(rec[0])
		if i == 0 && looksLikeHeader(rec) {
			out.Skipped = append(out.Skipped, DirectoryCSVError{Line: line, Code: "header", Text: name})
			continue
		}
		if name == "" {
			out.Errors = append(out.Errors, DirectoryCSVError{Line: line, Code: "noName"})
			continue
		}
		e := DirectoryEntry{Name: name}
		if len(rec) > 1 {
			e.Club = strings.TrimSpace(rec[1])
		}
		if len(rec) > 2 {
			raw := strings.TrimSpace(strings.ReplaceAll(rec[2], ",", "."))
			if raw != "" {
				v, err := strconv.ParseFloat(raw, 64)
				if err != nil || v < 0 {
					out.Errors = append(out.Errors, DirectoryCSVError{Line: line, Code: "badRating", Text: strings.TrimSpace(rec[2])})
					continue
				}
				e.Rating = v
			}
		}
		out.Rows = append(out.Rows, e)
	}
	return out, nil
}

// looksLikeHeader says whether a first line is a header rather than an entry.
//
// The rule is the rating column: a rating is a number or nothing. A first line that carries
// something else there is a heading — "rating", "cote", "Wertung" — and no list of words is
// needed to see it.
func looksLikeHeader(rec []string) bool {
	if len(rec) < 3 {
		return false
	}
	raw := strings.TrimSpace(strings.ReplaceAll(rec[2], ",", "."))
	if raw == "" {
		return false
	}
	_, err := strconv.ParseFloat(raw, 64)
	return err != nil
}

// separatorIsSemicolon guesses the separator on the first line. A French spreadsheet writes
// semicolons, and a file that will not open is a file the director stops using.
func separatorIsSemicolon(body string) bool {
	first := body
	if i := strings.IndexAny(body, "\r\n"); i >= 0 {
		first = body[:i]
	}
	return strings.Count(first, ";") > strings.Count(first, ",")
}
