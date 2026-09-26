/**
 * tabHandler.js — the App.svelte tab effect's panel logic, testable apart:
 * each "exclusive" tab (matches, stats, tournaments, collections) opens its
 * PANEL when active and closes it otherwise.
 */

import { PANEL, openPanel, closePanel } from '../stores/uiStore.js';

/**
 * Open `tab`'s panel and close the other exclusive tabs' panels; tabs without
 * a PANEL leave them untouched.
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
