/**
 * Opening the Stats panel with no database open used to tear the whole UI down.
 *
 * The reload effect calls loadLists(), and on the no-database branch that
 * function has no await at all: it runs to the end inside the effect. It then
 * read back the very $state it had just written (`dbEmpty = playerList.length
 * === 0`), which made the effect depend on what it writes. Svelte re-ran it
 * until it gave up with `effect_update_depth_exceeded` — and once Svelte gives
 * up, nothing on the page updates any more: clicking another panel tab left the
 * Stats panel on screen. Eight Playwright specs turned red on that, and so did
 * the application.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn().mockResolvedValue([]),
    GetAllTournaments: vi.fn().mockResolvedValue([]),
    GetStatsDateRange: vi.fn().mockResolvedValue(null),
    ComputeStats: vi.fn().mockResolvedValue({}),
    GetPlayerTable: vi.fn().mockResolvedValue([])
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn().mockResolvedValue(null),
    SaveStatsFilter: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../stores/uiStore.js', () => {
    const { writable } = require('svelte/store');
    return { dbMutationCounterStore: writable(0) };
});

// No database open — the branch of loadLists() that never awaits.
vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable(''), databaseLoadedStore: writable(false) };
});

import { dbMutationCounterStore } from '../stores/uiStore.js';
import StatsFilterBar from '../components/stats/StatsFilterBar.svelte';

describe('StatsFilterBar with no database open', () => {
    /** @type {string[]} */
    let svelteErrors;
    let restore;

    beforeEach(() => {
        dbMutationCounterStore.set(0);
        svelteErrors = [];
        // Svelte reports effect_update_depth_exceeded by throwing inside the
        // flush; the message reaches console.error rather than the test body.
        const spy = vi.spyOn(console, 'error').mockImplementation((...args) => {
            svelteErrors.push(args.map(String).join(' '));
        });
        restore = () => spy.mockRestore();
    });

    afterEach(() => restore());

    test('mounting does not loop the reload effect', async () => {
        render(StatsFilterBar);
        await waitFor(() => expect(svelteErrors.length + 1).toBeGreaterThan(0));
        await new Promise((r) => setTimeout(r, 50));

        expect(svelteErrors.join('\n')).not.toMatch(/effect_update_depth_exceeded/);
    });

    test('a database mutation re-runs the reload once, without looping', async () => {
        render(StatsFilterBar);
        await new Promise((r) => setTimeout(r, 20));

        dbMutationCounterStore.set(1);
        await new Promise((r) => setTimeout(r, 50));

        expect(svelteErrors.join('\n')).not.toMatch(/effect_update_depth_exceeded/);
    });
});
