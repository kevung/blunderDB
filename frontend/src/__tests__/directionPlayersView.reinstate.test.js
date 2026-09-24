/**
 * Réinscrire un joueur retiré (#439, D5.6).
 *
 * Le seul chemin de retour d'un retiré était un effet de bord : corriger sa fiche le remettait
 * « en jeu », sans le dire. Le retour est désormais un geste nommé, sur la ligne du retiré, et
 * lui seul.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import PlayersView from '../components/direction/PlayersView.svelte';

afterEach(cleanup);

const ROWS = [
    { id: 'ha', name: 'Hugo Andrieu', club: 'Lyon', rating: 5, state: 'withdrawn', wins: 1, losses: 1, lives: 1, byes: 0, opponents: [] },
    { id: 'lb', name: 'Léa Bonnet', club: 'Lyon', rating: 4, state: 'free', wins: 0, losses: 0, lives: 2, byes: 0, opponents: [] }
];

describe('réinscrire un retiré', () => {
    test('le bouton est sur la ligne du retiré, et sur elle seule', () => {
        const { container } = render(PlayersView, { props: { rows: ROWS, started: true } });
        const buttons = container.querySelectorAll('[data-testid="direction-player-reinstate"]');
        expect(buttons).toHaveLength(1);
        expect(buttons[0].closest('tr')?.textContent).toContain('Hugo Andrieu');
        // Il dit ce que le joueur récupère avant qu'on clique.
        expect(buttons[0].getAttribute('title')).toBeTruthy();
    });

    test('un clic réinscrit CE joueur', async () => {
        const onReinstate = vi.fn();
        const { container } = render(PlayersView, { props: { rows: ROWS, started: true, onReinstate } });
        await fireEvent.click(/** @type {Element} */ (container.querySelector('[data-testid="direction-player-reinstate"]')));
        expect(onReinstate).toHaveBeenCalledWith('ha');
    });
});
