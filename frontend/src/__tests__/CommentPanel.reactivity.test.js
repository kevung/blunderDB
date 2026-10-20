// @ts-nocheck — doublures de test : les signatures typées (Wails, stores) exigent des objets complets.
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

describe('CommentPanel — signed comments', () => {
    test('names the signer, and flags a note altered after signing', async () => {
        vi.mocked(GetCommentsByPosition).mockResolvedValue([
            {
                id: 1,
                positionId: 8,
                text: 'intact',
                author: 'Martin',
                origin: 'user',
                createdAt: '2026-01-01 10:00:00',
                signer: { name: 'Coach Martin', fingerprint: 'A3F1-9C24-7B05-E1D8', verified: true }
            },
            {
                id: 2,
                positionId: 8,
                text: 'altered',
                author: 'Martin',
                origin: 'user',
                createdAt: '2026-01-02 10:00:00',
                signer: { name: 'Coach Martin', fingerprint: 'A3F1-9C24-7B05-E1D8', verified: false }
            },
            { id: 3, positionId: 8, text: 'mine', author: 'Alice', origin: 'user', createdAt: '2026-01-03 10:00:00' }
        ]);
        positionStore.set({ id: 8 });
        const { container } = render(CommentPanel, { props: { visible: true, onClose: () => {} } });
        await new Promise((r) => setTimeout(r, 50));
        const signers = [...container.querySelectorAll('.msg-signer')];
        expect(signers).toHaveLength(2);
        const byText = Object.fromEntries(signers.map((s) => [s.closest('.msg').querySelector('.msg-text').textContent, s]));
        expect(byText.intact.textContent).toContain('Coach Martin');
        expect(byText.intact.classList.contains('unverified')).toBe(false);
        expect(byText.altered.classList.contains('unverified')).toBe(true);
        expect(byText.altered.getAttribute('title')).toContain('A3F1-9C24-7B05-E1D8');
    });

    test('shows the fingerprint beside the name, and says when a signature cannot be read', async () => {
        vi.mocked(GetCommentsByPosition).mockResolvedValue([
            {
                id: 1,
                positionId: 9,
                text: 'coach',
                author: '',
                origin: 'user',
                createdAt: '2026-01-01 10:00:00',
                signer: { name: 'Coach Alice', fingerprint: '21FA-ECC9-77E0-EC62', verified: true }
            },
            {
                id: 2,
                positionId: 9,
                text: 'namesake',
                author: '',
                origin: 'user',
                createdAt: '2026-01-02 10:00:00',
                signer: { name: 'Coach Alice', fingerprint: 'FA33-0578-1C2D-9E10', verified: true }
            },
            {
                id: 3,
                positionId: 9,
                text: 'damaged',
                author: '',
                origin: 'user',
                createdAt: '2026-01-03 10:00:00',
                signer: { name: '', fingerprint: '', verified: false }
            }
        ]);
        positionStore.set({ id: 9 });
        const { container } = render(CommentPanel, { props: { visible: true, onClose: () => {} } });
        await new Promise((r) => setTimeout(r, 50));
        const byText = Object.fromEntries([...container.querySelectorAll('.msg-signer')].map((s) => [s.closest('.msg').querySelector('.msg-text').textContent, s]));
        expect(byText.coach.textContent).toContain('21FA-ECC9-77E0-EC62');
        expect(byText.namesake.textContent).toContain('FA33-0578-1C2D-9E10');
        expect(byText.damaged.querySelector('.msg-fingerprint')).toBeNull();
        expect(byText.damaged.classList.contains('unverified')).toBe(true);
        expect(byText.damaged.textContent).not.toMatch(/ {2}/);
    });
});
