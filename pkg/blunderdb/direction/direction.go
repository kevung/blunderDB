package direction

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
)

// EngineVersion is the Nicomaque tag this build embeds. It is stored with every Direction and
// shown by the credit button, so a replay problem can be traced to the engine that wrote it.
// Keep it in step with the require line in go.mod.
const EngineVersion = "v0.2.0"

// State is where a Direction stands in its life (ADR-0047 § "Le cycle de vie").
type State string

const (
	// StateDraft: no match launched yet. Entries come and go, and the configuration still
	// changes — through an event like everything else, but freely.
	StateDraft State = "draft"
	// StateRunning: the first match froze the configuration into the created event. From here
	// the configuration only changes through an event.
	StateRunning State = "running"
	// StateFinished: the tournament is closed. Read-only, until a reopening event.
	StateFinished State = "finished"
)

// Record is the row that carries a Direction's own facts. The derived state is not in it.
type Record struct {
	TournamentID  int64
	FormatVersion int
	EngineVersion string
	State         State
	// Config is the configuration being edited, and is only meaningful while the Direction is
	// a draft: once running, the configuration of record is the one in the created event.
	Config    string
	OutputDir string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StoredEvent is one line of the log as persistence hands it back.
type StoredEvent struct {
	Seq     int
	Kind    string
	Time    time.Time
	Payload []byte
}

// Store is everything this package needs from persistence. It is deliberately small, and it is
// the whole reason the package does no SQL: the desktop wrapper and both storage backends
// implement it, so the logic is written once (CLAUDE.md, CLI/GUI/server parity).
//
// AppendEvent must be atomic with respect to the caller's transaction and must REFUSE to
// overwrite an existing sequence number — the log is append-only, and a silent overwrite would
// lose a decision the director made.
type Store interface {
	GetDirection(ctx context.Context, tournamentID int64) (Record, error)
	ListDirections(ctx context.Context) ([]Record, error)
	CreateDirection(ctx context.Context, rec Record) error
	UpdateDirection(ctx context.Context, rec Record) error
	DeleteDirection(ctx context.Context, tournamentID int64) error
	AppendEvent(ctx context.Context, tournamentID int64, ev StoredEvent) error
	LoadEvents(ctx context.Context, tournamentID int64) ([]StoredEvent, error)
}

// ErrNoDirection says this Tournament was never directed — it is a Tournament assembled from
// imported files, which is the ordinary case and not an error condition on its own.
var ErrNoDirection = errors.New("direction: this tournament has no direction")

// ErrFinished says the Direction is closed: it accepts no further decision until it is reopened.
var ErrFinished = errors.New("direction: the tournament is closed")

// Direction is an open Direction: its record, its log, and the state replayed from it.
type Direction struct {
	rec     Record
	store   Store
	journal tournoi.Journal
	st      *tournoi.State
}

// Open reads a Tournament's Direction and replays it. Replaying at every open is the point:
// nothing derived is stored, so a crash mid-tournament costs exactly nothing.
func Open(ctx context.Context, store Store, tournamentID int64) (*Direction, error) {
	rec, err := store.GetDirection(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	stored, err := store.LoadEvents(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("direction %d: loading events: %w", tournamentID, err)
	}
	d := &Direction{rec: rec, store: store}
	for _, se := range stored {
		var ev tournoi.Event
		if err := json.Unmarshal(se.Payload, &ev); err != nil {
			return nil, fmt.Errorf("direction %d: event %d unreadable: %w", tournamentID, se.Seq, err)
		}
		d.journal = append(d.journal, ev)
	}
	if len(d.journal) == 0 {
		// Only a Direction whose creation failed half-way has an empty log; there is no
		// tournament state to make up for it.
		return d, nil
	}
	if d.st, err = tournoi.Replay(d.journal); err != nil {
		return nil, fmt.Errorf("direction %d: replay: %w", tournamentID, err)
	}
	return d, nil
}

// Create starts a Direction on a Tournament that has none: it writes the created event at once,
// with the seed the draw will be reproducible from.
//
// The log therefore begins immediately, and that is deliberate. Entries have to survive — a
// director who typed twenty names and closed the laptop must find them again — and a draft that
// wrote nothing would have to keep them somewhere else, which is a second place for the same
// truth. "In preparation" is not "nothing is written": it is "no match has been launched yet",
// which is exactly what ADR-0047 says, and while that holds the configuration still changes
// freely (through an event, like everything else).
func Create(ctx context.Context, store Store, tournamentID int64, cfg tournoi.Config, seed int64, now time.Time) (*Direction, error) {
	if _, err := store.GetDirection(ctx, tournamentID); err == nil {
		return nil, fmt.Errorf("direction: tournament %d is already directed", tournamentID)
	} else if !errors.Is(err, ErrNoDirection) {
		return nil, err
	}
	if seed == 0 {
		// A draw must be reproducible, so the seed is recorded — but nobody should have to
		// invent one. The clock is as good a source as any, and it goes in the log.
		seed = now.UnixNano()
	}
	st, created, err := tournoi.New(cfg, seed, now)
	if err != nil {
		return nil, err
	}
	rec := Record{
		TournamentID:  tournamentID,
		FormatVersion: tournoi.JournalVersion,
		EngineVersion: EngineVersion,
		State:         StateDraft,
	}
	if err := store.CreateDirection(ctx, rec); err != nil {
		return nil, err
	}
	d := &Direction{rec: rec, store: store, st: st}
	if err := d.append(ctx, created); err != nil {
		return nil, err
	}
	return d, nil
}

// Record returns the Direction's own facts.
func (d *Direction) Record() Record { return d.rec }

// State returns the replayed tournament state, or nil while the Direction is a draft that has
// not started.
func (d *Direction) State() *tournoi.State { return d.st }

// Journal returns the log as it stands. The slice is the Direction's own; do not modify it.
func (d *Direction) Journal() tournoi.Journal { return d.journal }

// Config returns the configuration this Direction runs on: the one the log carries, as the
// created event wrote it and every configuration change since amended it.
func (d *Direction) Config() (tournoi.Config, error) {
	if d.st == nil {
		return tournoi.Config{}, ErrNoDirection
	}
	return d.st.Config, nil
}

// SetConfig changes the configuration. It is an EVENT, in preparation as afterwards, so that
// what the director decided stays readable in order (ADR-0047 §3.5). The engine validates it
// and refuses what would change the kind of a phase already begun.
func (d *Direction) SetConfig(ctx context.Context, cfg tournoi.Config) error {
	if d.st == nil {
		return ErrNoDirection
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	return d.Apply(ctx, tournoi.ConfigChangedEvent(cfg, time.Now()))
}

// SetOutputDir remembers where the standalone display page is written.
func (d *Direction) SetOutputDir(ctx context.Context, dir string) error {
	d.rec.OutputDir = dir
	return d.store.UpdateDirection(ctx, d.rec)
}

// Enter records one entry. It works in preparation and afterwards: a late arrival is an entry
// like any other, and the engine decides where they come in (ADR-0047 §4.4).
func (d *Direction) Enter(ctx context.Context, p tournoi.Player, now time.Time) error {
	return d.Apply(ctx, tournoi.PlayerAddedEvent(p, now))
}

// markRunning moves a Direction out of preparation. It is called the moment a match is
// launched, which is what ADR-0047 makes the boundary: not the created event, not the entries.
func (d *Direction) markRunning(ctx context.Context) error {
	if d.rec.State != StateDraft {
		return nil
	}
	d.rec.State = StateRunning
	return d.store.UpdateDirection(ctx, d.rec)
}

// Started says whether a match has been launched: the boundary between preparation and a
// tournament under way.
func (d *Direction) Started() bool { return d.rec.State != StateDraft }

// Reopen takes a closed tournament back, because a result was wrong. The final standings are
// recomputed at the next close; the log keeps everything.
func (d *Direction) Reopen(ctx context.Context, now time.Time) error {
	if d.st == nil {
		return ErrNoDirection
	}
	if err := d.st.Apply(tournoi.ReopenedEvent(now)); err != nil {
		return err
	}
	if err := d.append(ctx, tournoi.ReopenedEvent(now)); err != nil {
		return err
	}
	d.rec.State = StateRunning
	return d.store.UpdateDirection(ctx, d.rec)
}

// Propose asks the engine what to do now — at the host's clock.
//
// The engine has no clock of its own and refuses to invent one, so it takes the instant as an
// argument (Nicomaque v0.2.0). It matters: a micro-round's countdown and a break's warning are
// both answers to "what time is it", and calling the clockless Propose() would freeze them at
// the last recorded event. Returns nothing while the Direction is a draft.
func (d *Direction) Propose() []tournoi.Action {
	return d.ProposeAt(time.Now())
}

// ProposeAt asks the engine what to do at a given instant. Tests pass a fixed one so a run is
// reproducible; the panel passes the wall clock.
func (d *Direction) ProposeAt(now time.Time) []tournoi.Action {
	if d.st == nil {
		return nil
	}
	return d.st.ProposeAt(now)
}

// Warnings are the engine's standing complaints: a bracket match played by the wrong players, a
// score beyond the match length. They are shown, never blocking, and they disappear when their
// cause does — not because someone read them.
func (d *Direction) Warnings() []tournoi.Warning {
	if d.st == nil {
		return nil
	}
	return d.st.Warnings
}

// Apply records one decision: the engine judges the event first, and only an accepted event is
// written. The order matters — the engine refuses (a result on a match that is not running, a
// format change on a phase already drawn), and a log is a record of what happened, so a refused
// decision must leave nothing behind.
func (d *Direction) Apply(ctx context.Context, ev tournoi.Event) error {
	if d.st == nil {
		return fmt.Errorf("direction: the tournament has not started")
	}
	if d.st.Finished && ev.Kind != tournoi.EvNote {
		return ErrFinished
	}
	if err := d.st.Apply(ev); err != nil {
		return err
	}
	if err := d.append(ctx, ev); err != nil {
		return err
	}
	if ev.Kind == tournoi.EvMatchStarted {
		return d.markRunning(ctx)
	}
	return nil
}

// EventFor turns a proposal the director confirmed into the event that records it.
func (d *Direction) EventFor(a tournoi.Action, now time.Time) (tournoi.Event, error) {
	if d.st == nil {
		return tournoi.Event{}, fmt.Errorf("direction: the tournament has not started")
	}
	return d.st.EventFromAction(a, now)
}

// append writes one event at the next sequence number and keeps the in-memory log in step.
func (d *Direction) append(ctx context.Context, ev tournoi.Event) error {
	ev.Seq = len(d.journal)
	blob, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("direction: serialising event %s: %w", ev.Kind, err)
	}
	se := StoredEvent{Seq: ev.Seq, Kind: string(ev.Kind), Time: ev.Time, Payload: blob}
	if err := d.store.AppendEvent(ctx, d.rec.TournamentID, se); err != nil {
		return err
	}
	d.journal = append(d.journal, ev)
	d.rec.UpdatedAt = ev.Time
	return nil
}

// Finish closes the tournament and freezes the final standings.
func (d *Direction) Finish(ctx context.Context, now time.Time) error {
	if d.st == nil {
		return fmt.Errorf("direction: the tournament has not started")
	}
	if d.st.Finished {
		return ErrFinished
	}
	if err := d.Apply(ctx, tournoi.Event{Version: tournoi.JournalVersion, Kind: tournoi.EvFinished, Time: now}); err != nil {
		return err
	}
	d.rec.State = StateFinished
	return d.store.UpdateDirection(ctx, d.rec)
}

// Delete removes the Direction and its log. The Tournament and its Matches stay: the Matches
// merely lose the Slot they filled (ADR-0047).
func (d *Direction) Delete(ctx context.Context) error {
	return d.store.DeleteDirection(ctx, d.rec.TournamentID)
}

// Ranking is the current standings. Notes are codes; the frontend renders them.
func (d *Direction) Ranking() []tournoi.Rank {
	if d.st == nil {
		return nil
	}
	return d.st.Ranking()
}
