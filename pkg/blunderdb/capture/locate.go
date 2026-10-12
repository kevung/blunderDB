package capture

import (
	"math"
	"runtime"
	"sort"
	"sync"
)

// registration places a theme's board on the capture: (x0, y0) is the top-left
// corner of the left playing field, p the point width in pixels, hs a vertical
// stretch (1 = the theme's aspect ratio) that absorbs a non-uniform rescale.
type registration struct {
	x0, y0, p, hs float64
}

func (g registration) x(u float64) float64 { return g.x0 + u*g.p }
func (g registration) y(v float64) float64 { return g.y0 + v*g.p*g.hs }

// probe is one place the score looks at, in theme units, with the classes a
// correctly placed board may show there. Checkers are allowed almost
// everywhere; what pins the board down is the triangles' alternation, the
// field between them and the bar — a board placed one point off sees light
// where it expects dark.
type probe struct {
	u, v    float64
	allowed uint8 // bit set of classes
	half    float64
	// col and top name the point a probe belongs to; col is -1 for the bar
	// and the middle of the board.
	col int
	top bool
}

func bit(cs ...class) uint8 {
	var m uint8
	for _, c := range cs {
		m |= 1 << c
	}
	return m
}

func (t *theme) probes() []probe {
	var ps []probe
	checkers := bit(clsCheckerA, clsCheckerB)
	for _, top := range []bool{true, false} {
		for c := 0; c < 12; c++ {
			tri := clsDarkTri
			if t.light(c, top) {
				tri = clsLightTri
			}
			for k := 0; k < t.triSlots; k++ {
				ps = append(ps, probe{t.colX(c), t.slotY(k, top), checkers | bit(tri), 0.22, c, top})
			}
			// Between two points of a half, where four checkers' rims are
			// farthest apart, lies the field: no checker reaches there.
			if c%6 != 5 {
				for k := 1; k < t.triSlots; k++ {
					v := t.margin + float64(k)*t.step
					if !top {
						v = t.fieldH - v
					}
					ps = append(ps, probe{t.colX(c) + 0.5, v, bit(clsField), 0.06, -1, top})
				}
			}
		}
		for k := 0; k < t.triSlots; k++ {
			ps = append(ps, probe{6 + t.barW/2, t.slotY(k, top), checkers | bit(clsField, clsOther), 0.2, -1, top})
		}
	}
	for c := 0; c < 12; c++ {
		ps = append(ps, probe{t.colX(c), t.fieldH / 2, bit(clsField, clsOther, clsLightTri), 0.2, -1, false})
		// Just beyond the field's top and bottom edges the triangles have
		// ended: these two pin the board's height.
		ps = append(ps, probe{t.colX(c), -0.35, bit(clsField, clsOther), 0.12, -1, true})
		ps = append(ps, probe{t.colX(c), t.fieldH + 0.35, bit(clsField, clsOther), 0.12, -1, false})
	}
	return ps
}

// classMap is a raster already classified pixel by pixel (3×3 mean), for the
// coarse search.
type classMap struct {
	w, h int
	cls  []class
}

func (t *theme) classMap(r *raster) classMap {
	m := classMap{w: r.w, h: r.h, cls: make([]class, r.w*r.h)}
	for y := 0; y < r.h; y++ {
		for x := 0; x < r.w; x++ {
			m.cls[y*r.w+x] = t.classify(r.mean(float64(x-1), float64(y-1), float64(x+2), float64(y+2)))
		}
	}
	return m
}

type candidate struct {
	g     registration
	score float64
}

// locate finds where the theme's board sits, at whatever scale, and how well
// it fits there (the share of probes that agree, 0..1).
func (t *theme) locate(r *raster) (registration, float64) {
	ps := t.probes()
	const minP = 10.0
	type job struct {
		cm classMap
		pd float64
		f  int
	}
	var jobs []job
	maps := map[int]classMap{}
	for p := minP; p*t.fieldW() <= float64(r.w) && p*t.fieldH <= float64(r.h); p *= 1.035 {
		// Each scale is searched once, on a copy reduced so that a point is
		// 6 to 12 pixels wide: enough to tell the triangles apart.
		f := max(1, int(p/8))
		cm, ok := maps[f]
		if !ok {
			cm = t.classMap(r.downscale(f))
			maps[f] = cm
		}
		jobs = append(jobs, job{cm, p / float64(f), f})
	}
	results := make([][]candidate, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			results[i] = t.scan(j.cm, ps, j.pd, j.f)
			<-sem
		}()
	}
	wg.Wait()
	var cands []candidate
	for _, rs := range results {
		cands = append(cands, rs...)
	}
	if len(cands) == 0 {
		return registration{}, 0
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	best, bestScore := registration{}, -1.0
	for i := 0; i < len(cands) && i < 6; i++ {
		g, s := t.refine(r, ps, cands[i].g)
		if s > bestScore {
			best, bestScore = g, s
		}
	}
	agree := 0
	for _, q := range ps {
		if t.agrees(r, q, best) {
			agree++
		}
	}
	return best, float64(agree) / float64(len(ps))
}

