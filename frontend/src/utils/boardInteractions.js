// Mouse handling of the board: hit-testing of the drawn scene and the position edits each click
// performs in EDIT and EVAL mode, plus the double-click Pile gesture and the right-click menu gate.
// Everything comes through `deps` (no globals), so the flow runs in jsdom on plain stores.
// Every hit test goes through boardMouseToDrawing(): the canvas may be CSS-scaled (interface
// zoom, side layout) and raw client pixels drift.

import { get } from 'svelte/store';
import { boardMetrics, boardMouseToDrawing, checkerPointAndCountAt } from './boardGeometry.js';
import { EXCLUDE_EMPTY, sideLayout } from './boardScene.js';
import { OFF } from '../services/quizPlay.js';
import { dragStep, resetBoardPlay } from '../services/transcriptionPlay.js';
import { boardRightClick, diceClick, orderedDice, playClickedChecker } from '../services/boardMove.js';

// A second click on the same Except point within this delay blocks it. Detected by hand: each
// click recreates the two.js shapes, so native 'dblclick' sees two different DOM nodes.
const EXCEPT_DOUBLE_CLICK_MS = 450;

const MAX_CUBE_VALUE = 6; // log2 exponent: 64

/** @typedef {import('./boardGeometry.js').BoardMetrics} BoardMetrics */
/** @typedef {import('./boardGeometry.js').BoardPosition} BoardPosition */
/** @typedef {import('./boardConfig.js').BoardConfig} BoardConfig */
/** @typedef {{ x: number, y: number, button: number }} PressPoint */

/** @param {string} mode */
function isEditable(mode) {
    return mode === 'EDIT' || mode === 'EVAL';
}

/**
 * A real roll: both dice on a face. Anything else is "no dice" (a cube decision).
 * @param {number[]} dice
 */
function hasRoll(dice) {
    return dice[0] >= 1 && dice[0] <= 6 && dice[1] >= 1 && dice[1] <= 6;
}

/**
 * Le clic tombe-t-il sur le plateau de sortie de `player` ? Boîte du « (n OFF) » de drawBearoff,
 * élargie de moitié pour ne pas viser le texte au pixel.
 *
 * @param {number} x
 * @param {number} y
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} playerOnRoll
 * @param {number} player
 */
export function hitTestBearoffTray(x, y, geom, cfg, playerOnRoll, player) {
    const side = sideLayout(geom, cfg, playerOnRoll);
    const cs = geom.checkerSize;
    const trayY = player === 0 ? side.bearoff1Y : side.bearoff2Y;
    return Math.abs(x - side.bearoffX) <= 1.5 * cs && Math.abs(y - trayY) <= 0.75 * cs;
}

/**
 * Which side control a drawing-space point falls on: { die, playerRect, score }, each 0|1|null
 * (0 = bottom player). A die wins over its rectangle; a score box is carved out of it.
 * @param {number} x
 * @param {number} y
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} playerOnRoll
 * @returns {{ die: number|null, playerRect: number|null, score: number|null }}
 */
