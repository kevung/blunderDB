/**
 * App.tabHandler.test.js
 *
 * Tests the tab-handler panel-management logic extracted in
 * `frontend/src/services/tabHandler.js`.
 *
 * Strategy: call `applyTabPanels(tab)` directly (the same function the
 * App.svelte $effect calls) and assert the resulting `openPanels` Set state.
 * This avoids mounting App.svelte and its many complex mocks while giving
 * deterministic coverage of the panel open/close logic.
 *
 * These tests lock the fix for S2 (Stats transitions broken):
 *   - PANEL.STATS is now opened when tab === 'stats'
 *   - PANEL.TOURNAMENT is now opened when tab === 'tournaments'
 *   - PANEL.COLLECTION is now opened when tab === 'collections'
 */

import { describe, test, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';

import { openPanels, PANEL, openPanel } from '../stores/uiStore.js';
import { applyTabPanels } from '../services/tabHandler.js';

// ── Setup ─────────────────────────────────────────────────────────────────────

beforeEach(() => {
    openPanels.set(new Set());
});

// ── Helpers ───────────────────────────────────────────────────────────────────

/** The set of "exclusive" panel ids managed by applyTabPanels. */
const MANAGED = [PANEL.MATCH, PANEL.STATS, PANEL.TOURNAMENT, PANEL.COLLECTION];

/** Assert exactly one managed panel is open. */
function expectOnlyOpen(expectedPanel) {
    for (const p of MANAGED) {
        if (p === expectedPanel) {
            expect(get(openPanels).has(p), `${p} should be open`).toBe(true);
        } else {
            expect(get(openPanels).has(p), `${p} should be closed`).toBe(false);
        }
    }
}

/** Assert all managed panels are closed. */
function expectAllClosed() {
    for (const p of MANAGED) {
        expect(get(openPanels).has(p), `${p} should be closed`).toBe(false);
    }
}

// ── Tests: individual tabs ────────────────────────────────────────────────────

describe('applyTabPanels — individual tab', () => {
    test('matches → opens MATCH, closes others', () => {
        applyTabPanels('matches');
        expectOnlyOpen(PANEL.MATCH);
    });

    test('stats → opens STATS, closes others', () => {
        applyTabPanels('stats');
        expectOnlyOpen(PANEL.STATS);
    });

    test('tournaments → opens TOURNAMENT, closes others', () => {
        applyTabPanels('tournaments');
        expectOnlyOpen(PANEL.TOURNAMENT);
    });

    test('collections → opens COLLECTION, closes others', () => {
        applyTabPanels('collections');
        expectOnlyOpen(PANEL.COLLECTION);
    });

    test('analysis (non-panel tab) → all managed panels closed', () => {
        applyTabPanels('analysis');
        expectAllClosed();
    });

    test('epc (non-panel tab) → all managed panels closed', () => {
        applyTabPanels('epc');
        expectAllClosed();
    });

    test('anki (non-panel tab) → all managed panels closed', () => {
        applyTabPanels('anki');
        expectAllClosed();
    });

    test('search (non-panel tab) → all managed panels closed', () => {
        applyTabPanels('search');
        expectAllClosed();
    });
});

// ── Tests: transitions (S2 regression lock) ───────────────────────────────────

describe('applyTabPanels — transitions (S2 fix)', () => {
    test('matches → stats: STATS opens, MATCH closes', () => {
        applyTabPanels('matches');
        expect(get(openPanels).has(PANEL.MATCH)).toBe(true);

        applyTabPanels('stats');
        expect(get(openPanels).has(PANEL.STATS)).toBe(true);
        expect(get(openPanels).has(PANEL.MATCH)).toBe(false);
    });

    test('epc → stats: STATS opens, all others closed', () => {
        applyTabPanels('epc');
        applyTabPanels('stats');
        expectOnlyOpen(PANEL.STATS);
    });

    test('stats → anki → stats: STATS reopens cleanly', () => {
        applyTabPanels('stats');
        applyTabPanels('anki');
        expect(get(openPanels).has(PANEL.STATS)).toBe(false);

        applyTabPanels('stats');
        expectOnlyOpen(PANEL.STATS);
    });

    test('stats → matches: MATCH opens, STATS closes', () => {
        applyTabPanels('stats');
        applyTabPanels('matches');
        expect(get(openPanels).has(PANEL.MATCH)).toBe(true);
        expect(get(openPanels).has(PANEL.STATS)).toBe(false);
    });

    test('tournaments → stats → collections: correct panel at each step', () => {
        applyTabPanels('tournaments');
        expectOnlyOpen(PANEL.TOURNAMENT);

        applyTabPanels('stats');
        expectOnlyOpen(PANEL.STATS);

        applyTabPanels('collections');
        expectOnlyOpen(PANEL.COLLECTION);
    });
});

// ── Tests: rapid switching ────────────────────────────────────────────────────

describe('applyTabPanels — rapid switching', () => {
    test('5 rapid Match ↔ Stats switches end in consistent state', () => {
        for (let i = 0; i < 5; i++) {
            applyTabPanels(i % 2 === 0 ? 'stats' : 'matches');
        }
        // i goes 0→stats, 1→matches, 2→stats, 3→matches, 4→stats → ends on stats
        expectOnlyOpen(PANEL.STATS);
    });

    test('panels managed by applyTabPanels do not bleed onto unmanaged panels', () => {
        // Put a non-managed panel in openPanels before switching
        openPanel(PANEL.ANALYSIS);
        applyTabPanels('stats');

        // ANALYSIS was not managed by applyTabPanels — must still be open
        expect(get(openPanels).has(PANEL.ANALYSIS)).toBe(true);
        // STATS must be open
        expect(get(openPanels).has(PANEL.STATS)).toBe(true);
    });
});
