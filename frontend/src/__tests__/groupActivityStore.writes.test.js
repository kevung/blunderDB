import { describe, it, expect } from 'vitest';
import { get } from 'svelte/store';
import { holdingSwitch, writingOnScreen, groupSwitchingStore } from '../stores/groupActivityStore.js';

/** A promise and the function that settles it. */
function deferred() {
    /** @type {(v?: unknown) => void} */
    let resolve = () => {};
    const promise = new Promise((r) => (resolve = r));
    return { promise, resolve };
}

describe('writes to the library on screen and switches', () => {
    it('a switch waits for the write sent before it', async () => {
        /** @type {string[]} */
        const order = [];
        const answer = deferred();
        const write = writingOnScreen(async () => {
            await answer.promise;
            order.push('write');
        });
        const sw = holdingSwitch(async () => {
            order.push('switch');
            return true;
        });
        await Promise.resolve();
        expect(order).toEqual([]);
        answer.resolve();
        await Promise.all([write, sw]);
        expect(order).toEqual(['write', 'switch']);
        expect(get(groupSwitchingStore)).toBe(false);
    });

    it('a write is refused while a switch is in flight', async () => {
        const answer = deferred();
        const sw = holdingSwitch(() => answer.promise);
        let ran = false;
        const result = await writingOnScreen(async () => {
            ran = true;
        });
        expect(result).toBeUndefined();
        expect(ran).toBe(false);
        answer.resolve();
        await sw;
    });

    it('a failed write does not hold the switch back', async () => {
        const write = writingOnScreen(() => Promise.reject(new Error('refused')));
        const sw = holdingSwitch(async () => true);
        await expect(write).rejects.toThrow('refused');
        expect(await sw).toBe(true);
    });
});
