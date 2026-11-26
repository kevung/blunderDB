package arkqr

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"math/rand/v2"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The paper archive is printed, folded in four and scanned. With no scanner
// at hand, the sheet is drawn from the geometry Page lays out (same
// quadrants, same module size), rasterised at a scanner's resolution, damaged
// by one defect at a time, and read by the decoder the import uses. The
// ladders below are the defects' strengths; the first one that breaks the
// reading is the limit this file records.
//
// The simulated sheet leaves out what is not a code: the labels under the
// codes, the header and foot lines and the fold marks of the margins, which
// can only help a reader that finds none of them. A scanner's bed, its lid
// shadow and a photograph's lighting are not drawn either: the white of the
// paper reaches the frame.
//
// The trial costs minutes under the race detector, which checks nothing in
// pixel arithmetic: it runs in the nightly full suite only (no -race, no
// -short).

// archiveSheets lays out what a small archive prints: the matches of the
// corpus and two tranches of position lists, each with its parity. A
// tranche's three codes are 60 mm wide each: the archive spreads over
// several sheets, scanned one by one.
func archiveSheets(t *testing.T, mod float64) (sheets [][4][]placed, matches [][]byte, tranches map[int][]domain.Position) {
	t.Helper()
	tranches = map[int][]domain.Position{}
	ps := randomPositions(t, 2*Tranche)
	matches = corpusPayloads(t)
	frames, _, err := Frames(matches)
	if err != nil {
		t.Fatal(err)
	}
	for k, first := range []int{1, 146} {
		data, err := EncodePositions(first, ps[k*Tranche:(k+1)*Tranche])
		if err != nil {
			t.Fatal(err)
		}
		pf, err := PositionFrames(data)
		if err != nil {
			t.Fatal(err)
		}
		frames = append(frames, pf...)
		tr, _ := DecodePositions(data)
		tranches[first] = tr.Positions
	}
	codes := make([]Code, len(frames))
	for i, f := range frames {
		codes[i] = Code{Frame: f}
	}
	if sheets, err = layout(codes, mod); err != nil {
		t.Fatal(err)
	}
	return sheets, matches, tranches
}

// gray is a sheet as the scanner sees it: 0 black, 1 white.
type gray struct {
	w, h int
	v    []float32
}

func newGray(w, h int) *gray {
	g := &gray{w: w, h: h, v: make([]float32, w*h)}
	for i := range g.v {
		g.v[i] = 1
	}
	return g
}

func (g *gray) at(x, y int) float32 {
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return 1
	}
	return g.v[y*g.w+x]
}

// sample is a bilinear read, white outside the sheet.
func (g *gray) sample(x, y float64) float32 {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := float32(x-x0), float32(y-y0)
	ix, iy := int(x0), int(y0)
	a := g.at(ix, iy)*(1-fx) + g.at(ix+1, iy)*fx
	b := g.at(ix, iy+1)*(1-fx) + g.at(ix+1, iy+1)*fx
	return a*(1-fy) + b*fy
}

// rasterise draws a sheet's codes at dpi: each dark module darkens the pixels
// it covers in proportion to the area it covers, as a scanner integrates.
func rasterise(sheet [4][]placed, dpi float64) *gray {
	pxmm := dpi / 25.4
	g := newGray(int(math.Ceil(pageW*pxmm)), int(math.Ceil(pageH*pxmm)))
	dark := make([]float32, len(g.v))
	for qi, quad := range sheet {
		left := margin + float64(qi%2)*(quadW+gutter)
		top := margin + float64(qi/2)*(quadH+gutter)
		for _, p := range quad {
			n := len(p.mods)
			mod := p.side / float64(n)
			for y, row := range p.mods {
				for x, on := range row {
					if !on {
						continue
					}
					x0 := (left + p.x + float64(x)*mod) * pxmm
					y0 := (top + p.y + float64(y)*mod) * pxmm
					x1, y1 := x0+mod*pxmm, y0+mod*pxmm
					for py := int(y0); py < int(math.Ceil(y1)) && py < g.h; py++ {
						oy := math.Min(y1, float64(py+1)) - math.Max(y0, float64(py))
						for px := int(x0); px < int(math.Ceil(x1)) && px < g.w; px++ {
							ox := math.Min(x1, float64(px+1)) - math.Max(x0, float64(px))
							dark[py*g.w+px] += float32(ox * oy)
						}
					}
				}
			}
		}
	}
	for i := range g.v {
		g.v[i] = 1 - min(1, dark[i])
	}
	return g
}

