/**
 * TranscriptionPanel.triangle.test.js — T2.1 : le triangle câblé au panneau.
 *
 * L'équivalence clic/frappe est tenue par la machine (transcriptionKeys.mouse.test.js).
 * Ce qui ne se voit qu'ici : que le clic ATTEIGNE le moteur — deux `enter_die`
 * dans l'ordre, puis la demande des candidats — que le triangle soit posé à
 * côté du clavier et non à sa place, et que l'ouverture montre une rangée de
 * six dés plutôt qu'un triangle, un dé par camp.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    TranscriptionMAT: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(),
    EvaluatePositionImmediate: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));

import { ApplyTranscriptionGesture, ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const POSITION = {
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
};

function annotated({ expects = 'checker' } = {}) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects, side: 0, position: POSITION, crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

const state = (ann) => ({ id: 1, annotated: ann });

const PLAYS = [{ notation: '8/5 6/5', steps: [] }];
const RANKED = [{ index: 0, move: '8/5 6/5', equity: 0.1 }];

const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);
const diceCells = () => [...document.querySelectorAll('.transcription-panel button')].filter((b) => /^\d{1,2}$/.test(b.textContent.trim()));

async function openedPanel(expects = 'checker') {
    transcriptionStore.set(state(annotated({ expects })));
    render(TranscriptionPanel);
    await tick();
}

beforeEach(() => {
    vi.clearAllMocks();
    ListTranscriptions.mockResolvedValue([]);
    ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(annotated())));
    LegalMoves.mockResolvedValue(PLAYS);
    EvaluatePositionImmediate.mockResolvedValue({ moves: RANKED });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('le triangle des jets dans le panneau', () => {
    test('vingt et une cases sont offertes à côté des deux cases du jet', async () => {
        await openedPanel();
        expect(diceCells().filter((b) => b.textContent.trim().length === 2)).toHaveLength(21);
    });

    test('un clic sur la case 31 envoie les deux dés, dans l-ordre, puis demande les candidats', async () => {
        await openedPanel();
        const cell = diceCells().find((b) => b.textContent.trim() === '31');
        await fireEvent.click(cell);

        await vi.waitFor(() =>
            expect(gestures()).toEqual([
                { Kind: 'enter_die', Die: 3 },
                { Kind: 'enter_die', Die: 1 },
                { Kind: 'select_candidate', Candidate: 0 }
            ])
        );
        // `LegalMoves` est appelée aussi pour armer le coup joué au plateau —
        // une fois par jet, T2.3 — donc l'appel visé ici est nommé par son jet
        // et non par son rang dans la liste des appels.
        const [pos] = LegalMoves.mock.calls.find(([p]) => p.dice[0] === 3 && p.dice[1] === 1);
        expect(pos.dice).toEqual([3, 1]);
    });

    // Le clavier finit le tour : sans ce retour du focus, la touche qui suit le
    // clic tomberait sur le bouton et le budget serait faux d'un geste.
    test('le focus revient au panneau après le clic', async () => {
        await openedPanel();
        await fireEvent.click(diceCells().find((b) => b.textContent.trim() === '31'));
        expect(document.activeElement?.id).toBe('transcriptionPanel');
    });

    test('l-ouverture montre six dés, pas le triangle', async () => {
        await openedPanel('opening');
        const faces = diceCells().map((b) => b.textContent.trim());
        expect(faces).toEqual(['1', '2', '3', '4', '5', '6']);

        await fireEvent.click(diceCells()[5]);
        await vi.waitFor(() => expect(gestures()).toEqual([{ Kind: 'enter_die', Die: 6 }]));
    });

    // Devant une réponse au videau il n'y a pas de dé à donner : une cible qui
    // ne répond à rien vaut moins que pas de cible.
    test('aucune cible pendant l-attente d-une réponse au double', async () => {
        await openedPanel('take');
        expect(diceCells()).toHaveLength(0);
    });
});
