// Panel height remembered per tab, persisted through Config like every other layout
// preference (the webview's localStorage is not guaranteed to survive a restart). A tab with
// no remembered height uses the default, never the one last dragged on another tab.
import { GetTabPanelHeights, SaveTabPanelHeight } from '../../wailsjs/go/main/Config.js';
import { DEFAULT_PANEL_HEIGHT } from '../stores/panelLayoutStore.js';
import { logger } from './logger.js';

// Key under which Config carries the single height of a config written before heights were per
// tab: it stands in for every tab that has none of its own.
const FALLBACK_TAB = '*';

/** @type {Record<string, number>} */
let heights = {};

export async function initTabHeights() {
    try {
        heights = (await GetTabPanelHeights()) || {};
    } catch (err) {
        logger.error('Failed to load per-tab panel heights, using default:', err);
        heights = {};
    }
}

/** @param {string} tab @returns {number} */
export function tabPanelHeight(tab) {
    const h = heights[tab] ?? heights[FALLBACK_TAB];
    return Number.isFinite(h) && h > 0 ? h : DEFAULT_PANEL_HEIGHT;
}

/** @param {string} tab @param {number} height */
export function rememberTabHeight(tab, height) {
    if (!tab || !Number.isFinite(height) || height <= 0) return;
    heights = { ...heights, [tab]: Math.round(height) };
    Promise.resolve()
        .then(() => SaveTabPanelHeight(tab, Math.round(height)))
        .catch((err) => logger.error('Failed to save panel height:', err));
}
