/**
 * boardInteractions.test.js
 *
 * The board's mouse handling, driven end to end in jsdom: real MouseEvents on
 * a fake canvas whose getBoundingClientRect() can be scaled, plain writable
 * stores in place of the app's, and the click targets computed from the very
 * formulas the scene is drawn with (stackSlotCenter, cubeBox, sideLayout) —
 * a click "on the third checker of point 7" is a click where drawCheckers()
 * paints it.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { writable, get } from 'svelte/store';
import { boardMetrics } from '../utils/boardGeometry.js';
import { defaultBoardConfig } from '../utils/boardConfig.js';
import { EXCLUDE_EMPTY, stackSlotCenter, cubeBox, sideLayout } from '../utils/boardScene.js';
import { attachBoardInteractions, hitTestSideControls, applyCheckerEdit, applyCubeClick, applyScoreClick, applyStartingCheckers } from '../utils/boardInteractions.js';
import { newPlay, completedPlay } from '../services/quizPlay.js';
import { newBoardPlay } from '../services/transcriptionPlay.js';

const W = 1000;
const H = 720;
const RECT = { left: 37, top: 11 };

/** @typedef {import('../utils/boardGeometry.js').BoardMetrics} BoardMetrics */
/** @typedef {import('../utils/boardConfig.js').BoardConfig} BoardConfig */
/** @typedef {import('../utils/boardGeometry.js').BoardPosition} BoardPosition */

function makeCfg(orientation = 'right') {
    const base = defaultBoardConfig();
    return { ...base, widthFactor: 0.75, orientation, checker: { ...base.checker, sizeFactor: 0.97 } };
}

/** @returns {BoardPosition & { id: number }} */
function emptyPos() {
    return {
        id: 1,
        board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [15, 15] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [7, 7],
        player_on_roll: 0,
        decision_type: 0
    };
}

/** A mounted board: canvas + stores + deps, with the click helpers. */
function mount({ mode = 'EDIT', orientation = 'right', scale = 1, position = emptyPos(), mirrored = false, extra = /** @type {any} */ ({}) } = {}) {
    const canvas = document.createElement('div');
    document.body.appendChild(canvas);
    canvas.getBoundingClientRect = () => new DOMRect(RECT.left, RECT.top, W * scale, H * scale);
    const cfg = makeCfg(orientation);
    const geom = boardMetrics(W, H, cfg.widthFactor);
    const stores = {
        position: writable(position),
        structureMode: writable('include'),
        activeTab: writable('positions'),
        offeredCube: writable(false),
        anyModalOpen: writable(false),
        // Armé par les tests du quiz (#294) ; null partout ailleurs, donc le
        // clic retombe sur l'édition comme avant.
        quizPlay: writable(/** @type {any} */ (null)),
        // Le videau cliqué pendant une transcription (T2.5) : null tant que
        // rien n'a été demandé, et le panneau le remet à null en servant.
        transcriptionCube: writable(/** @type {string|null} */ (null)),
        // Le rappel de validation d'un mode passé à la grammaire d'ADR-0086 : null, la saisie
        // source puis destination reste.
        quizPlayValidate: writable(/** @type {(() => void)|null} */ (null))
    };
    const state = { mode, previousDice: [3, 1], cubeBox: cubeBox(geom, position, false) };
    const deps = {
        getMode: () => state.mode,
        getSize: () => ({ width: W, height: H }),
        cfg,
        getCubeBox: () => state.cubeBox,
        stores,
        getPreviousDice: () => state.previousDice,
        setPreviousDice: (/** @type {number[]} */ d) => (state.previousDice = d),
        reset: vi.fn(),
        openContextMenu: vi.fn(),
        resetQuizPlay: vi.fn(),
        quizDisplayMirrored: () => mirrored,
        ...extra
    };
    const detach = attachBoardInteractions(canvas, deps);

    // Drawing-space → client pixels, the inverse of boardMouseToDrawing().
    const client = (/** @type {{ x: number, y: number }} */ { x, y }) => ({ clientX: RECT.left + x * scale, clientY: RECT.top + y * scale });
    const fire = (/** @type {string} */ type, /** @type {{ x: number, y: number }} */ at, button = 0) => {
        const event = new MouseEvent(type, { bubbles: true, cancelable: true, button, ...client(at) });
        canvas.dispatchEvent(event);
        return event;
    };
    const click = (/** @type {{ x: number, y: number }} */ at, button = 0) => {
        fire('mousedown', at, button);
        fire('mouseup', at, button);
    };
    const drag = (/** @type {{ x: number, y: number }} */ from, /** @type {{ x: number, y: number }} */ to, button = 0) => {
        fire('mousedown', from, button);
        fire('mouseup', to, button);
    };
    const slot = (/** @type {number} */ point, /** @type {number} */ index) => /** @type {{ x: number, y: number }} */ (stackSlotCenter(geom, cfg, point, index));
    const pos = () => get(stores.position);
    return { canvas, cfg, geom, stores, state, deps, detach, fire, click, drag, slot, pos };
}

afterEach(() => {
    document.body.innerHTML = '';
});

// ── Checkers ────────────────────────────────────────────────────────────────

