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
    DeleteCommentEntry: vi.fn(() => Promise.resolve()),
    TrashCommentEntry: vi.fn(() => Promise.resolve()),
    Tags: vi.fn(() => Promise.resolve([])),
    RecommendedTags: vi.fn(() => Promise.resolve([]))
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetCommentAuthor: vi.fn(() => Promise.resolve('Alice'))
}));

import { GetCommentsByPosition } from '../../wailsjs/go/database/Database.js';
import { positionStore } from '../stores/positionStore';
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

describe('CommentPanel — thread of a position', () => {
    test('lists every author, the user own comments first', async () => {
        vi.mocked(GetCommentsByPosition).mockResolvedValue([
            { id: 1, positionId: 7, text: 'from Bob', author: 'Bob', origin: 'user', createdAt: '2026-01-01 10:00:00' },
            { id: 2, positionId: 7, text: 'from Alice', author: 'Alice', origin: 'user', createdAt: '2026-01-02 10:00:00' }
        ]);
        positionStore.set({ id: 7 });
        const { container } = render(CommentPanel, { props: { visible: true, onClose: () => {} } });
        await new Promise((r) => setTimeout(r, 50));
        const texts = [...container.querySelectorAll('.msg-text')].map((e) => e.textContent);
        expect(texts).toEqual(['from Alice', 'from Bob']);
        const authors = [...container.querySelectorAll('.msg-author')].map((e) => e.textContent);
        expect(authors[0]).toContain('Alice');
        expect(authors[1]).toBe('Bob');
    });
});
