/**
 * confirmService.test.js
 *
 * confirmAction() is the promise-based confirm/cancel dialog behind every destructive-action
 * confirmation (delete position, delete match, remove from collection, delete/reset Anki
 * deck, delete the bearoff download). Guards:
 *   - resolveConfirm(true/false) resolves the pending promise with that value.
 *   - confirmModalStore carries the message/labels while a confirmation is pending, and goes
 *     back to null once resolved.
 *   - a second confirmAction() call while one is still pending resolves the first one false —
 *     a stale confirmation can never fire after the situation that prompted it has moved on.
 */

import { describe, test, expect } from 'vitest';
import { get } from 'svelte/store';
import { confirmAction, resolveConfirm, confirmModalStore } from '../services/confirmService.js';

describe('confirmAction / resolveConfirm', () => {
    test('resolves true when confirmed', async () => {
        const pending = confirmAction('Delete this?', { confirmLabel: 'Delete' });
        expect(get(confirmModalStore)).toEqual({ message: 'Delete this?', confirmLabel: 'Delete', cancelLabel: '' });
        resolveConfirm(true);
        await expect(pending).resolves.toBe(true);
        expect(get(confirmModalStore)).toBeNull();
    });

    test('resolves false when cancelled', async () => {
        const pending = confirmAction('Delete this?');
        resolveConfirm(false);
        await expect(pending).resolves.toBe(false);
        expect(get(confirmModalStore)).toBeNull();
    });

    test('a second confirmAction resolves the first one false', async () => {
        const first = confirmAction('First?');
        const second = confirmAction('Second?');
        expect(get(confirmModalStore).message).toBe('Second?');
        await expect(first).resolves.toBe(false);
        resolveConfirm(true);
        await expect(second).resolves.toBe(true);
    });

    test('resolveConfirm without a pending confirmation is a no-op', () => {
        expect(() => resolveConfirm(true)).not.toThrow();
        expect(get(confirmModalStore)).toBeNull();
    });
});