describe('checker clicks land on the point they were drawn on', () => {
    for (const orientation of ['right', 'left']) {
        test(`every point, orientation ${orientation}: the third drawn slot puts three checkers there`, () => {
            const b = mount({ orientation });
            for (let p = 1; p <= 24; p++) {
                b.stores.position.set(emptyPos());
                b.click(b.slot(p, 2));
                const points = b.pos().board.points;
                expect(points[p], `point ${p}`).toEqual({ checkers: 3, color: 0 });
                expect(
                    points.filter((pt) => pt.checkers > 0),
                    `only point ${p}`
                ).toHaveLength(1);
            }
            b.detach();
        });
    }

    test('the right button places the other colour', () => {
        const b = mount();
        b.click(b.slot(7, 0), 2);
        expect(b.pos().board.points[7]).toEqual({ checkers: 1, color: 1 });
    });

    test('the bars are colour-fixed whichever button is used', () => {
        const b = mount();
        b.click(b.slot(0, 0), 0);
        b.click(b.slot(25, 0), 2);
        expect(b.pos().board.points[0].color).toBe(1);
        expect(b.pos().board.points[25].color).toBe(0);
    });

    test('bearoff follows the checkers placed', () => {
        const b = mount();
        b.click(b.slot(6, 4)); // five checkers
        b.click(b.slot(19, 1), 2); // two of the other colour
        expect(b.pos().board.bearoff).toEqual([10, 13]);
    });

    test('a drag from point 3 to point 6 fills the four points with the taller count', () => {
        const b = mount();
        b.drag(b.slot(3, 0), b.slot(6, 1));
        const points = b.pos().board.points;
        for (const p of [3, 4, 5, 6]) expect(points[p], `point ${p}`).toEqual({ checkers: 2, color: 0 });
        expect(points.filter((pt) => pt.checkers > 0)).toHaveLength(4);
    });

    test('nothing happens outside EDIT/EVAL, or on a click outside the board', () => {
        const b = mount({ mode: 'NORMAL' });
        b.click(b.slot(7, 0));
        expect(b.pos()).toEqual(emptyPos());
        b.state.mode = 'EDIT';
        const updates = vi.fn();
        const unsub = b.stores.position.subscribe(updates);
        updates.mockClear();
        b.click({ x: 5, y: 5 });
        expect(updates).not.toHaveBeenCalled(); // no store tick for a no-op
        unsub();
    });

    test('CSS-scaled canvas (90 %): a click on point 1 stays on point 1 — the historical drift bug', () => {
        // Before boardMouseToDrawing(), raw client pixels were used; at 90 %
        // interface scale the error grows towards the board edge and the
        // first point's centre fell into the second point's column.
        const b = mount({ scale: 0.9 });
        b.click(b.slot(1, 0));
        expect(b.pos().board.points[1]).toEqual({ checkers: 1, color: 0 });
        expect(b.pos().board.points[2].checkers).toBe(0);
        b.stores.position.set(emptyPos());
        b.click(b.slot(24, 0));
        expect(b.pos().board.points[24]).toEqual({ checkers: 1, color: 0 });
    });

    test('a search structure is not capped at 15 checkers per colour', () => {
        const b = mount();
        b.stores.activeTab.set('search');
        for (const p of [1, 2, 3, 4]) b.click(b.slot(p, 4)); // 4 × 5 = 20
        expect(b.pos().board.points.reduce((n, p) => n + p.checkers, 0)).toBe(20);
        expect(b.pos().board.bearoff[0]).toBe(0); // clamped, never negative
    });
});

describe('applyCheckerEdit', () => {
    test('caps a real position at 15 per colour, counting the other points', () => {
        const pos = emptyPos();
        pos.board.points[13] = { checkers: 12, color: 0 };
        applyCheckerEdit(pos, 6, 5, 0, false);
        expect(pos.board.points[6]).toEqual({ checkers: 3, color: 0 });
        expect(pos.board.bearoff[0]).toBe(0);
    });

    test('clicking the fifth checker of a tall stack adds one more', () => {
        const pos = emptyPos();
        pos.board.points[6] = { checkers: 7, color: 0 };
        applyCheckerEdit(pos, 6, 5, 0, false);
        expect(pos.board.points[6].checkers).toBe(8);
        applyCheckerEdit(pos, 6, 2, 0, false);
        expect(pos.board.points[6].checkers).toBe(2);
    });
});

// ── Except structure ────────────────────────────────────────────────────────

describe('Except structure: double-click blocks a point, a click unblocks it', () => {
    test('two quick clicks on the same point mark it must-be-empty', () => {
        const b = mount();
        b.stores.structureMode.set('exclude');
        b.click(b.slot(5, 0));
        b.click(b.slot(5, 0));
        expect(b.pos().board.points[5]).toEqual({ checkers: 1, color: EXCLUDE_EMPTY });
    });

    test('a click on a blocked point clears it and does not immediately re-block', () => {
        const b = mount();
        b.stores.structureMode.set('exclude');
        b.click(b.slot(5, 0));
        b.click(b.slot(5, 0));
        b.click(b.slot(5, 0));
        expect(b.pos().board.points[5]).toEqual({ checkers: 0, color: -1 });
        b.click(b.slot(5, 0));
        expect(b.pos().board.points[5]).toEqual({ checkers: 1, color: 0 }); // a plain checker again
    });

    test('slow clicks do not block', () => {
        vi.useFakeTimers();
        const b = mount();
        b.stores.structureMode.set('exclude');
        b.click(b.slot(5, 0));
        vi.advanceTimersByTime(600);
        b.click(b.slot(5, 0));
        expect(b.pos().board.points[5]).toEqual({ checkers: 1, color: 0 });
        vi.useRealTimers();
    });
});

// ── Cube ────────────────────────────────────────────────────────────────────

