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
/** @typedef {{ running: boolean, server: string, hasToken: boolean, lastSuccess: string, error?: { kind: string, message: string, allowance?: { axis: string, allowed: number, held: number, new: number } } | null, result?: { sent: SyncTotals, received: SyncTotals } | null, skipped?: SyncSkip[] | null }} SyncState */
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
    const msg = syncStatusMessage(s);
    if (msg) statusBarTextStore.set(msg);
}

/**
 * What the status bar says when a run ends, since nobody opens the tab to look: a failure, or
 * the records that did not get through. Nothing for a run under way or one that left nothing behind.
 * @param {SyncState} s
 */
export function syncStatusMessage(s) {
    if (s.running) return null;
    if (s.error) {
        const w = syncErrorWords(s.error);
        if (!w.key) return w.verbatim;
        return tMsg(w.key, w.params.axis ? { ...w.params, axis: tMsg(String(w.params.axis)) } : w.params);
    }
    const left = syncSkipped(s).length + syncUnapplied(s.result);
    return left > 0 ? tMsg('sync.skipped.status', { count: left }) : null;
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
    return ['interrupted', 'invalid', 'reference'].includes(reason) ? `sync.skipped.${reason}` : 'sync.skipped.too_large';
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

const ERROR_KINDS = ['refused', 'unavailable', 'rate_limited', 'too_large', 'blocked', 'address', 'no_token', 'no_library', 'library_changed', 'tool_locked', 'invalid_records'];

/** The server words an exceeded allowance itself: its message is shown as is, not translated. @param {string} kind */
export function syncErrorIsVerbatim(kind) {
    return kind === 'allowance' || kind === 'server_refusal';
}

const ALLOWANCE_AXES = ['positions', 'matches', 'storage', 'collections'];

/**
 * The words of a failed run: the i18n key with its values, and the technical detail to show under it.
 * An exceeded allowance is worded here from the server's figures, in the interface language; the server's
 * own message is only the fallback when it gave no figures, and is never shown twice.
 * @param {{kind: string, message: string, allowance?: {axis: string, allowed: number, held: number, new: number}}} error
 * @returns {{key: string, params: Record<string, string|number>, verbatim: string, detail: string}}
 */
export function syncErrorWords(error) {
    const a = error.allowance;
    if (error.kind === 'allowance' && a) {
        const axis = ALLOWANCE_AXES.includes(a.axis) ? a.axis : 'other';
        return { key: 'sync.error.allowance', params: { axis: `sync.allowance.axis.${axis}`, allowed: a.allowed, held: a.held, new: a.new }, verbatim: '', detail: '' };
    }
    if (syncErrorIsVerbatim(error.kind)) return { key: '', params: {}, verbatim: error.message, detail: '' };
    return { key: syncErrorKey(error.kind), params: {}, verbatim: '', detail: error.message };
}

/** The i18n key wording a failed run; an unforeseen kind falls back to the generic one. @param {string} kind */
export function syncErrorKey(kind) {
    return ERROR_KINDS.includes(kind) ? `sync.error.${kind}` : 'sync.error.other';
}
