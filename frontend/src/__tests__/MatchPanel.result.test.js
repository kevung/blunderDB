/**
 * MatchPanel.result.test.js
 *
 * The match list shows each match's stored result: the final score, the winner's points in
 * bold and named in the tooltip; an unfinished match is muted; a match without games shows a dash.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() =>
        Promise.resolve([
            { id: 1, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, final_score: [4, 7], winner: -1 },
            { id: 2, player1_name: 'Carol', player2_name: 'Dan', match_length: 5, final_score: [3, 1], winner: 0 },
            { id: 3, player1_name: 'Erin', player2_name: 'Finn', match_length: 3, winner: 0 }
        ])
    ),
    CountMatches: vi.fn(() => Promise.resolve(3)),
    GetMatchByID: vi.fn(() => Promise.resolve(null)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import { databasePathStore } from '../stores/databaseStore.js';
import MatchPanel from '../components/MatchPanel.svelte';

afterEach(() => {
    cleanup();
    databasePathStore.set('');
});

describe('MatchPanel result column', () => {
    test('shows the final score, the winner and an unfinished match', async () => {
        databasePathStore.set('/tmp/x.db');
        const { container } = render(MatchPanel);
        await vi.waitFor(() => expect(container.querySelectorAll('[data-testid="match-result"]').length).toBe(3));
        const cells = [...container.querySelectorAll('[data-testid="match-result"]')];
        expect(cells[0].textContent.trim()).toBe('4–7');
        expect(cells[0].querySelector('.winner')?.textContent).toBe('7');
        expect(cells[0].querySelector('.match-result')?.getAttribute('title')).toContain('Bob');
        expect(cells[1].querySelector('.match-result.unfinished')).not.toBeNull();
        expect(cells[1].querySelector('.winner')).toBeNull();
        expect(cells[2].textContent.trim()).toBe('—');
    });
});
