import { tMsg, translate } from '../i18n';
import { get } from 'svelte/store';
import { CopyImageToClipboard } from '../../wailsjs/go/gui/App.js';
import { ClipboardSetText } from '../../wailsjs/runtime/runtime.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, clipboardPositionStore } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { commentTextStore, statusBarModeStore } from '../stores/uiStore.js';
import { matchContextStore } from '../stores/positionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { generateXGID } from './positionService.js';
import { logger } from '../utils/logger.js';
import { cubeDecision, cubeTurnability, isMoneyPosition } from '../utils/cubeDecision.js';
import { cubeRows, cubeInfoRows, cubeFactRows, checkerRows, playedCubePredicate, playedMovePredicate } from '../utils/analysisRows.js';

// Write a PNG rendered from a <canvas> to the clipboard, walking the image
// clipboard's fallback ladder (see docs/adr/0004). Rung 1 is the WebView's own
// clipboard — it needs no external tool, so it is tried first. Only if the
// WebView declines does it hand off to the Go backend, which tries an external
// tool (xclip/wl-copy) and, failing that, saves the PNG to a file. Returns
// { method: 'clipboard' } or { method: 'file', path } so the caller can tell the
// user where the image ended up. Throws only if every rung — including the file
// save — fails.
async function writeCanvasToClipboard(canvas) {
    // Rung 1: native WebView clipboard.
    try {
        if (navigator.clipboard && typeof window.ClipboardItem === 'function') {
            const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/png'));
            if (blob) {
                await navigator.clipboard.write([new window.ClipboardItem({ 'image/png': blob })]);
                return { method: 'clipboard' };
            }
        }
    } catch (err) {
        logger.log('Native clipboard image write unavailable, falling back to backend:', err);
    }
    // Rung 2+: Go backend. Returns the saved file path when it fell back to a
    // file, or an empty string when it reached the system clipboard.
    const dataUrl = canvas.toDataURL('image/png');
    const base64Data = dataUrl.replace(/^data:image\/png;base64,/, '');
    const savedPath = await CopyImageToClipboard(base64Data);
    return savedPath ? { method: 'file', path: savedPath } : { method: 'clipboard' };
}