describe('cube clicks', () => {
    test('EDIT: left takes the cube to the bottom player at 2, right to the top player', () => {
        const b = mount();
        b.click(b.state.cubeBox, 0);
        expect(b.pos().cube).toEqual({ owner: 0, value: 1 });
        b.stores.position.set(emptyPos());
        b.click(b.state.cubeBox, 2);
        expect(b.pos().cube).toEqual({ owner: 1, value: 1 });
    });

    test('EDIT: the owner raises with its own button, lowers with the other, back to centred at 1', () => {
        const pos = emptyPos();
        pos.cube = { owner: 0, value: 1 };
        expect(applyCubeClick({ ...pos, cube: { owner: 0, value: 1 } }, 0).cube).toEqual({ owner: 0, value: 2 });
        expect(applyCubeClick({ ...pos, cube: { owner: 0, value: 1 } }, 2).cube).toEqual({ owner: -1, value: 0 });
        expect(applyCubeClick({ ...pos, cube: { owner: 1, value: 3 } }, 0).cube).toEqual({ owner: 1, value: 2 });
        expect(applyCubeClick({ ...pos, cube: { owner: 1, value: 6 } }, 2).cube).toEqual({ owner: 1, value: 6 }); // 64 is the ceiling
    });

    test('EVAL: clicks cycle the owner and pin the value', () => {
        const b = mount({ mode: 'EVAL' });
        const seen = [];
        for (let i = 0; i < 3; i++) {
            b.click(b.state.cubeBox, 0);
            seen.push({ ...b.pos().cube });
        }
        expect(seen).toEqual([
            { owner: 0, value: 1 },
            { owner: 1, value: 1 },
            { owner: -1, value: 0 }
        ]);
        b.click(b.state.cubeBox, 2);
        expect(b.pos().cube).toEqual({ owner: 1, value: 1 }); // right-click goes backwards
    });

    test('offered cube (take/pass search): the value moves, the cube stays centred and at least a double', () => {
        const b = mount();
        b.stores.offeredCube.set(true);
        b.stores.position.update((p) => ({ ...p, decision_type: 1, cube: { owner: -1, value: 1 } }));
        b.click(b.state.cubeBox, 0);
        expect(b.pos().cube).toEqual({ owner: -1, value: 2 });
        b.click(b.state.cubeBox, 2);
        b.click(b.state.cubeBox, 2);
        expect(b.pos().cube).toEqual({ owner: -1, value: 1 });
    });

    test('a click beside the cube does nothing to it', () => {
        const b = mount();
        b.click({ x: b.state.cubeBox.x, y: b.state.cubeBox.y + b.state.cubeBox.size }, 0);
        expect(b.pos().cube).toEqual({ owner: -1, value: 0 });
    });
});

// ── Dice, player rectangles, scores ─────────────────────────────────────────

/**
 * @param {BoardMetrics} geom
 * @param {BoardConfig} cfg
 * @param {number} playerOnRoll
 */
function sideTargets(geom, cfg, playerOnRoll) {
    const side = sideLayout(geom, cfg, playerOnRoll);
    return {
        die: (/** @type {number} */ i) => ({ x: side.diceX + i * (side.diceSize + side.diceGap), y: side.diceY }),
        rect: (/** @type {number} */ player) => ({
            x: side.scoreX,
            y: (player === 0 ? side.bearoff1Y + side.score1Y : side.bearoff2Y + side.score2Y) / 2 - (player === 0 ? 0.3 : -0.3) * geom.checkerSize
        }),
        score: (/** @type {number} */ player) => ({ x: side.scoreX, y: player === 0 ? side.score1Y : side.score2Y })
    };
}

describe('hitTestSideControls', () => {
    const cfg = makeCfg();
    const geom = boardMetrics(W, H, cfg.widthFactor);
    const t = sideTargets(geom, cfg, 0);

    test('tells dice, player rectangles and score boxes apart', () => {
        expect(hitTestSideControls(t.die(0).x, t.die(0).y, geom, cfg, 0)).toEqual({ die: 0, playerRect: null, score: null });
        expect(hitTestSideControls(t.die(1).x, t.die(1).y, geom, cfg, 0)).toEqual({ die: 1, playerRect: null, score: null });
        expect(hitTestSideControls(t.rect(0).x, t.rect(0).y, geom, cfg, 0)).toMatchObject({ die: null, playerRect: 0 });
        expect(hitTestSideControls(t.rect(1).x, t.rect(1).y, geom, cfg, 0)).toMatchObject({ die: null, playerRect: 1 });
        expect(hitTestSideControls(t.score(0).x, t.score(0).y, geom, cfg, 0)).toEqual({ die: null, playerRect: null, score: 0 });
        expect(hitTestSideControls(t.score(1).x, t.score(1).y, geom, cfg, 0)).toEqual({ die: null, playerRect: null, score: 1 });
        expect(hitTestSideControls(geom.originX, geom.originY, geom, cfg, 0)).toEqual({ die: null, playerRect: null, score: null });
    });

    test('the dice follow the player on roll', () => {
        const top = sideTargets(geom, cfg, 1).die(0);
        expect(hitTestSideControls(top.x, top.y, geom, cfg, 1).die).toBe(0);
        expect(hitTestSideControls(top.x, top.y, geom, cfg, 0).die).toBeNull();
    });
});

describe('the Pile on a double click outside the frame', () => {
    test('toggles the Pile where the double click has no other meaning', () => {
        const togglePile = vi.fn();
        const b = mount({ mode: 'NORMAL', extra: { togglePile } });
        b.fire('dblclick', b.slot(7, 0));
        expect(togglePile).not.toHaveBeenCalled();
        b.fire('dblclick', { x: 5, y: 5 });
        expect(togglePile).toHaveBeenCalledTimes(1);
        // On the dice, it is not outside: two clicks there are their own gesture.
        b.fire('dblclick', sideTargets(b.geom, b.cfg, 0).die(0));
        expect(togglePile).toHaveBeenCalledTimes(1);
    });

    test.each([['EDIT'], ['EVAL'], ['TRANSCRIBE']])('toggles the Pile in %s too, and in a play in progress: the reset is in the menu', (mode) => {
        const togglePile = vi.fn();
        const b = mount({ mode, extra: { togglePile } });
        b.fire('dblclick', { x: 5, y: 5 });
        expect(togglePile).toHaveBeenCalledTimes(1);
        b.stores.quizPlay.set(newPlay(emptyPos(), []));
        b.fire('dblclick', { x: 5, y: 5 });
        expect(togglePile).toHaveBeenCalledTimes(2);
        expect(b.deps.reset).not.toHaveBeenCalled();
        expect(b.deps.resetQuizPlay).not.toHaveBeenCalled();
    });

    test('never over a modal', () => {
        const togglePile = vi.fn();
        const b = mount({ mode: 'EDIT', extra: { togglePile } });
        b.stores.anyModalOpen.set(true);
        b.fire('dblclick', { x: 5, y: 5 });
        expect(togglePile).not.toHaveBeenCalled();
    });

    test('the margins around the drawing count as outside', () => {
        const togglePile = vi.fn();
        const container = document.createElement('div');
        document.body.appendChild(container);
        const b = mount({ mode: 'NORMAL', extra: { togglePile, container } });
        container.appendChild(b.canvas);
        container.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
        expect(togglePile).toHaveBeenCalledTimes(1);
        b.detach();
        container.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
        expect(togglePile).toHaveBeenCalledTimes(1);
    });

    test('in EDIT the margins toggle the Pile as well', () => {
        const togglePile = vi.fn();
        const container = document.createElement('div');
        document.body.appendChild(container);
        const b = mount({ mode: 'EDIT', extra: { togglePile, container } });
        container.appendChild(b.canvas);
        container.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
        expect(togglePile).toHaveBeenCalledTimes(1);
    });
});

