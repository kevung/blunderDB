import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

const contrast = vi.fn();
const load = vi.fn();
vi.mock('../../wailsjs/go/database/Database.js', () => ({ PlayerContrast: (...args) => contrast(...args) }));
vi.mock('../services/positionLoader.js', () => ({ loadPositionsFromSelection: (...args) => load(...args) }));

const { default: PlayerComparison } = await import('../components/stats/PlayerComparison.svelte');

const row = (name) => ({ name, decisions: 100, pr: 5, pr_checker: 5, pr_cube: 5, snowie_er: 5, blunders: 2, matches: 3, wins: 1, losses: 2 });

afterEach(() => {
    cleanup();
    contrast.mockReset();
    load.mockReset();
});

describe('PlayerComparison contrast', () => {
    test('asks the backend only on demand, then opens the positions', async () => {
        contrast.mockResolvedValue({
            player_a: 'Alice',
            player_b: 'Bob',
            threshold_mp: 50,
            common_positions: 4,
            positions: [
                { position_id: 7, well_played: 'b' },
                { position_id: 3, well_played: 'a' }
            ]
        });
        const { container } = render(PlayerComparison, { a: row('Alice'), b: row('Bob') });
        expect(contrast).not.toHaveBeenCalled();
        await fireEvent.click(container.querySelector('[data-testid="player-contrast"] button'));
        await waitFor(() => expect(contrast).toHaveBeenCalledTimes(1));
        expect(contrast.mock.calls[0].slice(0, 2)).toEqual(['Alice', 'Bob']);
        const open = await waitFor(() => container.querySelectorAll('[data-testid="player-contrast"] button')[0]);
        await fireEvent.click(open);
        expect(load).toHaveBeenCalledWith([7, 3]);
    });

    test('says so when no common position contrasts', async () => {
        contrast.mockResolvedValue({ common_positions: 5, positions: [] });
        const { container } = render(PlayerComparison, { a: row('Alice'), b: row('Bob') });
        await fireEvent.click(container.querySelector('[data-testid="player-contrast"] button'));
        await waitFor(() => expect(container.querySelector('[data-testid="player-contrast"] p')).not.toBeNull());
        expect(container.querySelector('[data-testid="player-contrast"] button')).toBeNull();
    });
});
