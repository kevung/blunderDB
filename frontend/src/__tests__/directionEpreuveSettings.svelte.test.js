/**
 * Réglages d'une épreuve : la suisse sautée est annoncée avant le premier lancement, et le
 * filtre des Joueurs se voit et s'efface d'un clic.
 */
/* global $state -- rune compilée par le greffon Svelte dans un fichier .svelte.test.js */
import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { namedConfigs } from '../stores/directionStore.js';

import DirectionSettings from '../components/direction/DirectionSettings.svelte';
import PlayersView from '../components/direction/PlayersView.svelte';

afterEach(cleanup);

const q = (/** @type {Element} */ c, /** @type {string} */ id) => c.querySelector(`[data-testid="${id}"]`);

describe('préréglage suisse_tableau', () => {
    test('à 8 joueurs, 2 vies et une bascule à 16, la suisse serait sautée : alerte', async () => {
        const config = $state(namedConfigs[0].build('Open'));
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft', entrantCount: 8 } });
        expect(q(container, 'direction-settings-swiss-skipped-1')).toBeTruthy();
    });

    test('à 16 joueurs, pas d alerte ; une bascule à 0 désactive aussi l alerte', async () => {
        const config = $state(namedConfigs[0].build('Open'));
        const { container } = render(DirectionSettings, { props: { config, directionState: 'draft', entrantCount: 16 } });
        expect(q(container, 'direction-settings-swiss-skipped-1')).toBeNull();
        await fireEvent.input(/** @type {Element} */ (q(container, 'direction-settings-target-1')), { target: { value: '40' } });
        await tick();
        expect(q(container, 'direction-settings-swiss-skipped-1')).toBeTruthy();
        await fireEvent.input(/** @type {Element} */ (q(container, 'direction-settings-target-1')), { target: { value: '0' } });
        await tick();
        expect(q(container, 'direction-settings-swiss-skipped-1')).toBeNull();
    });
});

describe('filtre des Joueurs', () => {
    const rows = [
        { id: 'a', name: 'Ana Roux', club: 'Lyon', rating: 5, state: 'free', wins: 0, losses: 0, lives: 2, byes: 0, opponents: [] },
        { id: 'b', name: 'Bea Sol', club: 'Nice', rating: 4, state: 'free', wins: 0, losses: 0, lives: 2, byes: 0, opponents: [] }
    ];

    test('un filtre actif montre une puce, que × efface', async () => {
        const { container } = render(PlayersView, { props: { rows } });
        expect(q(container, 'direction-player-filter-chip')).toBeNull();
        await fireEvent.input(/** @type {Element} */ (q(container, 'direction-player-filter')), { target: { value: 'ana' } });
        const chip = /** @type {Element} */ (q(container, 'direction-player-filter-chip'));
        expect(chip.textContent).toContain('ana');
        await fireEvent.click(chip);
        expect(q(container, 'direction-player-filter-chip')).toBeNull();
        expect(/** @type {HTMLInputElement} */ (q(container, 'direction-player-filter')).value).toBe('');
    });
});
