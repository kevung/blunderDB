/**
 * The documentation gallery feeds the Stats filter bar a mocked player list.
 * The bar keys its player options on `Name`, so a list that is not made of
 * PlayerFrequency rows gives every option the same key and crashes the render
 * with each_key_duplicate.
 */

import { describe, test, expect, vi } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';
import { showcasePlayerNames } from '../../tests/e2e/helpers/showcasePlayers.js';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn().mockResolvedValue([]),
    GetAllTournaments: vi.fn().mockResolvedValue([]),
    GetStatsDateRange: vi.fn().mockResolvedValue({ DateFrom: '2025-01-01', DateTo: '2026-01-01' })
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn().mockResolvedValue(null),
    SaveStatsFilter: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../stores/databaseStore.js', () => {
    const { writable } = require('svelte/store');
    return { databasePathStore: writable('/some/db.db'), databaseLoadedStore: writable(true) };
});

import { GetAllPlayerNames } from '../../wailsjs/go/database/Database.js';
import StatsFilterBar from '../components/stats/StatsFilterBar.svelte';

describe('StatsFilterBar with the showcase player list', () => {
    test('mounts one option per player, keyed by a defined name', async () => {
        /** @type {any} */ (GetAllPlayerNames).mockResolvedValue(showcasePlayerNames);
        const { container } = render(StatsFilterBar);
        await waitFor(() => {
            const names = [...container.querySelectorAll('#fb-player option')].map((o) => /** @type {HTMLOptionElement} */ (o).value).filter(Boolean);
            expect(names).toEqual(showcasePlayerNames.map((p) => p.Name));
        });
    });
});
