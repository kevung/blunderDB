import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';
import TheatreMiniBoard from '../components/TheatreMiniBoard.svelte';
import { positionStore } from '../stores/positionStore.js';
import { boardOrientationStore } from '../stores/boardOrientationStore.js';

const position = {
    board: { points: Array.from({ length: 26 }, (_, i) => ({ checkers: i === 6 ? 5 : 0, color: i === 6 ? 0 : -1 })), bearoff: [3, 2] },
    cube: { owner: -1, value: 0 },
    dice: [3, 1],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
};

afterEach(() => {
    cleanup();
    boardOrientationStore.set('right');
    positionStore.set(/** @type {any} */ (null));
});

describe('the theatre mini board follows the board orientation', () => {
    test('changing the orientation redraws the mini board', async () => {
        positionStore.set(/** @type {any} */ (position));
        const { container } = render(TheatreMiniBoard);
        await tick();
        const before = container.innerHTML;
        expect(before).toContain('<svg');
        boardOrientationStore.set('left');
        await tick();
        const after = container.innerHTML;
        expect(after).toContain('<svg');
        expect(after).not.toBe(before);
    });
});
