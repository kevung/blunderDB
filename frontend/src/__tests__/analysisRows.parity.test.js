/**
 * analysisRows.parity.test.js
 *
 * "Copy board with analysis" paints the analysis on a canvas, and used to
 * replay the DOM tables' formatting rules by hand — four formatEquity, six
 * duplicated .toFixed(2) — so the image could say something the screen did
 * not (it captioned a centred cube "Redouble"). Both surfaces now read
 * utils/analysisRows.js; this test holds them to it: every string the DOM
 * tables render is a string the canvas hands fillText, in the same order,
 * with the same row highlighted, in whatever language is active.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/gui/App.js', () => ({ CopyImageToClipboard: vi.fn() }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardSetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ generateXGID: () => 'XGID-STUB' }));

const { paintAnalysisStrip, analysisStrip } = await import('../services/clipboardService.js');
const { default: CubeVerdictTable } = await import('../components/CubeVerdictTable.svelte');
const { default: CandidateMovesTable } = await import('../components/CandidateMovesTable.svelte');
const { cubeDecision, cubeTurnability } = await import('../utils/cubeDecision.js');
const { playedCubePredicate, playedMovePredicate } = await import('../utils/analysisRows.js');
const { language } = await import('../i18n');

const PLAYED_BG = '#fff3cd';

// A 2D context that keeps what it is asked to write, and the ground each
// string was written on.
function fakeContext() {
    const painted = [];
    let lastFill = null;
    const ctx = {
        fillStyle: '#000',
        strokeStyle: '#000',
        lineWidth: 1,
        font: '',
        textAlign: 'left',
        textBaseline: 'alphabetic',
        fillRect() {
            lastFill = this.fillStyle;
        },
        fillText(text) {
            painted.push({ text: String(text), bg: lastFill });
        },
        strokeRect() {},
        beginPath() {},
        moveTo() {},
        lineTo() {},
        stroke() {},
        scale() {},
        drawImage() {}
    };
    return { ctx, painted };
}

function domTexts(container) {
    return [...container.querySelectorAll('th, td')].map((el) => el.textContent.trim());
}

// The DOM sequence must occur, contiguous and in order, in the painted one.
function indexOfRun(haystack, needle) {
    outer: for (let i = 0; i + needle.length <= haystack.length; i++) {
        for (let j = 0; j < needle.length; j++) if (haystack[i + j] !== needle[j]) continue outer;
        return i;
    }
    return -1;
}

const position = { cube: { owner: -1, value: 0 }, score: [-1, -1], player_on_roll: 0 };

const cubeAnalysis = {
    analysisType: 'DoublingCube',
    analysisEngineVersion: 'eXtreme Gammon 2.19',
    playedCubeAction: 'No Double',
    playedCubeActions: ['No Double'],
    doublingCubeAnalysis: {
        analysisDepth: 'XG Roller++',
        playerWinChances: 72.15,
        playerGammonChances: 20.5,
        playerBackgammonChances: 1.02,
        opponentWinChances: 27.85,
        opponentGammonChances: 5.1,
        opponentBackgammonChances: 0,
        cubelessNoDoubleEquity: 0.612,
        cubelessDoubleEquity: 1.224,
        cubefulNoDoubleEquity: 0.85,
        cubefulNoDoubleError: -0.15,
        cubefulDoubleTakeEquity: 1.0,
        cubefulDoubleTakeError: 0,
        cubefulDoublePassEquity: 1.0,
        cubefulDoublePassError: 0,
        bestCubeAction: 'Double/Take'
    }
};

const checkerAnalysis = {
    analysisType: 'CheckerMove',
    playedMove: '13/10 24/23',
    playedMoves: ['13/10 24/23'],
    checkerAnalysis: {
        moves: [
            {
                index: 0,
                move: '8/5 6/5',
                analysisDepth: '4-ply',
                analysisEngine: 'XG',
                equity: 0.201,
                playerWinChance: 58.4,
                playerGammonChance: 15.25,
                playerBackgammonChance: 0.6,
                opponentWinChance: 41.6,
                opponentGammonChance: 8.8,
                opponentBackgammonChance: 0.2
            },
            {
                index: 1,
                move: '24/23 13/10',
                analysisDepth: '4-ply',
                analysisEngine: 'XG',
                equity: 0.123,
                equityError: -0.078,
                playerWinChance: 55.123,
                playerGammonChance: 12.5,
                playerBackgammonChance: 0.4,
                opponentWinChance: 44.877,
                opponentGammonChance: 9.99,
                opponentBackgammonChance: 0
            },
            {
                index: 2,
                move: '24/21 13/12',
                analysisDepth: '2-ply',
                analysisEngine: 'GNU',
                equity: -0.02,
                equityError: -0.221,
                playerWinChance: 50,
                playerGammonChance: 10,
                playerBackgammonChance: 0.1,
                opponentWinChance: 50,
                opponentGammonChance: 11,
                opponentBackgammonChance: 0.5
            }
        ]
    }
};

function paint(analysis, extra = {}) {
    const { ctx, painted } = fakeContext();
    paintAnalysisStrip(ctx, { analysis, position, isMatchMode: false, y: 0, width: 900, ...extra });
    return painted;
}

describe('the copied image paints exactly what the tables show', () => {
    afterEach(() => {
        cleanup();
        language.set('en');
    });

    test.each(['en', 'fr', 'ja'])('cube decision, %s', (lang) => {
        language.set(lang);
        const { container } = render(CubeVerdictTable, {
            props: {
                decision: cubeDecision({ cubeAnalysis: cubeAnalysis.doublingCubeAnalysis, turnability: cubeTurnability(position), stored: true }),
                cubeAnalysis: cubeAnalysis.doublingCubeAnalysis,
                cubeValue: position.cube.value,
                isPlayedCubeAction: playedCubePredicate(cubeAnalysis, false),
                engineVersionFallback: cubeAnalysis.analysisEngineVersion,
                // `position.score` is [-1, -1] (money) — the same isMoney
                // paintAnalysisStrip now derives from the same position
                // (#190/C.3 point 4), kept in sync here so the DOM's equity
                // header and the canvas's stay identical.
                isMoney: true
            }
        });
        const dom = domTexts(container);
        const painted = paint(cubeAnalysis);
        const texts = painted.map((p) => p.text);

        // 3 headers, 3 × 3 option cells, 2 verdict cells, 2 × 2 info cells.
        expect(dom).toHaveLength(18);
        expect(dom).toContain('+0.850');
        expect(dom).toContain('Double/Take');
        const at = indexOfRun(texts, dom);
        expect(at, `DOM strings must be a run of the painted ones\nDOM: ${dom.join(' | ')}\ncanvas: ${texts.join(' | ')}`).toBeGreaterThanOrEqual(0);
        // Nothing painted after that run: the DOM table is the last block.
        expect(texts.slice(at)).toEqual(dom);

        // The same row is marked played on both surfaces.
        const playedRows = [...container.querySelectorAll('tr.played')].map((tr) => tr.querySelector('td').textContent.trim());
        expect(playedRows).toHaveLength(1);
        const paintedPlayed = painted.filter((p) => p.bg === PLAYED_BG).map((p) => p.text);
        expect(paintedPlayed[0]).toBe(playedRows[0]);
        expect(paintedPlayed).toHaveLength(3);
    });

    test('a centred cube is captioned "No Double" on the image as on the screen (it used to say "Redouble")', () => {
        const texts = paint(cubeAnalysis).map((p) => p.text);
        expect(texts).toContain('No Double');
        expect(texts.some((s) => s.includes('Redouble'))).toBe(false);
    });

    test.each(['en', 'fr'])('checker candidates, %s', (lang) => {
        language.set(lang);
        const { container } = render(CandidateMovesTable, {
            props: {
                moves: checkerAnalysis.checkerAnalysis.moves,
                sortColumn: '',
                isPlayedMove: playedMovePredicate(checkerAnalysis, false),
                isMoney: true
            }
        });
        const dom = domTexts(container);
        const painted = paint(checkerAnalysis);
        const texts = painted.map((p) => p.text);

        expect(dom).toHaveLength(11 * 4);
        expect(texts).toEqual(dom);

        const playedRows = [...container.querySelectorAll('tr.played')].map((tr) => tr.querySelector('td').textContent.trim());
        expect(playedRows).toEqual(['24/23 13/10']);
        const paintedPlayed = painted.filter((p) => p.bg === PLAYED_BG).map((p) => p.text);
        expect(paintedPlayed).toHaveLength(11);
        expect(paintedPlayed[0]).toBe('24/23 13/10');
    });

    test('the best move, whose error is absent by construction, reads +0.000 on both surfaces', () => {
        const { container } = render(CandidateMovesTable, { props: { moves: checkerAnalysis.checkerAnalysis.moves, sortColumn: '' } });
        const firstRow = [...container.querySelectorAll('tbody tr')[0].querySelectorAll('td')].map((td) => td.textContent.trim());
        expect(firstRow.slice(0, 3)).toEqual(['8/5 6/5', '+0.201', '+0.000']);
        expect(paint(checkerAnalysis).map((p) => p.text)).toContain('+0.000');
        expect(paint(checkerAnalysis).map((p) => p.text)).not.toContain('—');
    });

    test('the strip is sized by what it paints', () => {
        expect(analysisStrip(cubeAnalysis)).toEqual({ kind: 'cube', rows: 6 });
        expect(analysisStrip(checkerAnalysis)).toEqual({ kind: 'checker', rows: 4 });
        expect(analysisStrip({ checkerAnalysis: { moves: new Array(10).fill({ move: 'x' }) } })).toEqual({ kind: 'checker', rows: 7 });
        expect(analysisStrip({})).toBeNull();
        expect(analysisStrip(null)).toBeNull();
    });
});