export function hitTestSideControls(x, y, geom, cfg, playerOnRoll) {
    const side = sideLayout(geom, cfg, playerOnRoll);
    const cs = geom.checkerSize;
    const halfW = 0.75 * cs;
    const halfH = side.scoreHeight / 2;
    const inColumn = x >= side.scoreX - halfW && x <= side.scoreX + halfW;

    /** @type {number|null} */
    let die = null;
    for (let index = 0; index < 2; index++) {
        const dieX = side.diceX + index * (side.diceSize + side.diceGap);
        const half = side.diceSize / 2;
        if (x >= dieX - half && x <= dieX + half && y >= side.diceY - half && y <= side.diceY + half) die = index;
    }

    /** @type {number|null} */
    let score = null;
    /** @type {number|null} */
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

/**
 * True when (x, y) lies outside the board proper (triangles and bar).
 * @param {number} x
 * @param {number} y
 * @param {BoardMetrics} geom
 */
export function isOutsideBoard(x, y, geom) {
    const { originX, originY, boardWidth, boardHeight } = geom;
    return x < originX - boardWidth / 2 || x > originX + boardWidth / 2 || y < originY - boardHeight / 2 || y > originY + boardHeight / 2;
}

/**
 * The checkers of a new game: 2 on the 24-point, 5 on the 13, 3 on the 8, 5 on the 6 for each
 * side (colour 0 counts from point 1, colour 1 from point 24), nothing borne off. Cube, score
 * and dice are the caller's.
 * @template {BoardPosition} P
 * @param {P} pos
 * @returns {P}
 */
export function applyStartingCheckers(pos) {
    pos.board.points = pos.board.points.map(() => ({ checkers: 0, color: -1 }));
    for (const [point, checkers] of [
        [24, 2],
        [13, 5],
        [8, 3],
        [6, 5]
    ]) {
        pos.board.points[point] = { checkers, color: 0 };
        pos.board.points[25 - point] = { checkers, color: 1 };
    }
    pos.board.bearoff = [0, 0];
    return pos;
}

/**
 * Put `count` checkers of the clicking button's colour on `point` (left → colour 0, right →
 * colour 1; bars are colour-fixed). Clicking the fifth checker of a stack of five or more adds
 * one. A blocked Except point is unblocked. `isSearchStructure` lifts the 15-per-colour cap.
 * @param {BoardPosition} pos
 * @param {number} point
 * @param {number} count
 * @param {number} button
 * @param {boolean} isSearchStructure
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

    // A search structure may exceed 15 per colour: clamp bearoff at 0 (irrelevant there).
    const onBoard = [0, 1].map((c) => pos.board.points.reduce((acc, p) => acc + (p.color === c ? p.checkers : 0), 0));
    pos.board.bearoff = [Math.max(0, 15 - onBoard[0]), Math.max(0, 15 - onBoard[1])];
    return pos;
}

/**
 * Cube click. EVAL: only the owner matters (money equities are in cube units), so clicks cycle
 * centred → bottom → top owns (right-click backwards). Offered cube (take/pass search): edit the
 * value, centred, at least a double. EDIT: a centred cube is taken by the clicking side; the
 * owner's button raises it, the other lowers it, back to centred at 1.
 * @param {BoardPosition} pos
 * @param {number} button
 * @param {{ evalMode?: boolean, offeredTakePass?: boolean }} [options]
 */
export function applyCubeClick(pos, button, { evalMode = false, offeredTakePass = false } = {}) {
    const up = (/** @type {number} */ v) => Math.min(v + 1, MAX_CUBE_VALUE);
    if (evalMode) {
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
 * Score click: left lowers the away count (down to money, -1), right raises it (up to 99).
 * Money is symmetric: an away score facing a lone -1 is not a valid match state, so reaching or
 * leaving money on one side copies it to the other.
 * @param {BoardPosition} pos
 * @param {number} player
 * @param {number} button
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
 * Roll a die: left up (6 → 1), right down (1 → 6). A cleared die (0) steps down to 6, not into
 * negatives that would read as "no dice" forever.
 * @param {number} value
 * @param {number} button
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
 *   getMode()            current status-bar mode ('EDIT' / 'EVAL' edit the board)
 *   getSize()            { width, height } of the drawing surface
 *   cfg                  Board.svelte's boardCfg (orientation, widthFactor read live)
 *   getCubeBox()         { x, y, size } where the cube was last drawn
 *   stores               { position, structureMode, activeTab, offeredCube, anyModalOpen,
 *                          quizPlay, quizPlayValidate, transcriptionCube }; quizPlayValidate
 *                        holds the armed mode's validation: set, the play follows ADR-0086
 *   getPreviousDice()    dice saved when a player rectangle cleared them
 *   setPreviousDice(d)
 *   openContextMenu(at)  { x, y } client coordinates; in EDIT/EVAL only outside the frame and
 *                        its controls, where the right button edits nothing
 *   displayRoller()      the player on roll as drawn: the side the dice are drawn on
 *   togglePile()         the Pile gesture, on a double click outside the frame
 *   container            optional element around the canvas: its margins are outside the frame
 *   duel                 optional { holds(), press(hit), drop(from, to), context(hit) }: while
 *                        a Duel holds the board, every click is its own (services/duelBoard.js)
 *   logger               optional, `.log(...)`
 *
 * @param {HTMLElement} canvas
 * @param {any} deps
 */
export function attachBoardInteractions(canvas, deps) {
    const { cfg, stores } = deps;
    const log = (/** @type {unknown[]} */ ...args) => deps.logger?.log(...args);
    /** @type {PressPoint|null} */
    let startMousePos = null;
    /** @type {{ point: number, time: number }|null} */
    let lastExceptClick = null;
    const editable = () => isEditable(deps.getMode());
    const metrics = () => {
        const { width, height } = deps.getSize();
        return boardMetrics(width, height, cfg.widthFactor);
    };
    const toDrawing = (/** @type {MouseEvent} */ event) => {
        const { width, height } = deps.getSize();
        return boardMouseToDrawing(event.clientX, event.clientY, canvas.getBoundingClientRect(), width, height);
    };
    const checkerAt = (/** @type {number} */ x, /** @type {number} */ y) => {
        const { width, height } = deps.getSize();
        return checkerPointAndCountAt(x, y, width, height, cfg.widthFactor, cfg.orientation);
    };

    /**
     * @param {MouseEvent} event
     * @param {number} x
     * @param {number} y
     */
    function cubeClick(event, x, y) {
        const box = deps.getCubeBox();
        if (!box || Math.abs(x - box.x) > box.size / 2 || Math.abs(y - box.y) > box.size / 2) return;
        const mode = deps.getMode();
        stores.position.update((/** @type {BoardPosition} */ pos) =>
            applyCubeClick(pos, event.button, { evalMode: mode === 'EVAL', offeredTakePass: get(stores.offeredCube) && pos.decision_type === 1 })
        );
    }

    // EVAL shares this flow with EDIT: the Eval panel needs real dice for candidate moves and
    // none for a cube verdict, which is exactly EDIT's rectangle/die toggle.
    /**
     * @param {MouseEvent} event
     * @param {number} x
     * @param {number} y
     */
    function sideControlsClick(event, x, y) {
        const hit = hitTestSideControls(x, y, metrics(), cfg, get(stores.position).player_on_roll);
        if (hit.die === null && hit.playerRect === null && hit.score === null) return;
        log('side control clicked', hit);
        stores.position.update((/** @type {BoardPosition} */ pos) => {
            if (hit.die !== null) {
                pos.decision_type = 0;
                // Restore the cleared dice ONLY when they are cleared: with a roll on the board a
                // die click steps that die. Reinstating a stale [0, 0] would leave [n, 0], a half
                // roll every reader takes for "no dice", i.e. a cube decision.
                const base = hasRoll(pos.dice) ? pos.dice : deps.getPreviousDice();
                pos.dice = [base[0], base[1]]; // never alias previousDice
                pos.dice[hit.die] = stepDie(pos.dice[hit.die], event.button);
                // Both dice are set together or not at all (CONTEXT.md: dice → checker decision).
                const other = 1 - hit.die;
                if (pos.dice[other] < 1 || pos.dice[other] > 6) pos.dice[other] = 1;
            } else if (hit.playerRect !== null) {
                pos.player_on_roll = hit.playerRect;
                pos.decision_type = 1; // doubling cube decision
                deps.setPreviousDice([pos.dice[0], pos.dice[1]]);
                pos.dice = [0, 0];
            } else if (hit.score !== null) {
                applyScoreClick(pos, hit.score, event.button);
            }
            return pos;
        });
    }

    /**
     * Le point (ou le plateau de sortie) visé pendant une question de quiz, null hors du damier.
     * @param {number} x
     * @param {number} y
     */
    function quizTargetAt(x, y) {
        // Le plateau de sortie du camp au trait, en bas ou en haut selon l'affichage (en
        // transcription le joueur 1 reste en bas, le joueur 2 sort donc par le haut).
        const bearoffSide = deps.quizBearoffSide?.() ?? 0;
        if (hitTestBearoffTray(x, y, metrics(), cfg, get(stores.position).player_on_roll, bearoffSide)) return OFF;
        return pointAt(x, y);
    }

    /**
     * Le point du MODÈLE visé, null hors du damier. Joueur 2 au trait, le plateau est montré
     * retourné : conversion de mirrorPosition, 25 - p, qui échange aussi les barres.
     * @param {number} x
     * @param {number} y
     */
    function pointAt(x, y) {
        const { checkerPoint } = checkerAt(x, y);
        if (checkerPoint < 0 || checkerPoint > 25) return null;
        return deps.quizDisplayMirrored?.() ? 25 - checkerPoint : checkerPoint;
    }

    /**
     * Clic sur le videau en transcription : le camp au trait propose un double, comme la touche
     * `d`. Le plateau pose seulement la demande (`transcriptionCubeRequestStore`) ; le panneau,
     * qui tient le brouillon, décide, et ne refuse rien (Incohérence marquée, ADR-0044).
     * Le clic ne RÉPOND jamais : une cible unique porterait une seule des deux réponses selon un
     * état que l'œil ne relit pas ; prise et passe vivent dans la rangée `[T] [P]`.
     * Rend `true` quand le geste est pris. Essayé en PREMIER : un coup joué arme `quizPlay`, qui
     * avale sinon tous les clics du damier.
     * @param {MouseEvent} event
     * @param {number} x
     * @param {number} y
     */
    function transcriptionCubeClick(event, x, y) {
        if (!stores.transcriptionCube || deps.getMode() !== 'TRANSCRIBE') return false;
        // Le bouton droit ouvre le menu de la position, ici comme ailleurs.
        if (event.button !== 0) return false;
        const box = deps.getCubeBox();
        if (!box || Math.abs(x - box.x) > box.size / 2 || Math.abs(y - box.y) > box.size / 2) return false;
        log('transcription cube clicked');
        stores.transcriptionCube.set('double');
        return true;
    }

    /**
     * Le coup armé selon la grammaire d'ADR-0086 (`quizPlayValidate` posé par son mode), ou null.
     */
    function grammarPlay() {
        const play = stores.quizPlay ? get(stores.quizPlay) : null;
        if (!play || !stores.quizPlayValidate) return null;
        return get(stores.quizPlayValidate) ? play : null;
    }

    /**
     * Le jet dans l'ordre dessiné : le jet saisi du coup, sinon les dés de la position.
     * @param {any} play
     */
    function drawnDice(play) {
        return orderedDice(play.rolled ?? get(stores.position).dice, !!play.swapped);
    }

    // Le pion pressé d'un coup selon la grammaire : relâché sur place, c'est un clic (le premier
    // dé non joué) ; relâché ailleurs, un glissé vers ce point.
    /** @type {number|null} */
    let grammarPress = null;

    /**
     * Pression sur le plateau pendant un coup selon la grammaire. Rend `true` quand elle est prise.
     * @param {MouseEvent} event
     * @param {number} x
     * @param {number} y
     */
    function grammarMouseDown(event, x, y) {
        grammarPress = null;
        const play = grammarPlay();
        if (!play) return false;
        if (event.button !== 0) return true;
        const hit = hitAt(x, y);
        if (hit.kind === 'point') {
            grammarPress = hit.point;
        } else if (hit.kind === 'die') {
            const action = diceClick(play, drawnDice(play));
            if (action === 'validate') get(stores.quizPlayValidate)?.();
            else if (action === 'swap') stores.quizPlay.update((/** @type {any} */ s) => (s ? { ...s, swapped: !s.swapped } : s));
        }
        return true;
    }

    /**
     * Relâchement d'une pression du coup selon la grammaire. Rend `true` quand elle est prise.
     * @param {MouseEvent} event
     */
    function grammarMouseUp(event) {
        const from = grammarPress;
        grammarPress = null;
        if (from === null) return false;
        const { x, y } = toDrawing(event);
        const target = quizTargetAt(x, y);
        stores.quizPlay.update((/** @type {any} */ s) => {
            if (!s) return s;
            if (target === null || target === from) return playClickedChecker(s, from, drawnDice(s));
            return dragStep(s, from, target);
        });
        return true;
    }

    /**
     * What a drawing-space point falls on, for the Duel and the Pile gesture: a die (as drawn,
     * on the roller's side), the cube, a point of the model, the frame's outside, or nothing.
     * @param {number} x
     * @param {number} y
     * @returns {import('../services/duelBoard.js').BoardHit}
     */
    function hitAt(x, y) {
        const roller = deps.displayRoller?.() ?? get(stores.position).player_on_roll;
        const { die } = hitTestSideControls(x, y, metrics(), cfg, roller);
        if (die !== null) return { kind: 'die', index: die };
        const box = deps.getCubeBox();
        if (box && Math.abs(x - box.x) <= box.size / 2 && Math.abs(y - box.y) <= box.size / 2) return { kind: 'cube' };
        const point = pointAt(x, y);
        if (point !== null) return { kind: 'point', point };
        return isOutsideBoard(x, y, metrics()) ? { kind: 'outside' } : { kind: 'none' };
    }

    // Le pion pressé pendant un Duel : relâché sur place, c'est un clic (le dé de gauche) ;
    // relâché ailleurs, un glissé vers ce point.
    /** @type {number|null} */
    let duelPress = null;

    /**
     * @param {MouseEvent} event
     * @param {number} x
     * @param {number} y
     */
    function duelMouseDown(event, x, y) {
        duelPress = null;
        if (event.button !== 0) return;
        const hit = hitAt(x, y);
        if (hit.kind === 'point') duelPress = hit.point;
        else deps.duel.press(hit);
    }

    /** @param {MouseEvent} event */
    function duelMouseUp(event) {
        const from = duelPress;
        duelPress = null;
        if (from === null) return;
        const { x, y } = toDrawing(event);
        const target = quizTargetAt(x, y);
        if (target === null || target === from) deps.duel.press({ kind: 'point', point: from });
        else deps.duel.drop(from, target);
    }

    /** @param {MouseEvent} event */
    function onMouseDown(event) {
        event.preventDefault(); // no text or element selection
        if (document.activeElement && document.activeElement.matches('input, textarea, [contenteditable]')) {
            /** @type {HTMLElement} */ (document.activeElement).blur();
        }
        if (deps.duel?.holds()) {
            const { x, y } = toDrawing(event);
            duelMouseDown(event, x, y);
            return;
        }
        {
            const { x, y } = toDrawing(event);
            // Le videau d'abord : il est hors du damier, donc le coup joué au
            // plateau ne le vise pas, et il avale le clic.
            if (transcriptionCubeClick(event, x, y)) return;
            if (grammarMouseDown(event, x, y)) return;
        }
        if (!editable()) return;
        const { x, y } = toDrawing(event);
        startMousePos = { x, y, button: event.button };
        cubeClick(event, x, y);
        sideControlsClick(event, x, y);
    }

    /** @param {MouseEvent} event */
    function onMouseMove(event) {
        event.preventDefault();
    }

    /** @param {MouseEvent} event */
    function onMouseUp(event) {
        event.preventDefault();
        if (deps.duel?.holds()) {
            duelMouseUp(event);
            return;
        }
        // Avant la garde d'édition : le coup joué au plateau vit dans des modes qui n'éditent pas
        // la position (TRANSCRIBE, quiz).
        if (grammarMouseUp(event)) return;
        if (!editable() || !startMousePos) return;
        const end = { ...toDrawing(event), button: event.button };

        // Except structure: a quick second click on the same (empty) point blocks it.
        if (deps.getMode() === 'EDIT' && get(stores.structureMode) === 'exclude') {
            const { checkerPoint } = checkerAt(end.x, end.y);
            if (checkerPoint >= 1 && checkerPoint <= 24) {
                const isMarker = get(stores.position).board.points[checkerPoint]?.color === EXCLUDE_EMPTY;
                const now = Date.now();
                if (!isMarker && lastExceptClick && lastExceptClick.point === checkerPoint && now - lastExceptClick.time < EXCEPT_DOUBLE_CLICK_MS) {
                    lastExceptClick = null;
                    stores.position.update((/** @type {BoardPosition} */ pos) => {
                        pos.board.points = pos.board.points.map((/** @type {{ checkers: number, color: number }} */ p, /** @type {number} */ i) =>
                            i === checkerPoint ? { checkers: 1, color: EXCLUDE_EMPTY } : p
                        );
                        return pos;
                    });
                    return;
                }
                // A click that unblocks (applyCheckerEdit) must not seed a re-blocking double-click.
                lastExceptClick = isMarker ? null : { point: checkerPoint, time: now };
            } else {
                lastExceptClick = null;
            }
        }

        fillCheckersBetween(startMousePos, end);
    }

    // A press-and-release across several points fills them with the taller clicked count.
    /**
     * @param {PressPoint} startPos
     * @param {PressPoint} endPos
     */
    function fillCheckersBetween(startPos, endPos) {
        const start = checkerAt(startPos.x, startPos.y);
        const end = checkerAt(endPos.x, endPos.y);
        if (start.checkerPoint === -1 && end.checkerPoint === -1) return; // outside the board
        const count = Math.max(start.checkerCount, end.checkerCount);
        const from = Math.min(start.checkerPoint, end.checkerPoint);
        const to = Math.max(start.checkerPoint, end.checkerPoint);
        const isSearchStructure = get(stores.activeTab) === 'search';
        for (let point = from; point <= to; point++) {
            stores.position.update((/** @type {BoardPosition} */ pos) => applyCheckerEdit(pos, point, count, startPos.button, isSearchStructure));
        }
    }

    /** @param {MouseEvent} event */
    function onDoubleClick(event) {
        const { x, y } = toDrawing(event);
        // Hors du cadre, dans tous les modes, le double-clic met la position sur la Pile, ou l'en
        // retire. La remise à zéro est au menu contextuel (Board.svelte).
        if (hitAt(x, y).kind === 'outside' && !get(stores.anyModalOpen)) deps.togglePile?.();
    }

    /**
     * In EDIT and EVAL the right button edits the board (the other colour's checker, a die, the
     * cube, a score): the menu opens only where it is idle, outside the frame and its controls.
     * @param {number} x
     * @param {number} y
     */
    function rightButtonIdle(x, y) {
        if (hitAt(x, y).kind !== 'outside') return false;
        const hit = hitTestSideControls(x, y, metrics(), cfg, get(stores.position).player_on_roll);
        return hit.die === null && hit.playerRect === null && hit.score === null;
    }

    /**
     * Clic droit sur le damier pendant un coup selon la grammaire : au moins un pas joué, il les
     * reprend tous (jet saisi, origine et ordre des dés gardés). Rend `true` quand il est pris.
     * @param {number} x
     * @param {number} y
     */
    function grammarRightClick(x, y) {
        const play = grammarPlay();
        const kind = hitAt(x, y).kind;
        if (!play || (kind !== 'point' && kind !== 'none')) return false;
        if (boardRightClick(play) !== 'reset') return false;
        stores.quizPlay.update((/** @type {any} */ s) => (s ? { ...resetBoardPlay(s, get(stores.position)), swapped: !!s.swapped } : s));
        return true;
    }

    /** @param {MouseEvent} event */
    function onContextMenu(event) {
        event.preventDefault(); // no native menu, in every mode
        if (get(stores.anyModalOpen)) return;
        const { x, y } = toDrawing(event);
        if (deps.duel?.holds()) {
            if (!deps.duel.context(hitAt(x, y))) return;
        } else if (grammarRightClick(x, y)) {
            return;
        } else if (editable() && !rightButtonIdle(x, y)) return;
        deps.openContextMenu({ x: event.clientX, y: event.clientY });
    }

    // Les marges autour du dessin sont hors du cadre, elles aussi.
    const container = deps.container;
    const inMargin = (/** @type {Event} */ event) => !!container && !canvas.contains(/** @type {Node} */ (event.target));

    /** @param {MouseEvent} event */
    function onMarginDoubleClick(event) {
        if (!inMargin(event) || get(stores.anyModalOpen)) return;
        deps.togglePile?.();
    }

    // In the margins the menu opens where it carries something of its own: the Duel's menu, and
    // the reset entries of EDIT, EVAL and a play in progress.
    /** @param {MouseEvent} event */
    function onMarginContextMenu(event) {
        if (!inMargin(event) || get(stores.anyModalOpen)) return;
        if (deps.duel?.holds()) {
            event.preventDefault();
            if (deps.duel.context({ kind: 'outside' })) deps.openContextMenu({ x: event.clientX, y: event.clientY });
            return;
        }
        if (!editable() && !(stores.quizPlay && get(stores.quizPlay))) return;
        event.preventDefault();
        deps.openContextMenu({ x: event.clientX, y: event.clientY });
    }

    canvas.addEventListener('mousedown', onMouseDown);
    canvas.addEventListener('mousemove', onMouseMove);
    canvas.addEventListener('mouseup', onMouseUp);
    canvas.addEventListener('dblclick', onDoubleClick);
    canvas.addEventListener('contextmenu', onContextMenu);
    container?.addEventListener('dblclick', onMarginDoubleClick);
    container?.addEventListener('contextmenu', onMarginContextMenu);

    return function detach() {
        container?.removeEventListener('dblclick', onMarginDoubleClick);
        container?.removeEventListener('contextmenu', onMarginContextMenu);
        canvas.removeEventListener('mousedown', onMouseDown);
        canvas.removeEventListener('mousemove', onMouseMove);
        canvas.removeEventListener('mouseup', onMouseUp);
        canvas.removeEventListener('dblclick', onDoubleClick);
        canvas.removeEventListener('contextmenu', onContextMenu);
    };
}