describe('the right-click menu', () => {
    test('NORMAL: it opens anywhere on the board', () => {
        const b = mount({ mode: 'NORMAL' });
        expect(b.fire('contextmenu', b.slot(7, 0), 2).defaultPrevented).toBe(true);
        b.fire('contextmenu', { x: 5, y: 5 }, 2);
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(2);
    });

    test.each([['EDIT'], ['EVAL']])('%s: it opens outside the frame only, where the right button edits nothing', (mode) => {
        const b = mount({ mode });
        const side = sideTargets(b.geom, b.cfg, 0);
        for (const at of [b.slot(7, 0), side.die(0), b.state.cubeBox]) {
            expect(b.fire('contextmenu', at, 2).defaultPrevented, 'no native menu').toBe(true);
        }
        expect(b.deps.openContextMenu).not.toHaveBeenCalled();
        b.fire('contextmenu', { x: 5, y: 5 }, 2);
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(1);
    });

    test('a play in progress: it opens on the board, the right button plays nothing', () => {
        const b = mount({ mode: 'NORMAL' });
        b.stores.quizPlay.set(newPlay(emptyPos(), []));
        b.fire('contextmenu', b.slot(7, 0), 2);
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(1);
    });

    test('never over a modal', () => {
        const b = mount({ mode: 'EDIT' });
        b.stores.anyModalOpen.set(true);
        b.fire('contextmenu', { x: 5, y: 5 }, 2);
        expect(b.deps.openContextMenu).not.toHaveBeenCalled();
    });

    test.each([
        ['EDIT', false, true],
        ['NORMAL', true, true],
        ['NORMAL', false, false]
    ])('margins in %s (play in progress: %s): menu %s', (mode, playing, opens) => {
        const container = document.createElement('div');
        document.body.appendChild(container);
        const b = mount({ mode, extra: { container } });
        container.appendChild(b.canvas);
        if (playing) b.stores.quizPlay.set(newPlay(emptyPos(), []));
        container.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(opens ? 1 : 0);
    });
});

describe('applyStartingCheckers', () => {
    test('puts the fifteen checkers of each side where a game starts, nothing borne off', () => {
        const pos = applyStartingCheckers(emptyPos());
        const of = (/** @type {number} */ color) =>
            pos.board.points.flatMap((/** @type {{ checkers: number, color: number }} */ p, /** @type {number} */ i) => (p.color === color ? [[i, p.checkers]] : []));
        expect(of(0)).toEqual([
            [6, 5],
            [8, 3],
            [13, 5],
            [24, 2]
        ]);
        expect(of(1)).toEqual([
            [1, 2],
            [12, 5],
            [17, 3],
            [19, 5]
        ]);
        expect(pos.board.bearoff).toEqual([0, 0]);
    });
});

describe('a Duel holds the board', () => {
    function duelBoard(contextAnswer = true) {
        const duel = { holds: () => true, press: vi.fn(), drop: vi.fn(), context: vi.fn(() => contextAnswer) };
        const togglePile = vi.fn();
        const b = mount({ mode: 'DUEL', extra: { duel, togglePile, displayRoller: () => 0 } });
        b.stores.quizPlay.set(newPlay(emptyPos(), []));
        return { b, duel, togglePile };
    }

    test('a die is pressed at once, a checker on release; a drag drops', () => {
        const { b, duel } = duelBoard();
        b.fire('mousedown', sideTargets(b.geom, b.cfg, 0).die(1));
        expect(duel.press).toHaveBeenCalledWith({ kind: 'die', index: 1 });
        b.fire('mouseup', sideTargets(b.geom, b.cfg, 0).die(1));
        b.click(b.slot(8, 0));
        expect(duel.press).toHaveBeenLastCalledWith({ kind: 'point', point: 8 });
        b.drag(b.slot(8, 0), b.slot(5, 0));
        expect(duel.drop).toHaveBeenCalledWith(8, 5);
        // The right button presses nothing: the context menu event decides.
        duel.press.mockClear();
        b.click(b.slot(8, 0), 2);
        expect(duel.press).not.toHaveBeenCalled();
    });

    test('the cube is pressed as the cube', () => {
        const { b, duel } = duelBoard();
        const box = b.state.cubeBox;
        b.fire('mousedown', { x: box.x, y: box.y });
        expect(duel.press).toHaveBeenCalledWith({ kind: 'cube' });
    });

    test('a right click asks the Duel, which opens its menu or takes the gesture', () => {
        const opening = duelBoard(true);
        opening.b.fire('contextmenu', { x: 5, y: 5 }, 2);
        expect(opening.duel.context).toHaveBeenCalledWith({ kind: 'outside' });
        expect(opening.b.deps.openContextMenu).toHaveBeenCalledTimes(1);
        const taking = duelBoard(false);
        taking.b.fire('contextmenu', taking.b.slot(13, 0), 2);
        expect(taking.duel.context).toHaveBeenCalledWith({ kind: 'point', point: 13 });
        expect(taking.b.deps.openContextMenu).not.toHaveBeenCalled();
    });

    test('a double click outside toggles the Pile, never resets the move', () => {
        const { b, togglePile } = duelBoard();
        b.fire('dblclick', { x: 5, y: 5 });
        expect(togglePile).toHaveBeenCalledTimes(1);
        expect(b.deps.resetQuizPlay).not.toHaveBeenCalled();
    });
});

