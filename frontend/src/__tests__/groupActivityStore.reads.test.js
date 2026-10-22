import { describe, it, expect } from 'vitest';
import { readOfLibrary, libraryReplaced, isLibraryLeft } from '../stores/groupActivityStore.js';

/** A promise and the function that settles it. */
function deferred() {
    /** @type {(v?: unknown) => void} */
    let resolve = () => {};
    /** @type {(e?: unknown) => void} */
    let reject = () => {};
    const promise = new Promise((r, j) => ((resolve = r), (reject = j)));
    return { promise, resolve, reject };
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
        // Rejected, not left pending: a caller waiting on it lowers its loading flag.
        await expect(read).rejects.toSatisfy(isLibraryLeft);
    });

    it('a failure of the library left is not reported in the one coming', async () => {
        const answer = deferred();
        const read = readOfLibrary(() => answer.promise);
        libraryReplaced();
        answer.reject(new Error('gone'));
        await expect(read).rejects.toSatisfy(isLibraryLeft);
    });
});
