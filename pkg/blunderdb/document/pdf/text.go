package pdf

import (
	"bytes"
	"fmt"
	"math"
)

// Glyph is one character a page draws with a text operator.
type Glyph struct {
	// X and Y are the glyph's origin in the page's default user space
	// (points, y upwards).
	X, Y float64
	// Size is the font size as drawn: the Tf size scaled by the text and
	// current transformation matrices.
	Size float64
	// W is the advance to the next glyph, in user space.
	W float64
	// Upright is false when the glyph is rotated, skewed or mirrored.
	Upright bool
	Font    *Font
	// Text is the glyph's Unicode text, empty when the font does not say.
	Text string
	// m maps the glyph's text space to user space; adv and size are its
	// advance and font size in that space (Scan places words with them).
	m         matrix
	adv, size float64
}

type matrix [6]float64

var identity = matrix{1, 0, 0, 1, 0, 0}

func (m matrix) mul(n matrix) matrix {
	return matrix{
		m[0]*n[0] + m[1]*n[2], m[0]*n[1] + m[1]*n[3],
		m[2]*n[0] + m[3]*n[2], m[2]*n[1] + m[3]*n[3],
		m[4]*n[0] + m[5]*n[2] + n[4], m[4]*n[1] + m[5]*n[3] + n[5],
	}
}

type gstate struct {
	ctm              matrix
	font             *Font
	size, tc, tw, tz float64
	leading          float64
	fill             uint8 // the grey a stencil mask paints with
}

type interp struct {
	r      *Reader
	out    []Glyph
	g      gstate
	stack  []gstate
	tm     matrix
	tlm    matrix
	depth  int
	inForm map[*Stream]bool
	// collect records the images drawn (Scan).
	collect bool
	draws   []imageDraw
}

const (
	maxFormDepth = 8
	maxStack     = 256
	maxOperands  = 64
)

// Glyphs interprets the page's content and returns every glyph it draws, in
// drawing order.
func (r *Reader) Glyphs(p Page) (gs []Glyph, err error) {
	defer func() {
		if e := recover(); e != nil {
			gs, err = nil, malformed("%v", e)
		}
	}()
	data, err := r.content(p)
	if err != nil {
		return nil, err
	}
	in := &interp{r: r, g: gstate{ctm: identity, tz: 1}, tm: identity, tlm: identity, inForm: map[*Stream]bool{}}
	if err := in.run(data, p.resources); err != nil {
		return nil, err
	}
	return in.out, nil
}

func (in *interp) run(data []byte, res Dict) error {
	l := &lexer{b: data}
	var ops []any
	for {
		save := l.pos
		t := l.token()
		switch v := t.(type) {
		case eof:
			return nil
		case delimiter:
			if v == '[' || v == '<' {
				l.pos = save
				o, err := l.object(false, 0)
				if err != nil {
					return err
				}
				ops = append(ops, o)
			}
		case keyword:
			if v == "BI" && in.collect {
				if s := inlineImage(l); s != nil {
					in.record(s, res)
				}
			} else if v == "BI" {
				skipInlineImage(l)
			} else if err := in.op(string(v), ops, res); err != nil {
				return err
			}
			ops = ops[:0]
		default:
			if len(ops) >= maxOperands {
				ops = ops[:0]
			}
			ops = append(ops, v)
		}
	}
}

// skipInlineImage moves past "… ID <data> EI": the data is binary.
func skipInlineImage(l *lexer) {
	for {
		t := l.token()
		if _, ok := t.(eof); ok {
			return
		}
		if k, ok := t.(keyword); ok && k == "ID" {
			break
		}
	}
	l.pos++
	for i := l.pos; i+1 < len(l.b); i++ {
		if l.b[i] == 'E' && l.b[i+1] == 'I' && (i == 0 || isWhite(l.b[i-1])) && (i+2 >= len(l.b) || isWhite(l.b[i+2])) {
			l.pos = i + 2
			return
		}
	}
	l.pos = len(l.b)
}

func numArg(ops []any, i int) float64 {
	if i < 0 || i >= len(ops) {
		return 0
	}
	switch v := ops[i].(type) {
	case int64:
		return float64(v)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return v
	}
	return 0
}

