package videomux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// EBML element IDs, with their marker bits, as the Matroska specification writes them.
const (
	idEBML           = 0x1A45DFA3
	idSegment        = 0x18538067
	idSeekHead       = 0x114D9B74
	idSeek           = 0x4DBB
	idSeekID         = 0x53AB
	idSeekPosition   = 0x53AC
	idInfo           = 0x1549A966
	idTimestampScale = 0x2AD7B1
	idTracks         = 0x1654AE6B
	idTrackEntry     = 0xAE
	idTrackNumber    = 0xD7
	idTrackType      = 0x83
	idCues           = 0x1C53BB6B
	idCuePoint       = 0xBB
	idCueTrackPos    = 0xB7
	idCueTrack       = 0xF7
	idCueClusterPos  = 0xF1
	idCluster        = 0x1F43B675
	idClusterTime    = 0xE7
	idSimpleBlock    = 0xA3
	idBlockGroup     = 0xA0
	idBlock          = 0xA1
	idReferenceBlock = 0xFB
)

const trackTypeVideo = 1

// Bounds on what one check may read. A cue table lists one point per keyframe at most; a few
// hours of video hold tens of thousands. Clusters open with a timestamp and maybe a few audio
// blocks before the first video block.
const (
	maxMasterSize        = 16 << 20 // SeekHead, Info, Tracks or Cues read whole
	maxClusters          = 100_000
	maxElementsInCluster = 64
	maxTopLevel          = 4096 // top-level elements walked when no SeekHead names the Cues
	maxBlockDepth        = 8    // elements read inside a BlockGroup before its Block
	// DefaultTimeout bounds CheckFile when its context sets no deadline.
	DefaultTimeout = 10 * time.Second
)

// ErrNotMatroska is returned for a file that does not open with an EBML header.
var ErrNotMatroska = errors.New("videomux: not a Matroska/WebM file")

// Report is what Check found.
type Report struct {
	// Clusters is how many distinct clusters the cues reference and Check examined.
	Clusters int
	// Faulty holds the start, in the file's timeline, of each referenced cluster whose first
	// video block is not a keyframe: a seek near these instants freezes in WebKitGTK.
	Faulty []time.Duration
	// NoCues is true when the file has no cue table: nothing can be told about its seeks.
	NoCues bool
	// Partial is true when a bound or the deadline stopped the check before every cluster
	// was examined.
	Partial bool
}

// IsMatroskaPath reports whether path names a Matroska or WebM file by its extension.
func IsMatroskaPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".webm", ".mkv":
		return true
	}
	return false
}

// CheckFile opens path and runs Check on it, within DefaultTimeout when ctx sets no deadline.
func CheckFile(ctx context.Context, path string) (Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return Report{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Report{}, err
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}
	return Check(ctx, f, st.Size())
}

// Check examines the first video block of every cluster the cues of r reference.
func Check(ctx context.Context, r io.ReaderAt, size int64) (Report, error) {
	p := &parser{r: r, size: size, ctx: ctx}
	return p.check()
}

type parser struct {
	r    io.ReaderAt
	size int64
	ctx  context.Context
}

// element is an EBML element header: its ID, where its data starts, and its data size
// (-1 when the size is unknown, as a live-written Segment or Cluster may declare).
type element struct {
	id         uint64
	dataStart  int64
	dataSize   int64
	headerSize int64
}

func (e element) end(limit int64) int64 {
	if e.dataSize < 0 || e.dataStart+e.dataSize > limit {
		return limit
	}
	return e.dataStart + e.dataSize
}

var errMalformed = errors.New("videomux: malformed element")

// readHeader reads the element header at off, which must lie before limit.
func (p *parser) readHeader(off, limit int64) (element, error) {
	if off < 0 || off >= limit || off >= p.size {
		return element{}, io.ErrUnexpectedEOF
	}
	var buf [12]byte
	n, err := p.r.ReadAt(buf[:], off)
	if n == 0 {
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return element{}, err
	}
	b := buf[:n]
	id, idLen, ok := vint(b, true)
	if !ok || idLen > 4 {
		return element{}, errMalformed
	}
	sz, szLen, ok := vint(b[idLen:], false)
	if !ok {
		return element{}, errMalformed
	}
	e := element{id: id, dataStart: off + int64(idLen+szLen), headerSize: int64(idLen + szLen)}
	if sz == unknownSize(szLen) {
		e.dataSize = -1
	} else if sz > uint64(1<<62) {
		return element{}, errMalformed
	} else {
		e.dataSize = int64(sz)
	}
	return e, nil
}

