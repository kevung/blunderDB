/**
 * La file nomme le qualifié retiré qu'un repêchage remplace. Il n'est plus libre (la file ne
 * reçoit pour l'appariement que les joueurs libres) : son nom vient de la liste complète des
 * inscrits, pas de son identifiant.
 */
import { test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import ProposalList from '../components/direction/ProposalList.svelte';

afterEach(cleanup);

test('un repêchage nomme le retiré par son nom, pas par son identifiant', () => {
    const proposals = [{ kind: 'repechage', phase: 0, a: 'gaelle-tessier', b: 'ines-vautrin', label: { kind: 'repechage', section: 'poule:A', players: 1 } }];
    const { container } = render(ProposalList, {
        props: {
            proposals,
            players: [{ id: 'ines-vautrin', name: 'Inès Vautrin' }],
            entrants: [
                { id: 'ines-vautrin', name: 'Inès Vautrin' },
                { id: 'gaelle-tessier', name: 'Gaëlle Tessier' }
            ]
        }
    });
    const text = /** @type {Element} */ (container.querySelector('ul.queue')).textContent;
    expect(text).toContain('Gaëlle Tessier');
    expect(text).not.toContain('gaelle-tessier');
});