export function copyPosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    logger.log('copyPosition');
    const position = get(positionStore);
    const analysis = get(analysisStore);
    const comment = get(commentTextStore);
    const mode = get(statusBarModeStore);

    clipboardPositionStore.set(
        JSON.parse(
            JSON.stringify({
                board: position.board,
                cube: position.cube,
                dice: position.dice,
                score: position.score,
                player_on_roll: position.player_on_roll,
                decision_type: position.decision_type,
                has_jacoby: position.has_jacoby,
                has_beaver: position.has_beaver
            })
        )
    );

    // A scratch board — the Eval panel (EPC) and the search board (EDIT) —
    // shows a position no stored record describes, but analysisStore still
    // holds the record last opened: its xgid, its analysis and its comment all
    // belong to a DIFFERENT position. Copying the board there must therefore
    // regenerate the XGID from the board itself and carry nothing else, which
    // is exactly what this clipboard is for in the Eval panel: pasting the
    // position into XG, or into another blunderDB.
    const scratchBoard = mode === 'EPC' || mode === 'EDIT';
    const xgid = !scratchBoard && analysis.xgid ? analysis.xgid : generateXGID(position);

    let clipboardContent = `XGID=${xgid}\n\n`;

    clipboardContent += `Position:\n`;
    clipboardContent += `Board: ${JSON.stringify(position.board)}\n`;
    clipboardContent += `Cube: ${JSON.stringify(position.cube)}\n`;
    clipboardContent += `Dice: ${position.dice.join(', ')}\n`;
    clipboardContent += `Score: ${position.score.join(', ')}\n`;
    clipboardContent += `Player on roll: ${position.player_on_roll}\n`;
    clipboardContent += `Decision type: ${position.decision_type}\n\n`;

    if (scratchBoard) {
        writeTextToClipboard(clipboardContent)
            .then(() => {
                logger.log('Scratch-board position copied to clipboard');
                setStatusBarMessage(tMsg('status.positionCopied'));
            })
            .catch((err) => {
                logger.error('Error copying to clipboard:', err);
                setStatusBarMessage(tMsg('status.errorCopyingClipboard'));
            });
        return;
    }

    clipboardContent += `Analysis:\n`;
    if (analysis.analysisType === 'DoublingCube') {
        clipboardContent += `Doubling Cube Analysis:\n`;
        clipboardContent += `Analysis Depth: "${analysis.doublingCubeAnalysis.analysisDepth}"\n`;
        clipboardContent += `Player Win Chances: ${analysis.doublingCubeAnalysis.playerWinChances}%\n`;
        clipboardContent += `Player Gammon Chances: ${analysis.doublingCubeAnalysis.playerGammonChances}%\n`;
        clipboardContent += `Player Backgammon Chances: ${analysis.doublingCubeAnalysis.playerBackgammonChances}%\n`;
        clipboardContent += `Opponent Win Chances: ${analysis.doublingCubeAnalysis.opponentWinChances}%\n`;
        clipboardContent += `Opponent Gammon Chances: ${analysis.doublingCubeAnalysis.opponentGammonChances}%\n`;
        clipboardContent += `Opponent Backgammon Chances: ${analysis.doublingCubeAnalysis.opponentBackgammonChances}%\n`;
        clipboardContent += `Cubeless No Double Equity: ${analysis.doublingCubeAnalysis.cubelessNoDoubleEquity}\n`;
        clipboardContent += `Cubeless Double Equity: ${analysis.doublingCubeAnalysis.cubelessDoubleEquity}\n`;
        clipboardContent += `Cubeful No Double Equity: ${analysis.doublingCubeAnalysis.cubefulNoDoubleEquity}\n`;
        clipboardContent += `Cubeful No Double Error: ${analysis.doublingCubeAnalysis.cubefulNoDoubleError}\n`;
        clipboardContent += `Cubeful Double Take Equity: ${analysis.doublingCubeAnalysis.cubefulDoubleTakeEquity}\n`;
        clipboardContent += `Cubeful Double Take Error: ${analysis.doublingCubeAnalysis.cubefulDoubleTakeError}\n`;
        clipboardContent += `Cubeful Double Pass Equity: ${analysis.doublingCubeAnalysis.cubefulDoublePassEquity}\n`;
        clipboardContent += `Cubeful Double Pass Error: ${analysis.doublingCubeAnalysis.cubefulDoublePassError}\n`;
        clipboardContent += `Best Cube Action: ${analysis.doublingCubeAnalysis.bestCubeAction}\n`;
        clipboardContent += `Wrong Pass Percentage: ${analysis.doublingCubeAnalysis.wrongPassPercentage}%\n`;
        clipboardContent += `Wrong Take Percentage: ${analysis.doublingCubeAnalysis.wrongTakePercentage}%\n`;

        if (comment && comment.trim() !== '') {
            clipboardContent += `\n${comment}\n\n`;
        }
    } else if (analysis.analysisType === 'CheckerMove') {
        clipboardContent += `Checker Move Analysis:\n`;
        analysis.checkerAnalysis.moves.forEach((move) => {
            clipboardContent += `Move ${move.index}: ${move.move}\n`;
            clipboardContent += `Analysis Depth: "${move.analysisDepth}"\n`;
            clipboardContent += `Equity: ${move.equity}\n`;
            if (move.equityError !== undefined) {
                clipboardContent += `Equity Error: ${move.equityError}\n`;
            }
            clipboardContent += `Player Win Chance: ${move.playerWinChance}%\n`;
            clipboardContent += `Player Gammon Chance: ${move.playerGammonChance}%\n`;
            clipboardContent += `Player Backgammon Chance: ${move.playerBackgammonChance}%\n`;
            clipboardContent += `Opponent Win Chance: ${move.opponentWinChance}%\n`;
            clipboardContent += `Opponent Gammon Chance: ${move.opponentGammonChance}%\n`;
            clipboardContent += `Opponent Backgammon Chance: ${move.opponentBackgammonChance}%\n\n`;
        });

        if (comment && comment.trim() !== '') {
            clipboardContent += `\n${comment}\n\n`;
        }
    }

    if (analysis.analysisEngineVersion) {
        clipboardContent += `eXtreme Gammon Version: ${analysis.analysisEngineVersion}\n`;
    }

    writeTextToClipboard(clipboardContent)
        .then(() => {
            logger.log('Position, analysis, and comment copied to clipboard');
            setStatusBarMessage(tMsg('status.positionCopied'));
        })
        .catch((err) => {
            logger.error('Error copying to clipboard:', err);
            setStatusBarMessage(tMsg('status.errorCopyingClipboard'));
        });
}

