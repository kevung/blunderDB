// Mouse handling of the board: hit-testing of the drawn scene and the position edits each click
// performs in EDIT and EVAL mode, plus double-click reset and the right-click menu gate.
// Everything comes through `deps` (no globals), so the flow runs in jsdom on plain stores.
// Every hit test goes through boardMouseToDrawing(): the canvas may be CSS-scaled (interface
// zoom, side layout) and raw client pixels drift.

import { get } from 'svelte/store';
import { boardMetrics, boardMouseToDrawing, checkerPointAndCountAt } from './boardGeometry.js';
import { EXCLUDE_EMPTY, sideLayout } from './boardScene.js';
import { OFF, playHop, selectSource } from '../services/quizPlay.js';
import { canPlayFree, dragStep, freeClick, hasMoverChecker } from '../services/transcriptionPlay.js';

// A second click on the same Except point within this delay blocks it. Detected by hand: each
// click recreates the two.js shapes, so native 'dblclick' sees two different DOM nodes.
const EXCEPT_DOUBLE_CLICK_MS = 450;

const MAX_CUBE_VALUE = 6; // log2 exponent: 64

function isEditable(mode) {
    return mode === 'EDIT' || mode === 'EVAL';
}

/** A real roll: both dice on a face. Anything else is "no dice" (a cube decision). */
function hasRoll(dice) {
    return dice[0] >= 1 && dice[0] <= 6 && dice[1] >= 1 && dice[1] <= 6;
}

/**
 * Le clic tombe-t-il sur le plateau de sortie de `player` ? Boîte du « (n OFF) » de drawBearoff,
 * élargie de moitié pour ne pas viser le texte au pixel.
 *
 * @param {number} x
 * @param {number} y
 * @param {any} geom
 * @param {any} cfg
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
 * Put `count` checkers of the clicking button's colour on `point` (left → colour 0, right →
 * colour 1; bars are colour-fixed). Clicking the fifth checker of a stack of five or more adds
 * one. A blocked Except point is unblocked. `isSearchStructure` lifts the 15-per-colour cap.
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
 */
export function applyCubeClick(pos, button, { evalMode = false, offeredTakePass = false } = {}) {
    const up = (v) => Math.min(v + 1, MAX_CUBE_VALUE);
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
 *                          quizPlay, transcriptionCube }
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
    // Le point d'où un glissé est parti, quand la pression a choisi une source.
    let boardPress = null;

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
        stores.position.update((pos) => applyCubeClick(pos, event.button, { evalMode: mode === 'EVAL', offeredTakePass: get(stores.offeredCube) && pos.decision_type === 1 }));
    }

    // EVAL shares this flow with EDIT: the Eval panel needs real dice for candidate moves and
    // none for a cube verdict, which is exactly EDIT's rectangle/die toggle.
    function sideControlsClick(event, x, y) {
        const hit = hitTestSideControls(x, y, metrics(), cfg, get(stores.position).player_on_roll);
        if (hit.die === null && hit.playerRect === null && hit.score === null) return;
        log('side control clicked', hit);
        stores.position.update((pos) => {
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
            } else {
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
     * Joue le clic sur le coup en cours. Rend `true` quand le quiz a pris la main (l'édition ne
     * doit pas voir ce clic). Un clic qu'aucun coup légal n'autorise ne fait rien, sans message.
     * @param {MouseEvent} event
     * @param {number} x
     * @param {number} y
     */
    function quizClick(event, x, y) {
        const state = stores.quizPlay ? get(stores.quizPlay) : null;
        if (!state) return false;
        if (event.button !== 0) return true;
        const target = quizTargetAt(x, y);
        if (target === null) return true;
        stores.quizPlay.update((/** @type {any} */ s) => {
            // Coup SORTI DES RÈGLES (ADR-0052) : même geste, sans coup légal pour le contraindre.
            if (s.free) return freeClick(s, target);
            if (s.selected === null) return selectSource(s, target);
            const played = playHop(s, s.selected, target);
            // Un clic qui ne joue rien re-choisit une source, sans déselection préalable.
            return played === s ? selectSource(s, target) : played;
        });
        // Une pression qui vient de choisir une source ouvre un glissé : relâché ailleurs, le pas
        // est joué en un geste (ux.md §4.1). Deux clics donnent le même état.
        const after = get(stores.quizPlay);
        boardPress = state.selected === null && after?.selected === target ? target : null;
        // Jet connu : une pression sur un pion du camp au trait sans coup légal ouvre aussi un
        // glissé, qui pose le pion hors règles (ADR-0052). La pression seule ne choisit rien.
        if (boardPress === null && after && canPlayFree(after) && after.steps.length === state.steps.length && hasMoverChecker(after, target)) {
            boardPress = target;
        }
        return true;
    }

    /**
     * Fin d'un glissé sur `to`. Rend `true` si c'était un glissé du coup en cours. Un pas légal
     * est joué ; sinon, jet connu, le pion est posé où il est lâché (`dragStep`, ADR-0052).
     * @param {MouseEvent} event
     */
    function boardPlayDrop(event) {
        const from = boardPress;
        boardPress = null;
        if (from === null || !stores.quizPlay) return false;
        const { x, y } = toDrawing(event);
        const target = quizTargetAt(x, y);
        if (target === null || target === from) return true;
        stores.quizPlay.update((/** @type {any} */ s) => (s ? dragStep(s, from, target) : s));
        return true;
    }

    function onMouseDown(event) {
        event.preventDefault(); // no text or element selection
        if (document.activeElement && document.activeElement.matches('input, textarea, [contenteditable]')) {
            /** @type {HTMLElement} */ (document.activeElement).blur();
        }
        {
            const { x, y } = toDrawing(event);
            // Le videau d'abord : il est hors du damier, donc le coup joué au
            // plateau ne le vise pas, et il avale le clic.
            if (transcriptionCubeClick(event, x, y)) return;
            if (quizClick(event, x, y)) return;
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
        // Avant la garde d'édition : le coup joué au plateau vit dans des modes qui n'éditent pas
        // la position (TRANSCRIBE, quiz).
        if (boardPlayDrop(event)) return;
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
                    stores.position.update((pos) => {
                        pos.board.points = pos.board.points.map((p, i) => (i === checkerPoint ? { checkers: 1, color: EXCLUDE_EMPTY } : p));
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
        const { x, y } = toDrawing(event);
        if (stores.quizPlay && get(stores.quizPlay)) {
            // Double-clic hors damier : remet le coup à zéro, pas la position.
            if (isOutsideBoard(x, y, metrics())) deps.resetQuizPlay?.();
            return;
        }
        if (!editable()) return;
        if (isOutsideBoard(x, y, metrics())) deps.reset();
    }

    // Right-click menu only where the right button is otherwise idle: in EDIT and EVAL it
    // places the other colour's checker.
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
