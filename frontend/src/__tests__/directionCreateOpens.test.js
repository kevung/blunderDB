/**
 * « Diriger » crée la Direction et l'ouvre : la page remplace le plateau sans autre clic.
 */
import { test, expect, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => {
    const orig = await importOriginal();
    return Object.fromEntries(Object.keys(/** @type {object} */ (orig)).map((k) => [k, vi.fn().mockResolvedValue(null)]));
});

import { createDirection, openDirectionIdStore, directionPageShownStore, defaultConfig } from '../stores/directionStore.js';
import { activeTabStore } from '../stores/uiStore.js';

test('createDirection ouvre la Direction créée', async () => {
    activeTabStore.set('tournaments');
    await createDirection(7, defaultConfig('Open'));
    expect(get(openDirectionIdStore)).toBe(7);
    expect(get(directionPageShownStore)).toBe(true);
});
