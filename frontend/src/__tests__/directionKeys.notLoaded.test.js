/**
 * Tant que la vue Direction (chunk différé) n'est pas montée, le plateau est à l'écran : les
 * gardes de la page ne doivent pas avaler les touches.
 */
import { test, expect, afterEach } from 'vitest';
import { directionPageShown, directionLeavesToPage } from '../services/directionKeys.js';
import { activeTabStore } from '../stores/uiStore.js';
import { openDirectionIdStore, directionViewLoadedStore } from '../stores/directionStore.js';

afterEach(() => {
    activeTabStore.set('matches');
    openDirectionIdStore.set(null);
    directionViewLoadedStore.set(false);
});

test('une page Direction ouverte mais pas encore chargée ne prend aucune touche', () => {
    activeTabStore.set('tournaments');
    openDirectionIdStore.set(1);
    const end = new KeyboardEvent('keydown', { key: 'End' });
    expect(directionPageShown()).toBe(false);
    expect(directionLeavesToPage(end)).toBe(false);
    directionViewLoadedStore.set(true);
    expect(directionPageShown()).toBe(true);
    expect(directionLeavesToPage(end)).toBe(true);
});
