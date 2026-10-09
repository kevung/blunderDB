import { must } from './helpers/must.js';
import { get } from 'svelte/store';
import { expect, vi } from 'vitest';
import { confirmModalStore, resolveConfirm } from '../services/confirmService.js';

/**
 * Answers the themed confirm dialog (confirmService) the way a user would: waits
 * for it to be up, returns its message, and resolves it with `answer`.
 *
 * @param {boolean} answer
 * @returns {Promise<string>}
 */
export async function answerConfirm(answer) {
    await vi.waitFor(() => expect(get(confirmModalStore)).not.toBeNull());
    const message = must(get(confirmModalStore)).message;
    resolveConfirm(answer);
    await Promise.resolve();
    await Promise.resolve();
    return message;
}
