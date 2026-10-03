/**
 * Supprimer la direction d'un tournoi se confirme par le dialogue thémé ; refusé, rien n'est supprimé.
 */
import { test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { answerConfirm } from './confirmHelper.js';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => {
    const orig = await importOriginal();
    return Object.fromEntries(Object.keys(/** @type {object} */ (orig)).map((k) => [k, vi.fn().mockResolvedValue(null)]));
});

import DirectionView from '../components/direction/DirectionView.svelte';
import { directionStore, openDirectionIdStore } from '../stores/directionStore.js';
import { DeleteDirection } from '../../wailsjs/go/database/Database.js';

afterEach(() => {
    cleanup();
    directionStore.set(null);
    openDirectionIdStore.set(null);
});

test('supprimer la direction se confirme ; refuser ne supprime rien', async () => {
    openDirectionIdStore.set(1);
    render(DirectionView);
    directionStore.set(/** @type {any} */ ({ tournamentId: 1, state: 'running', config: { phases: [], tables: { count: 0 } }, proposals: [], running: [], warnings: [] }));
    await tick();
    await fireEvent.click(screen.getByTestId('direction-tab-settings'));
    const del = /** @type {HTMLElement} */ (document.querySelector('.actions button.danger'));
    expect(del).not.toBeNull();

    await fireEvent.click(del);
    await answerConfirm(false);
    expect(DeleteDirection).not.toHaveBeenCalled();

    await fireEvent.click(del);
    await answerConfirm(true);
    await vi.waitFor(() => expect(DeleteDirection).toHaveBeenCalledWith(1));
});
