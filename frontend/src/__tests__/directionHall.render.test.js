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

/** @param {any} table @param {any} tid @param {any} event @param {any} eventIndex @param {any} id @param {any} [extra] */
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

/** @type {any} */
const hall = {
    rencontreId: 1,
    name: 'Festival',
    events: [
        { tournamentId: 3, name: 'Principal', index: 0, names: { e: 'Emma', f: 'Fanny' } },
        { tournamentId: 4, name: 'Speed', index: 1, names: { c: 'Chloé', d: 'David', g: 'Gaspard', h: 'Hugo' } }
    ],
    queue: [
        { tournamentId: 4, eventIndex: 1, action: { kind: 'start_match', a: 'c', b: 'd', length: 3, table: 3, key: 'k1' } },
        { tournamentId: 3, eventIndex: 0, action: { kind: 'start_match', a: 'e', b: 'f', length: 7, table: 4, key: 'k2' } }
    ],
    held: [{ tournamentId: 4, eventIndex: 1, action: { kind: 'start_match', a: 'g', b: 'h', length: 3, reason: 'player_busy', key: 'k3' } }],
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

    test("une seule file, dans l'ordre reçu, et « Lancer » vise la bonne épreuve", async () => {
        const act = vi.fn(async (fn) => (await fn(), true));
        const { getByTestId } = render(HallView, { props: { hall, act } });
        const rows = getByTestId('hall-queue').querySelectorAll('li');
        expect(rows).toHaveLength(2);
        expect(rows[0].textContent).toContain('Speed');
        expect(rows[0].textContent).toContain('Chloé');
        expect(rows[1].textContent).toContain('Principal');
        await fireEvent.click(/** @type {HTMLElement} */ (rows[0].querySelector('button.go')));
        expect(hallConfirmProposal).toHaveBeenCalledWith(4, hall.queue[0].action);
    });

    test('un match retenu sans table reste dans la file, sans bouton « Lancer »', () => {
        const act = vi.fn();
        const { getByTestId } = render(HallView, { props: { hall, act } });
        const row = getByTestId('hall-held-4');
        expect(row.textContent).toContain('Hugo');
        expect(row.querySelector('button')).toBeNull();
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