func unknownSize(length int) uint64 { return 1<<(7*uint(length)) - 1 }

// vint decodes an EBML variable-length integer. keepMarker keeps the length marker bit, as
// element IDs are written; sizes and track numbers drop it.
func vint(b []byte, keepMarker bool) (uint64, int, bool) {
	if len(b) == 0 || b[0] == 0 {
		return 0, 0, false
	}
	length := 1
	for mask := byte(0x80); b[0]&mask == 0; mask >>= 1 {
		length++
	}
	if length > 8 || len(b) < length {
		return 0, 0, false
	}
	v := uint64(b[0])
	if !keepMarker {
		v &= uint64(0xFF >> uint(length))
	}
	for i := 1; i < length; i++ {
		v = v<<8 | uint64(b[i])
	}
	return v, length, true
}

// readData reads an element's whole data, refusing anything above maxMasterSize.
func (p *parser) readData(e element) ([]byte, error) {
	if e.dataSize < 0 || e.dataSize > maxMasterSize {
		return nil, errMalformed
	}
	if e.dataStart+e.dataSize > p.size {
		return nil, io.ErrUnexpectedEOF
	}
	buf := make([]byte, e.dataSize)
	if _, err := p.r.ReadAt(buf, e.dataStart); err != nil && !(errors.Is(err, io.EOF) && len(buf) > 0) {
		return nil, err
	}
	return buf, nil
}

// children walks the elements packed in b, calling fn with each one's ID and data.
func children(b []byte, fn func(id uint64, data []byte) error) error {
	for len(b) > 0 {
		id, idLen, ok := vint(b, true)
		if !ok || idLen > 4 {
			return errMalformed
		}
		sz, szLen, ok := vint(b[idLen:], false)
		if !ok {
			return errMalformed
		}
		start := idLen + szLen
		if sz == unknownSize(szLen) || sz > uint64(len(b)-start) {
			return errMalformed
		}
		end := start + int(sz)
		if err := fn(id, b[start:end]); err != nil {
			return err
		}
		b = b[end:]
	}
	return nil
}

func uintOf(b []byte) uint64 {
	var v uint64
	for i, c := range b {
		if i == 8 {
			break
		}
		v = v<<8 | uint64(c)
	}
	return v
}

func (p *parser) check() (Report, error) {
	head, err := p.readHeader(0, p.size)
	if err != nil || head.id != idEBML {
		return Report{}, ErrNotMatroska
	}
	seg, err := p.readHeader(head.end(p.size), p.size)
	if err != nil {
		return Report{}, err
	}
	if seg.id != idSegment {
		return Report{}, ErrNotMatroska
	}
	segEnd := seg.end(p.size)

	top := map[uint64]int64{} // first offset of each top-level element of interest
	if err := p.locateTopLevel(seg, segEnd, top); err != nil {
		return Report{}, err
	}

	scale := uint64(1_000_000)
	if off, ok := top[idInfo]; ok {
		if s, err := p.timestampScale(off, segEnd); err == nil && s > 0 {
			scale = s
		}
	}
	video, err := p.videoTrack(top, segEnd)
	if err != nil {
		return Report{}, err
	}
	cuesOff, ok := top[idCues]
	if !ok {
		return Report{NoCues: true}, nil
	}
	clusters, partial, err := p.cueClusters(cuesOff, segEnd, video)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Partial: partial}
	for _, rel := range clusters {
		if p.ctx.Err() != nil {
			rep.Partial = true
			break
		}
		off := seg.dataStart + int64(rel)
		ts, key, found, err := p.firstVideoBlock(off, segEnd, video)
		if err != nil {
			rep.Partial = true
			continue
		}
		if !found {
			continue
		}
		rep.Clusters++
		if !key {
			rep.Faulty = append(rep.Faulty, time.Duration(ts*scale))
		}
	}
	return rep, nil
}