describe('dice and player rectangles', () => {
    test('left click rolls a die up (6 wraps to 1), right click down (1 wraps to 6)', () => {
        const b = mount();
        const t = sideTargets(b.geom, b.cfg, 0);
        b.click(t.die(0), 0);
        expect(b.pos().dice).toEqual([4, 1]);
        b.click(t.die(1), 2);
        expect(b.pos().dice).toEqual([4, 6]);
        b.stores.position.update((p) => ({ ...p, dice: [6, 6] }));
        b.state.previousDice = [6, 6];
        b.click(t.die(0), 0);
        expect(b.pos().dice[0]).toBe(1);
    });

    // The Eval panel's own report: a position pasted from the analysis panel
    // carries its roll, but previousDice is still the [0, 0] enterEvalMode
    // seeds an Eval session with. A die click used to restore that [0, 0]
    // over the pasted roll, leaving half a roll — read as "no dice", i.e. a
    // cube decision on a board plainly asking a checker question, and the
    // candidate moves vanished with it.
    test('a die click steps the roll already on the board instead of restoring a stale one', () => {
        const b = mount({ position: { ...emptyPos(), dice: [6, 5] } });
        b.state.previousDice = [0, 0];
        const t = sideTargets(b.geom, b.cfg, 0);
        b.click(t.die(0), 0);
        expect(b.pos().dice).toEqual([1, 5]);
        b.click(t.die(1), 2);
        expect(b.pos().dice).toEqual([1, 4]);
    });

    test('stepping a cleared die brings the other one along — never half a roll', () => {
        const b = mount({ position: { ...emptyPos(), dice: [0, 0], decision_type: 1 } });
        b.state.previousDice = [0, 0];
        const t = sideTargets(b.geom, b.cfg, 0);
        b.click(t.die(0), 0);
        expect(b.pos(), 'a die click asks for a checker decision').toMatchObject({ decision_type: 0, dice: [1, 1] });
        // Right-clicking a cleared die wraps to 6 rather than walking below 0,
        // which used to leave the position permanently dice-less.
        const c = mount({ position: { ...emptyPos(), dice: [0, 0] } });
        c.state.previousDice = [0, 0];
        c.click(sideTargets(c.geom, c.cfg, 0).die(1), 2);
        expect(c.pos().dice).toEqual([1, 6]);
    });

    test('stepping a die never writes through to the remembered roll', () => {
        const b = mount();
        const t = sideTargets(b.geom, b.cfg, 0);
        b.click(t.rect(1)); // clears the dice, remembering [3, 1]
        b.click(sideTargets(b.geom, b.cfg, 1).die(0), 0);
        expect(b.pos().dice).toEqual([4, 1]);
        expect(b.state.previousDice, 'the snapshot is a copy, not the live array').toEqual([3, 1]);
    });

    test("a player's rectangle makes it a cube decision for that player and remembers the dice", () => {
        const b = mount();
        const t = sideTargets(b.geom, b.cfg, 0);
        b.click(t.rect(1));
        expect(b.pos()).toMatchObject({ player_on_roll: 1, decision_type: 1, dice: [0, 0] });
        expect(b.state.previousDice).toEqual([3, 1]);
        // The dice now sit on the top side; a click there restores and bumps them.
        b.click(sideTargets(b.geom, b.cfg, 1).die(0), 0);
        expect(b.pos()).toMatchObject({ decision_type: 0, dice: [4, 1] });
    });

    test('the bottom rectangle keeps the bottom player on roll', () => {
        const b = mount();
        b.click(sideTargets(b.geom, b.cfg, 0).rect(0));
        expect(b.pos()).toMatchObject({ player_on_roll: 0, decision_type: 1, dice: [0, 0] });
    });
});

describe('score clicks', () => {
    test('left lowers the away count, right raises it', () => {
        const b = mount();
        const t = sideTargets(b.geom, b.cfg, 0);
        b.click(t.score(0), 0);
        expect(b.pos().score).toEqual([6, 7]);
        b.click(t.score(1), 2);
        expect(b.pos().score).toEqual([6, 8]);
    });

    test('money is symmetric: reaching -1 on one side sets the other; leaving it copies the score', () => {
        expect(applyScoreClick({ ...emptyPos(), score: [0, 5] }, 0, 0).score).toEqual([-1, -1]);
        expect(applyScoreClick({ ...emptyPos(), score: [-1, -1] }, 1, 2).score).toEqual([0, 0]);
        expect(applyScoreClick({ ...emptyPos(), score: [99, 4] }, 0, 2).score).toEqual([99, 4]);
    });
});

// ── Double-click, context menu, detach ──────────────────────────────────────

