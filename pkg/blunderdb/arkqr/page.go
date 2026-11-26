package arkqr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
)

// Code is one frame to print, with the line read under it.
type Code struct {
	Frame Frame
	Label string
}

// Geometry of an A4 sheet folded in four, in millimetres. The quadrants
// leave a white gutter on both fold lines: no code straddles a fold.
const (
	pageW, pageH = 210.0, 297.0
	margin       = 8.0
	gutter       = 12.0
	quadW        = (pageW - 2*margin - gutter) / 2
	quadH        = (pageH - 2*margin - gutter) / 2
	moduleMM     = 0.5
	codeGap      = 3.0
	labelH       = 6.5
)

type placed struct {
	code       Code
	mods       [][]bool
	x, y, side float64
}

// Page renders codes as a self-contained, printable HTML document of A4
// sheets: codes fill the four quadrants of each sheet in reading order, a
// line under each. The frames are embedded as well, so the file imports
// without reading a code. origin, when set, is printed at the foot of each
// sheet (the producer's watermark).
func Page(title, origin string, codes []Code) ([]byte, error) {
	sheets, err := layout(codes)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	esc := html.EscapeString
	fmt.Fprintf(&b, `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="generator" content="blunderDB paper archive, format %d">
<title>%s</title>
<style>
@page { size: A4; margin: 0 }
html, body { margin: 0; padding: 0; background: #fff; color: #000; font-family: sans-serif }
.sheet { position: relative; width: %.1fmm; height: %.1fmm; overflow: hidden; page-break-after: always; break-after: page }
.sheet:last-child { page-break-after: auto; break-after: auto }
.quad { position: absolute; width: %.2fmm; height: %.2fmm }
.code { position: absolute }
.code svg { display: block; shape-rendering: crispEdges }
.lab { font-size: 2.3mm; line-height: 2.8mm; overflow: hidden; white-space: nowrap; text-overflow: ellipsis }
.head, .foot { position: absolute; left: %.1fmm; right: %.1fmm; font-size: 2.4mm; color: #555 }
.head { top: 2.5mm } .foot { bottom: 2.5mm }
.fold { position: absolute; background: #999 }
@media screen { body { background: #ddd } .sheet { margin: 8mm auto; background: #fff; box-shadow: 0 0 2mm #888 } }
</style></head><body>
`, FormatVersion, esc(title), pageW, pageH, quadW, quadH, margin, margin)
	for si, sheet := range sheets {
		fmt.Fprintf(&b, `<section class="sheet"><div class="head">%s · %d/%d</div>`, esc(title), si+1, len(sheets))
		// Fold marks, in the margins only.
		fmt.Fprintf(&b, `<div class="fold" style="left:%.1fmm;top:0;width:.2mm;height:4mm"></div>`, pageW/2)
		fmt.Fprintf(&b, `<div class="fold" style="left:%.1fmm;bottom:0;width:.2mm;height:4mm"></div>`, pageW/2)
		fmt.Fprintf(&b, `<div class="fold" style="top:%.1fmm;left:0;height:.2mm;width:4mm"></div>`, pageH/2)
		fmt.Fprintf(&b, `<div class="fold" style="top:%.1fmm;right:0;height:.2mm;width:4mm"></div>`, pageH/2)
		for qi, quad := range sheet {
			left := margin + float64(qi%2)*(quadW+gutter)
			top := margin + float64(qi/2)*(quadH+gutter)
			fmt.Fprintf(&b, `<div class="quad" style="left:%.2fmm;top:%.2fmm">`, left, top)
			for _, p := range quad {
				n := len(p.mods)
				fmt.Fprintf(&b, `<div class="code" style="left:%.2fmm;top:%.2fmm;width:%.2fmm">`, p.x, p.y, p.side)
				fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%.2fmm" height="%.2fmm"><rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="%s"/></svg>`,
					n, n, p.side, p.side, n, n, svgPath(p.mods))
				fmt.Fprintf(&b, `<div class="lab">%s</div></div>`, esc(p.code.Label))
			}
			b.WriteString(`</div>`)
		}
		if origin != "" {
			fmt.Fprintf(&b, `<div class="foot">%s</div>`, esc(origin))
		}
		b.WriteString("</section>\n")
	}
	b.WriteString(`<script type="application/vnd.blunderdb.archive">` + "\n")
	for _, c := range codes {
		b.WriteString(base64.StdEncoding.EncodeToString(c.Frame.Bytes()) + "\n")
	}
	b.WriteString("</script>\n</body></html>\n")
	return b.Bytes(), nil
}

// layout places the codes: rows inside a quadrant, quadrants in reading
// order, a new sheet after the fourth.
func layout(codes []Code) ([][4][]placed, error) {
	var sheets [][4][]placed
	var cur [4][]placed
	q, x, y, rowH := 0, 0.0, 0.0, 0.0
	for _, c := range codes {
		mods, err := Modules(c.Frame)
		if err != nil {
			return nil, err
		}
		side := min(quadW, float64(len(mods))*moduleMM)
		h := side + labelH
		if x > 0 && x+side > quadW {
			x, y, rowH = 0, y+rowH+codeGap, 0
		}
		if y > 0 && y+h > quadH {
			x, y, rowH = 0, 0, 0
			q++
			if q == 4 {
				sheets = append(sheets, cur)
				cur, q = [4][]placed{}, 0
			}
		}
		cur[q] = append(cur[q], placed{c, mods, x, y, side})
		x += side + codeGap
		rowH = max(rowH, h)
	}
	if len(cur[0]) > 0 {
		sheets = append(sheets, cur)
	}

	return sheets, nil
}
