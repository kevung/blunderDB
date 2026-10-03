import { get, writable } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { ConfigureMCPHost, GetMCPHostStatus } from '../../wailsjs/go/gui/App.js';
import { GetMCPHost, SaveMCPHost } from '../../wailsjs/go/main/Config.js';
import { viewStore } from '../stores/viewStore.js';
import { statusBarModeStore, statusBarTextStore } from '../stores/uiStore';
import { processCommand } from '../commandProcessor.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';

// The MCP server the window serves on localhost, and the screen side of its display tools:
// the Go side only asks (open a view, show a position), the window acts on its own state, a
// view of its tabs and a search through the command bar, as if the user had typed it.

export const DEFAULT_MCP_PORT = 8765;

/** @type {import('svelte/store').Writable<{ running: boolean, url: string, write: boolean, error: string }>} */
export const mcpHostStatusStore = writable(stopped());

function stopped() {
    return { running: false, url: '', write: false, error: '' };
}

let subscribed = false;

/** Listen to the display tools and start the server if the user enabled it. */
export async function initMCPHost() {
    if (!subscribed) {
        subscribed = true;
        EventsOn('mcp:open-view', (/** @type {{ name: string, query: string }} */ e) => openSearchView(e.name, e.query));
        EventsOn('mcp:show-position', (/** @type {{ id: number }} */ e) => showPositionById(e.id));
    }
    try {
        await applyMCPHost(await GetMCPHost());
    } catch (error) {
        logger.error('Error starting the MCP server:', error);
    }
}

/**
 * Start, restart or stop the server to match a setting.
 * @param {{ on: boolean, port: number, write: boolean }} s
 */
export async function applyMCPHost(s) {
    const status = await ConfigureMCPHost({ enabled: !!s.on, port: s.port || DEFAULT_MCP_PORT, write: !!s.write });
    mcpHostStatusStore.set(status ?? stopped());
    return status;
}

/**
 * Persist a setting, then apply it.
 * @param {{ on: boolean, port: number, write: boolean }} s
 */
export async function saveMCPHost(s) {
    await SaveMCPHost(/** @type {any} */ (s));
    return applyMCPHost(s);
}

export async function refreshMCPHostStatus() {
    mcpHostStatusStore.set((await GetMCPHostStatus()) ?? stopped());
}

/**
 * A new view tab named `name`, the search `query` run in it, shown. Only from the library
 * modes a search runs in: in a match, a collection or a transcription the view would inherit a
 * mode that refuses the search, so the request is refused by name instead.
 * @param {string} name
 * @param {string} query
 * @returns {boolean} whether the view was opened
 */
export function openSearchView(name, query) {
    const mode = get(statusBarModeStore);
    if (mode !== 'NORMAL' && mode !== 'EDIT') {
        statusBarTextStore.set(tMsg('assistant.viewNeedsNormalMode'));
        return false;
    }
    viewStore.addView();
    viewStore.renameView(get(viewStore.activeViewId), name);
    processCommand(`s ${query}`.trim());
    return true;
}

/** @param {number} id */
export function showPositionById(id) {
    return openSearchView(`id ${id}`, `id${id}`);
}
