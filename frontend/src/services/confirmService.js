import { writable } from 'svelte/store';

/**
 * Promise-based confirm/cancel dialog (WarningModal in "confirm" mode). Kept
 * apart from the exclusive `activeModal` store so it can layer over a modal
 * that triggers a destructive action instead of replacing it. null, or
 * { message, confirmLabel, cancelLabel } while pending.
 */
export const confirmModalStore = writable(null);

let pendingResolve = null;

/**
 * Show a confirm/cancel dialog; resolves to whether the user confirmed. A call
 * still pending when a new one starts resolves false first, so a stale
 * confirmation never fires.
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