describe('double-click and context menu', () => {
    test('right-click on a point opens the menu in NORMAL mode only, never when a modal is open, and always eats the native menu', () => {
        const b = mount({ mode: 'NORMAL' });
        const at = b.slot(7, 0);
        let event = b.fire('contextmenu', at, 2);
        expect(event.defaultPrevented).toBe(true);
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(1);
        expect(b.deps.openContextMenu.mock.calls[0][0]).toEqual({ x: RECT.left + at.x, y: RECT.top + at.y });

        b.stores.anyModalOpen.set(true);
        b.fire('contextmenu', at, 2);
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(1);
        b.stores.anyModalOpen.set(false);

        for (const mode of ['EDIT', 'EVAL']) {
            b.state.mode = mode;
            event = b.fire('contextmenu', at, 2);
            expect(event.defaultPrevented).toBe(true);
            expect(b.deps.openContextMenu).toHaveBeenCalledTimes(1);
        }
    });

    test('detach removes every listener', () => {
        const b = mount({ mode: 'NORMAL' });
        b.detach();
        b.state.mode = 'EDIT';
        b.click(b.slot(7, 0));
        b.fire('dblclick', { x: 5, y: 5 });
        b.state.mode = 'NORMAL';
        const event = b.fire('contextmenu', b.slot(7, 0), 2);
        expect(b.pos()).toEqual(emptyPos());
        expect(b.deps.reset).not.toHaveBeenCalled();
        expect(b.deps.openContextMenu).not.toHaveBeenCalled();
        expect(event.defaultPrevented).toBe(false);
    });
});

describe('mousedown blurs a focused text field', () => {
    beforeEach(() => {
        document.body.innerHTML = '';
    });
    test('so board shortcuts are not typed into the input', () => {
        const input = document.createElement('input');
        document.body.appendChild(input);
        input.focus();
        expect(document.activeElement).toBe(input);
        const b = mount({ mode: 'NORMAL' });
        b.fire('mousedown', b.slot(7, 0));
        expect(document.activeElement).not.toBe(input);
    });
});

// ── Le coup du quiz joué sur le plateau (#294) ───────────────────────────────
//
// Ce que le réducteur ne peut pas vérifier seul : que le point CLIQUÉ devienne
// le bon point du modèle. Une position dont le joueur 2 est au trait est
// affichée en miroir, et la conversion 25 - p est exactement le détail qui se
// découvre six mois plus tard, en jouant un coup qui part du mauvais point.
describe('the quiz move is played on the board', () => {
    /**
     * Un plateau monté, puis armé d'un coup de quiz.
     * @param {{mirrored?: boolean, plays: any[], position: any}} opts
     */
    function mountQuiz({ mirrored = false, plays, position }) {
        const b = mount({ mode: 'NORMAL', position, mirrored });
        b.stores.quizPlay.set(newPlay(position, plays));
        return b;
    }

    /**
     * @param {Record<number, [number, number]>} stacks
     * @param {number} [mover]
     */
    function posWith(stacks, mover = 0) {
        const p = emptyPos();
        p.board.bearoff = [0, 0];
        for (const [pt, [n, color]] of Object.entries(stacks)) p.board.points[Number(pt)] = { checkers: n, color };
        p.player_on_roll = mover;
        return p;
    }

    // 2-1 : le moteur ne rend que 13/11, le 1 restant injouable dans ce plateau de test.
    const play13to11 = [{ steps: [{ from: 13, to: 11, hit: false }], notation: '13/11', result: {} }];

    test('the clicked source and destination move the checker', () => {
        const position = { ...posWith({ 13: [5, 0] }), dice: [2, 1] };
        const b = mountQuiz({ plays: play13to11, position });
        b.click(b.slot(13, 0));
        b.click(b.slot(11, 0));
        const state = get(b.stores.quizPlay);
        expect(state.steps).toEqual([{ from: 13, to: 11 }]);
        expect(state.board.points[11].checkers).toBe(1);
        b.detach();
    });

    test('a mirrored board maps the clicked point back to the model', () => {
        const position = { ...posWith({ 13: [5, 0] }), dice: [2, 1] };
        const b = mountQuiz({ mirrored: true, plays: play13to11, position });
        // En miroir, le point 13 du modèle est dessiné là où le 12 le serait.
        b.click(b.slot(12, 0));
        b.click(b.slot(14, 0));
        expect(get(b.stores.quizPlay).steps).toEqual([{ from: 13, to: 11 }]);
        b.detach();
    });

    test('the position itself never moves: it is the question', () => {
        const position = { ...posWith({ 13: [5, 0] }), dice: [2, 1] };
        const b = mountQuiz({ plays: play13to11, position });
        b.click(b.slot(13, 0));
        b.click(b.slot(11, 0));
        expect(b.pos().board.points[13].checkers).toBe(5);
        b.detach();
    });

    test('a click no legal play offers moves nothing, and says nothing', () => {
        const position = { ...posWith({ 13: [5, 0] }), dice: [2, 1] };
        const b = mountQuiz({ plays: play13to11, position });
        b.click(b.slot(13, 0));
        b.click(b.slot(9, 0));
        expect(get(b.stores.quizPlay).steps).toEqual([]);
        b.detach();
    });

    test('with no move armed, editing is untouched', () => {
        const b = mount({ mode: 'EDIT' });
        b.click(b.slot(13, 2));
        expect(b.pos().board.points[13]).toEqual({ checkers: 3, color: 0 });
        b.detach();
    });
});

