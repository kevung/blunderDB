package changeset

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/anki"
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The families below have no trash of their own: a trash a peer sends is a
// delete here, as for a lesson.

func decode(what string, raw json.RawMessage, into any) error {
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("%s payload: %w", what, err)
	}
	return nil
}

// --- transcription ---

// transcriptionPayload is a draft with the match it produced, by uuid. The
// revision stays out: it counts the writes of one library's row.
type transcriptionPayload struct {
	FormatVersion string `json:"formatVersion"`
	Match         string `json:"match,omitempty"`
	Label         string `json:"label"`
	Document      string `json:"document"`
	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// matchHere is the id of the match a uuid names, 0 when this library lacks it
// (the draft then stands as one that produced none).
func matchHere(ctx context.Context, s storage.Stores, scope, uuid string) (int64, error) {
	if uuid == "" {
		return 0, nil
	}
	id, err := uuidsOf(s).ObjectID(ctx, scope, "match", uuid)
	if errors.Is(err, storage.ErrNotFound) {
		noteMissing(ctx, uuid)
		return 0, nil
	}
	return id, err
}

func saveTranscription(ctx context.Context, s storage.Stores, scope string, id int64, p transcriptionPayload) (int64, error) {
	mid, err := matchHere(ctx, s, scope, p.Match)
	if err != nil {
		return 0, err
	}
	t := &storage.Transcription{ID: id, FormatVersion: p.FormatVersion, MatchID: mid, Label: p.Label, Document: p.Document}
	id, err = s.Transcriptions().Save(ctx, scope, t)
	if err != nil {
		return 0, err
	}
	return id, s.Transcriptions().KeepDates(ctx, scope, id, p.CreatedAt, p.UpdatedAt)
}

var transcriptionCodec = codec{
	rank: 5,
	// A tie says neither side knows of the other's edit: a draft that names the
	// match it produced is kept over the same draft standing alone (its match
	// had not arrived), whatever the digests say.
	merge: func(winner, loser json.RawMessage) (json.RawMessage, error) {
		var w, l transcriptionPayload
		if err := decode("transcription", winner, &w); err != nil {
			return nil, err
		}
		if err := decode("transcription", loser, &l); err != nil {
			return nil, err
		}
		if w.Match != "" || l.Match == "" {
			return winner, nil
		}
		w.Match = l.Match
		return json.Marshal(w)
	},
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		t, err := s.Transcriptions().Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		p := transcriptionPayload{FormatVersion: t.FormatVersion, Label: t.Label, Document: t.Document,
			CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
		if t.MatchID != 0 {
			if p.Match, err = uuidsOf(s).ObjectUUID(ctx, scope, "match", t.MatchID); err != nil {
				return nil, err
			}
		}
		return p, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p transcriptionPayload
		if err := decode("transcription", raw, &p); err != nil {
			return 0, err
		}
		return saveTranscription(ctx, s, scope, 0, p)
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p transcriptionPayload
		if err := decode("transcription", raw, &p); err != nil {
			return err
		}
		_, err := saveTranscription(ctx, s, scope, id, p)
		return err
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, _ bool) error {
		return s.Transcriptions().Delete(ctx, scope, id)
	},
}

// --- tournament ---

// tournamentPayload is a tournament's header and note. Which matches belong to
// it is the match's to say (it names the tournament). updated_at stays out: it
// moves at every attachment, here and on the peer alike.
type tournamentPayload struct {
	Name      string `json:"name"`
	Date      string `json:"date,omitempty"`
	Location  string `json:"location,omitempty"`
	Comment   string `json:"comment,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// joinText settles two texts of one field: the one that is not empty, the one
// that holds the other, else both in byte order, so either side builds the same.
func joinText(a, b string) string {
	switch {
	case a == b || b == "" || strings.Contains(a, b):
		return a
	case a == "" || strings.Contains(b, a):
		return b
	case a > b:
		a, b = b, a
	}
	return a + "\n\n" + b
}

// firstOf is the first of two texts that is not empty.
func firstOf(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

var tournamentCodec = codec{
	// Before the matches, which name it.
	rank: -1,
	// A tie says neither side knows of the other's edit: a note or a place one
	// side lacks is not erased for that, and two notes are both kept.
	merge: func(winner, loser json.RawMessage) (json.RawMessage, error) {
		var w, l tournamentPayload
		if err := decode("tournament", winner, &w); err != nil {
			return nil, err
		}
		if err := decode("tournament", loser, &l); err != nil {
			return nil, err
		}
		w.Comment = joinText(w.Comment, l.Comment)
		w.Date, w.Location = firstOf(w.Date, l.Date), firstOf(w.Location, l.Location)
		if w.CreatedAt == "" || (l.CreatedAt != "" && l.CreatedAt < w.CreatedAt) {
			w.CreatedAt = l.CreatedAt
		}
		return json.Marshal(w)
	},
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		t, err := s.Tournaments().Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		return tournamentPayload{Name: t.Name, Date: t.Date, Location: t.Location, Comment: t.Comment, CreatedAt: t.CreatedAt}, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p tournamentPayload
		if err := decode("tournament", raw, &p); err != nil {
			return 0, err
		}
		id, err := s.Tournaments().Create(ctx, scope, p.Name, p.Date, p.Location)
		if err != nil {
			return 0, err
		}
		return id, fillTournament(ctx, s, scope, id, p)
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p tournamentPayload
		if err := decode("tournament", raw, &p); err != nil {
			return err
		}
		if err := s.Tournaments().Update(ctx, scope, id, p.Name, p.Date, p.Location); err != nil {
			return err
		}
		return fillTournament(ctx, s, scope, id, p)
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, _ bool) error {
		return s.Tournaments().Delete(ctx, scope, id)
	},
}

func fillTournament(ctx context.Context, s storage.Stores, scope string, id int64, p tournamentPayload) error {
	if err := s.Tournaments().UpdateComment(ctx, scope, id, p.Comment); err != nil {
		return err
	}
	return s.Tournaments().KeepDates(ctx, scope, id, p.CreatedAt, "")
}

// --- anki deck ---

// deckPayload is a deck's name, source and scheduling settings. A deck drawn
// from a collection names it by uuid; a deck drawn from a search keeps the
// search. The counters are the library's to compute.
type deckPayload struct {
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	SourceType       string  `json:"sourceType"`
	SourceCollection string  `json:"sourceCollection,omitempty"`
	SourceCommand    string  `json:"sourceCommand,omitempty"`
	RequestRetention float64 `json:"requestRetention"`
	MaximumInterval  float64 `json:"maximumInterval"`
	EnableFuzz       bool    `json:"enableFuzz"`
	SessionLimit     *int    `json:"sessionLimit"`
}

// portableSource turns the local position ids a search deck's source lists into
// board hashes, which two libraries share; localSource turns them back.
func portableSource(ctx context.Context, s storage.Stores, scope, source string) (string, error) {
	doc, isDoc := domain.SearchSourceDocument(source)
	ids := domain.SearchSourceIDs(source)
	if !isDoc && len(ids) == 0 {
		return source, nil
	}
	if doc == nil {
		doc = map[string]json.RawMessage{}
	}
	refs := make([]string, 0, len(ids))
	for _, id := range ids {
		ref, err := refOf(ctx, s, scope, id)
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			return "", err
		}
		refs = append(refs, ref)
	}
	delete(doc, "ids")
	// The command and the payload replay on the peer's own numbering: one that
	// names a row by id selects other rows there, so it is dropped, as every
	// other path a deck travels by drops it. The refs stay.
	if raw, ok := doc["command"]; ok {
		var command string
		if json.Unmarshal(raw, &command) != nil || commandNamesRows(command) {
			delete(doc, "command")
			delete(doc, "position")
		}
	}
	if raw, ok := doc["payload"]; ok && domain.SearchPayloadNamesRows(raw) {
		delete(doc, "payload")
	}
	raw, err := json.Marshal(refs)
	if err != nil {
		return "", err
	}
	doc["refs"] = raw
	out, err := json.Marshal(doc)
	return string(out), err
}

// commandNamesRows says a command-bar query selects by row id (like<n>, id<n>,
// ma<n>, tn<n>).
func commandNamesRows(command string) bool {
	f, _ := searchquery.Parse(command)
	return f.LikeTargetID != 0 || f.PositionIDsFilter != "" || f.MatchIDsFilter != "" || f.TournamentIDsFilter != ""
}

func localSource(ctx context.Context, s storage.Stores, scope, source string) (string, error) {
	doc, isDoc := domain.SearchSourceDocument(source)
	raw, has := doc["refs"]
	if !isDoc || !has {
		return source, nil
	}
	var refs []string
	if err := json.Unmarshal(raw, &refs); err != nil {
		return "", fmt.Errorf("deck payload: %w", err)
	}
	ids := make([]int64, 0, len(refs))
	for _, ref := range refs {
		id, found, err := findPosition(ctx, s, scope, ref)
		if err != nil {
			return "", err
		}
		if !found {
			noteMissing(ctx, ref)
			continue
		}
		ids = append(ids, id)
	}
	delete(doc, "refs")
	idsRaw, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	doc["ids"] = idsRaw
	out, err := json.Marshal(doc)
	return string(out), err
}

// ankiStates is the backend's Anki state; a backend that keeps a journal but
// no decks (the contract is optional) answers every call with an error rather
// than a panic.
func ankiStates(s storage.Stores) storage.AnkiStateStore {
	if p, ok := s.(storage.AnkiStateProvider); ok {
		return p.AnkiStates()
	}
	return noAnkiState{}
}

var errNoAnkiState = errors.New("changeset: the backend keeps no Anki decks and cards")

type noAnkiState struct{}

func (noAnkiState) Deck(context.Context, string, int64) (*domain.AnkiDeck, error) {
	return nil, errNoAnkiState
}
func (noAnkiState) PutDeck(context.Context, string, domain.AnkiDeck) (int64, error) {
	return 0, errNoAnkiState
}
func (noAnkiState) Card(context.Context, string, int64) (*domain.AnkiCard, error) {
	return nil, errNoAnkiState
}
func (noAnkiState) FindCard(context.Context, string, int64, string, string) (int64, bool, error) {
	return 0, false, errNoAnkiState
}
func (noAnkiState) PutCard(context.Context, string, domain.AnkiCard) (int64, error) {
	return 0, errNoAnkiState
}

func deckFrom(ctx context.Context, s storage.Stores, scope string, id int64, p deckPayload) (domain.AnkiDeck, error) {
	d := domain.AnkiDeck{ID: id, Name: p.Name, Description: p.Description, SourceType: p.SourceType,
		SourceCommand: p.SourceCommand, RequestRetention: p.RequestRetention, MaximumInterval: p.MaximumInterval,
		EnableFuzz: p.EnableFuzz, SessionLimit: p.SessionLimit}
	if p.SourceType == domain.AnkiSourceSearch {
		var err error
		if d.SourceCommand, err = localSource(ctx, s, scope, p.SourceCommand); err != nil {
			return d, err
		}
	}
	if p.SourceCollection != "" {
		cid, err := uuidsOf(s).ObjectID(ctx, scope, "collection", p.SourceCollection)
		switch {
		case err == nil:
			d.SourceID = cid
		case errors.Is(err, storage.ErrNotFound):
			noteMissing(ctx, p.SourceCollection)
		default:
			return d, err
		}
	}
	return d, nil
}

var deckCodec = codec{
	rank: 7,
	// A tie says neither side knows of the other's edit: a description one side
	// lacks is not erased for that, and two descriptions are both kept. The
	// settings are the winner's, the one the digests settle.
	merge: func(winner, loser json.RawMessage) (json.RawMessage, error) {
		var w, l deckPayload
		if err := decode("deck", winner, &w); err != nil {
			return nil, err
		}
		if err := decode("deck", loser, &l); err != nil {
			return nil, err
		}
		w.Description = joinText(w.Description, l.Description)
		if w.SourceType == l.SourceType {
			w.SourceCollection, w.SourceCommand = firstOf(w.SourceCollection, l.SourceCollection), firstOf(w.SourceCommand, l.SourceCommand)
		}
		if w.SessionLimit == nil {
			w.SessionLimit = l.SessionLimit
		}
		return json.Marshal(w)
	},
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		d, err := ankiStates(s).Deck(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		p := deckPayload{Name: d.Name, Description: d.Description, SourceType: d.SourceType, SourceCommand: d.SourceCommand,
			RequestRetention: d.RequestRetention, MaximumInterval: d.MaximumInterval, EnableFuzz: d.EnableFuzz, SessionLimit: d.SessionLimit}
		if d.SourceType == domain.AnkiSourceSearch {
			if p.SourceCommand, err = portableSource(ctx, s, scope, d.SourceCommand); err != nil {
				return nil, err
			}
		}
		if d.SourceType == domain.AnkiSourceCollection && d.SourceID != 0 {
			u, err := uuidsOf(s).ObjectUUID(ctx, scope, "collection", d.SourceID)
			if err != nil && !errors.Is(err, storage.ErrNotFound) {
				return nil, err
			}
			p.SourceCollection = u
		}
		return p, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p deckPayload
		if err := decode("deck", raw, &p); err != nil {
			return 0, err
		}
		d, err := deckFrom(ctx, s, scope, 0, p)
		if err != nil {
			return 0, err
		}
		return ankiStates(s).PutDeck(ctx, scope, d)
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p deckPayload
		if err := decode("deck", raw, &p); err != nil {
			return err
		}
		d, err := deckFrom(ctx, s, scope, id, p)
		if err != nil {
			return err
		}
		_, err = ankiStates(s).PutDeck(ctx, scope, d)
		return err
	},
	// The cards go with the deck, each with an entry of its own in the journal.
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, _ bool) error {
		return s.Anki().DeleteDeck(ctx, scope, id)
	},
}

// --- anki card ---

// cardPayload is a card with the deck it belongs to and what it asks about: a
// board by its hash, or a score by its key. The scheduling state is the card's
// whole content, so a review made on either side travels as the later write.
type cardPayload struct {
	Deck          string  `json:"deck"`
	Kind          string  `json:"kind"`
	Position      string  `json:"position,omitempty"`
	Key           string  `json:"key,omitempty"`
	Due           string  `json:"due"`
	Stability     float64 `json:"stability"`
	Difficulty    float64 `json:"difficulty"`
	ElapsedDays   int     `json:"elapsedDays"`
	ScheduledDays int     `json:"scheduledDays"`
	Reps          int     `json:"reps"`
	Lapses        int     `json:"lapses"`
	State         int     `json:"state"`
	LastReview    string  `json:"lastReview,omitempty"`
	Suspended     bool    `json:"suspended,omitempty"`
	BuriedUntil   string  `json:"buriedUntil,omitempty"`
}

// cardKey is the key a card of this payload bears here: the id of the position
// it names, or its score. false when the position is not held.
func cardKey(ctx context.Context, s storage.Stores, scope string, p cardPayload) (key string, pid int64, ok bool, err error) {
	if p.Kind != domain.AnkiKindPosition {
		return p.Key, 0, true, nil
	}
	pid, found, err := findPosition(ctx, s, scope, p.Position)
	if err != nil || !found {
		return "", 0, false, err
	}
	return strconv.FormatInt(pid, 10), pid, true, nil
}

func (p cardPayload) card(id, deckID, positionID int64, key string) domain.AnkiCard {
	return domain.AnkiCard{ID: id, DeckID: deckID, Kind: p.Kind, Key: key, PositionID: positionID, Due: p.Due,
		Stability: p.Stability, Difficulty: p.Difficulty, ElapsedDays: p.ElapsedDays, ScheduledDays: p.ScheduledDays,
		Reps: p.Reps, Lapses: p.Lapses, State: p.State, LastReview: p.LastReview, Suspended: p.Suspended, BuriedUntil: p.BuriedUntil}
}

// reviewInstant is a last review in the one layout both backends write, so two
// libraries compare the same instant as the same text whatever layout a row
// was filled with; a text no layout reads stays as it is.
func reviewInstant(s string) string {
	for _, layout := range []string{anki.TimeLayout, time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(anki.TimeLayout)
		}
	}
	return s
}

// ahead orders two states of one card by how far its reviews went: the later
// last review, then the more reviews, then the more lapses.
func (p cardPayload) ahead(o cardPayload) int {
	if c := cmp.Compare(reviewInstant(p.LastReview), reviewInstant(o.LastReview)); c != 0 {
		return c
	}
	if c := cmp.Compare(p.Reps, o.Reps); c != 0 {
		return c
	}
	return cmp.Compare(p.Lapses, o.Lapses)
}

var cardCodec = codec{
	rank: 8,
	// Two states of one card that nothing orders (a tie, or a draw that met a
	// card reviewed elsewhere) keep the planning whose reviews went further, a
	// draw carrying no review; the suspension and the burial are gestures, and
	// stay the last write's. Winner is the later of the two.
	merge: func(winner, loser json.RawMessage) (json.RawMessage, error) {
		var w, l cardPayload
		if err := decode("card", winner, &w); err != nil {
			return nil, err
		}
		if err := decode("card", loser, &l); err != nil {
			return nil, err
		}
		if l.ahead(w) <= 0 {
			return winner, nil
		}
		l.Suspended, l.BuriedUntil = w.Suspended, w.BuriedUntil
		return json.Marshal(l)
	},
	// The cards of two libraries that drew the same question meet under two
	// uuids: the planning that went furthest is the card's.
	twins: true,
	read: func(ctx context.Context, s storage.Stores, scope string, id int64) (any, error) {
		c, err := ankiStates(s).Card(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		p := cardPayload{Kind: c.Kind, Due: c.Due, Stability: c.Stability, Difficulty: c.Difficulty, ElapsedDays: c.ElapsedDays,
			ScheduledDays: c.ScheduledDays, Reps: c.Reps, Lapses: c.Lapses, State: c.State, LastReview: c.LastReview,
			Suspended: c.Suspended, BuriedUntil: c.BuriedUntil}
		if p.Deck, err = uuidsOf(s).ObjectUUID(ctx, scope, "anki_deck", c.DeckID); err != nil {
			return nil, err
		}
		if c.Kind == domain.AnkiKindPosition {
			if p.Position, err = refOf(ctx, s, scope, c.PositionID); err != nil {
				return nil, err
			}
		} else {
			p.Key = c.Key
		}
		return p, nil
	},
	// The same question in the same deck under two uuids (two libraries drew
	// the card before they met): the older uuid is kept on both sides.
	locate: func(ctx context.Context, s storage.Stores, scope string, r Record) (int64, string, bool, error) {
		var p cardPayload
		if len(r.Payload) == 0 || json.Unmarshal(r.Payload, &p) != nil {
			return 0, "", false, nil
		}
		deck, err := uuidsOf(s).ObjectID(ctx, scope, "anki_deck", p.Deck)
		if errors.Is(err, storage.ErrNotFound) {
			return 0, "", false, nil
		}
		if err != nil {
			return 0, "", false, err
		}
		key, _, ok, err := cardKey(ctx, s, scope, p)
		if err != nil || !ok {
			return 0, "", false, err
		}
		id, found, err := ankiStates(s).FindCard(ctx, scope, deck, p.Kind, key)
		if err != nil || !found {
			return 0, "", false, err
		}
		have, err := uuidsOf(s).ObjectUUID(ctx, scope, "anki_card", id)
		if err != nil {
			return 0, "", false, err
		}
		if have == "" || r.UUID >= have {
			return id, have, have != "", nil
		}
		journal, _, err := providers(s)
		if err != nil {
			return 0, "", false, err
		}
		stamp, serr := journal.Entry(ctx, scope, have)
		if err := uuidsOf(s).SetObjectUUID(ctx, scope, "anki_card", id, r.UUID); err != nil {
			return 0, "", false, err
		}
		// The rename is no edit: it must not look newer than what the peer holds.
		op, at := storage.ChangeCreate, floor
		if serr == nil {
			op, at = stamp.Op, stamp.ChangedAt
		}
		if err := journal.Mark(ctx, scope, "anki_card", r.UUID, op, at); err != nil {
			return 0, "", false, err
		}
		return id, r.UUID, true, nil
	},
	create: func(ctx context.Context, s storage.Stores, scope string, raw json.RawMessage) (int64, error) {
		var p cardPayload
		if err := decode("card", raw, &p); err != nil {
			return 0, err
		}
		deck, err := uuidsOf(s).ObjectID(ctx, scope, "anki_deck", p.Deck)
		if errors.Is(err, storage.ErrNotFound) {
			noteMissing(ctx, p.Deck)
			return 0, storage.ErrConflict
		}
		if err != nil {
			return 0, err
		}
		key, pid, ok, err := cardKey(ctx, s, scope, p)
		if err != nil {
			return 0, err
		}
		if !ok {
			noteMissing(ctx, p.Position)
			return 0, storage.ErrConflict
		}
		return ankiStates(s).PutCard(ctx, scope, p.card(0, deck, pid, key))
	},
	update: func(ctx context.Context, s storage.Stores, scope string, id int64, raw json.RawMessage) error {
		var p cardPayload
		if err := decode("card", raw, &p); err != nil {
			return err
		}
		_, err := ankiStates(s).PutCard(ctx, scope, p.card(id, 0, 0, ""))
		return err
	},
	remove: func(ctx context.Context, s storage.Stores, scope string, id int64, _ bool) error {
		return s.Anki().RemoveCard(ctx, scope, id)
	},
}
