import { get, writable } from 'svelte/store';
import { AssistantAsk, AssistantConfirm, AssistantCancel, AssistantReset, AssistantPresets } from '../../wailsjs/go/gui/App.js';
import { GetAssistant, SaveAssistant } from '../../wailsjs/go/main/Config.js';
import { logger } from '../utils/logger.js';

// The in-app assistant, a client of the window's own MCP tools (ADR-0064). Strictly opt-in:
// off until the user turns it on, and a remote provider only once its privacy warning is
// accepted. The API key never passes through here: the Go side reads it from the keyring.

/** @typedef {{ on: boolean, preset: string, baseURL: string, model: string, write: boolean, remoteAck: string }} AssistantSettings */
/** @typedef {{ id: string, name: string, baseURL: string, model: string, remote: boolean, needsKey: boolean }} Preset */
/** @typedef {{ kind: string, text?: string, tool?: string, args?: string, result?: string, free?: boolean }} Entry */

/** @returns {AssistantSettings} */
function defaults() {
    return { on: false, preset: 'ollama', baseURL: '', model: '', write: false, remoteAck: '' };
}

/** @type {import('svelte/store').Writable<AssistantSettings>} */
export const assistantSettingsStore = writable(defaults());
/** @type {import('svelte/store').Writable<Preset[]>} */
export const assistantPresetsStore = writable([]);
/** @type {import('svelte/store').Writable<Entry[]>} */
export const assistantEntriesStore = writable([]);
/** @type {import('svelte/store').Writable<{ tool: string, title: string, args: string } | null>} */
export const assistantPendingStore = writable(null);
export const assistantBusyStore = writable(false);
export const assistantErrorStore = writable('');

export async function loadAssistantSettings() {
    try {
        const [saved, presets] = await Promise.all([GetAssistant(), AssistantPresets()]);
        assistantSettingsStore.set({ ...defaults(), ...(saved || {}), preset: saved?.preset || 'ollama' });
        assistantPresetsStore.set(presets || []);
    } catch (error) {
        logger.error('Error loading the assistant setting:', error);
    }
}

/** @param {AssistantSettings} s */
export async function saveAssistantSettings(s) {
    assistantSettingsStore.set(s);
    await SaveAssistant(/** @type {any} */ (s));
}

/**
 * @param {AssistantSettings} s
 * @param {Preset[]} presets
 * @returns {Preset | undefined}
 */
export function presetOf(s, presets) {
    return presets.find((p) => p.id === s.preset) ?? presets[0];
}

/**
 * The address a sentence actually goes to: the user's, else the preset's.
 * @param {AssistantSettings} s
 * @param {Preset[]} presets
 */
export function effectiveURL(s, presets) {
    return (s.baseURL || presetOf(s, presets)?.baseURL || '').trim();
}

/**
 * Whether an address is off this machine: any host that is not loopback, whatever preset it came
 * from; an unreadable address counts as remote. Mirrors assistant.IsRemote, which the Go side
 * checks again before sending anything.
 * @param {string} url
 */
export function isRemoteURL(url) {
    let host;
    try {
        host = new URL(url).hostname.toLowerCase();
    } catch {
        return true;
    }
    if (!host) return true;
    if (host === 'localhost' || host.endsWith('.localhost') || host === '[::1]' || host === '::1') return false;
    return !/^127(\.\d{1,3}){3}$/.test(host);
}

/**
 * Whether the sentences would leave the machine for an address the user has not accepted.
 * @param {AssistantSettings} s
 * @param {Preset[]} presets
 */
export function needsPrivacyAck(s, presets) {
    const url = effectiveURL(s, presets);
    return isRemoteURL(url) && s.remoteAck !== url;
}

/** @param {{ entries?: Entry[], pending?: any }} turn */
function applyTurn(turn) {
    assistantEntriesStore.update((es) => [...es, ...(turn?.entries || [])]);
    assistantPendingStore.set(turn?.pending || null);
}

/** @param {unknown} error */
function message(error) {
    const e = /** @type {any} */ (error);
    return e?.message || String(error);
}

/**
 * Send one sentence. Refused while the assistant is off, while a remote provider's warning is
 * unanswered, and while a prepared change awaits its confirmation.
 * @param {string} text
 */
export async function askAssistant(text) {
    const s = get(assistantSettingsStore);
    const sentence = text.trim();
    if (!s.on || !sentence || get(assistantBusyStore) || get(assistantPendingStore)) return false;
    if (needsPrivacyAck(s, get(assistantPresetsStore))) {
        assistantErrorStore.set('privacy');
        return false;
    }
    assistantErrorStore.set('');
    assistantBusyStore.set(true);
    try {
        const turn = await AssistantAsk(/** @type {any} */ ({ preset: s.preset, baseURL: s.baseURL, model: s.model, write: !!s.write, text: sentence }));
        applyTurn(turn);
        return true;
    } catch (error) {
        assistantEntriesStore.update((es) => [...es, { kind: 'user', text: sentence }]);
        assistantErrorStore.set(/privacy/.test(message(error)) ? 'privacy' : message(error));
        return false;
    } finally {
        assistantBusyStore.set(false);
    }
}

/** @param {boolean} accept */
export async function confirmAssistant(accept) {
    if (!get(assistantPendingStore) || get(assistantBusyStore)) return;
    assistantBusyStore.set(true);
    try {
        applyTurn(await AssistantConfirm(accept));
    } catch (error) {
        assistantPendingStore.set(null);
        assistantErrorStore.set(message(error));
    } finally {
        assistantBusyStore.set(false);
    }
}

export function cancelAssistant() {
    AssistantCancel();
}

export async function resetAssistant() {
    await AssistantReset();
    assistantEntriesStore.set([]);
    assistantPendingStore.set(null);
    assistantErrorStore.set('');
}
