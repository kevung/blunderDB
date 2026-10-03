package ingest

import (
	"sync"
	"time"
)

// BatchProgress is where an ImportFiles run stands. Rate and ETA are
// measured on bytes read: a file's size predicts its parse time far better
// than its rank in the list.
type BatchProgress struct {
	FilesDone       int     `json:"filesDone"`
	FilesTotal      int     `json:"filesTotal"`
	Imported        int     `json:"imported"`
	Duplicates      int     `json:"duplicates"`
	Failed          int     `json:"failed"`
	Positions       int     `json:"positions"`
	BytesRead       int64   `json:"bytesRead"`
	BytesTotal      int64   `json:"bytesTotal"`
	ElapsedSeconds  float64 `json:"elapsedSeconds"`
	PositionsPerSec float64 `json:"positionsPerSec"`
	// ETASeconds is -1 until enough has been read to say anything.
	ETASeconds float64 `json:"etaSeconds"`
	// CurrentFile is the last file decided.
	CurrentFile string `json:"currentFile"`
	Done        bool   `json:"done"`
}

// ProgressInterval bounds how often a ProgressMeter emits: four times a
// second is smooth to the eye and costs nothing over an hours-long import.
const ProgressInterval = 250 * time.Millisecond

// ProgressMeter folds read sizes and file outcomes into BatchProgress and
// emits it at most once per interval, plus once at the end. Safe for the
// pipeline's concurrent readers and its writer.
type ProgressMeter struct {
	mu       sync.Mutex
	p        BatchProgress
	start    time.Time
	last     time.Time
	interval time.Duration
	emit     func(BatchProgress)
	now      func() time.Time
}

// NewProgressMeter starts a meter over files files of bytesTotal bytes.
func NewProgressMeter(files int, bytesTotal int64, interval time.Duration, emit func(BatchProgress)) *ProgressMeter {
	m := &ProgressMeter{interval: interval, emit: emit, now: time.Now}
	m.start = m.now()
	m.p = BatchProgress{FilesTotal: files, BytesTotal: bytesTotal, ETASeconds: -1}
	return m
}

// Read records size bytes read.
func (m *ProgressMeter) Read(size int64) {
	m.mu.Lock()
	m.p.BytesRead += size
	m.maybeEmit(false)
	m.mu.Unlock()
}

// File records one decided file.
func (m *ProgressMeter) File(o FileOutcome) {
	m.mu.Lock()
	m.p.FilesDone++
	m.p.CurrentFile = o.Path
	switch o.Status {
	case FileDuplicate:
		m.p.Duplicates++
	case FileFailed:
		m.p.Failed++
	default:
		m.p.Imported++
	}
	m.p.Positions += o.Positions
	m.maybeEmit(false)
	m.mu.Unlock()
}

// Finish emits the final state, whatever the interval.
func (m *ProgressMeter) Finish() BatchProgress {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.p.Done = true
	m.maybeEmit(true)
	return m.p
}

func (m *ProgressMeter) maybeEmit(force bool) {
	now := m.now()
	if !force && now.Sub(m.last) < m.interval {
		return
	}
	m.last = now
	el := now.Sub(m.start).Seconds()
	m.p.ElapsedSeconds = el
	if el > 0 {
		m.p.PositionsPerSec = float64(m.p.Positions) / el
	}
	m.p.ETASeconds = -1
	if m.p.BytesRead > 0 && m.p.BytesTotal > 0 && el > 1 {
		m.p.ETASeconds = el * float64(m.p.BytesTotal-m.p.BytesRead) / float64(m.p.BytesRead)
	}
	if m.emit != nil {
		m.emit(m.p)
	}
}
