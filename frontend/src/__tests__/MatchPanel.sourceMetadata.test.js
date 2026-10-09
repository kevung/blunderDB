/**
 * MatchPanel.sourceMetadata.test.js
 *
 * The match sheet shows what the source file said of the match — ratings,
 * transcriber, session rules, the program that wrote it — and nothing for a
 * match whose file said none of it.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() =>
        Promise.resolve([
            {
                id: 7,
                player1_name: 'Alice',
                player2_name: 'Bob',
                match_length: 0,
                match_date: '2026-01-15',
                game_count: 2,
                player1_elo: 1854.7,
                player1_experience: 205,
                player2_elo: 1600,
                transcriber: 'Carol',
                has_jacoby: true,
                has_beaver: false,
                engine_version: 'eXtreme Gammon, file format 30'
            },
            { id: 8, player1_name: 'Dave', player2_name: 'Erin', match_length: 5, match_date: '2026-01-16', game_count: 1 }
        ])
    ),
    CountMatches: vi.fn(() => Promise.resolve(1)),
    GetMatchByID: vi.fn(() => Promise.resolve(null)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    TrashMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    GetMatchMoveGrades: vi.fn(() => Promise.resolve([])),
    GetMatchTimeSummary: vi.fn(() => Promise.resolve(null)),
    GetMatchOrigin: vi.fn(() => Promise.resolve(null)),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import { openPanels, PANEL } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { ListMatches } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';
import { openSection } from './matchSectionHelper.js';

/**
 * The list is loaded twice at mount (onMount, then the visibility effect) and
 * the second load clears the selection when it resolves: wait for both before
 * touching a row.
 */
async function settle() {
    await vi.waitFor(() => expect(ListMatches).toHaveBeenCalledTimes(2));
    await new Promise((resolve) => setTimeout(resolve, 0));
    for (let i = 0; i < 4; i++) await tick();
}

async function select(container, name) {
    const cell = () => [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes(name));
    await vi.waitFor(() => expect(cell()).toBeTruthy());
    await fireEvent.click(cell());
    await vi.waitFor(() => expect(container.querySelector('.detail-pane')).not.toBeNull());
    await openSection(container, 'info');
    for (let i = 0; i < 4; i++) await tick();
    return container.querySelector('.detail-pane').textContent;
}

describe('MatchPanel — the source metadata on the match sheet', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
    });
    afterEach(cleanup);

    test('a match whose file states them shows ratings, transcriber, rules and program', async () => {
        const { container } = render(MatchPanel);
        await settle();
        const text = await select(container, 'Alice');
        expect(text).toContain('1855 (205) – 1600');
        expect(text).toContain('Carol');
        expect(text).toContain('Jacoby');
        expect(text).not.toContain('Beaver');
        expect(text).toContain('eXtreme Gammon, file format 30');
    });

    test('a match whose file says nothing shows no such row', async () => {
        const { container } = render(MatchPanel);
        await settle();
        const text = await select(container, 'Dave');
        expect(text).not.toContain('Jacoby');
        expect(text).not.toContain('eXtreme Gammon');
        expect(text).not.toContain(' – ');
    });
});
