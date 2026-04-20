import { get } from 'svelte/store';
import { CopyImageToClipboard } from '../../wailsjs/go/main/App.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, clipboardPositionStore } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { commentTextStore } from '../stores/uiStore.js';
import { matchContextStore } from '../stores/positionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { generateXGID } from './positionService.js';

export function copyPosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened');
        return;
    }
    console.log('copyPosition');
    const position = get(positionStore);
    const analysis = get(analysisStore);
    const comment = get(commentTextStore);

    clipboardPositionStore.set(JSON.parse(JSON.stringify({
        board: position.board,
        cube: position.cube,
        dice: position.dice,
        score: position.score,
        player_on_roll: position.player_on_roll,
        decision_type: position.decision_type,
        has_jacoby: position.has_jacoby,
        has_beaver: position.has_beaver,
    })));

    const xgid = analysis.xgid || generateXGID(position);

    let clipboardContent = `XGID=${xgid}\n\n`;

    clipboardContent += `Position:\n`;
    clipboardContent += `Board: ${JSON.stringify(position.board)}\n`;
    clipboardContent += `Cube: ${JSON.stringify(position.cube)}\n`;
    clipboardContent += `Dice: ${position.dice.join(', ')}\n`;
    clipboardContent += `Score: ${position.score.join(', ')}\n`;
    clipboardContent += `Player on roll: ${position.player_on_roll}\n`;
    clipboardContent += `Decision type: ${position.decision_type}\n\n`;

    clipboardContent += `Analysis:\n`;
    if (analysis.analysisType === "DoublingCube") {
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
    } else if (analysis.analysisType === "CheckerMove") {
        clipboardContent += `Checker Move Analysis:\n`;
        analysis.checkerAnalysis.moves.forEach(move => {
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

    navigator.clipboard.writeText(clipboardContent).then(() => {
        console.log('Position, analysis, and comment copied to clipboard');
        setStatusBarMessage('Position, analysis, and comment copied to clipboard');
    }).catch(err => {
        console.error('Error copying to clipboard:', err);
        setStatusBarMessage('Error copying to clipboard');
    });
}

export async function copyBoardImage() {
    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened');
        return;
    }
    try {
        const boardEl = document.getElementById('backgammon-board');
        if (!boardEl) {
            setStatusBarMessage('Board element not found');
            return;
        }
        const svgEl = boardEl.querySelector('svg');
        if (!svgEl) {
            setStatusBarMessage('Board SVG not found');
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
        const styleProps = ['fill', 'stroke', 'stroke-width', 'stroke-linecap', 'stroke-linejoin',
                            'stroke-miterlimit', 'opacity', 'font-family', 'font-size', 'font-weight',
                            'font-style', 'text-anchor', 'dominant-baseline', 'visibility', 'display'];
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

            const dataUrl = canvas.toDataURL('image/png');
            const base64Data = dataUrl.replace(/^data:image\/png;base64,/, '');
            try {
                await CopyImageToClipboard(base64Data);
                setStatusBarMessage('Board image copied to clipboard');
            } catch (err) {
                console.error('Failed to copy image to clipboard:', err);
                setStatusBarMessage('Failed to copy image to clipboard: ' + err);
            }
        };
        img.onerror = () => {
            URL.revokeObjectURL(url);
            setStatusBarMessage('Failed to render board image');
        };
        img.src = url;
    } catch (error) {
        console.error('Error copying board image:', error);
        setStatusBarMessage('Error copying board image');
    }
}

export async function copyBoardWithAnalysisImage() {
    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened');
        return;
    }
    try {
        const boardEl = document.getElementById('backgammon-board');
        if (!boardEl) {
            setStatusBarMessage('Board element not found');
            return;
        }
        const svgEl = boardEl.querySelector('svg');
        if (!svgEl) {
            setStatusBarMessage('Board SVG not found');
            return;
        }

        const analysis = get(analysisStore);
        const position = get(positionStore);
        if (!analysis || (!analysis.checkerAnalysis?.moves?.length && !analysis.doublingCubeAnalysis)) {
            setStatusBarMessage('No analysis data to export');
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
        const styleProps = ['fill', 'stroke', 'stroke-width', 'stroke-linecap', 'stroke-linejoin',
                            'stroke-miterlimit', 'opacity', 'font-family', 'font-size', 'font-weight',
                            'font-style', 'text-anchor', 'dominant-baseline', 'visibility', 'display'];
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
            const bgColor = '#f7f0e6';
            const font = '12px monospace';
            const headerFont = 'bold 12px monospace';
            const charWidth = 7.2;
            const rowHeight = 18;
            const padding = 10;
            const tablePadding = 4;

            const cubeValue = position.cube?.value || 1;
            const isCubeAnalysis = analysis.analysisType === 'DoublingCube' ||
                (!analysis.checkerAnalysis?.moves?.length && analysis.doublingCubeAnalysis);

            let analysisHeight = 0;
            let analysisWidth = svgWidth;

            if (isCubeAnalysis && analysis.doublingCubeAnalysis) {
                analysisHeight = padding + rowHeight * 9 + padding;
            } else if (analysis.checkerAnalysis?.moves?.length) {
                const moveCount = Math.min(analysis.checkerAnalysis.moves.length, 6);
                analysisHeight = padding + rowHeight * (moveCount + 1) + padding;
            }

            const totalHeight = svgHeight + analysisHeight;
            const canvas = document.createElement('canvas');
            canvas.width = analysisWidth * scale;
            canvas.height = totalHeight * scale;
            const ctx = canvas.getContext('2d');
            ctx.scale(scale, scale);

            ctx.fillStyle = bgColor;
            ctx.fillRect(0, 0, analysisWidth, totalHeight);

            ctx.drawImage(img, 0, 0, svgWidth, svgHeight);
            URL.revokeObjectURL(url);

            const startY = svgHeight + padding;
            ctx.font = font;
            ctx.textBaseline = 'middle';

            if (isCubeAnalysis && analysis.doublingCubeAnalysis) {
                drawCubeAnalysis(ctx, analysis.doublingCubeAnalysis, cubeValue, startY, analysisWidth, rowHeight, tablePadding, headerFont, font, charWidth);
            } else if (analysis.checkerAnalysis?.moves?.length) {
                drawCheckerAnalysis(ctx, analysis, startY, analysisWidth, rowHeight, tablePadding, headerFont, font, charWidth);
            }

            const dataUrl = canvas.toDataURL('image/png');
            const base64Data = dataUrl.replace(/^data:image\/png;base64,/, '');
            try {
                await CopyImageToClipboard(base64Data);
                setStatusBarMessage('Board + analysis image copied to clipboard');
            } catch (err) {
                console.error('Failed to copy image to clipboard:', err);
                setStatusBarMessage('Failed to copy image to clipboard: ' + err);
            }
        };
        img.onerror = () => {
            URL.revokeObjectURL(url);
            setStatusBarMessage('Failed to render board image');
        };
        img.src = url;
    } catch (error) {
        console.error('Error copying board+analysis image:', error);
        setStatusBarMessage('Error copying board+analysis image');
    }
}

