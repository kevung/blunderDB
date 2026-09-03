// canvasTable.js — a generic {header?, rows} table rasterizer for a 2D canvas
// context: column widths, cell borders, zebra striping, section rules.
//
// Extracted from clipboardService.js (fiche D.10, #210): "copy board with analysis"
// embedded this canvas rasterizer alongside the backgammon-specific glue that decides
// WHAT to paint (utils/analysisRows.js's rows). This module knows nothing about cube
// decisions or checker moves — it only knows how to lay cells on a grid — so
// clipboardService.js's paintAnalysisStrip is the only caller that still mixes the two,
// and it does so by calling into this module rather than by containing it.
const STRIP = { rowHeight: 18, padding: 10, cellPad: 4, font: '12px monospace', headerFont: 'bold 12px monospace' };
const INK = { border: '#ddd', section: '#ccc', header: '#f2f2f2', white: '#ffffff', even: '#fdfdfd', played: '#fff3cd' };

// splitWidth cuts a width into columns by fraction; the last column absorbs
// the rounding so the table ends exactly on its right edge.
function splitWidth(width, fractions) {
    const total = fractions.reduce((a, b) => a + b, 0);
    const widths = fractions.map((f) => Math.floor((width * f) / total));
    widths[widths.length - 1] += width - widths.reduce((a, b) => a + b, 0);
    return widths;
}

function paintCell(ctx, x, y, w, text, { bg = INK.white, bold = false, align = 'center' } = {}) {
    const h = STRIP.rowHeight;
    ctx.fillStyle = bg;
    ctx.fillRect(x, y, w, h);
    ctx.strokeStyle = INK.border;
    ctx.lineWidth = 0.5;
    ctx.strokeRect(x, y, w, h);
    ctx.fillStyle = '#000';
    ctx.font = bold ? STRIP.headerFont : STRIP.font;
    ctx.textBaseline = 'middle';
    ctx.textAlign = align;
    ctx.fillText(text, align === 'left' ? x + STRIP.cellPad : x + w / 2, y + h / 2);
}

// paintTable lays a {header?, rows} block on the canvas. A row whose cells
// stop short of the last column spans it with its last cell (the verdict,
// a cubeless equity). `sections` names the columns after which a heavier
// rule separates the groups, as the DOM table's borders do.
function paintTable(ctx, x0, y0, widths, { header = null, rows }, { boldLabels = false, leftLabels = false, labelBg = INK.white, zebra = false, sections = [] } = {}) {
    const h = STRIP.rowHeight;
    const edges = widths.reduce((acc, w) => [...acc, acc[acc.length - 1] + w], [x0]);
    let y = y0;

    function paintRow(cells) {
        cells.forEach((cell, i) => {
            const w = i === cells.length - 1 ? edges[widths.length] - edges[i] : widths[i];
            paintCell(ctx, edges[i], y, w, cell.text, cell);
        });
        ctx.strokeStyle = INK.section;
        ctx.lineWidth = 1.5;
        for (const s of sections) {
            ctx.beginPath();
            ctx.moveTo(edges[s + 1], y);
            ctx.lineTo(edges[s + 1], y + h);
            ctx.stroke();
        }
        y += h;
    }

    if (header) paintRow(header.map((text) => ({ text, bg: INK.header, bold: true })));
    rows.forEach((row, i) => {
        const bg = row.highlight ? INK.played : zebra && i % 2 === 1 ? INK.even : INK.white;
        paintRow([
            { text: row.label, bg: row.highlight ? bg : labelBg, bold: boldLabels || !!row.bold, align: leftLabels ? 'left' : 'center' },
            ...row.cells.map((text) => ({ text, bg, bold: !!row.bold }))
        ]);
    });
}

export { STRIP, INK, splitWidth, paintTable };
