/**
 * Le classement de saison d'un Événement (ADR-0061) : le barème saisi part tel quel, les lignes
 * s'affichent, l'Elo n'a de colonne que demandé.
 */

import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../stores/rencontreStore.js', () => ({
    seasonRanking: vi.fn(),
    seasonCSV: vi.fn(async () => 'rank;name\n')
}));

import SeasonRanking from '../components/direction/SeasonRanking.svelte';
import { seasonRanking } from '../stores/rencontreStore.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const flush = async () => {
    await tick();
    await Promise.resolve();
    await tick();
};

describe('le classement de saison', () => {
    test('le barème et l’Elo partent avec la Rencontre, les lignes s’affichent', async () => {
        vi.mocked(seasonRanking).mockResolvedValue(
            /** @type {any} */ ({
                events: [],
                elo: true,
                points: [10, 6],
                participation: 0,
                rows: [
                    { rank: 1, name: 'Alice', club: 'Lyon', total: 20, played: 2, elo: 1512.3 },
                    { rank: 2, name: 'Bob', total: 12, played: 2, elo: 1487.7 }
                ]
            })
        );
        const { getByTestId, getByText } = render(SeasonRanking, { props: { rencontreId: 4 } });
        await fireEvent.input(getByTestId('season-points'), { target: { value: '10, 6,' } });
        await fireEvent.click(getByTestId('season-elo'));
        await fireEvent.click(getByTestId('season-compute'));
        await flush();
        expect(seasonRanking).toHaveBeenCalledWith({ rencontreId: 4, points: [10, 6], elo: true });
        expect(getByTestId('season-table')).toBeTruthy();
        expect(getByText('Alice (Lyon)')).toBeTruthy();
        expect(getByText('1512.3')).toBeTruthy();
    });

    test('sans tournoi clos, le panneau le dit', async () => {
        vi.mocked(seasonRanking).mockResolvedValue(/** @type {any} */ ({ events: [], rows: [], elo: false }));
        const { getByTestId } = render(SeasonRanking, { props: { rencontreId: 1 } });
        await fireEvent.click(getByTestId('season-compute'));
        await flush();
        expect(getByTestId('season-empty')).toBeTruthy();
    });
});
