/**
 * CommentPanel.reactivity.test.js
 *
 * Regression: opening the comments tab with no database open (the displayed
 * position has id 0) used to throw `effect_update_depth_exceeded`. The mount
 * effect called loadComments(), whose no-DB branch ran synchronously (no await)
 * and both wrote and read `allComments` in the same pass — making the effect
 * read-and-write the same state, an infinite update loop. displayedComments is
 * now owned solely by the search $effect, so loadComments only writes
 * allComments. See CommentPanel.svelte.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetCommentsByPosition: vi.fn(() => Promise.resolve([])),
    SearchComments: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadPosition: vi.fn(() => Promise.resolve(null)),
    AddComment: vi.fn(() => Promise.resolve()),
    UpdateCommentEntry: vi.fn(() => Promise.resolve()),
    DeleteCommentEntry: vi.fn(() => Promise.resolve())
}));

import CommentPanel from '../components/CommentPanel.svelte';

afterEach(cleanup);

describe('CommentPanel — no infinite effect loop on mount', () => {
    test('mounts with no database open without an update-depth loop', async () => {
        // Svelte logs the infinite-loop error via console.error; spy on it.
        const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
        try {
            render(CommentPanel, { props: { visible: true, onClose: () => {} } });
            await tick();
            await tick();
            await new Promise((r) => setTimeout(r, 50));
        } finally {
            spy.mockRestore();
        }
        const logged = spy.mock.calls.map((c) => c.join(' '));
        const loopErr = logged.find((e) => /effect_update_depth_exceeded|update depth/.test(e));
        expect(loopErr, `console errors: ${logged.join('\n')}`).toBeUndefined();
    });
});
