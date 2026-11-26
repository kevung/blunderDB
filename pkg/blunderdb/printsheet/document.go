package printsheet

import (
	"html"
	"strconv"
	"strings"
)

// Body sizes of the fitting script, in points.
const (
	MaxBodyPt  = 9.0
	MinBodyPt  = 5.5
	StepBodyPt = 0.25
)

// styles is the style sheet every document shares. Paper white and black
// ink in every theme: it is a print.
const styles = `@page { size: A4; margin: 0 }
html, body { margin: 0; padding: 0; background: #fff; color: #111 }
body { --ink: #111; --soft: #555; --paper: #fff;
  --cond: "Arial Narrow", "Liberation Sans Narrow", "Roboto Condensed", "Helvetica Neue", Arial, sans-serif;
  --mono: "DejaVu Sans Mono", "Liberation Mono", Menlo, Consolas, monospace }
.sheet { position: relative; width: 210mm; height: 297mm; box-sizing: border-box; padding: 9mm 8mm; overflow: hidden;
  background: var(--paper); color: var(--ink); font-family: var(--cond); font-size: 7pt; line-height: 1.2;
  font-variant-numeric: tabular-nums; break-after: page; page-break-after: always }
.doc:last-child .sheet:last-child { break-after: auto; page-break-after: auto }
@media screen { body { background: #ddd } .sheet { margin: 8mm auto; box-shadow: 0 0 2mm #888 } }
.mh { display: flex; justify-content: space-between; gap: 2em; align-items: flex-end; border-bottom: .6pt solid var(--ink); padding-bottom: .35em }
.mt { font-size: 1.9em; font-weight: 600; line-height: 1.1 }
.mm { color: var(--soft) }
.st { border-collapse: collapse }
.st th, .st td { padding: 0 0 0 1.2em; text-align: right; font-weight: 400 }
.st th { color: var(--soft) } .st td:first-child { text-align: left; font-weight: 600; padding-left: 0 }
.mc { font-style: italic; padding-top: .3em; white-space: pre-wrap }
.lg { color: var(--soft); padding: .3em 0 .5em }
.tr { column-gap: 1.5em; column-rule: .3pt solid #aaa; column-width: 20em }
.tr.flow { column-width: 30em }
.gh { font-weight: 600; border-bottom: .3pt solid var(--ink); margin-top: .45em; break-after: avoid; display: flex; justify-content: space-between; gap: 1em }
.gh span { font-weight: 400; color: var(--soft) }
.rw { display: grid; grid-template-columns: 1.7em 1fr 1fr; break-inside: avoid }
.no { color: var(--soft) }
.tr.flow .gb { text-align: left }
.tr.flow .rw { display: inline }
.tr.flow .rw .no { font-weight: 600; color: var(--ink); margin-left: .5em }
.tr.flow .rw .no::after { content: "." }
.tr.flow .rw .c2:not(:empty)::before { content: " / "; color: #999 }
.d { font-weight: 600 }
.g { font-weight: 600 }
.bl { font-weight: 600; text-decoration: underline }
.ref { font-weight: 600; font-size: .82em; background: var(--ink); color: var(--paper); padding: 0 .3em; border-radius: .25em; margin-left: .2em; border: .4pt solid var(--ink) }
.ref.o { background: var(--paper); color: var(--ink) }
.tm { font-style: italic; color: var(--soft); margin-left: .25em; font-size: .88em }
.ph { font-weight: 600; font-size: 1.15em; border-bottom: .6pt solid var(--ink); margin: .8em 0 .25em; display: flex; justify-content: space-between }
.ph span { font-weight: 400; font-size: .87em; color: var(--soft) }
.pl { column-count: 2; column-gap: 1.8em; column-rule: .3pt solid #aaa }
.pe { break-inside: avoid; padding: .12em 0; display: grid; grid-template-columns: 2.2em max-content 1fr; column-gap: .6em }
.pe > .ref { margin: 0; justify-self: start; align-self: start }
.nt { font-family: var(--mono); font-size: .93em; font-weight: 600; white-space: nowrap; min-width: 34ch }
.wh { color: var(--soft) } .cm { font-style: italic; white-space: pre-wrap }
.pe > .cm { grid-column: 2 / -1 }
.qrb { break-inside: avoid; display: flex; gap: 1.2em; align-items: flex-end; margin-top: .9em; color: var(--soft) }
.qrb svg { flex: none; display: block; shape-rendering: crispEdges }
`

