/**
 * MatchPanel.transcriptionDrafts.test.js
 *
 * T1.8: a draft being transcribed is written to its row after every Action so
 * that a crash costs nothing but the dice half typed — and the other half of
 * that promise is being FOUND again. The Match panel is where a user looks for
 * a match, so the drafts that are not matches yet get one line each there
 * (integration.md §4), and clicking one brings the tab they are typed in
 * forward.
 *
 * What this pins is the pair: a line per draft the library reports, none at
 * all when it reports none.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { tick } from 'svelte';

const drafts = vi.hoisted(() => ({ rows: [] }));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllMatches: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve(drafts.rows)),
    DeleteMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import { openPanels, PANEL, activeTabStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { transcriptionListStore } from '../stores/transcriptionStore.js';
import { ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';

async function settle() {
    await vi.waitFor(() => expect(ListTranscriptions).toHaveBeenCalled());
    await new Promise((resolve) => setTimeout(resolve, 0));
    for (let i = 0; i < 4; i++) await tick();
}

describe('MatchPanel — the drafts being transcribed', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        drafts.rows = [];
        transcriptionListStore.set([]);
        activeTabStore.set('match');
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
    });
    afterEach(cleanup);

    test('one line per draft, naming it and opening the Transcription tab', async () => {
        drafts.rows = [
            { id: 3, label: 'Kévin vs Alice', player1: 'Kévin', player2: 'Alice', match_length: 7, action_count: 12, match_id: 0 },
            { id: 4, label: '', player1: '', player2: '', match_length: 0, action_count: 2, match_id: 0 }
        ];

        const { container } = render(MatchPanel);
        await settle();

        const lines = () => [...container.querySelectorAll('.draft-line')];
        await vi.waitFor(() => expect(lines()).toHaveLength(2));
        expect(lines()[0].textContent).toContain('Kévin vs Alice');
        // A draft that has not said who is playing still gets its line.
        expect(lines()[1].textContent.trim()).not.toBe('');

        await fireEvent.click(lines()[0]);
        expect(get(activeTabStore)).toBe('transcription');
    });

    test('no band at all when the library has no draft', async () => {
        const { container } = render(MatchPanel);
        await settle();

        expect(get(transcriptionListStore)).toEqual([]);
        expect(container.querySelector('.draft-band')).toBeNull();
        expect(container.querySelectorAll('.draft-line')).toHaveLength(0);
    });
});