type scanSet []*gray

func rasteriseAll(sheets [][4][]placed, dpi float64) scanSet {
	var out scanSet
	for _, sh := range sheets {
		out = append(out, rasterise(sh, dpi))
	}
	return out
}

// damage applies a defect to every sheet, each with its own noise.
func (set scanSet) damage(d defect, level, dpi float64) []image.Image {
	var out []image.Image
	for i, g := range set {
		rng := rand.New(rand.NewPCG(3, uint64(5+i)))
		out = append(out, d.apply(g, level, dpi, rng))
	}
	return out
}

func (set scanSet) images() []image.Image {
	var out []image.Image
	for _, g := range set {
		out = append(out, g.image())
	}
	return out
}

func (g *gray) image() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, g.w, g.h))
	for i, v := range g.v {
		img.Pix[i] = uint8(math.Round(float64(max(0, min(1, v))) * 255))
	}
	return img
}

// raceEnabled is set by race_on_test.go: the trial is pixel arithmetic that
// the race detector slows tenfold without checking anything.
var raceEnabled bool

func skipUnlessFull(t *testing.T) {
	t.Helper()
	if testing.Short() || raceEnabled {
		t.Skip("essai de feuilles rastérisées : suite complète sans -race")
	}
}

// A defect transforms a scan; the strength is its own unit (see ladders).
type defect struct {
	name   string
	unit   string
	levels []float64
	apply  func(g *gray, level, dpi float64, rng *rand.Rand) image.Image
}

func lift(f func(g *gray, level, dpi float64, rng *rand.Rand) *gray) func(*gray, float64, float64, *rand.Rand) image.Image {
	return func(g *gray, level, dpi float64, rng *rand.Rand) image.Image { return f(g, level, dpi, rng).image() }
}

func (g *gray) warp(inv func(x, y float64) (float64, float64)) *gray {
	out := newGray(g.w, g.h)
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			sx, sy := inv(float64(x), float64(y))
			out.v[y*g.w+x] = g.sample(sx, sy)
		}
	}
	return out
}

func (g *gray) blur(sigma float64) *gray {
	if sigma < 0.3 {
		return g
	}
	r := int(math.Ceil(3 * sigma))
	k := make([]float32, 2*r+1)
	var sum float32
	for i := range k {
		d := float64(i - r)
		k[i] = float32(math.Exp(-d * d / (2 * sigma * sigma)))
		sum += k[i]
	}
	for i := range k {
		k[i] /= sum
	}
	tmp, out := newGray(g.w, g.h), newGray(g.w, g.h)
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			var s float32
			for i, w := range k {
				s += w * g.at(x+i-r, y)
			}
			tmp.v[y*g.w+x] = s
		}
	}
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			var s float32
			for i, w := range k {
				s += w * tmp.at(x, y+i-r)
			}
			out.v[y*g.w+x] = s
		}
	}
	return out
}

// rotate turns the sheet about its centre inside the same frame: the corners
// leave it, and past some angle the codes of the outer quadrants with them.
func (g *gray) rotate(deg float64) *gray {
	a := deg * math.Pi / 180
	c, s := math.Cos(a), math.Sin(a)
	cx, cy := float64(g.w)/2, float64(g.h)/2
	return g.warp(func(x, y float64) (float64, float64) {
		dx, dy := x-cx, y-cy
		return cx + c*dx + s*dy, cy - s*dx + c*dy
	})
}