// Put text on the system clipboard through the Go backend rather than
// navigator.clipboard. navigator.clipboard.writeText() needs transient user
// activation; a keydown carrying Ctrl is treated as a shortcut and grants none
// in the WebView, which is exactly the difference between the toolbar button (a
// click, always activated — it worked) and Ctrl-C (it did not). The backend has
// no such requirement, and it is the same clipboard the paste side already reads
// with ClipboardGetText(). The WebView call stays as a fallback for hosts where
// the backend one is unavailable.
async function writeTextToClipboard(text) {
    try {
        if (await ClipboardSetText(text)) return;
        throw new Error('backend clipboard write returned false');
    } catch (err) {
        logger.log('Backend clipboard write unavailable, falling back to the WebView:', err);
        await navigator.clipboard.writeText(text);
    }
}

export async function copyBoardImage() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    try {
        const boardEl = document.getElementById('backgammon-board');
        if (!boardEl) {
            setStatusBarMessage(tMsg('status.boardElementNotFound'));
            return;
        }
        const svgEl = boardEl.querySelector('svg');
        if (!svgEl) {
            setStatusBarMessage(tMsg('status.boardSvgNotFound'));
            return;
        }
        const svgWidth = parseInt(svgEl.getAttribute('width')) || svgEl.clientWidth;
        const svgHeight = parseInt(svgEl.getAttribute('height')) || svgEl.clientHeight;

        const clonedSvg = /** @type {SVGSVGElement} */ (svgEl.cloneNode(true));
        clonedSvg.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
        clonedSvg.setAttribute('width', String(svgWidth));
        clonedSvg.setAttribute('height', String(svgHeight));

        const origElements = svgEl.querySelectorAll('*');
        const clonedElements = clonedSvg.querySelectorAll('*');
        const styleProps =
            'fill stroke stroke-width stroke-linecap stroke-linejoin stroke-miterlimit opacity font-family font-size font-weight font-style text-anchor dominant-baseline visibility display'.split(
                ' '
            );
        for (let i = 0; i < origElements.length; i++) {
            const orig = origElements[i];
            const cloned = clonedElements[i];
            if (!cloned || !(cloned instanceof SVGElement)) continue;
            const computed = window.getComputedStyle(orig);
            for (const prop of styleProps) {
                const val = computed.getPropertyValue(prop);
                if (val) {
                    cloned.style.setProperty(prop, val);
                }
            }
        }

        const serializer = new XMLSerializer();
        const svgString = serializer.serializeToString(clonedSvg);
        const svgBlob = new Blob([svgString], { type: 'image/svg+xml;charset=utf-8' });
        const url = URL.createObjectURL(svgBlob);

        const img = new Image();
        img.onload = async () => {
            const scale = 2;
            const canvas = document.createElement('canvas');
            canvas.width = svgWidth * scale;
            canvas.height = svgHeight * scale;
            const ctx = canvas.getContext('2d');
            ctx.scale(scale, scale);

            ctx.fillStyle = '#f7f0e6';
            ctx.fillRect(0, 0, svgWidth, svgHeight);

            ctx.drawImage(img, 0, 0, svgWidth, svgHeight);
            URL.revokeObjectURL(url);

            try {
                const res = await writeCanvasToClipboard(canvas);
                if (res.method === 'file') {
                    setStatusBarMessage(tMsg('status.boardImageSavedToFile', { path: res.path }));
                } else {
                    setStatusBarMessage(tMsg('status.boardImageCopied'));
                }
            } catch (err) {
                logger.error('Failed to copy image to clipboard:', err);
                setStatusBarMessage(tMsg('status.failedCopyImage', { err }));
            }
        };
        img.onerror = () => {
            URL.revokeObjectURL(url);
            setStatusBarMessage(tMsg('status.failedRenderBoard'));
        };
        img.src = url;
    } catch (error) {
        logger.error('Error copying board image:', error);
        setStatusBarMessage(tMsg('status.errorCopyingBoardImage'));
    }
}

