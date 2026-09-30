/**
 * L'onglet d'ouverture se choisit sur la Direction chargée, pas sur le vide qui la précède :
 * un tournoi en cours s'ouvre sur l'onglet Direction.
 */
import { test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => {
    const orig = await importOriginal();
    return Object.fromEntries(Object.keys(/** @type {object} */ (orig)).map((k) => [k, vi.fn().mockResolvedValue(null)]));
});

import DirectionView from '../components/direction/DirectionView.svelte';
import { directionStore, openDirectionIdStore } from '../stores/directionStore.js';

afterEach(() => {
    cleanup();
    directionStore.set(null);
    openDirectionIdStore.set(null);
});

test('un tournoi en cours, chargé après le montage, ouvre l’onglet Direction', async () => {
    openDirectionIdStore.set(1);
    directionStore.set(null);
    render(DirectionView);
    await tick();

    directionStore.set(/** @type {any} */ ({ tournamentId: 1, state: 'running', config: { phases: [], tables: { count: 0 } }, proposals: [], running: [], warnings: [] }));
    await tick();

    expect(screen.getByTestId('direction-tab-direction').classList.contains('active')).toBe(true);
});
