import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SuggestReferencePositions: vi.fn(),
    GetAllTournaments: vi.fn(async () => [])
}));
vi.mock('../services/recurringStudy.js', () => ({ collectionFromIds: vi.fn(async () => true), deckFromIds: vi.fn(), quizOnIds: vi.fn() }));
vi.mock('../services/positionLoader.js', () => ({ loadPositionsFromSelection: vi.fn() }));

import ReferenceSuggest from '../components/ReferenceSuggest.svelte';
import { SuggestReferencePositions } from '../../wailsjs/go/database/Database.js';
import { collectionFromIds, quizOnIds } from '../services/recurringStudy.js';
import { statsFilterStore } from '../stores/statsStore.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const ref = (id, extra = {}) => ({
    PositionID: id,
    GameType: 'holding',
    Kind: 'checker',
    Theme: 'blots',
    Gain: 0.031,
    Covered: 0.031,
    Errors: 4,
    Matches: 3,
    Excess: 0.012,
    FamilyErrors: 9,
    GapMP: 80,
    Close: false,
    Unstable: false,
    RolledOut: false,
    ...extra
});

const proposal = {
    NumDecisions: 500,
    ThresholdMP: 50,
    Radius: 12,
    Size: 20,
    Candidates: 30,
    Handled: 2,
    References: [ref(7), ref(9, { Kind: 'cube', Theme: 'offer_missed', AwayOnRoll: 3, AwayOpponent: 5, RolledOut: true, GapMP: 120 })]
};

describe('ReferenceSuggest', () => {
    test('asks the engine for the Statistics player over the whole library, then words each reason', async () => {
        statsFilterStore.set({ playerName: 'Alice', tournamentIDs: [4], dateFrom: '2025-01-01', dateTo: '', decisionType: 0, matchLength: [] });
        vi.mocked(SuggestReferencePositions).mockResolvedValue(proposal);
        render(ReferenceSuggest);
        await fireEvent.click(screen.getByTestId('suggest-run'));
        await waitFor(() => expect(screen.getAllByTestId('suggest-row')).toHaveLength(2));
        const [filter, matchIDs, size] = vi.mocked(SuggestReferencePositions).mock.calls[0];
        expect(filter.playerName).toBe('Alice');
        expect(filter.tournamentIDs).toEqual([]);
        expect(filter.dateFrom).toBe('');
        expect(matchIDs).toEqual([]);
        expect(size).toBe(20);
        const rows = screen.getAllByTestId('suggest-row');
        expect(rows[0].textContent).toContain('3.10 %');
        expect(rows[0].textContent).toMatch(/4/);
        expect(rows[1].textContent).toContain('3a-5a');
    });

    test('keeps only the checked positions, as a collection or a quiz', async () => {
        vi.mocked(SuggestReferencePositions).mockResolvedValue(proposal);
        vi.mocked(collectionFromIds).mockResolvedValue(true);
        const onCollectionCreated = vi.fn();
        render(ReferenceSuggest, { props: { onCollectionCreated } });
        await fireEvent.click(screen.getByTestId('suggest-run'));
        await waitFor(() => expect(screen.getAllByTestId('suggest-row')).toHaveLength(2));
        await fireEvent.click(screen.getAllByTestId('suggest-row')[0].querySelector('input[type=checkbox]'));
        await fireEvent.click(screen.getByTestId('suggest-quiz'));
        expect(quizOnIds).toHaveBeenCalledWith([9]);
        await fireEvent.click(screen.getByTestId('suggest-collection'));
        await waitFor(() => expect(collectionFromIds).toHaveBeenCalled());
        expect(vi.mocked(collectionFromIds).mock.calls[0][1]).toEqual([9]);
        await waitFor(() => expect(onCollectionCreated).toHaveBeenCalled());
    });

    test('says so when nothing is left to propose', async () => {
        vi.mocked(SuggestReferencePositions).mockResolvedValue({ ...proposal, References: [] });
        render(ReferenceSuggest);
        await fireEvent.click(screen.getByTestId('suggest-run'));
        await waitFor(() => expect(screen.getByTestId('reference-suggest').textContent).toMatch(/30/));
        expect(screen.queryAllByTestId('suggest-row')).toHaveLength(0);
    });
});
