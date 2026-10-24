import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { libraryMark, writingOnLibrary, libraryReplaced, holdingSwitch } from '../stores/groupActivityStore.js';
import { HoldLibrary, ReleaseLibrary } from '../stores/libraryHold.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { tMsg } from '../i18n';

/** A promise and the function that settles it. */
function deferred() {
    /** @type {(v?: unknown) => void} */
    let resolve = () => {};
    const promise = new Promise((r) => (resolve = r));
    return { promise, resolve };
}

beforeEach(() => {
    vi.mocked(HoldLibrary).mockReset().mockResolvedValue(7);
    vi.mocked(ReleaseLibrary).mockReset().mockResolvedValue(undefined);
    databasePathStore.set('/libs/a.db');
    statusBarTextStore.set('');
});

describe('a write naming the rows of the library it read them in', () => {
    it('is held in that library by the backend, and let go once answered', async () => {
        const mark = libraryMark();
        const write = vi.fn().mockResolvedValue('written');
        expect(await writingOnLibrary(mark, write)).toBe('written');
        expect(HoldLibrary).toHaveBeenCalledWith('/libs/a.db');
        expect(write).toHaveBeenCalledOnce();
        expect(ReleaseLibrary).toHaveBeenCalledWith(7);
    });

    it('is let go even when it fails', async () => {
        await expect(writingOnLibrary(libraryMark(), () => Promise.reject(new Error('boom')))).rejects.toThrow('boom');
        expect(ReleaseLibrary).toHaveBeenCalledWith(7);
    });

    it('is not sent once another library came on screen, and the status bar says so', async () => {
        const mark = libraryMark();
        libraryReplaced();
        databasePathStore.set('/libs/b.db');
        const write = vi.fn().mockResolvedValue(true);
        expect(await writingOnLibrary(mark, write)).toBeUndefined();
        expect(write).not.toHaveBeenCalled();
        expect(HoldLibrary).not.toHaveBeenCalled();
        expect(get(statusBarTextStore)).toEqual(tMsg('libraryGroups.writeDropped'));
    });

    it('is not sent when the backend carries another library, and the status bar says so', async () => {
        vi.mocked(HoldLibrary).mockRejectedValue(new Error('the library these rows were read in is no longer the open one'));
        const write = vi.fn().mockResolvedValue(true);
        expect(await writingOnLibrary(libraryMark(), write)).toBeUndefined();
        expect(write).not.toHaveBeenCalled();
        expect(ReleaseLibrary).not.toHaveBeenCalled();
        expect(get(statusBarTextStore)).toEqual(tMsg('libraryGroups.writeDropped'));
    });

    it('is not sent during a switch; a switch begun after it waits for its answer', async () => {
        const during = deferred();
        const sw = holdingSwitch(() => during.promise);
        const write = vi.fn().mockResolvedValue(true);
        expect(await writingOnLibrary(libraryMark(), write)).toBeUndefined();
        expect(write).not.toHaveBeenCalled();
        expect(get(statusBarTextStore)).toEqual(tMsg('libraryGroups.switching'));
        during.resolve();
        await sw;

        /** @type {string[]} */
        const order = [];
        const answer = deferred();
        const written = writingOnLibrary(libraryMark(), async () => (await answer.promise, order.push('write'), true));
        await Promise.resolve();
        const next = holdingSwitch(async () => (order.push('switch'), true));
        answer.resolve();
        await Promise.all([written, next]);
        expect(order).toEqual(['write', 'switch']);
    });
});
