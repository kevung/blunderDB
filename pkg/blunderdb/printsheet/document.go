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
.qrb { break-inside: avoid; display: flex; flex-wrap: wrap; gap: 1.2em; align-items: flex-end; margin-top: .9em; color: var(--soft) }
.qrb svg { flex: none; display: block; shape-rendering: crispEdges }
.pe.cont > .cm::before, .de.cont > .cm:first-child::before { content: "… " }
.dl { column-gap: 1.6em; column-rule: .3pt solid #aaa }
.dl.two { column-count: 2 } .dl.many { column-width: 27em } .dl.ans { column-width: 30em }
.de { display: flex; gap: .6em; align-items: baseline; break-inside: avoid }
.de .no { min-width: 1.9em; text-align: right; font-weight: 600; flex: none }
.de .nt { flex: none } .de .an { min-width: 0 }
.de.cont { display: block; padding-left: 2.5em }
.pg { position: absolute; bottom: 4mm; right: 8mm; color: var(--soft); font-size: 6pt }
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
  function height(x) { var h = 0, rs = x.getClientRects(); for (var k = 0; k < rs.length; k++) h += rs[k].height; return h; }
  // long tells whether entry x, just laid on sheet s at the smallest size,
  // is worth cutting there: comments taller than a third of the page, and
  // the rest of the entry fitting without them.
  function long(x, s) {
    var cms = x.querySelectorAll('.cm'), t = [], ok;
    if (!cms.length || height(x) <= avail(s) / 3) return false;
    cms.forEach(function (cm, k) { t[k] = cm.textContent; cm.textContent = ''; });
    ok = fits(s, MIN);
    cms.forEach(function (cm, k) { cm.textContent = t[k]; });
    fits(s, MIN);
    return ok;
  }
  function entries(pl) { return pl.querySelectorAll(':scope > .pe'); }
  // split cuts entry e, alone on sheet s and taller than it, at the last
  // word that still fits, and returns the rest as a continuation entry: its
  // comments, moved whole from the end, then the end of the one cut.
  function split(e, s) {
    if (!e) return null;
    var cms = e.querySelectorAll('.cm'), c = e.cloneNode(false), k;
    c.classList.add('cont');
    for (k = cms.length - 1; k >= 0; k--) {
      var cm = cms[k], text = cm.textContent;
      cm.textContent = '';
      if (!fits(s, MIN)) { cm.textContent = text; c.insertBefore(cm, c.firstChild); continue; }
      var lo = 0, hi = text.length;
      while (lo < hi) { var mid = (lo + hi + 1) >> 1; cm.textContent = text.slice(0, mid); if (fits(s, MIN)) lo = mid; else hi = mid - 1; }
      var cut = text.lastIndexOf(' ', lo);
      if (cut <= 0) cut = lo;
      cm.textContent = text.slice(0, cut);
      var rest = cm.cloneNode(false); rest.textContent = text.slice(cut).replace(/^\s+/, '');
      c.insertBefore(rest, c.firstChild);
      break;
    }
    return c.firstChild ? c : null;
  }
  function spill(doc, s, head) {
    while (!fit(s)) {
      var pl = s.querySelector('.pl');
      if (!pl || entries(pl).length < 1) return;
      var next = sheetAfter(doc, s, head), npl = document.createElement('div');
      npl.className = 'pl'; next.firstElementChild.appendChild(npl);
      var es = entries(pl);
      while (es.length > 1 && !fits(s, MIN)) { npl.insertBefore(es[es.length - 1], npl.firstChild); es = entries(pl); }
      if (!fits(s, MIN)) {
        var c = split(es[0], s);
        if (!c) { doc.removeChild(next); fit(s); return; }
        npl.insertBefore(c, npl.firstChild);
      }
      fit(s); s = next;
    }
  }
  // list lays a data-fill document out page by page: header and legend on
  // each, every page holding the lines that fit at the smallest size; with a
  // data-verso block, each front page is followed by the back that carries
  // the same numbers. A line taller than a page goes on over the next one.
  // A reimport code (.qrb, data-from) goes on the front page that holds the
  // first line of its tranche. Every page is then fitted and numbered.
  function list(doc) {
    var s = doc.querySelector('.sheet'), inner = s.firstElementChild;
    var dl = inner.querySelector(':scope > .dl'), v = inner.querySelector(':scope > [data-verso]');
    if (!dl) return;
    var es = Array.prototype.slice.call(dl.children), as = null, vframe = null, qs = {};
    es.forEach(function (e) { dl.removeChild(e); });
    var qrs = inner.querySelector(':scope > .qrs');
    if (qrs) {
      Array.prototype.slice.call(qrs.children).forEach(function (q) {
        var k = parseInt(q.getAttribute('data-from'), 10) - 1;
        if (k >= 0 && k < es.length) qs[k] = q; else return;
        qrs.removeChild(q);
      });
      inner.removeChild(qrs);
    }
    if (v) {
      var vdl = v.querySelector('.dl');
      as = Array.prototype.slice.call(vdl.children);
      as.forEach(function (e) { vdl.removeChild(e); });
      inner.removeChild(v);
      vframe = inner.cloneNode(true);
      vframe.replaceChild(v, vframe.querySelector(':scope > .dl'));
      var vlg = vframe.querySelector(':scope > .lg'); if (vlg) vframe.removeChild(vlg);
    }
    var rframe = inner.cloneNode(true), pages = [], i = 0, r = s, last = s;
    function page(frame) {
      var n = document.createElement('div'); n.className = 'sheet'; n.appendChild(frame.cloneNode(true));
      doc.insertBefore(n, last.nextSibling); last = n; return n;
    }
    for (;;) {
      var b = vframe ? page(vframe) : null, rp = r.querySelector('.dl'), bp = b && b.querySelector('[data-verso] .dl'), n = 0;
      pages.push(r); if (b) pages.push(b);
      while (i < es.length) {
        var e = es[i], a = as && as[i], q = e && qs[i];
        if (e) rp.appendChild(e);
        if (q) { rp.parentNode.appendChild(q); qs[i] = null; }
        if (a) bp.appendChild(a);
        if (!e && a && n === 0 && b) {
          // An answer going on past its back: the front of this sheet,
          // empty otherwise, carries it first, then its back.
          rp.appendChild(a);
          if (fits(r, MIN)) { i++; n++; continue; }
          var rest = split(a, r);
          bp.appendChild(rest);
          if (fits(b, MIN)) { i++; n++; continue; }
          as[i] = split(rest, b);
          if (!as[i]) i++;
          break;
        }
        var okR = fits(r, MIN), okB = !b || fits(b, MIN);
        if (okR && okB) { i++; n++; continue; }
        if (n > 0) {
          // A long line starts where the page has room rather than leaving
          // it nearly empty; a short one moves whole to the next page.
          var c = okB ? (e && long(e, r) ? split(e, r) : null) : (okR && long(a, b) ? split(a, b) : null);
          if (!c) {
            if (e) rp.removeChild(e);
            if (a) bp.removeChild(a);
            if (q) { q.parentNode.removeChild(q); qs[i] = q; }
            break;
          }
          if (okB) { es[i] = c; if (as) as[i] = null; } else { as[i] = c; es[i] = null; }
          break;
        }
        var c = okR ? split(a, b) : split(e, r);
        if (!c) i++;
        else if (okR) { as[i] = c; es[i] = null; }
        else { es[i] = c; if (as) as[i] = null; }
        break;
      }
      if (i >= es.length) break;
      r = page(rframe);
    }
    // What is left of the codes (a list that could not be coded says why)
    // closes the first page.
    if (qrs && qrs.firstChild) s.firstElementChild.appendChild(qrs);
    pages.forEach(function (x, k) {
      fit(x);
      var p = document.createElement('div'); p.className = 'pg'; p.textContent = (k + 1) + ' / ' + pages.length; x.appendChild(p);
    });
  }
  document.querySelectorAll('.doc').forEach(function (doc) {
    if (doc.hasAttribute('data-fill')) { list(doc); return; }
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

// ListSheet is a document laid out line by line over as many pages as it
// takes (the script's data-fill layout): content holds the header, the
// legend, the .dl block of lines and, optionally, the data-verso block of
// their backs.
func ListSheet(kind, content string) string {
	return `<section class="doc" data-doc="` + kind + `" data-fill><div class="sheet"><div>` + content + `</div></div></section>`
}
