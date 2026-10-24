import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const { answers, written } = vi.hoisted(() => ({
    /** @type {number[][]} what each call of the backend answers, in turn; [] once exhausted */
    answers: [],
    /** @type {(number | null | undefined)[]} */
    written: []
}));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LibrariesWrittenElsewhere: vi.fn(async () => answers.shift() ?? [])
}));
vi.mock('../stores/groupActivityStore.js', () => ({
    libraryWritten: vi.fn((/** @type {number} */ group) => written.push(group))
}));

import { LibrariesWrittenElsewhere } from '../../wailsjs/go/gui/App.js';
import { checkOutsideWrites, initOutsideWrites, stopOutsideWrites, OUTSIDE_WRITES_POLL_MS, OUTSIDE_WRITES_MAX_WAIT_MS } from '../services/outsideWritesService.js';

/** @param {'visible' | 'hidden'} state */
function setVisibility(state) {
    Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => state });
    document.dispatchEvent(new Event('visibilitychange'));
}

beforeEach(() => {
    answers.length = 0;
    written.length = 0;
    vi.mocked(LibrariesWrittenElsewhere).mockClear();
    vi.useFakeTimers();
});

afterEach(() => {
    stopOutsideWrites();
    vi.useRealTimers();
    setVisibility('visible');
});

describe('writes another process made to an open library', () => {
    it('reads a group again once its outside writes stop', async () => {
        answers.push([2], [2], []);
        await checkOutsideWrites();
        await checkOutsideWrites();
        expect(written).toEqual([]);
        await checkOutsideWrites();
        expect(written).toEqual([2]);
        await checkOutsideWrites();
        expect(written).toEqual([2]);
    });

    it('a long run of writes is read at most every max wait, not at every poll', async () => {
        const polls = OUTSIDE_WRITES_MAX_WAIT_MS / OUTSIDE_WRITES_POLL_MS;
        for (let i = 0; i < 3 * polls; i++) {
            answers.push([1]);
            await checkOutsideWrites();
            vi.advanceTimersByTime(OUTSIDE_WRITES_POLL_MS);
        }
        expect(written).toEqual([1, 1]);
    });

    it('a read asked at once during a check in flight is done when that check is answered', async () => {
        /** @type {(v: number[]) => void} */
        let answer = () => {};
        vi.mocked(LibrariesWrittenElsewhere).mockImplementationOnce(() => new Promise((r) => (answer = r)));
        const inFlight = checkOutsideWrites();
        await checkOutsideWrites({ now: true });
        expect(written).toEqual([]);
        answer([4]);
        await inFlight;
        expect(written).toEqual([4]);
    });

    it('nothing written outside: nothing read', async () => {
        await checkOutsideWrites();
        await checkOutsideWrites({ now: true });
        expect(written).toEqual([]);
    });

    it('the window taking focus reads what is pending at once', async () => {
        initOutsideWrites();
        await vi.advanceTimersByTimeAsync(0);
        answers.push([3]);
        await vi.advanceTimersByTimeAsync(OUTSIDE_WRITES_POLL_MS);
        expect(written).toEqual([]);
        window.dispatchEvent(new Event('focus'));
        await vi.advanceTimersByTimeAsync(0);
        expect(written).toEqual([3]);
    });

    it('asks every poll while visible, never while hidden', async () => {
        initOutsideWrites();
        await vi.advanceTimersByTimeAsync(0);
        const first = vi.mocked(LibrariesWrittenElsewhere).mock.calls.length;
        await vi.advanceTimersByTimeAsync(3 * OUTSIDE_WRITES_POLL_MS);
        expect(vi.mocked(LibrariesWrittenElsewhere).mock.calls.length).toBe(first + 3);
        setVisibility('hidden');
        const hidden = vi.mocked(LibrariesWrittenElsewhere).mock.calls.length;
        await vi.advanceTimersByTimeAsync(10 * OUTSIDE_WRITES_POLL_MS);
        expect(vi.mocked(LibrariesWrittenElsewhere).mock.calls.length).toBe(hidden);
        answers.push([1]);
        setVisibility('visible');
        await vi.advanceTimersByTimeAsync(0);
        expect(written).toEqual([1]);
    });
});
