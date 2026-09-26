/**
 * Inscrire une paire (ADR-0056 §4) : deux personnes, chacune avec son club et sa cote ; la cote
 * d'entrée proposée est leur moyenne, et une valeur saisie la remplace.
 */

import { test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import PlayersView from '../components/direction/PlayersView.svelte';

afterEach(cleanup);

/** @param {Element} el @param {string} v */
async function type(el, v) {
    await fireEvent.input(el, { target: { value: v } });
}

test('une paire part en deux personnes, cote moyenne par défaut', async () => {
    const onAddPair = vi.fn();
    const onAdd = vi.fn();
    const { container } = render(PlayersView, { props: { rows: [], onAddPair, onAdd } });
    await fireEvent.click(/** @type {Element} */ (container.querySelector('[data-testid="direction-player-pair"]')));
    const inputs = container.querySelectorAll('[data-testid="direction-player-entry"] input[type="text"]');
    await type(inputs[0], 'Ana Roux');
    await type(inputs[1], 'BC Ourcq');
    await type(inputs[2], '4');
    await type(/** @type {Element} */ (container.querySelector('[data-testid="direction-player-name2"]')), 'Bea Sol');
    await type(inputs[5], '6');
    const pairRating = /** @type {HTMLInputElement} */ (container.querySelector('[data-testid="direction-player-pair-rating"]'));
    expect(pairRating.placeholder).toBe('5.00');
    await fireEvent.submit(/** @type {Element} */ (container.querySelector('[data-testid="direction-player-entry"]')));
    expect(onAdd).not.toHaveBeenCalled();
    expect(onAddPair).toHaveBeenCalledWith(
        [
            { name: 'Ana Roux', club: 'BC Ourcq', rating: 4 },
            { name: 'Bea Sol', club: '', rating: 6 }
        ],
        0
    );
});
