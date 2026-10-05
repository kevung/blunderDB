/**
 * While a Duel holds the board, nothing may put a library position under it: no view change or
 * creation, no go-to-position, no paste, and Ctrl+H folds the Duel tab instead of leaving it.
 */
import { describe, test, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetDuelForm: vi.fn(), SaveDuelForm: vi.fn() }));

async function fresh() {
    vi.resetModules();
    const duel = await import('../stores/duelStore.js');
    const ui = await import('../stores/uiStore.js');
    const views = await import('../stores/viewStore.js');
    return { duel, ui, viewStore: views.viewStore };
}

const live = { state: { ended: false } };

describe('un Duel tient le plateau', () => {
    beforeEach(() => vi.resetModules());

    test('changer ou créer une vue est refusé', async () => {
        const { duel, viewStore } = await fresh();
        viewStore.addView();
        const [first] = get(viewStore.views);
        duel.duelStore.set(live);
        const active = get(viewStore.activeViewId);
        viewStore.switchTo(first.id);
        viewStore.addView();
        expect(get(viewStore.activeViewId)).toBe(active);
        expect(get(viewStore.views)).toHaveLength(2);
        duel.duelStore.set(null);
        viewStore.switchTo(first.id);
        expect(get(viewStore.activeViewId)).toBe(first.id);
    });

    test('Aller à la position ne s ouvre pas', async () => {
        const { duel, ui } = await fresh();
        duel.duelStore.set(live);
        ui.openModal(ui.MODAL.GO_TO_POSITION);
        ui.toggleModal(ui.MODAL.GO_TO_POSITION);
        expect(get(ui.activeModal)).toBeNull();
        duel.duelStore.set(null);
        ui.openModal(ui.MODAL.GO_TO_POSITION);
        expect(get(ui.activeModal)).toBe(ui.MODAL.GO_TO_POSITION);
    });

    test('Ctrl+H replie puis déplie l onglet Duel', async () => {
        const { duel, ui } = await fresh();
        const { toggleDuelPanel } = await import('../services/tabToggles.js');
        (await import('../stores/databaseStore.js')).databasePathStore.set('/tmp/x.db');
        ui.activeTabStore.set('duel');
        duel.duelStore.set(live);
        toggleDuelPanel();
        expect(get(duel.duelFoldedStore)).toBe(true);
        expect(get(ui.activeTabStore)).toBe('duel');
        toggleDuelPanel();
        expect(get(duel.duelFoldedStore)).toBe(false);
    });
});
