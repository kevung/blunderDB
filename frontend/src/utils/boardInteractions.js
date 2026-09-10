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
import { OFF, playHop, selectSource } from '../services/quizPlay.js';
import { freeClick, freeStep } from '../services/transcriptionPlay.js';
import { nextFilter, sourcesOf } from '../services/transcriptionFilter.js';

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
 * Le clic tombe-t-il sur le plateau de sortie du joueur `player` ?
 *
 * C'est la destination d'un pion sorti, et la seule qui ne soit pas un point :
 * la boîte est celle du « (n OFF) » que drawBearoff dessine, élargie de moitié
 * pour qu'on n'ait pas à viser le texte au pixel.
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
 *   stores               { position, structureMode, activeTab, offeredCube, anyModalOpen,
 *                          quizPlay, transcriptionFilter, transcriptionCandidates,
 *                          transcriptionCube }
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

    /**
     * Le point (ou le plateau de sortie) visé par un clic, pendant une
     * question de quiz. Rend null hors du damier.
     * @param {number} x
     * @param {number} y
     */
    function quizTargetAt(x, y) {
        // Le plateau de sortie visé est celui du camp au trait, en bas ou en
        // haut selon l'affichage : il n'est en bas que là où le camp au trait
        // descend. En transcription le joueur 1 reste en bas quel que soit le
        // trait, et le joueur 2 sort donc ses pions par le haut.
        const bearoffSide = deps.quizBearoffSide?.() ?? 0;
        if (hitTestBearoffTray(x, y, metrics(), cfg, get(stores.position).player_on_roll, bearoffSide)) return OFF;
        return pointAt(x, y);
    }

    /**
     * Le point du MODÈLE visé par un clic, ou null hors du damier. Une position
     * dont le joueur 2 est au trait est montrée retournée (#294) : le point
     * cliqué n'est alors pas le point du modèle, et la conversion est celle de
     * mirrorPosition — 25 - p, qui échange aussi les deux barres.
     * @param {number} x
     * @param {number} y
     */
    function pointAt(x, y) {
        const { checkerPoint } = checkerAt(x, y);
        if (checkerPoint < 0 || checkerPoint > 25) return null;
        return deps.quizDisplayMirrored?.() ? 25 - checkerPoint : checkerPoint;
    }

    /**
     * Le clic sur le VIDEAU pendant une transcription (T2.5) : le camp au trait
     * propose un double, exactement comme la touche `d`.
     *
     * Le plateau ne fait que POSER la demande dans un magasin que le panneau
     * lit (`transcriptionCubeRequestStore`) : c'est lui qui tient le brouillon
     * et l'aller-retour Wails, et lui seul sait ce que le document attend. Le
     * plateau ne juge donc pas davantage que la touche — un double sans le
     * videau, en partie Crawford ou au plafond reste transcriptible, et
     * l'Incohérence est marquée (ADR-0044).
     *
     * Ce que le clic ne fait PAS : répondre. Devant une offre, la cible unique
     * qu'est le videau porterait celle des deux réponses qu'on aurait choisie —
     * la prise, la passe restant sans cible — et la même cible créerait alors
     * deux Actions différentes selon un état que l'œil, occupé par la vidéo,
     * ne relit pas. Les deux réponses sont symétriques et vivent ensemble dans
     * la rangée `[T] [P]`, un clic chacune : le budget d'ux.md §4.2 est le même
     * (deux clics), et un clic tombé à contretemps n'y écrit rien.
     *
     * Rend `true` quand le geste est pris, sur le modèle de `quizClick` — et il
     * est essayé le PREMIER, parce qu'un coup joué au plateau arme `quizPlay`,
     * qui avale sinon tous les clics du damier.
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
     * Le clic qui réduit la liste des candidats d'une transcription (T2.2).
     *
     * Un clic sur un point d'où part au moins un candidat ne garde que les
     * coups qui en partent ; un second point réduit encore ; un clic sur le
     * point déjà filtré l'enlève, et un clic hors du damier lève tout. C'est le
     * geste du coup lointain : le rang douze coûte treize touches au clavier.
     *
     * Rien de tout cela ne touche au document : le filtre est un état
     * d'AFFICHAGE (services/transcriptionFilter.js), le moteur n'est pas
     * rappelé, aucune Action n'est créée.
     *
     * Rend `true` quand le geste est pris — sur le modèle de `quizClick`, pour
     * que l'édition, et le déplacement libre de pions qui viendra, ne voient
     * pas ce clic.
     * @param {MouseEvent} event
     * @param {number} x
     * @param {number} y
     */
    function transcriptionClick(event, x, y) {
        if (!stores.transcriptionFilter || !stores.transcriptionCandidates) return false;
        const candidates = get(stores.transcriptionCandidates);
        if (!candidates?.length) return false;
        // Le bouton droit ouvre le menu de la position, ici comme ailleurs.
        if (event.button !== 0) return false;

        const points = get(stores.transcriptionFilter);
        if (isOutsideBoard(x, y, metrics())) {
            if (!points.length) return false;
            stores.transcriptionFilter.set([]);
            return true;
        }

        const point = pointAt(x, y);
        if (point === null) return false;
        // Un point d'où ne part aucun candidat ne fait RIEN : le geste reste
        // disponible pour le déplacement libre de pions, qui a son propre état.
        if (!points.includes(point) && !sourcesOf(candidates).has(point)) return false;

        const next = nextFilter(points, point, candidates);
        if (next === null) return false;
        stores.transcriptionFilter.set(next);
        return true;
    }

    /**
     * Joue le clic sur le coup en cours, s'il y en a un. Rend `true` quand le
     * quiz a pris la main — l'édition ne doit alors pas voir ce clic.
     *
     * Un clic qu'aucun coup légal n'autorise ne fait RIEN : ni pion déplacé,
     * ni message. Le plateau n'a pas à expliquer pourquoi un pion ne peut pas
     * aller là ; il le montre en n'offrant que ce qui est jouable.
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
            // Le déplacement LIBRE d'une transcription (T2.4) : aucun coup légal
            // ne le contraint, et c'est la seule différence — le geste, lui, est
            // le même, source puis destination.
            if (s.free) return freeClick(s, target);
            if (s.selected === null) return selectSource(s, target);
            const played = playHop(s, s.selected, target);
            // Le clic qui ne joue rien re-choisit une source : on change d'avis
            // sur le pion à bouger sans avoir à déselectionner d'abord.
            return played === s ? selectSource(s, target) : played;
        });
        // Une pression qui vient de CHOISIR une source ouvre un glissé : si le
        // bouton se relâche ailleurs, le pas est joué et le pion aura suivi la
        // souris en un seul geste au lieu de deux clics (ux.md §4.1, un P B B
        // par pas). Deux clics restent possibles et donnent le même état.
        const after = get(stores.quizPlay);
        boardPress = state.selected === null && after?.selected === target ? target : null;
        return true;
    }

    /**
     * La fin d'un glissé : le pion lâché sur `to`. Rend `true` quand le geste
     * était bien un glissé du coup en cours — l'édition ne doit alors pas voir
     * ce relâchement.
     * @param {MouseEvent} event
     */
    function boardPlayDrop(event) {
        const from = boardPress;
        boardPress = null;
        if (from === null || !stores.quizPlay) return false;
        const { x, y } = toDrawing(event);
        const target = quizTargetAt(x, y);
        if (target === null || target === from) return true;
        stores.quizPlay.update((/** @type {any} */ s) => {
            if (!s || s.selected !== from) return s;
            return s.free ? freeStep(s, from, target) : playHop(s, from, target);
        });
        return true;
    }

    function onMouseDown(event) {
        event.preventDefault(); // no text or element selection
        // Blur any focused text field when clicking the board
        if (document.activeElement && document.activeElement.matches('input, textarea, [contenteditable]')) {
            /** @type {HTMLElement} */ (document.activeElement).blur();
        }
        {
            const { x, y } = toDrawing(event);
            // Le videau d'abord : il est hors du damier, donc aucun des deux
            // gestes qui suivent ne le vise, et tous deux avalent le clic.
            if (transcriptionCubeClick(event, x, y)) return;
            if (quizClick(event, x, y)) return;
            if (transcriptionClick(event, x, y)) return;
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
        // Avant la garde d'édition : le glissé du coup joué au plateau vit dans
        // un mode qui n'édite pas la position (TRANSCRIBE, et le quiz).
        if (boardPlayDrop(event)) return;
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
        const { x, y } = toDrawing(event);
        if (stores.quizPlay && get(stores.quizPlay)) {
            // Même geste qu'ailleurs — double-clic hors damier = remise à
            // zéro — mais ce qui est remis est le coup, pas la position : la
            // question, elle, ne change pas parce qu'on s'est trompé de pion.
            if (isOutsideBoard(x, y, metrics())) deps.resetQuizPlay?.();
            return;
        }
        if (!editable()) return;
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
