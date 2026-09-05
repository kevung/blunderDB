package bearoffgen

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Pausing a two-sided run means writing its state down, so that resuming
// continues the sweep instead of restarting it. The difference is not
// cosmetic: TS-06-11 is about half an hour of arithmetic on one core, and the
// domains beyond it are hours. A button labelled "Reprendre" that silently
// started over would be a lie the user only discovers by watching the
// percentage.
//
// The state is the table so far plus the diagonal reached — nothing else,
// since a diagonal reads only diagonals below it. So a checkpoint is the
// table's own bytes with a small header in front, and costs one write of the
// size the finished file will have.
//
// It lives next to the table as `<name>.ckpt`, a name neither Resolve nor
// Verify will ever mistake for a table. A `.part` remains what it always was:
// the debris of a run that died, of no use to anyone.

const (
	checkpointMagic  = "blunderDB-bearoff-checkpoint-1\n"
	checkpointHeader = 40 // magic, padded, then points, checkers, diagonal
)

// CheckpointPath is where a paused run's state lives for a domain.
func CheckpointPath(dir string, d Domain) string {
	return filepath.Join(dir, d.FileName()+".ckpt")
}

// ErrNoCheckpoint says there is no resumable state for this domain — the
// ordinary case, not a failure.
var ErrNoCheckpoint = errors.New("bearoffgen: no checkpoint")

func checkpointHead(st *TwoSidedState) []byte {
	head := make([]byte, checkpointHeader)
	copy(head, checkpointMagic)
	head[32] = byte(st.Points)
	head[33] = byte(st.Checkers)
	binary.LittleEndian.PutUint32(head[34:], uint32(st.Diagonal))
	return head
}

// WriteCheckpoint writes a paused run's state beside its table.
//
// It is written through a temporary file and renamed: a checkpoint half
// written when the machine goes down must not replace the one that was whole.
func WriteCheckpoint(dir string, st *TwoSidedState) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	d := Domain{Kind: TwoSidedKind, Points: st.Points, Checkers: st.Checkers}
	path := CheckpointPath(dir, d)

	f, err := os.CreateTemp(dir, "ckpt-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()

	if _, err := f.Write(checkpointHead(st)); err != nil {
		f.Close()
		return err
	}
	buf := make([]byte, 1<<16)
	for i := 0; i < len(st.Body); {
		k := 0
		for ; k < len(buf)-1 && i < len(st.Body); k, i = k+2, i+1 {
			binary.LittleEndian.PutUint16(buf[k:], uint16(st.Body[i]))
		}
		if _, err := f.Write(buf[:k]); err != nil {
			f.Close()
			return err
		}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// CheckpointProgress reads only a checkpoint's header, which is what a list of
// interrupted runs needs: the domain and how far it got. It never reads the
// body, so it stays instant on a 1.2 GB file.
func CheckpointProgress(dir string, d Domain) (done, total int64, err error) {
	f, err := os.Open(CheckpointPath(dir, d))
	if err != nil {
		return 0, 0, ErrNoCheckpoint
	}
	defer f.Close()

	head := make([]byte, checkpointHeader)
	if _, err := f.Read(head); err != nil {
		return 0, 0, ErrNoCheckpoint
	}
	if string(head[:len(checkpointMagic)]) != checkpointMagic {
		return 0, 0, ErrNoCheckpoint
	}
	if int(head[32]) != d.Points || int(head[33]) != d.Checkers {
		return 0, 0, ErrNoCheckpoint
	}
	n := NumPositions(d.Points, d.Checkers)
	diagonal := int(binary.LittleEndian.Uint32(head[34:]))
	if diagonal < 0 || diagonal > 2*n-1 {
		return 0, 0, ErrNoCheckpoint
	}
	return pairsThroughDiagonal(n, diagonal), int64(n) * int64(n), nil
}

// ReadCheckpoint loads a paused run's state. A file whose size contradicts its
// own header is discarded rather than resumed: half a table read as whole
// would silently produce a wrong one, and the run is only worth minutes.
func ReadCheckpoint(dir string, d Domain) (*TwoSidedState, error) {
	path := CheckpointPath(dir, d)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrNoCheckpoint
	}
	n := NumPositions(d.Points, d.Checkers)
	entries := int64(n) * int64(n) * planeCount
	if int64(len(raw)) != int64(checkpointHeader)+entries*2 {
		return nil, fmt.Errorf("bearoffgen: checkpoint %s is %d bytes, want %d", path, len(raw), int64(checkpointHeader)+entries*2)
	}
	if string(raw[:len(checkpointMagic)]) != checkpointMagic {
		return nil, fmt.Errorf("bearoffgen: %s is not a checkpoint", path)
	}
	if int(raw[32]) != d.Points || int(raw[33]) != d.Checkers {
		return nil, fmt.Errorf("bearoffgen: checkpoint %s holds another domain", path)
	}
	diagonal := int(binary.LittleEndian.Uint32(raw[34:]))
	if diagonal < 0 || diagonal > 2*n-1 {
		return nil, fmt.Errorf("bearoffgen: checkpoint %s stops at diagonal %d", path, diagonal)
	}

	st := &TwoSidedState{Points: d.Points, Checkers: d.Checkers, Diagonal: diagonal, Body: make([]int16, entries)}
	body := raw[checkpointHeader:]
	for i := range st.Body {
		st.Body[i] = int16(binary.LittleEndian.Uint16(body[i*2:]))
	}
	return st, nil
}

// RemoveCheckpoint drops a paused run's state — what "Supprimer" does to an
// interrupted run, and what finishing one does on its way out.
func RemoveCheckpoint(dir string, d Domain) error {
	err := os.Remove(CheckpointPath(dir, d))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
