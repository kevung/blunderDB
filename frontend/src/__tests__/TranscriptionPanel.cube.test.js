/**
 * TranscriptionPanel.cube.test.js — T1.4 : le videau et la fin de partie dans
 * le panneau.
 *
 * Les transitions du clavier sont tenues à part (transcriptionKeys.cube.test.js).
 * Ce qui est vérifié ici est ce que seul le panneau fait : traduire la commande
 * en geste Go SANS lui poser de camp — le moteur sait qui double et qui répond —
 * et AFFICHER ce que le Replay a dérivé, le videau, le score, la mention
 * Crawford, la fin du match et l'Incohérence de videau impossible.
 *
 * Le panneau ne dérive rien de tout cela : il est un client du moteur
 * (ADR-0045 règle 9). Un test qui le prendrait à recalculer un score serait un
 * test qui passe pour une mauvaise raison ; ceux-ci lui donnent des documents
 * annotés tout faits et regardent l'écran.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn()
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
import { transcriptionListStore, transcriptionStore, clearTranscription, transcriptionPromptStore, transcriptionInfoStore } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const board = () => ({ points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] });

function position({ cube = { owner: -1, value: 0 }, decision = 0 } = {}) {
    return { board: board(), cube, dice: [0, 0], score: [7, 7], player_on_roll: 0, decision_type: decision };
}

/**
 * Un document annoté comme le moteur le rend. Tout ce que le panneau affiche en
 * sort : rien n'est recalculé côté Svelte.
 */
function annotated({ expects = 'checker', side = 0, cube, crawford = false, score = [0, 0], finished = false, winner = -1, game = 1, flags = [] } = {}) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: flags.length ? [{ index: 0, inconsistencies: flags.map((kind) => ({ kind, detail: 'engine prose' })) }] : [],
        games: [],
        next: { expects, side, position: position({ cube, decision: expects === 'take' ? 1 : 0 }), crawford, game_number: game },
        score,
        finished,
        winner,
        cursor: 0
    };
}

const state = (ann) => ({ id: 1, annotated: ann });

function press(code) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

async function openedPanel(ann = annotated()) {
    transcriptionStore.set(state(ann));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);

beforeEach(() => {
    vi.clearAllMocks();
    ListTranscriptions.mockResolvedValue([]);
    ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(annotated())));
    LegalMoves.mockResolvedValue([]);
    EvaluatePositionImmediate.mockResolvedValue({ moves: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('le videau', () => {
    test('d envoie un double, sans camp : le moteur sait lequel', async () => {
        await openedPanel();
        await press('KeyD');
        await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: 'double' }));
    });

    test('t et p répondent à l’offre, et seulement à elle', async () => {
        await openedPanel(annotated({ expects: 'take', side: 1, cube: { owner: 1, value: 1 } }));
        await press('KeyT');
        await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: 'take' }));

        cleanup();
        ApplyTranscriptionGesture.mockClear();
        await openedPanel(annotated({ expects: 'take', side: 1, cube: { owner: 1, value: 1 } }));
        await press('KeyP');
        await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: 'pass' }));
    });

    test('une réponse attendue montre les deux touches et cache les dés', async () => {
        await openedPanel(annotated({ expects: 'take', side: 1, cube: { owner: 1, value: 1 } }));
        // La ligne d'instruction « t = take, p = pass » a disparu du panneau
        // (ADR-0048 décision 8) : une instruction permanente est l'aveu qu'un
        // geste ne se devine pas, et sa place est raccourcis.rst et l'aide
        // engendrée. Ce qui reste dit, et qui suffit : l'Action attendue, et la
        // rangée qui n'allume que les deux boutons qui répondent à quelque chose.
        await vi.waitFor(() => {
            const prompt = get(transcriptionPromptStore);
            expect(prompt?.key).toBe('transcription.answerPrompt');
            expect(prompt?.params?.player).toBe('Alice');
        });
        // La valeur affichée est celle OFFERTE : c'est ce que pèse le preneur.
        expect(get(transcriptionInfoStore)?.cubeKey).toBe('transcription.doubleOffered');
        expect(get(transcriptionInfoStore)?.cubeParams?.v).toBe(2);
        expect(document.querySelectorAll('.die')).toHaveLength(0);
    });

    test('le videau centré, puis possédé, est dit tel quel', async () => {
        await openedPanel();
        await vi.waitFor(() => expect(get(transcriptionInfoStore)?.cubeKey).toBe('transcription.cubeCentred'));

        cleanup();
        await openedPanel(annotated({ cube: { owner: 0, value: 2 } }));
        await vi.waitFor(() => {
            const info = get(transcriptionInfoStore);
            expect(info?.cubeKey).toBe('transcription.cubeOwned');
            expect(info?.cubeParams).toEqual({ v: 4, player: 'Kévin' });
        });
    });
});

describe('la fin de partie et la fin de match', () => {
    test('le score et la partie suivante viennent du Replay', async () => {
        await openedPanel(annotated({ expects: 'opening', score: [3, 2], game: 4 }));
        await vi.waitFor(() => {
            const info = get(transcriptionInfoStore);
            expect(info?.score).toEqual([3, 2]);
            expect(info?.gameNumber).toBe(4);
        });
    });

    test('la mention Crawford est celle que le moteur a dérivée', async () => {
        await openedPanel(annotated({ expects: 'opening', score: [6, 2], crawford: true, game: 7 }));
        await vi.waitFor(() => expect(get(transcriptionInfoStore)?.crawford).toBe(true));
    });

    test('le match terminé le dit, avec son vainqueur', async () => {
        await openedPanel(annotated({ expects: 'opening', score: [7, 4], finished: true, winner: 0 }));
        await vi.waitFor(() => {
            const prompt = get(transcriptionPromptStore);
            expect(prompt?.key).toBe('transcription.matchOver');
            expect(prompt?.params).toEqual({ player: 'Kévin', a: 7, b: 4 });
        });
    });
});

describe('les incohérences sont montrées, jamais refusées', () => {
    test('un videau impossible est nommé sous les dés', async () => {
        ApplyTranscriptionGesture.mockResolvedValue(state(annotated({ flags: ['impossible_cube'] })));
        await openedPanel();
        await press('KeyD');
        // L'Action existe : le geste est parti, et la phrase la commente.
        await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: 'double' }));
        expect(await screen.findByText(/impossible cube action/)).toBeTruthy();
    });

    test('la phrase est celle du panneau, pas la prose anglaise du moteur', async () => {
        ApplyTranscriptionGesture.mockResolvedValue(state(annotated({ flags: ['past_end'] })));
        await openedPanel();
        await press('KeyD');
        expect(await screen.findByText(/past the end of the match/)).toBeTruthy();
        expect(screen.queryByText(/engine prose/)).toBeNull();
    });
});
