// Mouse handling of the board, extracted from Board.svelte: hit-testing of
// the drawn scene (checkers, cube, dice, player rectangles, scores) and the
// position edits each click performs in EDIT and EPC mode, plus the
// double-click reset and the right-click menu gate.
//
// attachBoardInteractions(canvas, deps) wires the DOM listeners and returns
// the function that removes them. It reads nothing global: the current mode,
// drawing size, board config, cube box and stores all come through `deps`,
// so the whole flow runs in jsdom against plain writable stores.
//
// Every hit test starts from boardMetrics() and the same layout helpers the
// scene is drawn with (boardScene.js), then normalises the click through
// boardMouseToDrawing(): the canvas may be CSS-scaled (interface zoom, side
// layout) and raw client pixels drift — at 90 % scale a click on point 1
// used to land on point 2.

import { get } from 'svelte/store';
import { boardMetrics, boardMouseToDrawing, checkerPointAndCountAt } from './boardGeometry.js';
import { EXCLUDE_EMPTY, sideLayout } from './boardScene.js';

// A second click on the same Except point within this delay blocks it.
// Detected by hand because native 'dblclick' is unreliable here: each click
// redraws (recreates) the two.js shapes, so the two clicks land on different
// DOM nodes.
const EXCEPT_DOUBLE_CLICK_MS = 450;

const MAX_CUBE_VALUE = 6; // log2 exponent: 64

function isEditable(mode) {
    return mode === 'EDIT' || mode === 'EPC';
}

/** A real roll: both dice on a face. Anything else is "no dice" (a cube decision). */
function hasRoll(dice) {
    return dice[0] >= 1 && dice[0] <= 6 && dice[1] >= 1 && dice[1] <= 6;
}

/**
 * Which side control (if any) a drawing-space point falls on:
 *   { die: 0|1|null, playerRect: 0|1|null, score: 0|1|null }
 * Player 0 is the bottom player, player 1 the top one. A die wins over the
 * rectangle it overlaps; a score box is carved out of its rectangle.
 */
export function hitTestSideControls(x, y, geom, cfg, playerOnRoll) {
    const side = sideLayout(geom, cfg, playerOnRoll);
    const cs = geom.checkerSize;
    const halfW = 0.75 * cs;
    const halfH = side.scoreHeight / 2;
    const inColumn = x >= side.scoreX - halfW && x <= side.scoreX + halfW;

    let die = null;
    for (let index = 0; index < 2; index++) {
        const dieX = side.diceX + index * (side.diceSize + side.diceGap);
        const half = side.diceSize / 2;
        if (x >= dieX - half && x <= dieX + half && y >= side.diceY - half && y <= side.diceY + half) die = index;
    }

    let score = null;
    let playerRect = null;
    const rows = [
        [side.score1Y, side.bearoff1Y],
        [side.score2Y, side.bearoff2Y]
    ];
    rows.forEach(([scoreY, bearoffY], player) => {
        if (!inColumn) return;
        if (y >= scoreY - halfH && y <= scoreY + halfH) {
            score = player;
        } else if (die === null && y >= Math.min(bearoffY, scoreY) && y <= Math.max(bearoffY, scoreY)) {
            playerRect = player; // the dice sit inside this rectangle and take precedence
        }
    });
    return { die, playerRect, score };
}

/** True when (x, y) lies outside the board proper (triangles and bar). */
export function isOutsideBoard(x, y, geom) {
    const { originX, originY, boardWidth, boardHeight } = geom;
    return x < originX - boardWidth / 2 || x > originX + boardWidth / 2 || y < originY - boardHeight / 2 || y > originY + boardHeight / 2;
}

/**
 * Put `count` checkers of the clicking button's colour on `point` (left →
 * colour 0, right → colour 1; the bars are colour-fixed). Clicking the fifth
 * checker of a stack of five or more adds one instead. A blocked Except point
 * is unblocked. Bearoff counts follow. `isSearchStructure` lifts the
 * 15-per-colour cap: a pattern may ask for e.g. 3 checkers on each of 1-6.
 */