describe('le coup joué au plateau d’une transcription (T2.3, ADR-0052)', () => {
    /**
     * @param {Record<number, [number, number]>} stacks
     */
    function posWith(stacks, mover = 0) {
        const p = emptyPos();
        p.board.bearoff = [0, 0];
        p.dice = [0, 0];
        for (const [pt, [n, color]] of Object.entries(stacks)) p.board.points[Number(pt)] = { checkers: n, color };
        p.player_on_roll = mover;
        return p;
    }

    const step = (/** @type {number} */ from, /** @type {number} */ to) => ({ from, to, hit: false });
    const play = (/** @type {{ from: number, to: number, hit: boolean }[]} */ ...steps) => ({ steps, notation: '', result: {} });
    const POSITION = posWith({ 13: [5, 0], 8: [3, 0], 6: [5, 0] });
    const BY_ROLL = [
        { dice: [6, 1], plays: [play(step(13, 7), step(8, 7))] },
        { dice: [6, 6], plays: [play(step(13, 7), step(13, 7), step(8, 2), step(8, 2))] }
    ];

    /**
     * Un plateau monté sur un coup de transcription : l'union des jets, ou le
     * seul jet saisi (ADR-0052).
     */
    function mountPlay({ rolled = false } = {}) {
        const b = mount({ mode: 'TRANSCRIBE', position: POSITION });
        b.stores.quizPlay.set(rolled ? newBoardPlay(POSITION, [BY_ROLL[0]], { rolled: [6, 1] }) : newBoardPlay(POSITION, BY_ROLL));
        return b;
    }

    // Le budget d'ux.md §4.1 compte UN P B B par pas : c'est le glissé, pas
    // deux clics. Sans lui, quatre pas coûteraient huit gestes et le double
    // sortirait du budget.
    test('un glissé joue le pas en un seul geste', () => {
        const b = mountPlay();
        b.drag(b.slot(13, 0), b.slot(7, 0));
        expect(get(b.stores.quizPlay).steps).toEqual([{ from: 13, to: 7 }]);
        b.detach();
    });

    test('deux clics font exactement ce que fait le glissé', () => {
        const dragged = mountPlay();
        dragged.drag(dragged.slot(13, 0), dragged.slot(7, 0));
        const byDrag = get(dragged.stores.quizPlay);
        dragged.detach();

        const clicked = mountPlay();
        clicked.click(clicked.slot(13, 0));
        clicked.click(clicked.slot(7, 0));
        expect(get(clicked.stores.quizPlay).steps).toEqual(byDrag.steps);
        clicked.detach();
    });

    // La recette de la fiche : quatre pas, quatre gestes, et les deux dés du
    // double déduits sans qu'un chiffre ait été tapé.
    test('un double se joue en quatre glissés et déduit 6-6', () => {
        const b = mountPlay();
        const hops = [
            [13, 7],
            [13, 7],
            [8, 2],
            [8, 2]
        ];
        let gestures = 0;
        for (const [from, to] of hops) {
            b.drag(b.slot(from, 0), b.slot(to, 0));
            gestures += 1;
        }
        expect(gestures).toBe(4);
        const state = get(b.stores.quizPlay);
        expect(state.steps).toHaveLength(4);
        expect(completedPlay(state)).not.toBeNull();
        b.detach();
    });

    test('un glissé qu’aucun coup légal n’offre ne déplace rien', () => {
        const b = mountPlay();
        b.drag(b.slot(13, 0), b.slot(3, 0));
        expect(get(b.stores.quizPlay).steps).toEqual([]);
        b.detach();
    });

    // ADR-0052 : le jet saisi, le même geste pose le pion là où il est lâché.
    // C'est le coup qui a tenu à la table, et il n'est pas jugé (ADR-0044).
    test('le jet saisi, un glissé hors des règles pose le pion là où il est lâché', () => {
        const b = mountPlay({ rolled: true });
        b.drag(b.slot(13, 0), b.slot(3, 0));
        const state = get(b.stores.quizPlay);
        expect(state.free).toBe(true);
        expect(state.steps).toEqual([{ from: 13, to: 3 }]);
        expect(state.board.points[3]).toEqual({ checkers: 1, color: 0 });
        b.detach();
    });

    // Aucun coup de 6-1 ne part de 6 : la pression n'y choisit rien, mais elle
    // ouvre le glissé.
    test('le glissé libre part aussi d’un point qui n’est pas une source légale', () => {
        const b = mountPlay({ rolled: true });
        b.drag(b.slot(6, 0), b.slot(2, 0));
        const state = get(b.stores.quizPlay);
        expect(state.free).toBe(true);
        expect(state.steps).toEqual([{ from: 6, to: 2 }]);
        b.detach();
    });

    test('un clic sur ce point-là ne choisit toujours rien', () => {
        const b = mountPlay({ rolled: true });
        b.click(b.slot(6, 0));
        const state = get(b.stores.quizPlay);
        expect(state.selected).toBeNull();
        expect(state.steps).toEqual([]);
        b.detach();
    });

    test('le jet saisi, un glissé légal reste un pas légal', () => {
        const b = mountPlay({ rolled: true });
        b.drag(b.slot(13, 0), b.slot(7, 0));
        b.drag(b.slot(8, 0), b.slot(7, 1));
        const state = get(b.stores.quizPlay);
        expect(state.free).toBe(false);
        expect(completedPlay(state)).not.toBeNull();
        b.detach();
    });

    test('la position du brouillon ne bouge pas sous le coup joué', () => {
        const b = mountPlay();
        b.drag(b.slot(13, 0), b.slot(7, 0));
        expect(b.pos().board.points[13].checkers).toBe(5);
        b.detach();
    });
});

// ── Le videau cliqué pendant une transcription (T2.5) ───────────────────────
//
// Le plateau ne fait que POSER la demande dans un magasin ; c'est le panneau
// qui en fait un double, parce que lui seul tient le brouillon et sait ce que
// le document attend. Ce qui se mesure ici est donc la CIBLE : le videau, en
// mode TRANSCRIBE, au bouton gauche, et rien d'autre.