// crease models the two folds of a sheet folded in four, which cross the
// quadrants' neighbourhood: a soft shadow 4 mm wide on both sides of each fold
// (the paper bulges there and the light falls off, level being the share of
// light lost at the fold), a dark line where it is creased, and the halves
// shifted a pixel against each other. The shadow reaches the codes, which
// start 6 mm from the fold.
func (g *gray) crease(level, dpi float64) *gray {
	pxmm := dpi / 25.4
	sigma := 4 * pxmm
	line := max(1, int(math.Round(0.3*pxmm)))
	cx, cy := g.w/2, g.h/2
	shifted := g.warp(func(x, y float64) (float64, float64) {
		if x > float64(cx) {
			x--
		}
		if y > float64(cy) {
			y--
		}
		return x, y
	})
	out := newGray(g.w, g.h)
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			dx, dy := float64(abs(x-cx)), float64(abs(y-cy))
			shade := math.Exp(-dx*dx/(2*sigma*sigma)) + math.Exp(-dy*dy/(2*sigma*sigma))
			f := 1 - level*math.Min(1, shade)
			if min(abs(x-cx), abs(y-cy)) < line {
				f -= level
			}
			out.v[y*g.w+x] = shifted.v[y*g.w+x] * float32(max(0, f))
		}
	}
	return out
}

