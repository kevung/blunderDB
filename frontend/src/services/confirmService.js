import { writable } from 'svelte/store';

/**
 * Promise-based confirm/cancel dialog, backed by WarningModal in "confirm" mode.
 *
 * Deliberately decoupled from the exclusive `activeModal` store (stores/uiStore.js): a
 * destructive action can be triggered from inside another modal (e.g. deleting the bearoff
 * download from ConfigModal), and reusing `activeModal` there would replace that modal with
 * the confirm dialog instead of layering on top of it. confirmModalStore is null when no
 * confirmation is showing, or { message, confirmLabel, cancelLabel } while one is pending.
 */
export const confirmModalStore = writable(null);

let pendingResolve = null;

/**
 * Show a confirm/cancel dialog and resolve to whether the user confirmed. Any call still
 * pending when a new one starts resolves false first, so a stale confirmation can never fire
 * after the situation that prompted it has moved on.
 *
 * @param {string} message
 * @param {{confirmLabel?: string, cancelLabel?: string}} [options]
 * @returns {Promise<boolean>}
 */
export function confirmAction(message, { confirmLabel = '', cancelLabel = '' } = {}) {
    if (pendingResolve) {
        const resolvePrevious = pendingResolve;
        pendingResolve = null;
        resolvePrevious(false);
    }
    confirmModalStore.set({ message, confirmLabel, cancelLabel });
    return new Promise((resolve) => {
        pendingResolve = resolve;
    });
}

export function resolveConfirm(result) {
    confirmModalStore.set(null);
    const resolve = pendingResolve;
    pendingResolve = null;
    resolve?.(result);
}