function drawCubeAnalysis(ctx, cube, cubeValue, startY, totalWidth, rowHeight, pad, headerFont, font, charWidth) {
    const formatEq = (v) => (v >= 0 ? '+' : '') + (v || 0).toFixed(3);
    const getDecLabel = (d) => cubeValue >= 1 ? d.replace('Double', 'Redouble') : d;

    const colWidth = Math.floor(totalWidth / 3);
    const borderColor = '#ddd';
    const headerBg = '#f2f2f2';
    const whiteBg = '#ffffff';
    const playedBg = '#fff3cd';

    function drawCell(x, y, w, h, text, opts = {}) {
        ctx.fillStyle = opts.bg || whiteBg;
        ctx.fillRect(x, y, w, h);
        ctx.strokeStyle = borderColor;
        ctx.lineWidth = 0.5;
        ctx.strokeRect(x, y, w, h);
        ctx.fillStyle = '#000';
        ctx.font = opts.bold ? headerFont : font;
        ctx.textAlign = opts.align || 'center';
        const tx = opts.align === 'left' ? x + pad : x + w / 2;
        ctx.fillText(text, tx, y + h / 2);
    }

    const analysis = get(analysisStore);
    const matchCtx = get(matchContextStore);
    function normCubeAction(action) {
        const s = action.toLowerCase().replace(/\s+/g, '');
        if (s === 'double/take' || s === 'doubletake') return ['double', 'take'];
        if (s === 'double/pass' || s === 'doublepass') return ['double', 'pass'];
        if (s === 'nodouble' || s === 'nodoubleorredouble' || s === 'noredouble') return ['nodouble'];
        if (s === 'redouble') return ['double'];
        return [s];
    }
    function isPlayedAction(action) {
        const aParts = normCubeAction(action);
        if (matchCtx.isMatchMode) {
            if (analysis.playedCubeAction) {
                const pp = normCubeAction(analysis.playedCubeAction);
                return aParts.every(a => pp.includes(a));
            }
            return false;
        }
        const allParts = new Set();
        if (analysis.playedCubeActions?.length) {
            for (const pa of analysis.playedCubeActions) {
                for (const p of normCubeAction(pa)) allParts.add(p);
            }
        }
        if (allParts.size === 0 && analysis.playedCubeAction) {
            for (const p of normCubeAction(analysis.playedCubeAction)) allParts.add(p);
        }
        return allParts.size > 0 && aParts.every(a => allParts.has(a));
    }

    let y = startY;

    const leftX = 0;
    const leftW = colWidth;
    const cellW = Math.floor(leftW / 3);

    drawCell(leftX, y, cellW, rowHeight, '', { bg: headerBg, bold: true });
    drawCell(leftX + cellW, y, cellW, rowHeight, 'P', { bg: headerBg, bold: true });
    drawCell(leftX + cellW * 2, y, cellW, rowHeight, 'O', { bg: headerBg, bold: true });
    y += rowHeight;

    drawCell(leftX, y, cellW, rowHeight, 'W', { bold: true });
    drawCell(leftX + cellW, y, cellW, rowHeight, (cube.playerWinChances || 0).toFixed(2));
    drawCell(leftX + cellW * 2, y, cellW, rowHeight, (cube.opponentWinChances || 0).toFixed(2));
    y += rowHeight;

    drawCell(leftX, y, cellW, rowHeight, 'G', { bold: true });
    drawCell(leftX + cellW, y, cellW, rowHeight, (cube.playerGammonChances || 0).toFixed(2));
    drawCell(leftX + cellW * 2, y, cellW, rowHeight, (cube.opponentGammonChances || 0).toFixed(2));
    y += rowHeight;

    drawCell(leftX, y, cellW, rowHeight, 'B', { bold: true });
    drawCell(leftX + cellW, y, cellW, rowHeight, (cube.playerBackgammonChances || 0).toFixed(2));
    drawCell(leftX + cellW * 2, y, cellW, rowHeight, (cube.opponentBackgammonChances || 0).toFixed(2));
    y += rowHeight;

    drawCell(leftX, y, cellW, rowHeight, 'ND Eq', { bold: true });
    drawCell(leftX + cellW, y, cellW * 2, rowHeight, formatEq(cube.cubelessNoDoubleEquity));
    y += rowHeight;

    drawCell(leftX, y, cellW, rowHeight, 'D Eq', { bold: true });
    drawCell(leftX + cellW, y, cellW * 2, rowHeight, formatEq(cube.cubelessDoubleEquity));

    y = startY;
    const rightX = colWidth;
    const rightW = colWidth;
    const decW = Math.floor(rightW * 0.4);
    const eqW = Math.floor(rightW * 0.3);
    const errW = rightW - decW - eqW;

    drawCell(rightX, y, decW, rowHeight, 'Decision', { bg: headerBg, bold: true });
    drawCell(rightX + decW, y, eqW, rowHeight, 'Equity', { bg: headerBg, bold: true });
    drawCell(rightX + decW + eqW, y, errW, rowHeight, 'Error', { bg: headerBg, bold: true });
    y += rowHeight;

    const ndPlayed = isPlayedAction('No Double');
    const ndBg = ndPlayed ? playedBg : whiteBg;
    drawCell(rightX, y, decW, rowHeight, getDecLabel('No Double'), { bg: ndBg });
    drawCell(rightX + decW, y, eqW, rowHeight, formatEq(cube.cubefulNoDoubleEquity), { bg: ndBg });
    drawCell(rightX + decW + eqW, y, errW, rowHeight, formatEq(cube.cubefulNoDoubleError), { bg: ndBg });
    y += rowHeight;

    const dtPlayed = isPlayedAction('Double') && isPlayedAction('Take');
    const dtBg = dtPlayed ? playedBg : whiteBg;
    drawCell(rightX, y, decW, rowHeight, getDecLabel('Double/Take'), { bg: dtBg });
    drawCell(rightX + decW, y, eqW, rowHeight, formatEq(cube.cubefulDoubleTakeEquity), { bg: dtBg });
    drawCell(rightX + decW + eqW, y, errW, rowHeight, formatEq(cube.cubefulDoubleTakeError), { bg: dtBg });
    y += rowHeight;

    const dpPlayed = isPlayedAction('Double') && isPlayedAction('Pass');
    const dpBg = dpPlayed ? playedBg : whiteBg;
    drawCell(rightX, y, decW, rowHeight, getDecLabel('Double/Pass'), { bg: dpBg });
    drawCell(rightX + decW, y, eqW, rowHeight, formatEq(cube.cubefulDoublePassEquity), { bg: dpBg });
    drawCell(rightX + decW + eqW, y, errW, rowHeight, formatEq(cube.cubefulDoublePassError), { bg: dpBg });
    y += rowHeight;

    drawCell(rightX, y, decW, rowHeight, 'Best Action', { bold: true });
    drawCell(rightX + decW, y, eqW + errW, rowHeight, cube.bestCubeAction || '', { bold: true });

    y = startY;
    const infoX = colWidth * 2;
    const infoW = totalWidth - infoX;
    const infoLabelW = Math.floor(infoW * 0.5);
    const infoValW = infoW - infoLabelW;

    drawCell(infoX, y, infoLabelW, rowHeight, 'Analysis Depth', { bg: headerBg, bold: true });
    drawCell(infoX + infoLabelW, y, infoValW, rowHeight, cube.analysisDepth || '');
    y += rowHeight;

    drawCell(infoX, y, infoLabelW, rowHeight, 'Engine', { bg: headerBg, bold: true });
    drawCell(infoX + infoLabelW, y, infoValW, rowHeight, cube.analysisEngine || get(analysisStore).analysisEngineVersion || '');
}

