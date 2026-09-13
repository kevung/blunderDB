/**
 * Un score dont un côté vaut zéro (issue #408).
 *
 * Le backend encode les scores avec `omitempty` : un 7–0 arrive SANS `scoreB`. L'arbre
 * additionnait donc `7 + undefined`, obtenait NaN, et n'affichait aucun score ; la vue des
 * emplacements écrivait « 7– ». Ces tests montent les deux vues avec le JSON tel qu'il arrive.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import BracketsView from '../components/direction/BracketsView.svelte';
import SlotsView from '../components/direction/SlotsView.svelte';

afterEach(cleanup);

describe('un score blanchi garde son zéro', () => {
    test('dans l’arbre', () => {
        const phases = [
            {
                index: 1,
                kind: 'bracket',
                current: true,
                drawn: true,
                sections: [
                    {
                        name: 'main',
                        kind: 'bracket',
                        rounds: 1,
                        matches: [
                            { key: 'f', label: { kind: 'final' }, round: 0, length: 7, a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', matchId: 'M1', winner: 'a', scoreA: 7, done: true, running: false }
                        ]
                    }
                ]
            }
        ];
        const { container } = render(BracketsView, { props: { phases } });
        expect(container.querySelector('.outcome')?.textContent).toBe('7–0');
    });

    test('dans la liste des emplacements', () => {
        const slots = [
            { slotId: 's1', label: { kind: 'final' }, phase: 1, a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', length: 7, winner: 'a', winnerName: 'Alice', scoreA: 7, done: true, matchId: 3 }
        ];
        const { container } = render(SlotsView, { props: { slots } });
        const row = /** @type {HTMLElement} */ (container.querySelector('tbody tr'));
        expect(row.textContent).toContain('7–0');
    });
});
