// Package mets is the library's match equity tables as the three modes use
// them (ADR-0068): importing a gnubg .xml, the table a computation is valued
// with, and whether a stored analysis was valued with another one. GUI, CLI
// and daemon call it over the same storage contract, so none of them forks.
package mets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// BuiltInName names the table a library uses when none is current.
const BuiltInName = "Kazaross-XG2"

// MaxSourceBytes bounds an imported file: the largest gnubg table is a few
// tens of kilobytes.
const MaxSourceBytes = 1 << 20

// Import parses a gnubg .xml table and stores it in scope. A file holding the
// built-in values is not stored: it is Kazaross-XG2, returned with ID 0. A
// table already held is returned as held, under its first name. name falls
// back on the file's own <name>, then on fallbackName.
func Import(ctx context.Context, s storage.Storage, scope string, data []byte, fallbackName string) (*domain.MatchEquityTable, error) {
	if len(data) > MaxSourceBytes {
		return nil, fmt.Errorf("%w: %w: %d bytes, more than a match equity table holds", storage.ErrInvalid, engine.ErrMETFormat, len(data))
	}
	m, err := engine.ParseGnubgMET(data)
	if err != nil {
		// A file the caller brought: invalid input, not a fault.
		return nil, fmt.Errorf("%w: %w", storage.ErrInvalid, err)
	}
	digest := m.Digest()
	if digest == engine.KazarossXG2Digest() {
		return &domain.MatchEquityTable{Name: BuiltInName, Digest: digest}, nil
	}
	name := m.Name
	if name == "" {
		name = fallbackName
	}
	if name == "" {
		name = "MET"
	}
	id, err := s.MatchEquityTables().Save(ctx, scope, domain.MatchEquityTable{Name: name, Digest: digest, Source: string(data)})
	if err != nil {
		return nil, err
	}
	list, err := s.MatchEquityTables().List(ctx, scope)
	if err != nil {
		return nil, err
	}
	for _, t := range list {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, fmt.Errorf("match equity table %d: %w", id, storage.ErrNotFound)
}

// ImportFile is Import from a path, named after the file when the table
// names nothing.
func ImportFile(ctx context.Context, s storage.Storage, scope, path string) (*domain.MatchEquityTable, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxSourceBytes {
		return nil, fmt.Errorf("%w: %s is %d bytes, more than a match equity table holds", engine.ErrMETFormat, path, info.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Import(ctx, s, scope, data, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
}

// Current is the table computations in scope are valued with: its id (0 for
// the built-in one) and the parsed table (nil for the built-in one).
func Current(ctx context.Context, s storage.Storage, scope string) (int64, *engine.MET, error) {
	cur, err := s.MatchEquityTables().Current(ctx, scope)
	if err != nil || cur == nil {
		return 0, nil, err
	}
	m, err := engine.ParseGnubgMET([]byte(cur.Source))
	if err != nil {
		return 0, nil, fmt.Errorf("current match equity table %q: %w", cur.Name, err)
	}
	return cur.ID, m, nil
}

// Overview is the tables a library holds, the built-in one first, with the
// current one marked.
func Overview(ctx context.Context, s storage.Storage, scope string) ([]*domain.MatchEquityTable, error) {
	list, err := s.MatchEquityTables().List(ctx, scope)
	if err != nil {
		return nil, err
	}
	builtIn := &domain.MatchEquityTable{Name: BuiltInName, Digest: engine.KazarossXG2Digest(), Current: true}
	for _, t := range list {
		if t.Current {
			builtIn.Current = false
		}
	}
	return append([]*domain.MatchEquityTable{builtIn}, list...), nil
}

// Status is what an analysis panel says about the table behind a verdict.
type Status struct {
	// Name is the table the analysis was valued with.
	Name string `json:"name"`
	// Current is the library's current table.
	Current string `json:"current"`
	// Different: a match-play analysis valued with another table than the
	// current one, left out of every comparison. Never true at money.
	Different bool `json:"different"`
}

// AnalysisStatus reads, never writes, the table of the position's analysis.
func AnalysisStatus(ctx context.Context, s storage.Storage, scope string, positionID int64, money bool) (Status, error) {
	tables, err := Overview(ctx, s, scope)
	if err != nil {
		return Status{}, err
	}
	metID, err := s.MatchEquityTables().OfAnalysis(ctx, scope, positionID)
	if err != nil {
		return Status{}, err
	}
	var st Status
	var currentID int64
	for _, t := range tables {
		if t.ID == metID {
			st.Name = t.Name
		}
		if t.Current {
			st.Current, currentID = t.Name, t.ID
		}
	}
	st.Different = !money && metID != currentID
	return st, nil
}