function drawCheckerAnalysis(ctx, analysis, startY, totalWidth, rowHeight, pad, headerFont, font, charWidth) {
    const formatEq = (v) => (v >= 0 ? '+' : '') + (v || 0).toFixed(3);
    const borderColor = '#ddd';
    const headerBg = '#f2f2f2';
    const playedBg = '#fff3cd';
    const evenBg = '#fdfdfd';
    const sectionBorder = '#ccc';

    const cols = [
        { label: 'Move', frac: 0.18 },
        { label: 'Equity', frac: 0.08 },
        { label: 'Error', frac: 0.08 },
        { label: 'P W', frac: 0.07 },
        { label: 'P G', frac: 0.07 },
        { label: 'P B', frac: 0.07 },
        { label: 'O W', frac: 0.07 },
        { label: 'O G', frac: 0.07 },
        { label: 'O B', frac: 0.07 },
        { label: 'Depth', frac: 0.10 },
        { label: 'Engine', frac: 0.14 },
    ];

    let colPositions = [];
    let x = 0;
    for (const col of cols) {
        const w = Math.floor(totalWidth * col.frac);
        colPositions.push({ x, w, label: col.label });
        x += w;
    }
    colPositions[colPositions.length - 1].w = totalWidth - colPositions[colPositions.length - 1].x;

    const sectionBorders = [0, 2, 5, 8];

    function drawCell(cx, cy, cw, ch, text, opts = {}) {
        if (opts.bg) {
            ctx.fillStyle = opts.bg;
            ctx.fillRect(cx, cy, cw, ch);
        }
        ctx.strokeStyle = borderColor;
        ctx.lineWidth = 0.5;
        ctx.strokeRect(cx, cy, cw, ch);
        ctx.fillStyle = '#000';
        ctx.font = opts.bold ? headerFont : font;
        ctx.textAlign = opts.align || 'center';
        const tx = opts.align === 'left' ? cx + pad : cx + cw / 2;
        ctx.fillText(text, tx, cy + ch / 2);
    }

    function drawSectionBorders(cy) {
        ctx.strokeStyle = sectionBorder;
        ctx.lineWidth = 1.5;
        for (const colIdx of sectionBorders) {
            const col = colPositions[colIdx];
            const rx = col.x + col.w;
            ctx.beginPath();
            ctx.moveTo(rx, cy);
            ctx.lineTo(rx, cy + rowHeight);
            ctx.stroke();
        }
    }

    const matchCtx = get(matchContextStore);
    function normMove(m) {
        return m ? m.split(' ').sort().join(' ') : '';
    }
    function isPlayed(move) {
        if (!move.move) return false;
        const nm = normMove(move.move);
        if (matchCtx.isMatchMode) {
            return analysis.playedMove ? normMove(analysis.playedMove) === nm : false;
        }
        if (analysis.playedMoves?.length) {
            for (const pm of analysis.playedMoves) {
                if (normMove(pm) === nm) return true;
            }
        }
        if (analysis.playedMove) return normMove(analysis.playedMove) === nm;
        return false;
    }

    let y = startY;

    const moves = [...analysis.checkerAnalysis.moves].sort((a, b) => (b.equity || 0) - (a.equity || 0));
    const displayMoves = moves.slice(0, 6);

    for (const col of colPositions) {
        drawCell(col.x, y, col.w, rowHeight, col.label, { bg: headerBg, bold: true });
    }
    drawSectionBorders(y);
    y += rowHeight;

    for (let i = 0; i < displayMoves.length; i++) {
        const move = displayMoves[i];
        const played = isPlayed(move);
        const rowBg = played ? playedBg : (i % 2 === 1 ? evenBg : '#ffffff');

        const values = [
            move.move || '',
            formatEq(move.equity),
            formatEq(move.equityError),
            (move.playerWinChance || 0).toFixed(2),
            (move.playerGammonChance || 0).toFixed(2),
            (move.playerBackgammonChance || 0).toFixed(2),
            (move.opponentWinChance || 0).toFixed(2),
            (move.opponentGammonChance || 0).toFixed(2),
            (move.opponentBackgammonChance || 0).toFixed(2),
            move.analysisDepth || '',
            move.analysisEngine || '',
        ];

        for (let j = 0; j < colPositions.length; j++) {
            const col = colPositions[j];
            drawCell(col.x, y, col.w, rowHeight, values[j], { bg: rowBg, align: j === 0 ? 'left' : 'center' });
        }
        drawSectionBorders(y);
        y += rowHeight;
    }
}