func (in *interp) op(op string, ops []any, res Dict) error {
	n := func(i int) float64 { return numArg(ops, i) }
	last := func(k int) []any {
		if len(ops) < k {
			return nil
		}
		return ops[len(ops)-k:]
	}
	switch op {
	case "q":
		if len(in.stack) < maxStack {
			in.stack = append(in.stack, in.g)
		}
	case "Q":
		if k := len(in.stack); k > 0 {
			in.g = in.stack[k-1]
			in.stack = in.stack[:k-1]
		}
	case "cm":
		if a := last(6); a != nil {
			m := matrix{numArg(a, 0), numArg(a, 1), numArg(a, 2), numArg(a, 3), numArg(a, 4), numArg(a, 5)}
			in.g.ctm = m.mul(in.g.ctm)
		}
	case "BT":
		in.tm, in.tlm = identity, identity
	case "Tf":
		if a := last(2); a != nil {
			name, _ := a[0].(Name)
			fonts, _ := in.r.Resolve(res["Font"]).(Dict)
			in.g.font = in.r.font(fonts[name])
			in.g.size = numArg(a, 1)
		}
	case "Tc":
		in.g.tc = n(len(ops) - 1)
	case "Tw":
		in.g.tw = n(len(ops) - 1)
	case "Tz":
		in.g.tz = n(len(ops)-1) / 100
	case "TL":
		in.g.leading = n(len(ops) - 1)
	case "Td", "TD":
		if a := last(2); a != nil {
			tx, ty := numArg(a, 0), numArg(a, 1)
			if op == "TD" {
				in.g.leading = -ty
			}
			in.tlm = matrix{1, 0, 0, 1, tx, ty}.mul(in.tlm)
			in.tm = in.tlm
		}
	case "Tm":
		if a := last(6); a != nil {
			in.tlm = matrix{numArg(a, 0), numArg(a, 1), numArg(a, 2), numArg(a, 3), numArg(a, 4), numArg(a, 5)}
			in.tm = in.tlm
		}
	case "T*":
		in.nextLine()
	case "Tj":
		if a := last(1); a != nil {
			return in.show(a[0])
		}
	case "'":
		in.nextLine()
		if a := last(1); a != nil {
			return in.show(a[0])
		}
	case "\"":
		if a := last(3); a != nil {
			in.g.tw, in.g.tc = numArg(a, 0), numArg(a, 1)
			in.nextLine()
			return in.show(a[2])
		}
	case "TJ":
		if a := last(1); a != nil {
			arr, _ := a[0].(Array)
			for _, it := range arr {
				switch v := it.(type) {
				case String:
					if err := in.show(v); err != nil {
						return err
					}
				default:
					adj := -numArg([]any{v}, 0) / 1000 * in.g.size * in.g.tz
					in.tm = matrix{1, 0, 0, 1, adj, 0}.mul(in.tm)
				}
			}
		}
	case "Do":
		if a := last(1); a != nil {
			name, _ := a[0].(Name)
			return in.xobject(name, res)
		}
	case "g":
		in.setFill(ops, 1)
	case "rg":
		in.setFill(ops, 3)
	case "k":
		in.setFill(ops, 4)
	case "cs":
		in.g.fill = 0
	case "sc", "scn":
		k := 0
		for _, o := range ops {
			switch o.(type) {
			case int64, float64:
				k++
			}
		}
		if k == 1 || k == 3 || k == 4 {
			in.setFill(ops[:k], k)
		}
	}
	return nil
}

func (in *interp) nextLine() {
	in.tlm = matrix{1, 0, 0, 1, 0, -in.g.leading}.mul(in.tlm)
	in.tm = in.tlm
}

func (in *interp) show(o any) error {
	s, ok := o.(String)
	f := in.g.font
	if !ok || f == nil {
		return nil
	}
	step := 1
	if f.twoByte {
		step = 2
	}
	b := []byte(s)
	for i := 0; i+step <= len(b); i += step {
		code := uint32(b[i])
		if step == 2 {
			code = code<<8 | uint32(b[i+1])
		}
		if len(in.out) >= in.r.lim.MaxGlyphs {
			return fmt.Errorf("%w: a page draws more than %d glyphs", ErrTooLarge, in.r.lim.MaxGlyphs)
		}
		m := in.tm.mul(in.g.ctm)
		adv := f.advance(code)*in.g.size + in.g.tc
		if step == 1 && code == 32 {
			adv += in.g.tw
		}
		adv *= in.g.tz
		upright := math.Abs(m[1]) < 1e-6 && math.Abs(m[2]) < 1e-6 && m[0] > 0 && m[3] > 0
		size := in.g.size * math.Sqrt(math.Abs(m[0]*m[3]-m[1]*m[2]))
		if upright {
			size = in.g.size * m[3]
		}
		in.out = append(in.out, Glyph{
			X: m[4], Y: m[5], Size: size, W: adv * m[0], Upright: upright,
			Font: f, Text: f.text(code), m: m, adv: adv, size: in.g.size,
		})
		in.tm = matrix{1, 0, 0, 1, adv, 0}.mul(in.tm)
	}
	return nil
}

// form draws a form XObject: its content, in its own matrix and resources.
func (in *interp) form(name Name, res Dict) error {
	xo, _ := in.r.Resolve(res["XObject"]).(Dict)
	s, ok := in.r.Resolve(xo[name]).(*Stream)
	if !ok || s.Dict["Subtype"] != Name("Form") || in.depth >= maxFormDepth || in.inForm[s] {
		return nil
	}
	data, err := in.r.Decode(s)
	if err != nil {
		return nil // an unreadable form hides its glyphs, not the page's
	}
	sub, _ := in.r.Resolve(s.Dict["Resources"]).(Dict)
	if sub == nil {
		sub = res
	}
	saved, savedTM, savedTLM := in.g, in.tm, in.tlm
	if a, ok := in.r.Resolve(s.Dict["Matrix"]).(Array); ok && len(a) == 6 {
		m := matrix{in.r.num(a[0]), in.r.num(a[1]), in.r.num(a[2]), in.r.num(a[3]), in.r.num(a[4]), in.r.num(a[5])}
		in.g.ctm = m.mul(in.g.ctm)
	}
	in.depth++
	in.inForm[s] = true
	depth := len(in.stack)
	err = in.run(bytes.Clone(data), sub)
	in.stack = in.stack[:min(depth, len(in.stack))]
	delete(in.inForm, s)
	in.depth--
	in.g, in.tm, in.tlm = saved, savedTM, savedTLM
	return err
}
