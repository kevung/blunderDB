/**
 * TheatreMiniBoard.play.test.js — the theatre's board takes a play as the main board does
 * (ADR-0086): a click on a checker moves it by the first die left, a click on the dice validates
 * the finished play, a right click takes the play back. A turned board answers in model points.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';
import TheatreMiniBoard from '../components/TheatreMiniBoard.svelte';
import { positionStore } from '../stores/positionStore.js';
import { quizPlayStore, armBoardMove } from '../stores/quizPlayStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { transcriptionBoardSwapStore } from '../stores/transcriptionStore.js';
import { boardOrientationStore } from '../stores/boardOrientationStore.js';
import { newPlay } from '../services/quizPlay.js';
import { DIAGRAM_WIDTH, DIAGRAM_HEIGHT } from '../services/diagramService.js';
import { boardMetrics } from '../utils/boardGeometry.js';
import { defaultBoardConfig } from '../utils/boardConfig.js';
import { stackSlotCenter, sideLayout } from '../utils/boardScene.js';

// The mini board draws the diagram 24 units taller than its nominal height.
const W = DIAGRAM_WIDTH;
const H = DIAGRAM_HEIGHT + 24;

/** @param {Record<number, [number, number]>} stacks */
function posWith(stacks) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [pt, [n, color]] of Object.entries(stacks)) points[Number(pt)] = { checkers: n, color };
    return { board: { points, bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [3, 1], score: [7, 7], player_on_roll: 0, decision_type: 0 };
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
    ])
];

const geom = boardMetrics(W, H, defaultBoardConfig().widthFactor);
const cfg = { ...defaultBoardConfig(), orientation: /** @type {'right'} */ ('right') };
/** Where a screen point's first checker is drawn. */
const slot = (/** @type {number} */ point) => /** @type {{ x: number, y: number }} */ (stackSlotCenter(geom, cfg, point, 0));
const die = (/** @type {number} */ i) => {
    const side = sideLayout(geom, cfg, 0);
    return { x: side.diceX + i * (side.diceSize + side.diceGap), y: side.diceY };
};

const drawing = () => /** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-drawing"]'));

/**
 * @param {string} type
 * @param {{ x: number, y: number }} at
 * @param {number} [button]
 */
function fire(type, at, button = 0) {
    drawing().dispatchEvent(new MouseEvent(type, { clientX: at.x, clientY: at.y, button, bubbles: true, cancelable: true }));
}
async function click(/** @type {{ x: number, y: number }} */ at) {
    fire('pointerdown', at);
    fire('pointerup', at);
    await tick();
}
const steps = () => get(quizPlayStore)?.steps.map((/** @type {any} */ s) => [s.from, s.to]);

beforeEach(() => {
    // The drawing at its nominal size, at the window's origin.
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue(new DOMRect(0, 0, W, H));
    positionStore.set(/** @type {any} */ (position));
});

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    quizPlayStore.set(null);
    selectedMoveStore.set(null);
    transcriptionBoardSwapStore.set(false);
    boardOrientationStore.set('right');
    positionStore.set(/** @type {any} */ (null));
});

describe('the theatre board takes a play by the one grammar', () => {
    test('a click on a checker moves it by the first die left; the dice validate the finished play', async () => {
        const validate = vi.fn();
        armBoardMove(newPlay(position, plays), validate);
        render(TheatreMiniBoard);
        await tick();
        await click(slot(8));
        expect(steps()).toEqual([[8, 5]]);
        await click(slot(6));
        expect(steps()).toEqual([
            [8, 5],
            [6, 5]
        ]);
        expect(validate).not.toHaveBeenCalled();
        await click(die(0));
        expect(validate).toHaveBeenCalledTimes(1);
    });

    test('a right click on the board takes the play back', async () => {
        armBoardMove(newPlay(position, plays), () => {});
        render(TheatreMiniBoard);
        await tick();
        await click(slot(8));
        expect(steps()).toHaveLength(1);
        fire('contextmenu', slot(13), 2);
        await tick();
        expect(steps()).toEqual([]);
    });

    test('a turned board answers in model points', async () => {
        transcriptionBoardSwapStore.set(true);
        armBoardMove(newPlay(position, plays), () => {});
        render(TheatreMiniBoard);
        await tick();
        // Turned, the model's 8-point is drawn on the screen's 17-point.
        await click(slot(17));
        expect(steps()).toEqual([[8, 5]]);
    });

    test('without an armed play, a press drags the card and plays nothing', async () => {
        quizPlayStore.set(/** @type {any} */ (newPlay(position, plays)));
        render(TheatreMiniBoard);
        await tick();
        await click(slot(8));
        expect(steps()).toEqual([]);
    });

    test('the first step takes the candidate arrows off the drawing', async () => {
        armBoardMove(newPlay(position, plays), () => {});
        selectedMoveStore.set('13/10 10/9');
        render(TheatreMiniBoard);
        await tick();
        // The arrows' colour, as boardScene.drawMoveArrows paints them.
        const arrows = () => (drawing().innerHTML.match(/rgba\(255, 107, 107/g) ?? []).length;
        expect(arrows()).toBeGreaterThan(0);
        await click(slot(8));
        expect(arrows()).toBe(0);
        quizPlayStore.update((/** @type {any} */ s) => ({ ...s, steps: [], board: position.board }));
        await tick();
        expect(arrows()).toBeGreaterThan(0);
    });
});