// locateTopLevel fills top with the offsets of Info, Tracks, Cues and the first Cluster: from
// the SeekHead when there is one, and from a walk of the top-level elements for what it
// leaves out. The walk stops at the first Cluster, unless the Cues are still missing, in which
// case it skips over clusters by their sizes.
func (p *parser) locateTopLevel(seg element, segEnd int64, top map[uint64]int64) error {
	off := seg.dataStart
	for i := 0; i < maxTopLevel && off < segEnd; i++ {
		if err := p.ctx.Err(); err != nil {
			return err
		}
		e, err := p.readHeader(off, segEnd)
		if err != nil {
			break
		}
		switch e.id {
		case idSeekHead:
			if _, seen := top[idSeekHead]; !seen {
				top[idSeekHead] = off
				p.readSeekHead(e, seg.dataStart, top)
			}
		case idInfo, idTracks, idCues, idCluster:
			if _, seen := top[e.id]; !seen {
				top[e.id] = off
			}
		}
		_, haveCues := top[idCues]
		_, haveTracks := top[idTracks]
		if e.id == idCluster && haveCues && haveTracks {
			break
		}
		if e.dataSize < 0 {
			break
		}
		off = e.dataStart + e.dataSize
	}
	return nil
}

func (p *parser) readSeekHead(e element, segData int64, top map[uint64]int64) {
	data, err := p.readData(e)
	if err != nil {
		return
	}
	_ = children(data, func(id uint64, seek []byte) error {
		if id != idSeek {
			return nil
		}
		var target uint64
		var pos int64 = -1
		_ = children(seek, func(id uint64, v []byte) error {
			switch id {
			case idSeekID:
				target = uintOf(v)
			case idSeekPosition:
				if u := uintOf(v); u < uint64(p.size) {
					pos = int64(u)
				}
			}
			return nil
		})
		if pos < 0 {
			return nil
		}
		switch target {
		case idInfo, idTracks, idCues:
			if _, seen := top[target]; !seen {
				top[target] = segData + pos
			}
		}
		return nil
	})
}

func (p *parser) master(off, segEnd int64, want uint64) ([]byte, error) {
	e, err := p.readHeader(off, segEnd)
	if err != nil {
		return nil, err
	}
	if e.id != want {
		return nil, errMalformed
	}
	return p.readData(e)
}

func (p *parser) timestampScale(off, segEnd int64) (uint64, error) {
	data, err := p.master(off, segEnd, idInfo)
	if err != nil {
		return 0, err
	}
	var scale uint64
	_ = children(data, func(id uint64, v []byte) error {
		if id == idTimestampScale {
			scale = uintOf(v)
		}
		return nil
	})
	return scale, nil
}

// videoTrack returns the number of the first video track, 0 when the file names none (every
// block then counts).
func (p *parser) videoTrack(top map[uint64]int64, segEnd int64) (uint64, error) {
	off, ok := top[idTracks]
	if !ok {
		return 0, nil
	}
	data, err := p.master(off, segEnd, idTracks)
	if err != nil {
		return 0, nil
	}
	var video uint64
	_ = children(data, func(id uint64, entry []byte) error {
		if id != idTrackEntry || video != 0 {
			return nil
		}
		var number, kind uint64
		_ = children(entry, func(id uint64, v []byte) error {
			switch id {
			case idTrackNumber:
				number = uintOf(v)
			case idTrackType:
				kind = uintOf(v)
			}
			return nil
		})
		if kind == trackTypeVideo {
			video = number
		}
		return nil
	})
	return video, nil
}

// cueClusters returns the distinct cluster positions (relative to the Segment's data) the
// cues give for the video track, in file order.
func (p *parser) cueClusters(off, segEnd int64, video uint64) ([]uint64, bool, error) {
	data, err := p.master(off, segEnd, idCues)
	if err != nil {
		return nil, false, fmt.Errorf("videomux: reading the cues: %w", err)
	}
	seen := map[uint64]bool{}
	var out []uint64
	partial := false
	_ = children(data, func(id uint64, point []byte) error {
		if id != idCuePoint {
			return nil
		}
		return children(point, func(id uint64, pos []byte) error {
			if id != idCueTrackPos {
				return nil
			}
			var track, cluster uint64
			hasCluster := false
			_ = children(pos, func(id uint64, v []byte) error {
				switch id {
				case idCueTrack:
					track = uintOf(v)
				case idCueClusterPos:
					cluster, hasCluster = uintOf(v), true
				}
				return nil
			})
			if !hasCluster || (video != 0 && track != video) || seen[cluster] {
				return nil
			}
			if len(out) >= maxClusters {
				partial = true
				return errMalformed
			}
			seen[cluster] = true
			out = append(out, cluster)
			return nil
		})
	})
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, partial, nil
}