export function applyCheckerEdit(pos, point, count, button, isSearchStructure) {
    if (pos.board.points[point]?.color === EXCLUDE_EMPTY) {
        pos.board.points = pos.board.points.map((p, i) => (i === point ? { checkers: 0, color: -1 } : p));
        return pos;
    }
    const color = point === 0 ? 1 : point === 25 ? 0 : button === 2 ? 1 : 0;

    const totalOtherPoints = pos.board.points.reduce((acc, p, idx) => (idx !== point && p.color === color ? acc + p.checkers : acc), 0);
    const maxPerPoint = isSearchStructure ? 15 : 15 - totalOtherPoints;
    if (maxPerPoint <= 0) return pos;

    pos.board.points = pos.board.points.map((p, index) => {
        if (index !== point) return p;
        if (p.checkers >= 5 && p.color === color && count === 5) {
            return { ...p, checkers: Math.min(p.checkers + 1, maxPerPoint) };
        }
        return { ...p, checkers: Math.min(count, maxPerPoint), color };
    });
    pos.board.points = pos.board.points.map((p) => (p.checkers === 0 ? { ...p, color: -1 } : p));

    // A search structure can exceed 15 checkers per colour; clamp bearoff at 0
    // (it is irrelevant to structure search anyway).
    const onBoard = [0, 1].map((c) => pos.board.points.reduce((acc, p) => acc + (p.color === c ? p.checkers : 0), 0));
    pos.board.bearoff = [Math.max(0, 15 - onBoard[0]), Math.max(0, 15 - onBoard[1])];
    return pos;
}

/**
 * Cube click. EPC: only the owner matters (money equities are in units of
 * the current cube), so clicks cycle centred → bottom owns → top owns →
 * centred (right-click backwards) and pin the value. Offered cube (take/pass
 * search): edit the value while keeping it centred, at least a double. EDIT:
 * a centred cube is taken by the clicking side; the owner's own button
 * raises it, the other lowers it, back to centred at 1.
 */
export function applyCubeClick(pos, button, { epc = false, offeredTakePass = false } = {}) {
    const up = (v) => Math.min(v + 1, MAX_CUBE_VALUE);
    if (epc) {
        const cycle = [-1, 0, 1];
        const dir = button === 2 ? -1 : 1;
        const cur = cycle.indexOf(pos.cube.owner === undefined ? -1 : pos.cube.owner);
        const next = cycle[(cur + dir + 3) % 3];
        pos.cube.owner = next;
        pos.cube.value = next === -1 ? 0 : 1;
        return pos;
    }
    if (offeredTakePass) {
        if (button === 0) pos.cube.value = up(pos.cube.value);
        else if (button === 2) pos.cube.value = Math.max(pos.cube.value - 1, 1);
        pos.cube.owner = -1;
        return pos;
    }
    if (pos.cube.owner === -1) {
        pos.cube.value = up(pos.cube.value);
        pos.cube.owner = button === 0 ? 0 : 1;
    } else if (pos.cube.owner === 0) {
        if (button === 0) pos.cube.value = up(pos.cube.value);
        else if (button === 2) pos.cube.value = Math.max(pos.cube.value - 1, 0);
    } else if (pos.cube.owner === 1) {
        if (button === 0) pos.cube.value = Math.max(pos.cube.value - 1, 0);
        else if (button === 2) pos.cube.value = up(pos.cube.value);
    }
    if (pos.cube.value === 0) pos.cube.owner = -1;
    return pos;
}

/**
 * Score click: left lowers the away count (down to money, -1), right raises
 * it (up to 99). Money is symmetric — reaching -1 on one side sets the
 * other, and leaving money by editing one side alone copies the score to
 * the other: an away score with no opponent away score is not a valid
 * match state (EPC's own money default is [-1, -1], the only state where a
 * lone -1 is meaningful).
 */
export function applyScoreClick(pos, player, button) {
    const other = 1 - player;
    if (button === 0) pos.score[player] = Math.max(pos.score[player] - 1, -1);
    else if (button === 2) pos.score[player] = Math.min(pos.score[player] + 1, 99);
    if (pos.score[player] === -1) pos.score[other] = -1;
    else if (pos.score[other] === -1) pos.score[other] = pos.score[player];
    return pos;
}

/**
 * Roll a die: left click up (6 wraps to 1), right click down (1 wraps to 6).
 * A cleared die (0) is a valid starting point — a board asking a cube
 * question has no dice — so stepping down from it wraps to 6 rather than
 * walking into negatives, which read as "no dice" forever after.
 */
