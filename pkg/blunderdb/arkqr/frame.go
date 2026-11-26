package arkqr

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
)

// A Frame is what one QR code carries: one coded match — or one part of a
// match too long for a code —, or the parity of a group of ParityGroup data
// frames. Each frame names its archive, its rank and the archive's size, and
// ends with a CRC-32 of everything before it.
type Frame struct {
	Archive uint32
	Parity  bool
	// Index is the rank of a data frame, or the group of a parity frame.
	Index int
	// Total is the number of data frames of the archive.
	Total int
	// Body is a data frame's part number, part count and payload, or a
	// parity frame's body lengths and XOR.
	Body []byte
}

const (
	frameMagic = 0xA7 // and frame layout 1
	headerLen  = 10
	// ParityGroup is the number of data frames one parity frame covers.
	ParityGroup = 5
	// maxPart bounds the payload of one frame, so the densest code stays
	// readable on paper.
	maxPart = 1000
)

// ErrFrame is a QR code whose content is not a frame of an archive.
var ErrFrame = errors.New("arkqr: not an archive code")

// Bytes is the frame as a code carries it.
func (f Frame) Bytes() []byte {
	b := make([]byte, headerLen, headerLen+len(f.Body)+4)
	b[0] = frameMagic
	if f.Parity {
		b[1] = 1
	}
	binary.BigEndian.PutUint32(b[2:], f.Archive)
	binary.BigEndian.PutUint16(b[6:], uint16(f.Index))
	binary.BigEndian.PutUint16(b[8:], uint16(f.Total))
	b = append(b, f.Body...)
	return binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(b))
}

// ParseFrame reads a frame and checks its CRC.
func ParseFrame(b []byte) (Frame, error) {
	if len(b) < headerLen+4 || b[0] != frameMagic || b[1] > 1 {
		return Frame{}, ErrFrame
	}
	n := len(b) - 4
	if crc32.ChecksumIEEE(b[:n]) != binary.BigEndian.Uint32(b[n:]) {
		return Frame{}, fmt.Errorf("%w: checksum", ErrFrame)
	}
	f := Frame{
		Archive: binary.BigEndian.Uint32(b[2:]),
		Parity:  b[1] == 1,
		Index:   int(binary.BigEndian.Uint16(b[6:])),
		Total:   int(binary.BigEndian.Uint16(b[8:])),
		Body:    append([]byte(nil), b[headerLen:n]...),
	}
	if f.Total == 0 || (!f.Parity && (f.Index >= f.Total || len(f.Body) < 2)) {
		return Frame{}, fmt.Errorf("%w: header", ErrFrame)
	}
	return f, nil
}

// Text is the frame in Base45 (RFC 9285): the QR alphanumeric alphabet, which
// a code holds at 5.5 bits a character and every reader returns unaltered.
func (f Frame) Text() string { return base45(f.Bytes()) }

// ParseText reads a code's text.
func ParseText(s string) (Frame, error) {
	b, err := unbase45(s)
	if err != nil {
		return Frame{}, fmt.Errorf("%w: %w", ErrFrame, err)
	}
	return ParseFrame(b)
}

// Frames cuts coded matches into frames: one per match when it fits, data
// frames in match order, each group of ParityGroup followed by its parity.
// owner[i] is the match a frame belongs to, -1 for a parity frame.
func Frames(matches [][]byte) (frames []Frame, owner []int) {
	h := sha256.New()
	for _, m := range matches {
		h.Write(binary.BigEndian.AppendUint32(nil, uint32(len(m))))
		h.Write(m)
	}
	id := binary.BigEndian.Uint32(h.Sum(nil))
	var data []Frame
	var dataOwner []int
	for mi, m := range matches {
		parts := max(1, (len(m)+maxPart-1)/maxPart)
		size := (len(m) + parts - 1) / parts
		for p := range parts {
			chunk := m[min(len(m), p*size):min(len(m), (p+1)*size)]
			body := append([]byte{byte(p), byte(parts)}, chunk...)
			data = append(data, Frame{Archive: id, Index: len(data), Body: body})
			dataOwner = append(dataOwner, mi)
		}
	}
	for i := range data {
		data[i].Total = len(data)
	}
	for g := 0; g*ParityGroup < len(data); g++ {
		group := data[g*ParityGroup : min(len(data), (g+1)*ParityGroup)]
		for i, f := range group {
			frames = append(frames, f)
			owner = append(owner, dataOwner[g*ParityGroup+i])
		}
		frames = append(frames, Frame{Archive: id, Parity: true, Index: g, Total: len(data), Body: parityOf(group)})
		owner = append(owner, -1)
	}
	return frames, owner
}