describe('le clic sur le videau, en transcription', () => {
    const cubeOf = (/** @type {{ stores: { transcriptionCube: import('svelte/store').Readable<string|null> } }} */ b) => get(b.stores.transcriptionCube);

    test('un clic gauche sur le videau demande un double', () => {
        const b = mount({ mode: 'TRANSCRIBE' });
        b.click(b.state.cubeBox, 0);
        expect(cubeOf(b)).toBe('double');
        b.detach();
    });

    // La position n'est pas éditée : en TRANSCRIBE le plateau montre l'Action
    // du Cursor, il ne se modifie pas au clic (ADR-0045 règle 9).
    test('la position ne bouge pas', () => {
        const b = mount({ mode: 'TRANSCRIBE' });
        const before = JSON.stringify(b.pos());
        b.click(b.state.cubeBox, 0);
        expect(JSON.stringify(b.pos())).toBe(before);
        b.detach();
    });

    test('à côté du videau, rien n’est demandé', () => {
        const b = mount({ mode: 'TRANSCRIBE' });
        b.click({ x: b.state.cubeBox.x, y: b.state.cubeBox.y + b.state.cubeBox.size }, 0);
        expect(cubeOf(b)).toBeNull();
        b.detach();
    });

    // Le bouton droit ouvre le menu de la position, ici comme ailleurs.
    test('le bouton droit ne demande pas de double', () => {
        const b = mount({ mode: 'TRANSCRIBE' });
        b.click(b.state.cubeBox, 2);
        expect(cubeOf(b)).toBeNull();
        b.detach();
    });

    // En EDIT et en EVAL le videau s'ÉDITE (applyCubeClick) : la demande de
    // transcription ne doit pas s'y glisser.
    test('hors du mode TRANSCRIBE, le videau s’édite comme avant', () => {
        for (const mode of ['EDIT', 'EVAL', 'NORMAL']) {
            const b = mount({ mode });
            b.click(b.state.cubeBox, 0);
            expect(cubeOf(b), mode).toBeNull();
            b.detach();
        }
    });

    // Un coup joué au plateau arme `quizPlay`, qui avale TOUS les clics du
    // damier : le videau est essayé avant lui, sans quoi la cible serait morte
    // pendant tout un tour de pions.
    test('un coup en cours au plateau n’avale pas le clic du videau', () => {
        const b = mount({ mode: 'TRANSCRIBE' });
        b.stores.quizPlay.set(newBoardPlay(b.pos(), [], { rolled: [3, 1] }));
        b.click(b.state.cubeBox, 0);
        expect(cubeOf(b)).toBe('double');
        b.detach();
    });
});

// ── La grammaire d'ADR-0086 au plateau ───────────────────────────────────────
//
// Un mode qui pose son rappel de validation passe à la grammaire : le pion cliqué part du
// premier dé non joué, le clic sur les dés intervertit ou valide, le clic droit reprend. Sans
// rappel, rien ne change (les modes basculent un par un).
describe('ADR-0086: the play follows the one grammar once its mode opts in', () => {
    /** @param {Record<number, [number, number]>} stacks */
    function posWith(stacks) {
        const p = emptyPos();
        p.board.bearoff = [0, 0];
        for (const [pt, [n, color]] of Object.entries(stacks)) p.board.points[Number(pt)] = { checkers: n, color };
        return p;
    }
    const position = posWith({ 13: [5, 0], 8: [3, 0], 6: [5, 0] });
    /** @param {[number, number][]} list */
    const play = (list) => ({ steps: list.map(([from, to]) => ({ from, to })), notation: '', result: {} });
    const plays = [
        play([
            [8, 5],
            [6, 5]
        ]),
        play([
            [13, 10],
            [10, 9]
        ]),
        play([
            [8, 7],
            [8, 5]
        ])
    ];

    function mountGrammar({ optIn = true } = {}) {
        const b = mount({ mode: 'NORMAL', position });
        const validate = vi.fn();
        b.stores.quizPlay.set(newPlay(position, plays));
        if (optIn) b.stores.quizPlayValidate.set(validate);
        const dice = sideTargets(b.geom, b.cfg, 0);
        return { b, validate, dice };
    }
    const steps = (/** @type {any} */ b) => get(b.stores.quizPlay).steps.map((/** @type {any} */ s) => [s.from, s.to]);

    test('a click on a checker moves it by the left die, then the right one', () => {
        const { b } = mountGrammar();
        b.click(b.slot(8, 0));
        expect(steps(b)).toEqual([[8, 5]]);
        expect(get(b.stores.quizPlay).selected).toBeNull();
        b.click(b.slot(6, 0));
        expect(steps(b)).toEqual([
            [8, 5],
            [6, 5]
        ]);
        b.detach();
    });

    test('a click on the dice swaps them before the play, and validates it once finished', () => {
        const { b, validate, dice } = mountGrammar();
        b.click(dice.die(0));
        expect(get(b.stores.quizPlay).swapped).toBe(true);
        b.click(b.slot(8, 0));
        expect(steps(b)).toEqual([[8, 7]]);
        expect(validate).not.toHaveBeenCalled();
        b.click(b.slot(8, 0));
        expect(steps(b)).toEqual([
            [8, 7],
            [8, 5]
        ]);
        b.detach();
    });

    test('a finished play is validated by a click on the dice, never on its own', () => {
        const { b, validate, dice } = mountGrammar();
        b.click(b.slot(8, 0));
        b.click(b.slot(6, 0));
        expect(validate).not.toHaveBeenCalled();
        b.click(dice.die(1));
        expect(validate).toHaveBeenCalledTimes(1);
        b.detach();
    });

    test('a right click on the board takes the play back; with nothing played it opens the menu', () => {
        const { b } = mountGrammar();
        b.fire('contextmenu', b.slot(13, 0), 2);
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(1);
        b.click(b.slot(8, 0));
        b.fire('contextmenu', b.slot(13, 0), 2);
        expect(get(b.stores.quizPlay).steps).toEqual([]);
        expect(get(b.stores.quizPlay).board.points[8].checkers).toBe(3);
        expect(b.deps.openContextMenu).toHaveBeenCalledTimes(1);
        b.detach();
    });

    test('without the opt-in the click still chooses a source', () => {
        const { b } = mountGrammar({ optIn: false });
        b.click(b.slot(8, 0));
        expect(steps(b)).toEqual([]);
        expect(get(b.stores.quizPlay).selected).toBe(8);
        b.detach();
    });
});
