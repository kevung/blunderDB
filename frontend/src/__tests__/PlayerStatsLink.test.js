/** A player's name in a match header opens that player's statistics. */
import { describe, test, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../services/positionLoader.js', () => ({ openStatsFor: vi.fn(() => Promise.resolve()) }));

import { openStatsFor } from '../services/positionLoader.js';
import MatchDetailHeader from '../components/MatchDetailHeader.svelte';

describe('MatchDetailHeader', () => {
    test("each player's name opens their Stats", async () => {
        const match = { player1_name: 'Alice', player2_name: 'Bob', match_length: 7 };
        const { getAllByTestId } = render(MatchDetailHeader, { props: { match, games: [], onreview: () => {}, onexport: () => {}, onedit: () => {}, ondelete: () => {} } });
        const links = getAllByTestId('player-stats-link');
        expect(links.map((l) => l.textContent)).toEqual(['Alice', 'Bob']);
        await fireEvent.click(links[1]);
        expect(openStatsFor).toHaveBeenCalledWith({ playerName: 'Bob' });
    });
});
