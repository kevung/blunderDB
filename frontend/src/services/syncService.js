import { writable } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { SyncStatus, SyncNow, SyncSetToken } from '../../wailsjs/go/gui/App.js';
import { GetSync, SaveSync } from '../../wailsjs/go/main/Config.js';
import { logger } from '../utils/logger.js';

// Synchronisation with a remote library: an address and a token the user pastes. The Go side
// runs it (at startup if asked, on demand) and tells the window how it went by sync:status;
// the token goes straight to the system keyring and never comes back here.

/** @typedef {{ server: string, auto: boolean }} SyncSettings */
/** @typedef {{ running: boolean, server: string, hasToken: boolean, lastSuccess: string, error?: { kind: string, message: string } | null, result?: { sent: { records: number }, received: { records: number } } | null }} SyncState */

/** @type {import('svelte/store').Writable<SyncSettings>} */
export const syncSettingsStore = writable({ server: '', auto: false });
/** @type {import('svelte/store').Writable<SyncState>} */
export const syncStateStore = writable({ running: false, server: '', hasToken: false, lastSuccess: '' });

let subscribed = false;

export async function loadSync() {
    try {
        const [saved, state] = await Promise.all([GetSync(), SyncStatus()]);
        syncSettingsStore.set({ server: saved?.server ?? '', auto: !!saved?.auto });
        syncStateStore.set(state);
        if (!subscribed) {
            subscribed = true;
            EventsOn('sync:status', (/** @type {SyncState} */ s) => syncStateStore.set(s));
        }
    } catch (error) {
        logger.error('Error loading the sync setting:', error);
    }
}

/** @param {SyncSettings} s */
export async function saveSyncSettings(s) {
    await SaveSync(s);
    syncSettingsStore.set(s);
    syncStateStore.set(await SyncStatus());
}

/** @param {string} server @param {string} token */
export async function setSyncToken(server, token) {
    await SyncSetToken(server, token);
    syncStateStore.set(await SyncStatus());
}

export async function syncNow() {
    syncStateStore.set(await SyncNow());
}

const ERROR_KINDS = ['refused', 'unavailable', 'rate_limited', 'too_large', 'address', 'no_token', 'no_library'];

/** The i18n key wording a failed run; an unforeseen kind falls back to the generic one. @param {string} kind */
export function syncErrorKey(kind) {
    return ERROR_KINDS.includes(kind) ? `sync.error.${kind}` : 'sync.error.other';
}
