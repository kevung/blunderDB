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
import { EXCLUDE_EMPTY, stackSlotCenter, cubeBox, sideLayout } from '../utils/boardScene.js';
import { attachBoardInteractions, hitTestSideControls, applyCheckerEdit, applyCubeClick, applyScoreClick } from '../utils/boardInteractions.js';

const W = 1000;
const H = 720;
const RECT = { left: 37, top: 11 };

function makeCfg(orientation = 'right') {
    return { widthFactor: 0.75, orientation, checker: { sizeFactor: 0.97 } };
}

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
function mount({ mode = 'EDIT', orientation = 'right', scale = 1, position = emptyPos() } = {}) {
    const canvas = document.createElement('div');
    document.body.appendChild(canvas);
    canvas.getBoundingClientRect = () => ({ left: RECT.left, top: RECT.top, width: W * scale, height: H * scale });
    const cfg = makeCfg(orientation);
    const geom = boardMetrics(W, H, cfg.widthFactor);
    const stores = {
        position: writable(position),
        structureMode: writable('include'),
        activeTab: writable('positions'),
        offeredCube: writable(false),
        anyModalOpen: writable(false)
    };
    const state = { mode, previousDice: [3, 1], cubeBox: cubeBox(geom, position, false) };
    const deps = {
        getMode: () => state.mode,
        getSize: () => ({ width: W, height: H }),
        cfg,
        getCubeBox: () => state.cubeBox,
        stores,
        getPreviousDice: () => state.previousDice,
        setPreviousDice: (d) => (state.previousDice = d),
        reset: vi.fn(),
        openContextMenu: vi.fn()
    };
    const detach = attachBoardInteractions(canvas, deps);

    // Drawing-space → client pixels, the inverse of boardMouseToDrawing().
    const client = ({ x, y }) => ({ clientX: RECT.left + x * scale, clientY: RECT.top + y * scale });
    const fire = (type, at, button = 0) => {
        const event = new MouseEvent(type, { bubbles: true, cancelable: true, button, ...client(at) });
        canvas.dispatchEvent(event);
        return event;
    };
    const click = (at, button = 0) => {
        fire('mousedown', at, button);
        fire('mouseup', at, button);
    };
    const drag = (from, to, button = 0) => {
        fire('mousedown', from, button);
        fire('mouseup', to, button);
    };
    const slot = (point, index) => stackSlotCenter(geom, cfg, point, index);
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

    test('nothing happens outside EDIT/EPC, or on a click outside the board', () => {
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

    test('EPC: clicks cycle the owner and pin the value', () => {
        const b = mount({ mode: 'EPC' });
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

function sideTargets(geom, cfg, playerOnRoll) {
    const side = sideLayout(geom, cfg, playerOnRoll);
    return {
        die: (i) => ({ x: side.diceX + i * (side.diceSize + side.diceGap), y: side.diceY }),
        rect: (player) => ({ x: side.scoreX, y: (player === 0 ? side.bearoff1Y + side.score1Y : side.bearoff2Y + side.score2Y) / 2 - (player === 0 ? 0.3 : -0.3) * geom.checkerSize }),
        score: (player) => ({ x: side.scoreX, y: player === 0 ? side.score1Y : side.score2Y })
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
        expect(applyScoreClick({ score: [0, 5] }, 0, 0).score).toEqual([-1, -1]);
        expect(applyScoreClick({ score: [-1, -1] }, 1, 2).score).toEqual([0, 0]);
        expect(applyScoreClick({ score: [99, 4] }, 0, 2).score).toEqual([99, 4]);
    });
});

// ── Double-click, context menu, detach ──────────────────────────────────────

describe('double-click and context menu', () => {
    test('a double-click outside the board resets it, inside does not', () => {
        const b = mount();
        b.fire('dblclick', b.slot(7, 0));
        expect(b.deps.reset).not.toHaveBeenCalled();
        b.fire('dblclick', { x: 5, y: 5 });
        expect(b.deps.reset).toHaveBeenCalledTimes(1);
        b.state.mode = 'NORMAL';
        b.fire('dblclick', { x: 5, y: 5 });
        expect(b.deps.reset).toHaveBeenCalledTimes(1);
    });

    test('right-click opens the menu in NORMAL mode only, never when a modal is open, and always eats the native menu', () => {
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

        for (const mode of ['EDIT', 'EPC']) {
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