// firstVideoBlock reads the cluster at off up to its first block of the video track and
// returns the cluster's timestamp and whether that block is a keyframe. found is false when
// the cluster holds no video block within the bound.
func (p *parser) firstVideoBlock(off, segEnd int64, video uint64) (ts uint64, key, found bool, err error) {
	cl, err := p.readHeader(off, segEnd)
	if err != nil {
		return 0, false, false, err
	}
	if cl.id != idCluster {
		return 0, false, false, errMalformed
	}
	end := cl.end(segEnd)
	pos := cl.dataStart
	for i := 0; i < maxElementsInCluster && pos < end; i++ {
		e, err := p.readHeader(pos, end)
		if err != nil {
			return 0, false, false, err
		}
		if e.dataSize < 0 {
			return 0, false, false, errMalformed
		}
		switch e.id {
		case idClusterTime:
			if e.dataSize > 8 {
				return 0, false, false, errMalformed
			}
			b, err := p.readData(e)
			if err != nil {
				return 0, false, false, err
			}
			ts = uintOf(b)
		case idCluster:
			// The next cluster with an unknown-size one before it: this one held no block.
			return ts, false, false, nil
		case idSimpleBlock:
			track, flags, err := p.blockHeader(e)
			if err != nil {
				return 0, false, false, err
			}
			if video == 0 || track == video {
				return ts, flags&0x80 != 0, true, nil
			}
		case idBlockGroup:
			track, ref, err := p.blockGroup(e)
			if err != nil {
				return 0, false, false, err
			}
			if video == 0 || track == video {
				return ts, !ref, true, nil
			}
		}
		pos = e.dataStart + e.dataSize
	}
	return ts, false, false, nil
}

// blockHeader reads a block's track number and flags byte.
func (p *parser) blockHeader(e element) (track uint64, flags byte, err error) {
	var buf [11]byte
	n := int64(len(buf))
	if e.dataSize < n {
		n = e.dataSize
	}
	if e.dataStart+n > p.size {
		return 0, 0, io.ErrUnexpectedEOF
	}
	if _, err := p.r.ReadAt(buf[:n], e.dataStart); err != nil && !errors.Is(err, io.EOF) {
		return 0, 0, err
	}
	track, l, ok := vint(buf[:n], false)
	if !ok || int64(l)+3 > n {
		return 0, 0, errMalformed
	}
	return track, buf[l+2], nil
}

// blockGroup reads a BlockGroup's track number and whether it references another frame: a
// block that references none is a keyframe.
func (p *parser) blockGroup(g element) (track uint64, references bool, err error) {
	end := g.end(p.size)
	pos := g.dataStart
	gotTrack := false
	for i := 0; i < maxBlockDepth && pos < end; i++ {
		e, err := p.readHeader(pos, end)
		if err != nil {
			return 0, false, err
		}
		if e.dataSize < 0 {
			return 0, false, errMalformed
		}
		switch e.id {
		case idBlock:
			track, _, err = p.blockHeader(e)
			if err != nil {
				return 0, false, err
			}
			gotTrack = true
		case idReferenceBlock:
			references = true
		}
		pos = e.dataStart + e.dataSize
	}
	if !gotTrack {
		return 0, false, errMalformed
	}
	return track, references, nil
}

// RemuxCommand is the ffmpeg command that rewrites in with clusters long enough to start on
// keyframes, without re-encoding, quoted for the shell of goos ("windows" or any other).
func RemuxCommand(in, goos string) string {
	return fmt.Sprintf("ffmpeg -i %s -map 0 -c copy -cluster_time_limit 30000 %s",
		quote(in, goos), quote(RemuxOutput(in), goos))
}

// RemuxOutput names the repaired copy beside in: "x.webm" gives "x.remux.webm".
func RemuxOutput(in string) string {
	ext := filepath.Ext(in)
	return strings.TrimSuffix(in, ext) + ".remux" + ext
}

func quote(s, goos string) string {
	if goos == "windows" {
		// A Windows path cannot hold a double quote; cmd and PowerShell both take this.
		return `"` + s + `"`
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