var defects = []defect{
	{"pli", "ombre", []float64{0.1, 0.2, 0.3, 0.5}, lift(func(g *gray, level, dpi float64, _ *rand.Rand) *gray {
		return g.crease(level, dpi)
	})},
	{"rotation", "°", []float64{0.5, 1, 2, 3, 5, 8, 12}, lift(func(g *gray, deg, _ float64, _ *rand.Rand) *gray {
		return g.rotate(deg)
	})},
	{"perspective", "k", []float64{0.01, 0.02, 0.04, 0.08, 0.12}, lift(func(g *gray, k, _ float64, _ *rand.Rand) *gray {
		// A keystone: the far edge of the sheet is 1/(1+k) the size of the near one.
		cx, cy := float64(g.w)/2, float64(g.h)/2
		return g.warp(func(x, y float64) (float64, float64) {
			s := 1 + k*(y-cy)/float64(g.h)*2
			return cx + (x-cx)*s, cy + (y-cy)*(1+k*(y-cy)/float64(g.h))
		})
	})},
	{"flou", "mm", []float64{0.05, 0.1, 0.15, 0.2, 0.3}, lift(func(g *gray, mm, dpi float64, _ *rand.Rand) *gray {
		return g.blur(mm * dpi / 25.4)
	})},
	{"bruit", "σ", []float64{0.05, 0.1, 0.2, 0.3, 0.45}, lift(func(g *gray, sd, _ float64, rng *rand.Rand) *gray {
		out := newGray(g.w, g.h)
		for i, v := range g.v {
			out.v[i] = v + float32(rng.NormFloat64()*sd)
		}
		return out
	})},
	{"contraste", "écart", []float64{0.8, 0.6, 0.4, 0.25, 0.15, 0.1}, lift(func(g *gray, span, _ float64, _ *rand.Rand) *gray {
		// Faded toner on grey paper: black and white pulled to a narrow band.
		lo := float32(0.55 - span/2)
		out := newGray(g.w, g.h)
		for i, v := range g.v {
			out.v[i] = lo + v*float32(span)
		}
		return out
	})},
	{"jpeg", "qualité", []float64{90, 70, 50, 30, 15, 5}, func(g *gray, q, _ float64, _ *rand.Rand) image.Image {
		var b bytes.Buffer
		if err := jpeg.Encode(&b, g.image(), &jpeg.Options{Quality: int(q)}); err != nil {
			panic(err)
		}
		img, err := jpeg.Decode(&b)
		if err != nil {
			panic(err)
		}
		return img
	}},
	{"scan plausible", "k", []float64{0.5, 1, 1.5, 2, 3}, func(g *gray, k, dpi float64, rng *rand.Rand) image.Image {
		// What a household scanner gives, all at once and in proportion: a
		// slight blur, a small rotation, a faint crease, noise and JPEG.
		o := g.crease(0.1*k, dpi).rotate(0.6 * k).blur(0.06 * k * dpi / 25.4)
		for i := range o.v {
			o.v[i] += float32(rng.NormFloat64() * 0.03 * k)
		}
		var b bytes.Buffer
		if err := jpeg.Encode(&b, o.image(), &jpeg.Options{Quality: max(5, int(85-15*k))}); err != nil {
			panic(err)
		}
		img, err := jpeg.Decode(&b)
		if err != nil {
			panic(err)
		}
		return img
	}},
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// readBack runs the decoder the import uses on the damaged scan and checks
// the whole archive comes back: every match byte for byte, every tranche's
// positions, with the parity allowed to repair what it can.
func readBack(imgs []image.Image, matches [][]byte, tranches map[int][]domain.Position) (ok bool, frames int, why string) {
	var read []Frame
	for _, img := range imgs {
		r, err := scan(img)
		if err != nil {
			return false, 0, err.Error()
		}
		read = append(read, r...)
	}
	asm := Assemble(read)
	if len(asm.Missing) != 0 || !reflect.DeepEqual(asm.Matches, matches) {
		return false, len(read), fmt.Sprintf("%d matches of %d, missing %d", len(asm.Matches), len(matches), len(asm.Missing))
	}
	if len(asm.Tranches) != len(tranches) {
		return false, len(read), fmt.Sprintf("%d tranches of %d", len(asm.Tranches), len(tranches))
	}
	for _, data := range asm.Tranches {
		tr, err := DecodePositions(data)
		if err != nil || len(tr.Positions) != len(tranches[tr.First]) {
			return false, len(read), "tranche"
		}
		for i := range tr.Positions {
			a, _ := domain.EncodeArkID(&tr.Positions[i])
			b, _ := domain.EncodeArkID(&tranches[tr.First][i])
			if a != b {
				return false, len(read), "position"
			}
		}
	}
	return true, len(read), ""
}

// legacyModuleMM is the module the first archives were printed with: a sheet
// already printed at that size must still be read.
const legacyModuleMM = 0.5

// TestEssaiArchiveSansDefaut: a clean print is read back whole, at 150, 200
// and 300 dpi at today's module, and at 200 and 300 dpi at the legacy one.
func TestEssaiArchiveSansDefaut(t *testing.T) {
	skipUnlessFull(t)
	for mod, dpis := range map[float64][]float64{moduleMM: {150, 200, 300}, legacyModuleMM: {200, 300}} {
		sheets, matches, tranches := archiveSheets(t, mod)
		for _, dpi := range dpis {
			g := rasteriseAll(sheets, dpi)
			if ok, n, why := readBack(g.images(), matches, tranches); !ok {
				t.Errorf("module %g mm, %.0f dpi: %d codes read, %s", mod, dpi, n, why)
			}
		}
	}
}

// printing is a sheet as it was printed and scanned: its module, in
// millimetres, and the scanner's resolution.
type printing struct{ module, dpi float64 }

// reference is, per module and scanner resolution, the strength of each
// defect the archive must still be read through, as the grid measured it
// (one notch below the last one read).
//
// At 0.6 mm a module is 4.7 pixels at 200 dpi: the sheet takes a rotation of
// 8°, a noise of 0.1 and a household scan up to k = 2, as at 300 dpi. At
// 150 dpi (3.5 pixels) it is still read, noise aside.
//
// At the legacy 0.5 mm a module is four pixels at 200 dpi: the sheet survives
// soft defects and is lost to a noise of 0.05 or to a rotation of half a
// degree; at 150 dpi no sheet of this density is read.
var reference = map[printing]map[string]float64{
	{moduleMM, 150}:       {"pli": 0.3, "rotation": 5, "perspective": 0.01, "flou": 0.1, "contraste": 0.6, "jpeg": 15, "scan plausible": 0.5},
	{moduleMM, 200}:       {"pli": 0.3, "rotation": 5, "perspective": 0.02, "flou": 0.15, "bruit": 0.05, "contraste": 0.6, "jpeg": 15, "scan plausible": 1.5},
	{moduleMM, 300}:       {"pli": 0.3, "rotation": 5, "perspective": 0.04, "flou": 0.15, "bruit": 0.05, "contraste": 0.4, "jpeg": 15, "scan plausible": 1},
	{legacyModuleMM, 200}: {"pli": 0.3, "perspective": 0.04, "flou": 0.1, "contraste": 0.6, "jpeg": 15, "scan plausible": 0.5},
	{legacyModuleMM, 300}: {"pli": 0.3, "rotation": 5, "perspective": 0.04, "flou": 0.1, "bruit": 0.05, "contraste": 0.4, "jpeg": 15, "scan plausible": 1},
}

// TestEssaiArchiveSeuilsDeReference fixes the reading threshold: for each
// printing, every defect at its reference strength still reads the whole
// archive. The thresholds sit one notch below the last level the grid read,
// so that a different rounding of floating point (fused multiply-add on
// arm64) does not turn them red. A rotation of 12° is lost to the frame,
// which crops the corners where the outer codes sit, not to the decoder.
func TestEssaiArchiveSeuilsDeReference(t *testing.T) {
	skipUnlessFull(t)
	for pr, levels := range reference {
		sheets, matches, tranches := archiveSheets(t, pr.module)
		base := rasteriseAll(sheets, pr.dpi)
		for _, d := range defects {
			lv, ok := levels[d.name]
			if !ok {
				continue
			}
			if ok, n, why := readBack(base.damage(d, lv, pr.dpi), matches, tranches); !ok {
				t.Errorf("module %g mm, %.0f dpi, %s %g %s: %d codes read, %s", pr.module, pr.dpi, d.name, lv, d.unit, n, why)
			}
		}
	}
}

// TestEssaiArchiveGrille measures, for every resolution and every defect, the
// strongest level still read whole, and prints the table. It takes minutes:
// BLUNDERDB_ESSAI_GRILLE=1 go test ./pkg/blunderdb/arkqr -run Grille -v -timeout 1h
// BLUNDERDB_ESSAI_MODULE sets the module in millimetres (today's by default).
func TestEssaiArchiveGrille(t *testing.T) {
	if os.Getenv("BLUNDERDB_ESSAI_GRILLE") == "" {
		t.Skip("grille complète : BLUNDERDB_ESSAI_GRILLE=1")
	}
	mod := moduleMM
	if v := os.Getenv("BLUNDERDB_ESSAI_MODULE"); v != "" {
		var err error
		if mod, err = strconv.ParseFloat(v, 64); err != nil {
			t.Fatal(err)
		}
	}
	sheets, matches, tranches := archiveSheets(t, mod)
	var tab strings.Builder
	fmt.Fprintf(&tab, "module %g mm, %d feuilles\n%-12s", mod, len(sheets), "défaut")
	dpis := []float64{150, 200, 300}
	for _, dpi := range dpis {
		fmt.Fprintf(&tab, " | %-34s", fmt.Sprintf("%.0f dpi (dernier lu / premier perdu)", dpi))
	}
	tab.WriteString("\n")
	rows := map[string][]string{}
	for _, dpi := range dpis {
		t0 := time.Now()
		base := rasteriseAll(sheets, dpi)
		ok, n, why := readBack(base.images(), matches, tranches)
		t.Logf("%.0f dpi sans défaut: %v (%d codes, %s) %v", dpi, ok, n, why, time.Since(t0))
		for _, d := range defects {
			last, first := "aucun", "-"
			for _, lv := range d.levels {
				ok, n, why := readBack(base.damage(d, lv, dpi), matches, tranches)
				t.Logf("%.0f dpi %s %g: %v (%d codes) %s", dpi, d.name, lv, ok, n, why)
				if !ok {
					first = fmt.Sprintf("%g", lv)
					break
				}
				last = fmt.Sprintf("%g", lv)
			}
			rows[d.name] = append(rows[d.name], fmt.Sprintf("%s / %s %s", last, first, d.unit))
		}
	}
	for _, d := range defects {
		fmt.Fprintf(&tab, "%-12s", d.name)
		for _, c := range rows[d.name] {
			fmt.Fprintf(&tab, " | %-34s", c)
		}
		tab.WriteString("\n")
	}
	t.Log("\n" + tab.String())
}
