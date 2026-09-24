/**
 * « Corriger » dans l'Historique de la direction (#436, D5.3).
 *
 * Le bouton ne faisait que basculer sur la page Direction : le match étant fini, il n'était
 * plus sur la grille, et aucune fiche ne s'ouvrait. Corriger un résultat ancien n'avait aucun
 * chemin. Il ouvre désormais, sur la ligne choisie, le même panneau que la dernière décision.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import HistoryView from '../components/direction/HistoryView.svelte';

afterEach(cleanup);

const ENTRIES = [
    { seq: 3, kind: 'result', matchId: 'M1', a: 'ha', b: 'lb', aName: 'Hugo Andrieu', bName: 'Léa Bonnet', winner: 'ha', winnerName: 'Hugo Andrieu', correctable: true },
    { seq: 5, kind: 'result', matchId: 'M2', a: 'mc', b: 'nd', aName: 'Marc Colin', bName: 'Nadia Dubois', winner: 'mc', winnerName: 'Marc Colin', correctable: true }
];

describe("corriger un résultat depuis l'historique", () => {
    test('le panneau s’ouvre sur la ligne choisie, et sur elle seule', async () => {
        const { container } = render(HistoryView, { props: { entries: ENTRIES } });
        expect(container.querySelector('.correct')).toBeNull();

        const line = /** @type {Element} */ (container.querySelector('[data-testid="direction-history-5"]'));
        await fireEvent.click(/** @type {Element} */ (line.querySelector('[data-testid="direction-history-correct"]')));

        const panels = container.querySelectorAll('.correct');
        expect(panels).toHaveLength(1);
        const winners = [...panels[0].querySelectorAll('.winner')].map((b) => b.textContent?.trim());
        expect(winners).toEqual(['Marc Colin', 'Nadia Dubois']);
    });

    test('choisir le vainqueur corrige CE match, avec le score saisi', async () => {
        const onCorrect = vi.fn();
        const { container } = render(HistoryView, { props: { entries: ENTRIES, onCorrect } });
        const line = /** @type {Element} */ (container.querySelector('[data-testid="direction-history-3"]'));
        await fireEvent.click(/** @type {Element} */ (line.querySelector('[data-testid="direction-history-correct"]')));

        const inputs = container.querySelectorAll('.correct input');
        await fireEvent.input(inputs[0], { target: { value: '3' } });
        await fireEvent.input(inputs[1], { target: { value: '7' } });
        await fireEvent.click(/** @type {Element} */ (container.querySelector('.correct .winner:nth-of-type(2)')));

        expect(onCorrect).toHaveBeenCalledWith('M1', 'lb', 3, 7);
        // Le panneau se referme : la correction est faite, la ligne suivante le dira.
        expect(container.querySelector('.correct')).toBeNull();
    });

    test('un second clic sur « Corriger » referme le panneau', async () => {
        const { container } = render(HistoryView, { props: { entries: ENTRIES } });
        const button = /** @type {Element} */ (container.querySelector('[data-testid="direction-history-3"] [data-testid="direction-history-correct"]'));
        await fireEvent.click(button);
        expect(container.querySelector('.correct')).not.toBeNull();
        await fireEvent.click(button);
        expect(container.querySelector('.correct')).toBeNull();
    });
});
