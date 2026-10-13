package videomux

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"
)

// Hand-built files exercise what ffmpeg does not write for the fixtures: BlockGroups, and
// clusters of unknown size reached through a SeekHead.

func idBytes(id uint64) []byte {
	var b []byte
	for v := id; v > 0; v >>= 8 {
		b = append([]byte{byte(v)}, b...)
	}
	return b
}

// el encodes an element with an 8-byte size, or the unknown size when unknown is set.
func el(id uint64, unknown bool, children ...[]byte) []byte {
	data := bytes.Join(children, nil)
	size := make([]byte, 8)
	if unknown {
		binary.BigEndian.PutUint64(size, 0x01FFFFFFFFFFFFFF)
	} else {
		binary.BigEndian.PutUint64(size, uint64(len(data))|0x0100000000000000)
	}
	return append(append(idBytes(id), size...), data...)
}

func uintEl(id, v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return el(id, false, b)
}

func rawEl(id uint64, data []byte) []byte { return el(id, false, data) }

// block is a Block or SimpleBlock body: track 1 or 2 as a one-byte vint, timecode 0, flags.
func block(track byte, flags byte) []byte { return []byte{0x80 | track, 0, 0, flags, 0xAA, 0xBB} }

type synthCluster struct {
	unknown bool
	time    uint64
	blocks  [][]byte
}

// synthFile lays out EBML, then a Segment of SeekHead, Info, Tracks, the clusters and the
// Cues (one cue per cluster, on track 1). The SeekHead names Info, Tracks and Cues.
func synthFile(clusters []synthCluster, segUnknown bool) []byte {
	info := el(idInfo, false, uintEl(idTimestampScale, 1_000_000))
	tracks := el(idTracks, false,
		el(idTrackEntry, false, uintEl(idTrackNumber, 2), uintEl(idTrackType, 2)),
		el(idTrackEntry, false, uintEl(idTrackNumber, 1), uintEl(idTrackType, trackTypeVideo)))
	seekHead := func(infoPos, tracksPos, cuesPos uint64) []byte {
		seek := func(id, pos uint64) []byte {
			return el(idSeek, false, rawEl(idSeekID, idBytes(id)), uintEl(idSeekPosition, pos))
		}
		return el(idSeekHead, false, seek(idInfo, infoPos), seek(idTracks, tracksPos), seek(idCues, cuesPos))
	}
	pos := uint64(len(seekHead(0, 0, 0)))
	infoPos := pos
	pos += uint64(len(info))
	tracksPos := pos
	pos += uint64(len(tracks))
	var body [][]byte
	var cues [][]byte
	for _, c := range clusters {
		children := [][]byte{uintEl(idClusterTime, c.time)}
		children = append(children, c.blocks...)
		enc := el(idCluster, c.unknown, children...)
		cues = append(cues, el(idCuePoint, false, uintEl(0xB3, c.time),
			el(idCueTrackPos, false, uintEl(idCueTrack, 1), uintEl(idCueClusterPos, pos))))
		body = append(body, enc)
		pos += uint64(len(enc))
	}
	segment := [][]byte{seekHead(infoPos, tracksPos, pos), info, tracks}
	segment = append(segment, body...)
	segment = append(segment, el(idCues, false, cues...))
	head := el(idEBML, false, uintEl(0x4286, 1))
	return append(head, el(idSegment, segUnknown, segment...)...)
}

func checkBytes(t *testing.T, data []byte) Report {
	t.Helper()
	rep, err := Check(context.Background(), bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func TestCheck_BlockGroups(t *testing.T) {
	audio := el(idSimpleBlock, false, block(2, 0x80))
	data := synthFile([]synthCluster{
		// Keyframe: a BlockGroup with no ReferenceBlock, after an audio block.
		{time: 0, blocks: [][]byte{audio, el(idBlockGroup, false, rawEl(idBlock, block(1, 0)))}},
		// Delta: the BlockGroup references an earlier frame.
		{time: 4000, blocks: [][]byte{el(idBlockGroup, false, rawEl(idBlock, block(1, 0)), uintEl(idReferenceBlock, 1))}},
		{time: 9000, blocks: [][]byte{el(idSimpleBlock, false, block(1, 0x80))}},
	}, false)
	rep := checkBytes(t, data)
	if rep.Clusters != 3 || len(rep.Faulty) != 1 || rep.Faulty[0] != 4*time.Second {
		t.Errorf("report = %+v, want 3 clusters, the one at 4 s faulty", rep)
	}
}

func TestCheck_UnknownSizes(t *testing.T) {
	data := synthFile([]synthCluster{
		{unknown: true, time: 0, blocks: [][]byte{el(idSimpleBlock, false, block(1, 0x80))}},
		{unknown: true, time: 2000, blocks: [][]byte{el(idSimpleBlock, false, block(2, 0x80)), el(idSimpleBlock, false, block(1, 0))}},
		// No block before the next cluster: nothing to tell about this one.
		{unknown: true, time: 3000},
		{unknown: true, time: 5000, blocks: [][]byte{el(idSimpleBlock, false, block(1, 0x80))}},
	}, true)
	rep := checkBytes(t, data)
	if rep.Clusters != 3 || len(rep.Faulty) != 1 || rep.Faulty[0] != 2*time.Second {
		t.Errorf("report = %+v, want 3 clusters with a video block, the one at 2 s faulty", rep)
	}
	for n := len(data) - 1; n > 0; n -= 7 {
		_, _ = Check(context.Background(), bytes.NewReader(data[:n]), int64(n))
	}
}
