// One bottom-panel height for every tab, persisted through Config like every other layout
// preference (the webview's localStorage is not guaranteed to survive a restart). Switching
// tab never resizes the panel: a taller tab scrolls inside it, a shorter one leaves room.
// Config keeps the height under its shared "*" entry.
import { GetTabPanelHeights, SaveTabPanelHeight } from '../../wailsjs/go/main/Config.js';
import { DEFAULT_PANEL_HEIGHT } from '../stores/panelLayoutStore.js';
import { logger } from './logger.js';

const SHARED_KEY = '*';

let height = DEFAULT_PANEL_HEIGHT;

export async function initTabHeights() {
    try {
        const h = ((await GetTabPanelHeights()) || {})[SHARED_KEY];
        height = Number.isFinite(h) && h > 0 ? h : DEFAULT_PANEL_HEIGHT;
    } catch (err) {
        logger.error('Failed to load the panel height, using default:', err);
        height = DEFAULT_PANEL_HEIGHT;
    }
}

/** @returns {number} */
export function panelHeightValue() {
    return height;
}

/** @param {number} h */
export function rememberPanelHeight(h) {
    if (!Number.isFinite(h) || h <= 0) return;
    height = Math.round(h);
    Promise.resolve()
        .then(() => SaveTabPanelHeight(SHARED_KEY, height))
        .catch((err) => logger.error('Failed to save panel height:', err));
}