export async function copyBoardWithAnalysisImage() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    try {
        const boardEl = document.getElementById('backgammon-board');
        if (!boardEl) {
            setStatusBarMessage(tMsg('status.boardElementNotFound'));
            return;
        }
        const svgEl = boardEl.querySelector('svg');
        if (!svgEl) {
            setStatusBarMessage(tMsg('status.boardSvgNotFound'));
            return;
        }

        const analysis = get(analysisStore);
        const position = get(positionStore);
        const strip = analysisStrip(analysis);
        if (!strip) {
            setStatusBarMessage(tMsg('status.noAnalysisToExport'));
            return;
        }

        const svgWidth = parseInt(svgEl.getAttribute('width')) || svgEl.clientWidth;
        const svgHeight = parseInt(svgEl.getAttribute('height')) || svgEl.clientHeight;

        const clonedSvg = /** @type {SVGSVGElement} */ (svgEl.cloneNode(true));
        clonedSvg.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
        clonedSvg.setAttribute('width', String(svgWidth));
        clonedSvg.setAttribute('height', String(svgHeight));
        const origElements = svgEl.querySelectorAll('*');
        const clonedElements = clonedSvg.querySelectorAll('*');
        const styleProps = [
            'fill',
            'stroke',
            'stroke-width',
            'stroke-linecap',
            'stroke-linejoin',
            'stroke-miterlimit',
            'opacity',
            'font-family',
            'font-size',
            'font-weight',
            'font-style',
            'text-anchor',
            'dominant-baseline',
            'visibility',
            'display'
        ];
        for (let i = 0; i < origElements.length; i++) {
            const orig = origElements[i];
            const cloned = clonedElements[i];
            if (!cloned || !(cloned instanceof SVGElement)) continue;
            const computed = window.getComputedStyle(orig);
            for (const prop of styleProps) {
                const val = computed.getPropertyValue(prop);
                if (val) cloned.style.setProperty(prop, val);
            }
        }

        const serializer = new XMLSerializer();
        const svgString = serializer.serializeToString(clonedSvg);
        const svgBlob = new Blob([svgString], { type: 'image/svg+xml;charset=utf-8' });
        const url = URL.createObjectURL(svgBlob);

        const img = new Image();
        img.onload = async () => {
            const scale = 2;
            const stripHeight = STRIP.padding * 2 + STRIP.rowHeight * strip.rows;
            const totalHeight = svgHeight + stripHeight;
            const canvas = document.createElement('canvas');
            canvas.width = svgWidth * scale;
            canvas.height = totalHeight * scale;
            const ctx = canvas.getContext('2d');
            ctx.scale(scale, scale);

            ctx.fillStyle = '#f7f0e6';
            ctx.fillRect(0, 0, svgWidth, totalHeight);
            ctx.drawImage(img, 0, 0, svgWidth, svgHeight);
            URL.revokeObjectURL(url);

            paintAnalysisStrip(ctx, { analysis, position, isMatchMode: get(matchContextStore).isMatchMode, y: svgHeight + STRIP.padding, width: svgWidth });

            try {
                const res = await writeCanvasToClipboard(canvas);
                if (res.method === 'file') {
                    setStatusBarMessage(tMsg('status.boardImageSavedToFile', { path: res.path }));
                } else {
                    setStatusBarMessage(tMsg('status.boardAnalysisCopied'));
                }
            } catch (err) {
                logger.error('Failed to copy image to clipboard:', err);
                setStatusBarMessage(tMsg('status.failedCopyImage', { err }));
            }
        };
        img.onerror = () => {
            URL.revokeObjectURL(url);
            setStatusBarMessage(tMsg('status.failedRenderBoard'));
        };
        img.src = url;
    } catch (error) {
        logger.error('Error copying board+analysis image:', error);
        setStatusBarMessage(tMsg('status.errorCopyingBoardAnalysis'));
    }
}