function stepDie(value, button) {
    const v = value >= 1 && value <= 6 ? value : 0;
    if (button === 0) return (v % 6) + 1;
    if (button === 2) return v <= 1 ? 6 : v - 1;
    return value;
}

/**
 * Wire the board's mouse interactions to `canvas`. Returns the detach function.
 *
 * deps:
 *   getMode()            current status-bar mode ('EDIT' / 'EPC' edit the board)
 *   getSize()            { width, height } of the drawing surface
 *   cfg                  Board.svelte's boardCfg (orientation, widthFactor read live)
 *   getCubeBox()         { x, y, size } where the cube was last drawn
 *   stores               { position, structureMode, activeTab, offeredCube, anyModalOpen }
 *   getPreviousDice()    dice saved when a player rectangle cleared them
 *   setPreviousDice(d)
 *   reset()              blank the board (double-click outside, mode-specific)
 *   openContextMenu(at)  { x, y } client coordinates, NORMAL-like modes only
 *   logger               optional, `.log(...)`
 */
export function attachBoardInteractions(canvas, deps) {
    const { cfg, stores } = deps;
    const log = (...args) => deps.logger?.log(...args);
    let startMousePos = null;
    let lastExceptClick = null;

    const editable = () => isEditable(deps.getMode());
    const metrics = () => {
        const { width, height } = deps.getSize();
        return boardMetrics(width, height, cfg.widthFactor);
    };
    const toDrawing = (event) => {
        const { width, height } = deps.getSize();
        return boardMouseToDrawing(event.clientX, event.clientY, canvas.getBoundingClientRect(), width, height);
    };
    const checkerAt = (x, y) => {
        const { width, height } = deps.getSize();
        return checkerPointAndCountAt(x, y, width, height, cfg.widthFactor, cfg.orientation);
    };

    function cubeClick(event, x, y) {
        const box = deps.getCubeBox();
        if (!box || Math.abs(x - box.x) > box.size / 2 || Math.abs(y - box.y) > box.size / 2) return;
        const mode = deps.getMode();
        stores.position.update((pos) => applyCubeClick(pos, event.button, { epc: mode === 'EPC', offeredTakePass: get(stores.offeredCube) && pos.decision_type === 1 }));
    }

    // EPC mode shares this whole flow with EDIT mode: the Eval panel's
    // evaluation volet needs real dice to show candidate moves (no dice
    // means a cube verdict instead, EPCPanel.svelte) — a player's rectangle
    // clearing the dice to show the cube decision, then a die click
    // restoring/bumping them for a move decision, is exactly EDIT mode's
    // own toggle.
    function sideControlsClick(event, x, y) {
        const hit = hitTestSideControls(x, y, metrics(), cfg, get(stores.position).player_on_roll);
        if (hit.die === null && hit.playerRect === null && hit.score === null) return;
        log('side control clicked', hit);
        stores.position.update((pos) => {
            if (hit.die !== null) {
                pos.decision_type = 0;
                // Restore the dice the rectangle cleared — but ONLY when they
                // are actually cleared. With a roll already on the board (a
                // position loaded, pasted into the Eval panel, or just built
                // by hand) a die click steps THAT die; reinstating an older
                // roll used to overwrite both dice with a stale [0, 0], and
                // the click then left [n, 0] — half a roll, which every
                // reader of the position (EPCPanel's hasDiceSet, the engine's
                // own hasDice) takes for "no dice", i.e. a cube decision on a
                // board plainly asking a checker question.
                const base = hasRoll(pos.dice) ? pos.dice : deps.getPreviousDice();
                pos.dice = [base[0], base[1]]; // never alias previousDice
                pos.dice[hit.die] = stepDie(pos.dice[hit.die], event.button);
                // The two dice are set together or not at all (CONTEXT.md:
                // dice set → checker decision, no dice → cube decision).
                // Stepping one die of a cleared pair means "make this a
                // checker decision", so the other one comes along.
                const other = 1 - hit.die;
                if (pos.dice[other] < 1 || pos.dice[other] > 6) pos.dice[other] = 1;
            } else if (hit.playerRect !== null) {
                pos.player_on_roll = hit.playerRect;
                pos.decision_type = 1; // doubling cube decision
                deps.setPreviousDice([pos.dice[0], pos.dice[1]]);
                pos.dice = [0, 0];
            } else {
                applyScoreClick(pos, hit.score, event.button);
            }
            return pos;
        });
    }

    function onMouseDown(event) {
        event.preventDefault(); // no text or element selection
        // Blur any focused text field when clicking the board
        if (document.activeElement && document.activeElement.matches('input, textarea, [contenteditable]')) {
            /** @type {HTMLElement} */ (document.activeElement).blur();
        }
        if (!editable()) return;
        const { x, y } = toDrawing(event);
        startMousePos = { x, y, button: event.button };
        cubeClick(event, x, y);
        sideControlsClick(event, x, y);
    }

    function onMouseMove(event) {
        event.preventDefault();
    }

    function onMouseUp(event) {
        event.preventDefault();
        if (!editable() || !startMousePos) return;
        const end = { ...toDrawing(event), button: event.button };

        // In the Except structure, a quick second click on the same point
        // blocks it (must be empty).
        if (deps.getMode() === 'EDIT' && get(stores.structureMode) === 'exclude') {
            const { checkerPoint } = checkerAt(end.x, end.y);
            if (checkerPoint >= 1 && checkerPoint <= 24) {
                const isMarker = get(stores.position).board.points[checkerPoint]?.color === EXCLUDE_EMPTY;
                const now = Date.now();
                if (!isMarker && lastExceptClick && lastExceptClick.point === checkerPoint && now - lastExceptClick.time < EXCEPT_DOUBLE_CLICK_MS) {
                    lastExceptClick = null;
                    stores.position.update((pos) => {
                        pos.board.points = pos.board.points.map((p, i) => (i === checkerPoint ? { checkers: 1, color: EXCLUDE_EMPTY } : p));
                        return pos;
                    });
                    return;
                }
                // A click on a blocked point unblocks it (applyCheckerEdit); don't
                // let it seed a double-click that would immediately re-block.
                lastExceptClick = isMarker ? null : { point: checkerPoint, time: now };
            } else {
                lastExceptClick = null;
            }
        }

        fillCheckersBetween(startMousePos, end);
    }

    // A press-and-release across several points fills them all with the
    // taller of the two clicked counts (a drag along the home board).
    function fillCheckersBetween(startPos, endPos) {
        const start = checkerAt(startPos.x, startPos.y);
        const end = checkerAt(endPos.x, endPos.y);
        if (start.checkerPoint === -1 && end.checkerPoint === -1) return; // outside the board
        const count = Math.max(start.checkerCount, end.checkerCount);
        const from = Math.min(start.checkerPoint, end.checkerPoint);
        const to = Math.max(start.checkerPoint, end.checkerPoint);
        const isSearchStructure = get(stores.activeTab) === 'search';
        for (let point = from; point <= to; point++) {
            stores.position.update((pos) => applyCheckerEdit(pos, point, count, startPos.button, isSearchStructure));
        }
    }

    function onDoubleClick(event) {
        if (!editable()) return;
        const { x, y } = toDrawing(event);
        if (isOutsideBoard(x, y, metrics())) deps.reset();
    }

    // Right-clicking the board opens actions on the position it shows, but
    // ONLY in the modes where the right button is otherwise idle. In EDIT
    // and EPC the right button already means "place the other colour's
    // checker", so the menu stays out of their way.
    function onContextMenu(event) {
        event.preventDefault(); // no native menu, in every mode
        if (editable()) return;
        if (get(stores.anyModalOpen)) return;
        deps.openContextMenu({ x: event.clientX, y: event.clientY });
    }

    canvas.addEventListener('mousedown', onMouseDown);
    canvas.addEventListener('mousemove', onMouseMove);
    canvas.addEventListener('mouseup', onMouseUp);
    canvas.addEventListener('dblclick', onDoubleClick);
    canvas.addEventListener('contextmenu', onContextMenu);

    return function detach() {
        canvas.removeEventListener('mousedown', onMouseDown);
        canvas.removeEventListener('mousemove', onMouseMove);
        canvas.removeEventListener('mouseup', onMouseUp);
        canvas.removeEventListener('dblclick', onDoubleClick);
        canvas.removeEventListener('contextmenu', onContextMenu);
    };
}
