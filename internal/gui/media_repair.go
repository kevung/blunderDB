package gui

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/videomux"
)

// VideoRepair is what RepairVideoMux produced: the sound copy to play, or why there is none.
type VideoRepair struct {
	// Path is the copy whose seeks land; "" when the repair did not happen.
	Path string `json:"path"`
	// NoFFmpeg says the repair needs ffmpeg and none is on the PATH.
	NoFFmpeg bool `json:"noFFmpeg"`
	// Cancelled says every pane waiting on it closed or changed source.
	Cancelled bool `json:"cancelled"`
	// Reason is the failure, one line; "" on success, cancellation or without ffmpeg.
	Reason string `json:"reason"`
}

// repairJob is the one repair running for a file: every pane that asks for it waits on the
// same run, and the run stops when the last of them gives it up.
type repairJob struct {
	done    chan struct{}
	cancel  context.CancelFunc
	holders int
	// stopping: cancelled, its ffmpeg still exiting; a new request waits for it to end so
	// two runs never write the same copy.
	stopping bool
	result   VideoRepair
}

// repairOptions lets tests point the repair at a stand-in ffmpeg and a private cache.
var repairOptions = videomux.RepairOptions{}

// RepairVideoMux makes a sound copy of a .webm/.mkv video a pane is serving (registered
// through MediaURL) and returns it once checked; the original is only read. A pane asking
// while the same file is being repaired waits on that run. The call blocks for as long as
// ffmpeg runs, off the interface thread: Wails answers every binding from its own goroutine.
func (a *App) RepairVideoMux(path string) VideoRepair {
	if !videomux.IsMatroskaPath(path) || a.VideoSourceKind(path) != "file" {
		return VideoRepair{Reason: "not a local Matroska or WebM file"}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return VideoRepair{Reason: err.Error()}
	}
	h := &a.media
	h.mu.Lock()
	if _, served := h.byPath[abs]; !served {
		h.mu.Unlock()
		return VideoRepair{Reason: "the video is not open"}
	}
	job, running := h.repairs[abs]
	for running && job.stopping {
		h.mu.Unlock()
		<-job.done
		h.mu.Lock()
		job, running = h.repairs[abs]
	}
	if !running {
		ctx, cancel := context.WithCancel(context.Background())
		job = &repairJob{done: make(chan struct{}), cancel: cancel}
		if h.repairs == nil {
			h.repairs = map[string]*repairJob{}
		}
		h.repairs[abs] = job
		go h.runRepair(ctx, abs, job)
	}
	job.holders++
	h.mu.Unlock()
	<-job.done
	return job.result
}

func (h *mediaHost) runRepair(ctx context.Context, abs string, job *repairJob) {
	out, err := videomux.Repair(ctx, abs, repairOptions)
	switch {
	case err == nil:
		job.result = VideoRepair{Path: out}
	case errors.Is(err, videomux.ErrNoFFmpeg):
		job.result = VideoRepair{NoFFmpeg: true}
	case ctx.Err() != nil:
		job.result = VideoRepair{Cancelled: true}
	default:
		job.result = VideoRepair{Reason: err.Error()}
	}
	h.mu.Lock()
	if h.repairs[abs] == job {
		delete(h.repairs, abs)
	}
	h.mu.Unlock()
	job.cancel()
	close(job.done)
}

// CancelVideoRepair gives up one pane's interest in the repair of path (it closed or changed
// source); the run stops when no pane waits on it any more. Nothing running: ignored.
func (a *App) CancelVideoRepair(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	h := &a.media
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dropRepairLocked(abs, false)
}

// dropRepairLocked takes one holder off the repair of abs, or all of them; called with h.mu
// held.
func (h *mediaHost) dropRepairLocked(abs string, all bool) {
	job, ok := h.repairs[abs]
	if !ok {
		return
	}
	if job.holders--; all || job.holders <= 0 {
		job.stopping = true
		job.cancel()
	}
}
