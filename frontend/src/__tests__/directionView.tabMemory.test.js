/**
 * L'onglet d'une Direction se choisit pour son tournoi : celui où elle fut laissée, sinon celui
 * de son état — jamais celui d'un autre tournoi, et il survit à un rechargement.
 */
import { test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => {
    const orig = await importOriginal();
    return Object.fromEntries(Object.keys(/** @type {object} */ (orig)).map((k) => [k, vi.fn().mockResolvedValue(null)]));
});

import * as DB from '../../wailsjs/go/database/Database.js';
import DirectionView from '../components/direction/DirectionView.svelte';
import { directionStore, openDirectionIdStore, restoreDirection, closeDirection } from '../stores/directionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore } from '../stores/uiStore.js';

const view = (/** @type {number} */ id, /** @type {string} */ state) =>
    /** @type {any} */ ({ tournamentId: id, state, config: { phases: [], tables: { count: 0 } }, proposals: [], running: [], warnings: [] });
const active = () => document.querySelector('[data-testid^="direction-tab-"].active')?.getAttribute('data-testid');

beforeEach(() => {
    localStorage.clear();
    databasePathStore.set('/tmp/a.db');
});
afterEach(() => {
    cleanup();
    closeDirection();
    directionStore.set(null);
    databasePathStore.set('');
});

test('une autre Direction ouverte sans fermer la première ne reprend pas son onglet', async () => {
    openDirectionIdStore.set(1);
    directionStore.set(view(1, 'draft'));
    render(DirectionView);
    await tick();
    screen.getByTestId('direction-tab-standings').click();
    await tick();
    expect(active()).toBe('direction-tab-standings');

    openDirectionIdStore.set(2);
    directionStore.set(view(2, 'draft'));
    await tick();
    await tick();
    expect(active()).toBe('direction-tab-settings');
});

test('rouverte après démontage, une Direction retrouve son onglet', async () => {
    openDirectionIdStore.set(1);
    directionStore.set(view(1, 'running'));
    const first = render(DirectionView);
    await tick();
    screen.getByTestId('direction-tab-standings').click();
    await tick();
    first.unmount();

    render(DirectionView);
    await tick();
    await tick();
    expect(active()).toBe('direction-tab-standings');
});

test('après un rechargement, la dernière Direction se rouvre', async () => {
    localStorage.setItem('blunderdb.direction', JSON.stringify({ dbPath: '/tmp/a.db', openId: 7, tabs: { 7: 'standings' } }));
    vi.mocked(DB.ListDirections).mockResolvedValue(/** @type {any} */ ([{ tournamentId: 7 }]));
    activeTabStore.set('matches');

    await restoreDirection();

    expect(get(openDirectionIdStore)).toBe(7);
    expect(get(activeTabStore)).toBe('tournaments');
});

test('une Direction fermée ne se rouvre pas, ni celle d’une autre base', async () => {
    localStorage.setItem('blunderdb.direction', JSON.stringify({ dbPath: '/tmp/other.db', openId: 7, tabs: {} }));
    vi.mocked(DB.ListDirections).mockResolvedValue(/** @type {any} */ ([{ tournamentId: 7 }]));
    await restoreDirection();
    expect(get(openDirectionIdStore)).toBeNull();
});