// BodySizes is the scale the fitting script tries, largest first: MaxBodyPt
// down to MinBodyPt by StepBodyPt. Generated here so that Go tests the scale
// the browser walks.
func BodySizes() []float64 {
	var out []float64
	for i := 0; ; i++ {
		fs := MaxBodyPt - float64(i)*StepBodyPt
		if fs < MinBodyPt-StepBodyPt/2 {
			return out
		}
		out = append(out, fs)
	}
}

// bodySizesJS is BodySizes as a JavaScript array literal.
func bodySizesJS() string {
	parts := make([]string, 0, 16)
	for _, fs := range BodySizes() {
		parts = append(parts, strconv.FormatFloat(fs, 'f', -1, 64))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// fitScript sizes each sheet's body to the largest of SIZES that fits, moves
// the data-verso block to a second sheet when it cannot, runs a data-flow
// transcription as text when its lines still overflow, and spreads the
// positions of a verso still too long at the smallest size over further
// sheets, so that no position is ever hidden by the sheet's edge.
const fitScript = `(function () {
  var SIZES = /*SIZES*/, MIN = SIZES[SIZES.length - 1];
  function avail(s) { var c = getComputedStyle(s); return s.clientHeight - parseFloat(c.paddingTop) - parseFloat(c.paddingBottom); }
  function fits(s, fs) { s.style.fontSize = fs + 'pt'; return s.firstElementChild.scrollHeight <= avail(s); }
  function fit(s) {
    for (var i = 0; i < SIZES.length; i++) { if (fits(s, SIZES[i])) { s.dataset.fs = SIZES[i]; delete s.dataset.overflow; return true; } }
    s.dataset.fs = MIN; s.dataset.overflow = '1'; return false;
  }
  function flow(s) { var t = s.querySelector('[data-flow]'); if (!t || t.classList.contains('flow')) return false; t.classList.add('flow'); return true; }
  function sheetAfter(doc, s, head) {
    var n = document.createElement('div'); n.className = 'sheet';
    var inner = document.createElement('div'); n.appendChild(inner);
    if (head) inner.appendChild(head.cloneNode(true));
    doc.insertBefore(n, s.nextSibling);
    return n;
  }
  function entries(pl) { return pl.querySelectorAll(':scope > .pe'); }
  function spill(doc, s, head) {
    while (!fit(s)) {
      var pl = s.querySelector('.pl');
      if (!pl || entries(pl).length < 2) return;
      var next = sheetAfter(doc, s, head), npl = document.createElement('div');
      npl.className = 'pl'; next.firstElementChild.appendChild(npl);
      var es = entries(pl);
      while (es.length > 1 && !fits(s, MIN)) { npl.insertBefore(es[es.length - 1], npl.firstChild); es = entries(pl); }
      fit(s); s = next;
    }
  }
  document.querySelectorAll('.doc').forEach(function (doc) {
    var s = doc.querySelector('.sheet');
    if (!s || fit(s)) return;
    var v = s.querySelector('[data-verso]'), head = s.querySelector('.mh');
    if (v) {
      var back = sheetAfter(doc, s, head);
      back.firstElementChild.appendChild(v);
      if (!fit(s) && flow(s)) fit(s);
      spill(doc, back, head);
    } else if (flow(s)) fit(s);
  });
})();`

// Document wraps documents — each a <section class="doc"> holding its sheets
// — into one printable HTML file: the shared style sheet, the fitting script,
// and tail (an element read by a machine, such as arkqr.EmbedFrames) at the
// end of the body.
func Document(lang, title string, docs []string, tail string) []byte {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"" + html.EscapeString(lang) + "\"><head><meta charset=\"utf-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	b.WriteString("<meta name=\"generator\" content=\"blunderDB print sheet\">")
	b.WriteString("<title>" + html.EscapeString(title) + "</title>\n<style>\n" + styles + "</style></head><body>\n")
	for _, d := range docs {
		b.WriteString(d)
		b.WriteString("\n")
	}
	b.WriteString(tail)
	b.WriteString("<script>" + strings.Replace(fitScript, "/*SIZES*/", bodySizesJS(), 1) + "</script>\n</body></html>\n")
	return []byte(b.String())
}

// Sheet is one document of a single first sheet: the fitting script adds the
// verso when it needs one.
func Sheet(kind, content string) string {
	return `<section class="doc" data-doc="` + kind + `"><div class="sheet"><div>` + content + `</div></div></section>`
}
