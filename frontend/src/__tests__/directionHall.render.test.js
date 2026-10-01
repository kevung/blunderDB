/**
 * La Salle : une grille pour toutes les épreuves d'une Rencontre, chaque case marquée de son
 * épreuve, et les propositions groupées par épreuve. Chaque geste part vers l'épreuve de sa
 * case ou de sa proposition — jamais vers l'épreuve ouverte par défaut.
 */

import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

vi.mock('../stores/directionStore', () => ({
    hallEnterResult: vi.fn(async () => null),
    hallEnterForfeit: vi.fn(async () => null),
    hallMoveMatch: vi.fn(async () => null),
    hallCancelMatch: vi.fn(async () => null),
    hallConfirmProposal: vi.fn(async () => null)
}));

import HallView from '../components/direction/HallView.svelte';
import { hallConfirmProposal } from '../stores/directionStore';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const match = (table, tid, event, eventIndex, id, extra = {}) => ({
    table,
    free: false,
    unavailable: false,
    reserved: false,
    matchId: id,
    a: 'a',
    b: 'b',
    aName: 'Alice',
    bName: 'Bob',
    length: 7,
    tournamentId: tid,
    event,
    eventIndex,
    ...extra
});

const hall = {
    rencontreId: 1,
    name: 'Festival',
    events: [
        { tournamentId: 3, name: 'Principal', index: 0, proposals: [], names: {} },
        {
            tournamentId: 4,
            name: 'Speed',
            index: 1,
            proposals: [
                { kind: 'start_match', a: 'c', b: 'd', length: 3, table: 3, key: 'k1' },
                { kind: 'wait', reason: 'matches_running' }
            ],
            names: { c: 'Chloé', d: 'David' }
        }
    ],
    cells: [
        match(1, 3, 'Principal', 0, 'm1'),
        match(2, 4, 'Speed', 1, 'm1'),
        { table: 3, free: true, unavailable: false, reserved: false, eventIndex: -1 },
        match(0, 3, 'Principal', 0, 'm7', { noTable: true }),
        match(0, 4, 'Speed', 1, 'm7', { noTable: true })
    ]
};

describe('la Salle', () => {
    test('une case par table, marquée de son épreuve ; un même id de match dans deux épreuves fait deux cases', () => {
        const act = vi.fn(async (fn) => (await fn(), true));
        const { container, getByTestId, getAllByTestId } = render(HallView, { props: { hall, act } });
        expect(container.querySelectorAll('.cell')).toHaveLength(5);
        expect(getByTestId('direction-table-1').textContent).toContain('Principal');
        expect(getByTestId('direction-table-2').textContent).toContain('Speed');
        expect(getByTestId('direction-table-3').querySelector('[data-testid="hall-event-chip"]')).toBeNull();
        expect(getByTestId('direction-table-none-3-m7').textContent).toContain('Principal');
        expect(getByTestId('direction-table-none-4-m7').textContent).toContain('Speed');
        expect(getAllByTestId('hall-event-chip')).toHaveLength(4);
    });

    test('les propositions sont groupées par épreuve, et « Lancer » vise la bonne', async () => {
        const act = vi.fn(async (fn) => (await fn(), true));
        const { getByTestId, queryByTestId } = render(HallView, { props: { hall, act } });
        expect(queryByTestId('hall-proposals-3')).toBeNull();
        const group = getByTestId('hall-proposals-4');
        expect(group.textContent).toContain('Chloé');
        expect(group.querySelectorAll('li')).toHaveLength(1);
        await fireEvent.click(/** @type {HTMLElement} */ (group.querySelector('button.go')));
        expect(hallConfirmProposal).toHaveBeenCalledWith(4, hall.events[1].proposals[0]);
    });
});

describe('la Salle en erreur', () => {
    test("une erreur de lecture se montre au lieu d'un chargement sans fin", () => {
        const act = vi.fn();
        const { getByTestId, queryByText } = render(HallView, { props: { hall: null, error: 'boom', act } });
        expect(getByTestId('hall-error').textContent).toContain('boom');
        expect(queryByText(/Loading|Chargement/)).toBeNull();
    });

    test('une épreuve qui ne se rejoue pas est nommée, les autres restent', () => {
        const act = vi.fn();
        const broken = { ...hall, events: [{ ...hall.events[0], error: 'no direction' }, hall.events[1]] };
        const { getByTestId } = render(HallView, { props: { hall: broken, act } });
        expect(getByTestId('hall-event-error-3').textContent).toContain('Principal');
        expect(getByTestId('direction-table-2').textContent).toContain('Speed');
    });
});
