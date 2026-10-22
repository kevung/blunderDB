import { describe, it, expect } from 'vitest';
import { readOfLibrary, libraryReplaced } from '../stores/groupActivityStore.js';

/** A promise and the function that settles it. */
function deferred() {
    /** @type {(v?: unknown) => void} */
    let resolve = () => {};
    /** @type {(e?: unknown) => void} */
    let reject = () => {};
    const promise = new Promise((r, j) => ((resolve = r), (reject = j)));
    return { promise, resolve, reject };
}

/** Whether p settled within a few turns of the event loop. */
async function settles(/** @type {Promise<unknown>} */ p) {
    let settled = false;
    p.then(
        () => (settled = true),
        () => (settled = true)
    );
    for (let i = 0; i < 5; i++) await Promise.resolve();
    return settled;
}

describe('reads of the library on screen', () => {
    it('a read answers as usual while the library stays on screen', async () => {
        const answer = deferred();
        const read = readOfLibrary(() => answer.promise);
        answer.resolve(['T1']);
        expect(await read).toEqual(['T1']);
    });

    it('a read answered after another library came on screen is dropped, not shown in it', async () => {
        const answer = deferred();
        const read = readOfLibrary(() => answer.promise);
        libraryReplaced();
        answer.resolve(['of the library left']);
        expect(await settles(read)).toBe(false);
    });

    it('a failure of the library left is not reported in the one coming', async () => {
        const answer = deferred();
        const read = readOfLibrary(() => answer.promise);
        libraryReplaced();
        answer.reject(new Error('gone'));
        expect(await settles(read)).toBe(false);
    });
});
