import { writable } from 'svelte/store';
import { libraryOnScreenMark, libraryStillOnScreen } from '../stores/libraryEpoch.js';

/**
 * Promise-based confirm/cancel dialog (WarningModal in "confirm" mode). Kept
 * apart from the exclusive `activeModal` store so it can layer over a modal
 * that triggers a destructive action instead of replacing it. null, or
 * { message, confirmLabel, cancelLabel } while pending.
 */
export const confirmModalStore = writable(null);

/** @type {((result: unknown) => void) | null} */
let pendingResolve = null;

/**
 * Show a confirm/cancel dialog; resolves to whether the user confirmed. A call
 * still pending when a new one starts resolves false first, so a stale
 * confirmation never fires. So does one answered after another library came
 * on screen: what it confirms names rows (ids read, a list counted) of the
 * library left, and acting on them would write into the one coming.
 *
 * @param {string} message
 * @param {{confirmLabel?: string, cancelLabel?: string, choices?: {value: string, label: string, primary?: boolean}[], tone?: 'danger' | 'primary'}} [options]
 *   `tone: 'primary'` for a gesture that deletes nothing (the confirm button is not red).
 * @returns {Promise<boolean>}
 */
export function confirmAction(message, { confirmLabel = '', cancelLabel = '', choices = [], tone = 'danger' } = {}) {
    if (pendingResolve) {
        const resolvePrevious = pendingResolve;
        pendingResolve = null;
        resolvePrevious(false);
    }
    confirmModalStore.set({ message, confirmLabel, cancelLabel, choices, tone });
    const mark = libraryOnScreenMark();
    return new Promise((resolve) => {
        pendingResolve = (/** @type {unknown} */ answer) => resolve(answer && libraryStillOnScreen(mark) ? answer : false);
    });
}

/**
 * A dialog with several answers (and a cancel): resolves to the chosen
 * `value`, or null when dismissed. `primary` marks the one Enter picks.
 *
 * @param {string} message
 * @param {{value: string, label: string, primary?: boolean}[]} choices
 * @param {{cancelLabel?: string}} [options]
 * @returns {Promise<string | null>}
 */
export async function chooseAction(message, choices, { cancelLabel = '' } = {}) {
    const answer = await confirmAction(message, { cancelLabel, choices });
    return typeof answer === 'string' ? answer : null;
}

export function resolveConfirm(result) {
    confirmModalStore.set(null);
    const resolve = pendingResolve;
    pendingResolve = null;
    resolve?.(result);
}
