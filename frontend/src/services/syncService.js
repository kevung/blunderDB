import { writable } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { SyncStatus, SyncNow, SyncSetToken, SyncAtStartup } from '../../wailsjs/go/gui/App.js';
import { GetSync, SaveSync } from '../../wailsjs/go/main/Config.js';
import { statusBarTextStore } from '../stores/uiStore';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';

// Synchronisation with a remote library: an address and a token the user pastes. The Go side
// runs it (at startup if asked, on demand) and tells the window how it went by sync:status;
// the token goes straight to the system keyring and never comes back here.

/** @typedef {{ server: string, auto: boolean }} SyncSettings */
/** @typedef {{ running: boolean, server: string, hasToken: boolean, lastSuccess: string, error?: { kind: string, message: string } | null, result?: { sent: SyncTotals, received: SyncTotals } | null, skipped?: SyncSkip[] | null }} SyncState */
/** @typedef {{ direction: string, family: string, uuid: string, size?: number, reason: string }} SyncSkip */
/** @typedef {{ records: number, conflicts: number, missing: number, unsupported: number, unsynchronizable?: number }} SyncTotals */

/** @type {import('svelte/store').Writable<SyncSettings>} */
export const syncSettingsStore = writable({ server: '', auto: false });
/** @type {import('svelte/store').Writable<SyncState>} */
export const syncStateStore = writable({ running: false, server: '', hasToken: false, lastSuccess: '' });

let subscribed = false;

/** @param {SyncState} s */
function onStatus(s) {
    syncStateStore.set(s);
    // A failed run is told on the status bar too: nobody opens the tab to look for it.
    if (!s.running && s.error) statusBarTextStore.set(syncErrorIsVerbatim(s.error.kind) ? s.error.message : tMsg(syncErrorKey(s.error.kind)));
}

function subscribe() {
    if (subscribed) return;
    subscribed = true;
    EventsOn('sync:status', onStatus);
}

/** Called once the user's library is open: listens, and starts the run asked for at startup. */
export async function initSync() {
    subscribe();
    try {
        await SyncAtStartup();
    } catch (error) {
        logger.error('Error starting the sync:', error);
    }
}

/**
 * What a run could not apply, summed: records in conflict, with a reference this side lacks, or
 * of a family it does not carry. They count as passed on for the cursor, so they must be shown.
 * The records set aside are listed apart (syncSkipped): they come back at the next run.
 * @param {SyncState['result']} r
 */
export function syncUnapplied(r) {
    let n = 0;
    for (const side of [r?.sent, r?.received]) n += (side?.conflicts ?? 0) + (side?.missing ?? 0) + (side?.unsupported ?? 0);
    return n;
}

/** The records set aside, still waiting to go through. @param {SyncState} s */
export function syncSkipped(s) {
    return s?.skipped ?? [];
}

/** The i18n key wording why a record was set aside. @param {string} reason */
export function syncSkipReasonKey(reason) {
    return reason === 'interrupted' ? 'sync.skipped.interrupted' : 'sync.skipped.too_large';
}

export async function loadSync() {
    try {
        const [saved, state] = await Promise.all([GetSync(), SyncStatus()]);
        syncSettingsStore.set({ server: saved?.server ?? '', auto: !!saved?.auto });
        syncStateStore.set(state);
        subscribe();
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

const ERROR_KINDS = ['refused', 'unavailable', 'rate_limited', 'too_large', 'blocked', 'address', 'no_token', 'no_library', 'library_changed', 'tool_locked'];

/** The server words an exceeded allowance itself: its message is shown as is, not translated. @param {string} kind */
export function syncErrorIsVerbatim(kind) {
    return kind === 'allowance';
}

/** The i18n key wording a failed run; an unforeseen kind falls back to the generic one. @param {string} kind */
export function syncErrorKey(kind) {
    return ERROR_KINDS.includes(kind) ? `sync.error.${kind}` : 'sync.error.other';
}