// ---------------------------------------------------------------------------
// The analysis strip painted under the board. Every string it paints comes
// from utils/analysisRows.js — the rows CubeVerdictTable and
// CandidateMovesTable render — so the image cannot say something the screen
// does not; what remains here is geometry: where a cell goes and what colour
// sits behind it.
// ---------------------------------------------------------------------------

const STRIP = { rowHeight: 18, padding: 10, cellPad: 4, font: '12px monospace', headerFont: 'bold 12px monospace' };
const INK = { border: '#ddd', section: '#ccc', header: '#f2f2f2', white: '#ffffff', even: '#fdfdfd', played: '#fff3cd' };
// The image ranks the candidates itself and keeps the top of the list.
const MAX_IMAGE_MOVES = 6;
// The cube strip is as tall as its tallest block: the facts grid (a header
// and five rows).
const CUBE_STRIP_ROWS = 6;

// analysisStrip says which block the image gets — a cube record, or the top
// of a checker list — and how many rows it takes. Null when there is nothing
// to paint.
export function analysisStrip(analysis) {
    const cube = analysis?.doublingCubeAnalysis;
    const moves = analysis?.checkerAnalysis?.moves ?? [];
    const isCube = analysis?.analysisType === 'DoublingCube' || (!moves.length && !!cube);
    if (isCube && cube) return { kind: 'cube', rows: CUBE_STRIP_ROWS };
    if (moves.length) return { kind: 'checker', rows: Math.min(moves.length, MAX_IMAGE_MOVES) + 1 };
    return null;
}

// paintAnalysisStrip paints the strip at y. Exported for the parity test:
// what it hands fillText is what the DOM tables show.
export function paintAnalysisStrip(ctx, { analysis, position, isMatchMode = false, y, width }) {
    const strip = analysisStrip(analysis);
    if (!strip) return;
    const t = translate;
    // ADR-0016 point 6 / #190/C.3: the same referential the DOM tables read
    // off position, so the copied image's equity header never drifts from
    // the screen's.
    const isMoney = isMoneyPosition(position);

    if (strip.kind === 'cube') {
        const cube = analysis.doublingCubeAnalysis;
        const decision = cubeDecision({ cubeAnalysis: cube, turnability: cubeTurnability(position), stored: true });
        const block = cubeRows(decision, { t, cubeValue: position?.cube?.value ?? 0, isPlayedCubeAction: playedCubePredicate(analysis, isMatchMode), isMoney });
        const third = Math.floor(width / 3);

        paintTable(ctx, 0, y, splitWidth(third, [1, 1, 1]), cubeFactRows(cube), { boldLabels: true });
        paintTable(ctx, third, y, splitWidth(third, [0.4, 0.3, 0.3]), { header: block.header, rows: [...block.rows, { label: block.verdict.label, cells: [block.verdict.text], bold: true }] });
        paintTable(
            ctx,
            third * 2,
            y,
            splitWidth(width - third * 2, [0.5, 0.5]),
            { rows: cubeInfoRows(cube, { t, engineFallback: analysis.analysisEngineVersion }) },
            { boldLabels: true, labelBg: INK.header }
        );
        return;
    }

    const moves = [...analysis.checkerAnalysis.moves].sort((a, b) => (b.equity || 0) - (a.equity || 0)).slice(0, MAX_IMAGE_MOVES);
    const block = checkerRows(moves, { t, isPlayedMove: playedMovePredicate(analysis, isMatchMode), isMoney });
    const widths = splitWidth(width, [0.18, 0.08, 0.08, 0.07, 0.07, 0.07, 0.07, 0.07, 0.07, 0.1, 0.14]);
    paintTable(ctx, 0, y, widths, block, { leftLabels: true, zebra: true, sections: [0, 2, 5, 8] });
}

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