// parityOf is the lengths of the group's bodies, then their XOR.
func parityOf(group []Frame) []byte {
	width := 0
	for _, f := range group {
		width = max(width, len(f.Body))
	}
	out := make([]byte, 2*ParityGroup+width)
	for i, f := range group {
		binary.BigEndian.PutUint16(out[2*i:], uint16(len(f.Body)))
		for j, c := range f.Body {
			out[2*ParityGroup+j] ^= c
		}
	}
	return out
}

// Assembly is what a set of frames gives back.
type Assembly struct {
	// Matches are the coded matches, in archive order, of every archive met.
	Matches [][]byte
	// Repaired counts the data frames rebuilt from a parity frame.
	Repaired int
	// Missing lists, per archive, the ranks no frame and no parity restored.
	Missing map[uint32][]int
}

// Assemble rebuilds the coded matches from frames read in any order, any
// number of times, from one or several archives. A group that lost a single
// data frame gets it back from its parity frame.
func Assemble(frames []Frame) Assembly {
	type archive struct {
		total  int
		data   map[int]Frame
		parity map[int]Frame
	}
	arcs := map[uint32]*archive{}
	var ids []uint32
	for _, f := range frames {
		a := arcs[f.Archive]
		if a == nil {
			a = &archive{total: f.Total, data: map[int]Frame{}, parity: map[int]Frame{}}
			arcs[f.Archive] = a
			ids = append(ids, f.Archive)
		}
		if f.Parity {
			a.parity[f.Index] = f
		} else {
			a.data[f.Index] = f
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	res := Assembly{Missing: map[uint32][]int{}}
	for _, id := range ids {
		a := arcs[id]
		for g := 0; g*ParityGroup < a.total; g++ {
			lo, hi := g*ParityGroup, min(a.total, (g+1)*ParityGroup)
			lost := -1
			n := 0
			for r := lo; r < hi; r++ {
				if _, ok := a.data[r]; !ok {
					lost, n = r, n+1
				}
			}
			p, ok := a.parity[g]
			if n != 1 || !ok || len(p.Body) < 2*ParityGroup {
				continue
			}
			size := int(binary.BigEndian.Uint16(p.Body[2*(lost-lo):]))
			body := append([]byte(nil), p.Body[2*ParityGroup:]...)
			for r := lo; r < hi; r++ {
				if r == lost {
					continue
				}
				for j, c := range a.data[r].Body {
					if j < len(body) {
						body[j] ^= c
					}
				}
			}
			if size > len(body) || size < 2 {
				continue
			}
			a.data[lost] = Frame{Archive: id, Index: lost, Total: a.total, Body: body[:size]}
			res.Repaired++
		}
		for r := 0; r < a.total; {
			f, ok := a.data[r]
			if !ok {
				res.Missing[id] = append(res.Missing[id], r)
				r++
				continue
			}
			parts := int(f.Body[1])
			if f.Body[0] != 0 || parts == 0 {
				r++
				continue
			}
			var m []byte
			whole := true
			for p := range parts {
				g, ok := a.data[r+p]
				if !ok || int(g.Body[0]) != p || int(g.Body[1]) != parts {
					whole = false
					if !ok {
						res.Missing[id] = append(res.Missing[id], r+p)
					}
					continue
				}
				m = append(m, g.Body[2:]...)
			}
			if whole {
				res.Matches = append(res.Matches, m)
			}
			r += parts
		}
	}
	return res
}

const base45Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

func base45(b []byte) string {
	var sb strings.Builder
	for i := 0; i+1 < len(b); i += 2 {
		n := int(b[i])<<8 | int(b[i+1])
		sb.WriteByte(base45Alphabet[n%45])
		sb.WriteByte(base45Alphabet[n/45%45])
		sb.WriteByte(base45Alphabet[n/2025])
	}
	if len(b)%2 == 1 {
		n := int(b[len(b)-1])
		sb.WriteByte(base45Alphabet[n%45])
		sb.WriteByte(base45Alphabet[n/45])
	}
	return sb.String()
}

func unbase45(s string) ([]byte, error) {
	val := func(c byte) (int, error) {
		if i := strings.IndexByte(base45Alphabet, c); i >= 0 {
			return i, nil
		}
		return 0, fmt.Errorf("base45: character %q", c)
	}
	if len(s)%3 == 1 {
		return nil, errors.New("base45: length")
	}
	out := make([]byte, 0, len(s)*2/3)
	for i := 0; i < len(s); i += 3 {
		var n, mul int
		mul = 1
		for j := i; j < min(len(s), i+3); j++ {
			v, err := val(s[j])
			if err != nil {
				return nil, err
			}
			n += v * mul
			mul *= 45
		}
		if i+3 <= len(s) {
			if n > 0xFFFF {
				return nil, errors.New("base45: value")
			}
			out = append(out, byte(n>>8), byte(n))
		} else {
			if n > 0xFF {
				return nil, errors.New("base45: value")
			}
			out = append(out, byte(n))
		}
	}
	return out, nil
}