// agrees says whether a probe shows one of the classes it allows.
func (t *theme) agrees(r *raster, q probe, g registration) bool {
	hw := math.Max(q.half*g.p, 0.5)
	cx, cy := g.x(q.u), g.y(q.v)
	cl, _ := t.nearest(r.mean(cx-hw, cy-hw, cx+hw, cy+hw))
	return q.allowed&(1<<cl) != 0
}

// scan tries every placement of one scale on the class map and keeps the two
// best, in full-resolution coordinates.
func (t *theme) scan(cm classMap, ps []probe, p float64, f int) []candidate {
	type off struct {
		dx, dy int
		mask   uint8
	}
	offs := make([]off, len(ps))
	for i, q := range ps {
		offs[i] = off{int(math.Round(q.u * p)), int(math.Round(q.v * p)), q.allowed}
	}
	need := len(ps)
	var out []candidate
	bestA, bestB := -1, -1
	var gA, gB registration
	minX, minY, maxX, maxY := 0, 0, 0, 0
	for _, o := range offs {
		minX, minY = min(minX, o.dx), min(minY, o.dy)
		maxX, maxY = max(maxX, o.dx), max(maxY, o.dy)
	}
	for y0 := -minY; y0+maxY < cm.h; y0++ {
		for x0 := -minX; x0+maxX < cm.w; x0++ {
			loss := 0
			limit := min(2*need-bestB, need)
			for _, o := range offs {
				cl := cm.cls[(y0+o.dy)*cm.w+x0+o.dx]
				switch {
				case o.mask&(1<<cl) == 0:
					loss += 2
				case cl == clsCheckerA || cl == clsCheckerB:
					loss++
				}
				if loss >= limit {
					break
				}
			}
			s := 2*need - loss
			if s > bestB {
				g := registration{float64(x0 * f), float64(y0 * f), p * float64(f), 1}
				if s > bestA {
					bestB, gB = bestA, gA
					bestA, gA = s, g
				} else {
					bestB, gB = s, g
				}
			}
		}
	}
	if bestA >= 0 {
		out = append(out, candidate{gA, float64(bestA)})
	}
	if bestB >= 0 {
		out = append(out, candidate{gB, float64(bestB)})
	}
	return out
}

// fit scores a placement at full resolution: the probes that agree, minus a
// small term for how far each colour is from its swatch so that the descent
// still has a slope where the count is flat.
func (t *theme) fit(r *raster, ps []probe, g registration) float64 {
	s := 0.0
	for _, q := range ps {
		hw := math.Max(q.half*g.p, 0.5)
		cx, cy := g.x(q.u), g.y(q.v)
		c := r.mean(cx-hw, cy-hw, cx+hw, cy+hw)
		cl, d := t.nearest(c)
		if q.allowed&(1<<cl) != 0 {
			s++
			if cl != clsCheckerA && cl != clsCheckerB {
				s++
			}
			if cl != clsOther {
				s += 0.5 * (1 - d/t.maxDist)
			}
		}
	}
	return s
}

func (t *theme) refine(r *raster, ps []probe, g registration) (registration, float64) {
	best := t.fit(r, ps, g)
	steps := [4]float64{0.3 * g.p, 0.3 * g.p, 0.04 * g.p, 0.02}
	for round := 0; round < 6; round++ {
		improved := true
		for improved {
			improved = false
			for dim := 0; dim < 4; dim++ {
				for _, sign := range []float64{-1, 1} {
					h := g
					switch dim {
					case 0:
						h.x0 += sign * steps[0]
					case 1:
						h.y0 += sign * steps[1]
					case 2:
						h.p += sign * steps[2]
					case 3:
						h.hs += sign * steps[3]
					}
					if h.hs < 0.97 || h.hs > 1.03 {
						continue
					}
					if s := t.fit(r, ps, h); s > best {
						g, best, improved = h, s, true
					}
				}
			}
		}
		for i := range steps {
			steps[i] /= 2
		}
	}
	return g, best
}
