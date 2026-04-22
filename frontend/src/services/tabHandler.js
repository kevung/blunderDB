/**
 * tabHandler.js
 *
 * Pure panel-management logic for the TabbedPanel tab handler in App.svelte.
 * Extracted here so it can be unit-tested independently of the Svelte component.
 *
 * Rule: for each "exclusive" tab (matches, stats, tournaments, collections) the
 * corresponding PANEL is opened when that tab is active and closed for every
 * other tab. The if/else structure mirrors the App.svelte $effect so that
 * changes here are automatically reflected in both production code and tests.
 */

import { PANEL, openPanel, closePanel } from '../stores/uiStore.js';

/**
 * Open the panel that corresponds to `tab` and close the panels of all other
 * "exclusive" tabs. Tabs that have no associated PANEL (analysis, comments,
 * search, epc, anki, metadata, log) leave those panels untouched.
 *
 * @param {string} tab - The newly active tab id.
 */
export function applyTabPanels(tab) {
    if (tab === 'matches') openPanel(PANEL.MATCH);
    else closePanel(PANEL.MATCH);

    if (tab === 'stats') openPanel(PANEL.STATS);
    else closePanel(PANEL.STATS);

    if (tab === 'tournaments') openPanel(PANEL.TOURNAMENT);
    else closePanel(PANEL.TOURNAMENT);

    if (tab === 'collections') openPanel(PANEL.COLLECTION);
    else closePanel(PANEL.COLLECTION);
}
