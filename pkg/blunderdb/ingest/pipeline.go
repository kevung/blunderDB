package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The file pipeline imports a list of files with N readers and one writer.
// Reading a file — parse, mapping, Zobrist hashing — touches no database, so
// it runs on N goroutines; the writer alone opens transactions, one per group
// of files, and writes in file order so match ids do not depend on which
// reader finished first.
//
// A file whose bytes repeat an earlier file of the same list is not parsed:
// the writer reports it as a duplicate of the first. Corpora are often copied
// several times over, and parsing is the dominant cost of a duplicate.

// File outcomes, as FileOutcome.Status reports them.
const (
	FileImported  = "imported"  // a new match was written
	FileEnriched  = "enriched"  // a cross-format duplicate enriched an existing match
	FileDuplicate = "duplicate" // the match (or the file) was already there
	FilePosition  = "position"  // a single-position file was written
	FileFailed    = "failed"    // unreadable or refused; Error says why
)

// FileOutcome is what one file of the list became.
type FileOutcome struct {
	Index  int    `json:"index"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Status string `json:"status"`
	// MatchID is the written or enriched match, or for a duplicate the stored
	// match that already covers the file.
	MatchID      int64  `json:"match_id,omitempty"`
	PositionID   int64  `json:"position_id,omitempty"`
	Positions    int    `json:"positions"`
	FlagsApplied int    `json:"flags_applied,omitempty"`
	Player1      string `json:"player1,omitempty"`
	Player2      string `json:"player2,omitempty"`
	Games        int    `json:"games,omitempty"`
	Error        string `json:"error,omitempty"`
	// DuplicateOf is the index of an earlier file of the same list with the
	// same bytes, -1 when the duplicate (if any) was found in the database.
	DuplicateOf int `json:"duplicate_of"`
}

// Succeeded reports whether the file brought something in.
func (o FileOutcome) Succeeded() bool {
	return o.Status == FileImported || o.Status == FileEnriched || o.Status == FilePosition
}

// TxBeginner is the part of storage.Storage the pipeline writes through.
type TxBeginner interface {
	BeginTx(ctx context.Context) (storage.Tx, error)
}

// PipelineOptions tunes ImportFiles. The zero value is usable.
type PipelineOptions struct {
	// Workers is the number of reading goroutines; 0 means one per CPU.
	Workers int
	// FilesPerTx is how many files the writer groups in one transaction;
	// 0 means DefaultFilesPerTx.
	FilesPerTx int
	// Scope is the storage scope (tenant), empty for SQLite.
	Scope string
	// ImportBatchID stamps every written match.
	ImportBatchID int64
	// Lock, when set, is taken around each transaction and returns its
	// unlock. The writer takes it for a group only once every file of the
	// group has been read, so it is held for writing, never for waiting on a
	// reader.
	Lock func() (unlock func())
	// OnCommit is called with the outcomes of each committed group, in file
	// order, while Lock is held.
	OnCommit func([]FileOutcome)
	// OnRead is called, from a reader goroutine, when a file has been read
	// (parsed or not), with its size: progress counts bytes as they are read.
	OnRead func(size int64)
}

// DefaultFilesPerTx bounds the work a cancellation or a failed write throws
// away while amortising each commit's fsync over several files.
const DefaultFilesPerTx = 32

type readFile struct {
	index    int
	path     string
	size     int64
	digest   [sha256.Size]byte
	dupOf    int  // earlier index with the same bytes, -1 otherwise
	hashed   bool // the digest is valid: the file could be read
	graph    *MatchGraph
	position []PositionGraph
	err      error
}

// mapFile reads one file through the mapper its extension names.
func mapFile(path string) (*MatchGraph, []PositionGraph, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xg":
		g, err := MapXG(path)
		return g, nil, err
	case ".sgf", ".mat", ".txt":
		g, err := MapGnuBG(path)
		return g, nil, err
	case ".bgf":
		g, err := MapBGF(path)
		return g, nil, err
	case ".ogxm":
		g, err := MapOGXM(path)
		return g, nil, err
	case ".xgp":
		p, err := MapXGPPosition(path)
		return nil, p, err
	}
	return nil, nil, fmt.Errorf("unsupported file type: %s", filepath.Ext(path))
}

func digestFile(path string) ([sha256.Size]byte, int64, error) {
	var sum [sha256.Size]byte
	f, err := os.Open(path)
	if err != nil {
		return sum, 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return sum, n, err
	}
	copy(sum[:], h.Sum(nil))
	return sum, n, nil
}

// ImportFiles imports paths through store and returns one outcome per file
// actually decided, in file order. A group's transaction (and Options.Lock)
// opens only when all the group's files are read. On cancellation the group
// being written is rolled back, earlier groups stay committed, and the error
// is ctx.Err().
func ImportFiles(ctx context.Context, store TxBeginner, paths []string, opts PipelineOptions) ([]FileOutcome, error) {
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > len(paths) {
		workers = max(1, len(paths))
	}
	perTx := opts.FilesPerTx
	if perTx <= 0 {
		perTx = DefaultFilesPerTx
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// slots bounds the files read but not yet written: readers cannot run
	// away from a slow writer and hold the whole corpus in memory.
	inFlight := 2*workers + perTx
	slots := make(chan struct{}, inFlight)
	jobs := make(chan int)
	results := make(chan *readFile, inFlight)

	go func() {
		defer close(jobs)
		for i := range paths {
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				return
			}
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	claims := newDigestClaims()
	done := make(chan struct{})
	for w := 0; w < workers; w++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := range jobs {
				rf := &readFile{index: i, path: paths[i], dupOf: -1}
				rf.digest, rf.size, rf.err = digestFile(rf.path)
				rf.hashed = rf.err == nil
				if rf.err == nil {
					if first := claims.claim(rf.digest, i); first < i {
						rf.dupOf = first
					} else if ctx.Err() == nil {
						rf.graph, rf.position, rf.err = mapFile(rf.path)
					}
				}
				if opts.OnRead != nil {
					opts.OnRead(rf.size)
				}
				select {
				case results <- rf:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		for w := 0; w < workers; w++ {
			<-done
		}
		close(results)
	}()

	wr := &pipelineWriter{ctx: ctx, store: store, opts: opts, perTx: perTx, seen: map[[sha256.Size]byte]FileOutcome{}}
	pending := map[int]*readFile{}
	next := 0
	// ready holds the files, in file order, that are read and not yet written.
	// A group is handed to the writer only once all its files are read, so the
	// lock and the transaction are never held while the writer waits for a
	// reader.
	var ready []*readFile
	writeReady := func() error {
		for _, item := range ready {
			if err := wr.add(item); err != nil {
				return err
			}
			<-slots
		}
		ready = ready[:0]
		return nil
	}
	for rf := range results {
		pending[rf.index] = rf
		for {
			item, ok := pending[next]
			if !ok {
				break
			}
			delete(pending, next)
			next++
			ready = append(ready, item)
			if len(ready) >= perTx {
				if err := writeReady(); err != nil {
					cancel()
					return wr.out, err
				}
			}
		}
	}
	if ctx.Err() == nil {
		if err := writeReady(); err != nil {
			cancel()
			return wr.out, err
		}
	}
	if err := ctx.Err(); err != nil {
		wr.abort()
		return wr.out, err
	}
	if err := wr.flush(); err != nil {
		return wr.out, err
	}
	return wr.out, nil
}

// digestClaims remembers, per content digest, the lowest file index seen.
type digestClaims struct {
	mu sync.Mutex
	m  map[[sha256.Size]byte]int
}

func newDigestClaims() *digestClaims {
	return &digestClaims{m: map[[sha256.Size]byte]int{}}
}

// claim records index for digest and returns the lowest index known for it.
// A later file finding a lower index skips its parse; an earlier file finding
// a higher one parses anyway (the writer, in file order, decides).
func (c *digestClaims) claim(digest [sha256.Size]byte, index int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if first, ok := c.m[digest]; ok && first < index {
		return first
	}
	c.m[digest] = index
	return index
}

// pipelineWriter writes read files in order, perTx files per transaction.
type pipelineWriter struct {
	ctx    context.Context
	store  TxBeginner
	opts   PipelineOptions
	perTx  int
	tx     storage.Tx
	unlock func()
	group  []*readFile
	staged []FileOutcome
	out    []FileOutcome
	// seen maps a content digest to the outcome of the first file with it.
	seen map[[sha256.Size]byte]FileOutcome
	// groupSeen is what seen will gain if the group commits.
	groupSeen map[[sha256.Size]byte]FileOutcome
}

func (w *pipelineWriter) begin() error {
	if w.tx != nil {
		return nil
	}
	if w.opts.Lock != nil {
		w.unlock = w.opts.Lock()
	}
	tx, err := w.store.BeginTx(w.ctx)
	if err != nil {
		w.release()
		return err
	}
	w.tx = tx
	w.groupSeen = map[[sha256.Size]byte]FileOutcome{}
	return nil
}

func (w *pipelineWriter) release() {
	if w.unlock != nil {
		w.unlock()
		w.unlock = nil
	}
}

// lookup returns the outcome of an earlier file with the same bytes.
func (w *pipelineWriter) lookup(d [sha256.Size]byte) (FileOutcome, bool) {
	if o, ok := w.groupSeen[d]; ok {
		return o, true
	}
	o, ok := w.seen[d]
	return o, ok
}

func (w *pipelineWriter) add(rf *readFile) error {
	if err := w.ctx.Err(); err != nil {
		w.abort()
		return err
	}
	if err := w.begin(); err != nil {
		return err
	}
	o, err := w.write(w.tx, rf)
	if err != nil {
		if w.ctx.Err() != nil {
			w.abort()
			return w.ctx.Err()
		}
		// A write failed mid-group: the transaction holds part of that file.
		// Roll the group back and write its files one transaction each.
		return w.replay(append(w.group, rf))
	}
	w.group = append(w.group, rf)
	w.staged = append(w.staged, o)
	// A file refused after it was hashed is remembered too: its copies get its
	// error, not a generic one.
	if rf.hashed && rf.dupOf < 0 {
		w.groupSeen[rf.digest] = o
	}
	if len(w.group) >= w.perTx {
		return w.flush()
	}
	return nil
}

// write decides one file inside tx. A returned error means tx is spoiled; a
// file the pipeline could not read is an outcome, not an error.
func (w *pipelineWriter) write(tx storage.Tx, rf *readFile) (FileOutcome, error) {
	o := FileOutcome{Index: rf.index, Path: rf.path, Size: rf.size, DuplicateOf: -1}
	if rf.err != nil {
		o.Status, o.Error = FileFailed, rf.err.Error()
		return o, nil
	}
	if first, ok := w.lookup(rf.digest); ok {
		o.DuplicateOf = first.Index
		if first.Status == FileFailed {
			o.Status, o.Error = FileFailed, first.Error
			return o, nil
		}
		o.Status = FileDuplicate
		o.MatchID, o.PositionID = first.MatchID, first.PositionID
		o.Player1, o.Player2, o.Games = first.Player1, first.Player2, first.Games
		return o, nil
	}
	if rf.position != nil {
		var firstID int64
		for i := range rf.position {
			id, err := WritePosition(w.ctx, tx, w.opts.Scope, &rf.position[i])
			if err != nil {
				return o, err
			}
			if i == 0 {
				firstID = id
			}
		}
		o.Status, o.PositionID, o.Positions = FilePosition, firstID, len(rf.position)
		return o, nil
	}
	g := rf.graph
	if g == nil {
		o.Status, o.Error = FileFailed, "nothing to import"
		return o, nil
	}
	g.ImportBatchID = w.opts.ImportBatchID
	o.Player1, o.Player2, o.Games = g.Match.Player1Name, g.Match.Player2Name, len(g.Games)
	res, err := WriteMatch(w.ctx, tx, w.opts.Scope, g, nil)
	if err != nil {
		return o, err
	}
	o.MatchID = res.MatchID
	switch {
	case res.Skipped:
		o.Status, o.FlagsApplied = FileDuplicate, res.FlagsApplied
	case res.Enriched:
		o.Status, o.Positions = FileEnriched, res.SavedPositions
	default:
		o.Status, o.Positions = FileImported, res.SavedPositions
	}
	// The graph is no longer needed once written; drop it before the group
	// commits so a group of large files does not stay in memory.
	rf.graph = nil
	return o, nil
}

func (w *pipelineWriter) flush() error {
	if w.tx == nil {
		return nil
	}
	err := w.tx.Commit()
	w.tx = nil
	if err != nil {
		w.release()
		w.group, w.staged = nil, nil
		return err
	}
	w.commitStaged()
	w.release()
	return nil
}

// commitStaged publishes the staged outcomes once their transaction holds.
func (w *pipelineWriter) commitStaged() {
	for d, o := range w.groupSeen {
		w.seen[d] = o
	}
	if w.opts.OnCommit != nil && len(w.staged) > 0 {
		w.opts.OnCommit(w.staged)
	}
	w.out = append(w.out, w.staged...)
	w.group, w.staged, w.groupSeen = nil, nil, nil
}

func (w *pipelineWriter) abort() {
	if w.tx != nil {
		_ = w.tx.Rollback()
		w.tx = nil
	}
	w.release()
	w.group, w.staged, w.groupSeen = nil, nil, nil
}

// replay rolls the group back and writes each of its files in a transaction
// of its own, so one refused file costs only itself. Graphs already written
// were dropped, and a write can leave ids on a graph: every file is read again.
func (w *pipelineWriter) replay(files []*readFile) error {
	w.abort()
	for _, rf := range files {
		if rf.err == nil && rf.dupOf < 0 {
			rf.graph, rf.position, rf.err = mapFile(rf.path)
		}
		if err := w.begin(); err != nil {
			return err
		}
		o, err := w.write(w.tx, rf)
		if err != nil {
			if w.ctx.Err() != nil {
				w.abort()
				return w.ctx.Err()
			}
			w.abort()
			o = FileOutcome{Index: rf.index, Path: rf.path, Size: rf.size, Status: FileFailed, Error: err.Error(), DuplicateOf: -1}
			// A file refused at write time is decided: record it outside any
			// transaction, as a group of one with nothing to commit.
			if w.opts.Lock != nil {
				w.unlock = w.opts.Lock()
			}
			w.staged = []FileOutcome{o}
			w.groupSeen = map[[sha256.Size]byte]FileOutcome{rf.digest: o}
			w.commitStaged()
			w.release()
			continue
		}
		w.staged = append(w.staged, o)
		if rf.err == nil && rf.dupOf < 0 {
			w.groupSeen[rf.digest] = o
		}
		if err := w.flush(); err != nil {
			return err
		}
	}
	return nil
}
