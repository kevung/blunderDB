/**
 * TranscriptionPanel.creation.test.js — T1.2 : créer un brouillon, l'ouvrir.
 *
 * Ce qui est vérifié, c'est fonctionnel.md §1.1 et §6 flux 1-2 : la LONGUEUR est
 * le seul champ exigé, elle est proposée à celle du dernier brouillon (sinon 7),
 * `0` est une partie d'argent et c'est ce qui déplie Jacoby et le beaver — les
 * deux règles de session, jamais des Actions (ADR-0028, ADR-0044). Puis le
 * premier coup : ses deux dés sont le jet d'ouverture, le plus fort joue.
 *
 * Les allers-retours Wails sont simulés ; les vrais stores pilotent le composant.
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
// Le classement des candidats est l'affaire de T1.3 et de son propre fichier ;
// ici il ne doit qu'exister, pour que la saisie du premier coup ne bute pas
// dessus.
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn().mockResolvedValue([]),
    EvaluatePositionImmediate: vi.fn().mockResolvedValue({ moves: [] })
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));

import { ListTranscriptions, CreateTranscription, OpenTranscription, ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, transcriptionKeyStore, clearTranscription, transcriptionPromptStore } from '../stores/transcriptionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

// Un document annoté minimal, tel que le moteur Go le renvoie : ce que le
// panneau lit, et rien de plus (il ne dérive ni score, ni Crawford, ni trait).
/** @param {{expects?: string, side?: number, length?: number, actions?: any[], cursor?: number, score?: number[], entry?: any}} [options] */
function annotated({ expects = 'checker', side = 0, length = 7, actions = [], cursor = 0, score = [0, 0], entry = undefined } = {}) {
    return {
        document: { header: { match_length: length, player1: '', player2: '' }, actions, cursor },
        actions: [],
        games: [],
        next: { expects, game_start: actions.length === 0, side, position: { board: { points: [], bearoff: [0, 0] }, dice: [0, 0] }, crawford: false },
        entry,
        score,
        cursor
    };
}

function stateFor(/** @type {any} */ ann) {
    return { id: 1, annotated: ann };
}

beforeEach(() => {
    vi.clearAllMocks();
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (CreateTranscription).mockResolvedValue(stateFor(annotated()));
    /** @type {any} */ (OpenTranscription).mockResolvedValue(stateFor(annotated()));
    /** @type {any} */ (ApplyTranscriptionGesture).mockResolvedValue(stateFor(annotated()));
    transcriptionListStore.set([]);
    clearTranscription();
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('le formulaire de création', () => {
    test('propose 7 quand la bibliothèque ne tient aucun brouillon', async () => {
        render(TranscriptionPanel);
        await fireEvent.click(await screen.findByText('New transcription'));
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Length')).value).toBe('7');
    });

    // « défaut : celle du dernier brouillon, sinon 7 » — la liste arrive triée
    // du plus récemment modifié au plus ancien.
    test('propose la longueur du dernier brouillon', async () => {
        /** @type {any} */ (ListTranscriptions).mockResolvedValue([{ id: 2, match_length: 11, action_count: 3 }]);
        render(TranscriptionPanel);
        await screen.findByText('11 pts');
        await fireEvent.click(await screen.findByText('New transcription'));
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Length')).value).toBe('11');
    });

    test('une longueur non renseignée bloque la création, et rien d’autre ne la bloque', async () => {
        render(TranscriptionPanel);
        await fireEvent.click(await screen.findByText('New transcription'));
        const input = screen.getByLabelText('Length');
        const create = /** @type {HTMLButtonElement} */ (screen.getByText('Create'));

        await fireEvent.input(input, { target: { value: '' } });
        expect(create.disabled).toBe(true);

        await fireEvent.input(input, { target: { value: 'sept' } });
        expect(create.disabled).toBe(true);

        // Aucun autre champ n'est demandé : la longueur seule suffit.
        await fireEvent.input(input, { target: { value: '5' } });
        expect(create.disabled).toBe(false);
    });

    test('Jacoby et le beaver n’apparaissent qu’en partie d’argent', async () => {
        render(TranscriptionPanel);
        await fireEvent.click(await screen.findByText('New transcription'));
        const input = screen.getByLabelText('Length');

        expect(screen.queryByText('Jacoby')).toBeNull();
        expect(screen.queryByText('Beaver')).toBeNull();

        await fireEvent.input(input, { target: { value: '0' } });
        expect(screen.getByText('Jacoby')).toBeTruthy();
        expect(screen.getByText('Beaver')).toBeTruthy();

        await fireEvent.input(input, { target: { value: '7' } });
        expect(screen.queryByText('Jacoby')).toBeNull();
    });

    test('crée le brouillon avec la longueur donnée et l’ouvre', async () => {
        render(TranscriptionPanel);
        await fireEvent.click(await screen.findByText('New transcription'));
        await fireEvent.input(screen.getByLabelText('Length'), { target: { value: '5' } });
        await fireEvent.click(screen.getByText('Create'));
        await tick();

        expect(CreateTranscription).toHaveBeenCalledWith({ match_length: 5, jacoby: false, beaver: false });
        // L'Action attendue habite la barre d'état (ADR-0048 décision 2, qui
        // applique enfin ux.md §5) ; ce test monte le panneau seul et lit donc
        // le magasin que la barre lit.
        await vi.waitFor(() => expect(get(transcriptionPromptStore)?.key).toBe('transcription.firstPlayPrompt'));
    });

    test('une partie d’argent porte les règles de session cochées', async () => {
        render(TranscriptionPanel);
        await fireEvent.click(await screen.findByText('New transcription'));
        await fireEvent.input(screen.getByLabelText('Length'), { target: { value: '0' } });
        await fireEvent.click(screen.getByText('Create'));
        await tick();

        expect(CreateTranscription).toHaveBeenCalledWith({ match_length: 0, jacoby: true, beaver: false });
    });
});

describe('le premier coup d’une partie', () => {
    async function openedPanel() {
        // Un jet sans coup serait une danse, que le panneau enregistre seul.
        /** @type {any} */ (LegalMoves).mockResolvedValue([{ notation: '13/8 13/11' }]);
        /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [{ index: 0, move: '13/8 13/11', equity: 0.1 }] });
        transcriptionStore.set(stateFor(annotated()));
        const rendered = render(TranscriptionPanel);
        await tick();
        document.getElementById('transcriptionPanel')?.focus();
        return rendered;
    }

    function press(/** @type {string} */ code) {
        const digit = /^Digit([1-9])$/.exec(code);
        return fireEvent.keyDown(document, { code, key: digit ? digit[1] : code });
    }

    test('le second dé ne valide rien : le coup reste à choisir', async () => {
        await openedPanel();
        await press('Digit6');
        await press('Digit3');
        await tick();

        expect(/** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ c) => c[1])).toEqual([
            { Kind: 'enter_die', Die: 6 },
            { Kind: 'enter_die', Die: 3 }
        ]);
    });

    test('un double se saisit tel quel : aucune relance n’est transcrite', async () => {
        await openedPanel();
        await press('Digit4');
        await press('Digit4');
        await tick();

        expect(get(transcriptionKeyStore).dice).toEqual([4, 4]);
        expect(/** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ c) => c[1].Kind)).not.toContain('validate');
    });

    test('le plus fort des deux dés a le trait', async () => {
        // Le moteur donne le coup au camp du dé le plus fort dès le second dé.
        /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation((/** @type {any} */ _id, /** @type {any} */ gesture) =>
            Promise.resolve(
                stateFor(
                    gesture.Die === 5
                        ? annotated({ entry: { at: 0, replacing: false, side: 1, dice: [2, 5], selected: false, kind: 'checker', game_start: true } })
                        : annotated({ entry: { at: 0, replacing: false, side: 0, dice: [2, 0], selected: false, kind: 'checker', game_start: true } })
                )
            )
        );
        // Le jet du gagnant a des coups : sans cela ce serait une danse, et la
        // machine repartirait à zéro (ce que couvre TranscriptionPanel.turn).
        /** @type {any} */ (LegalMoves).mockResolvedValue([{ notation: '13/8 13/11' }]);
        /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [{ index: 0, move: '13/8 13/11', equity: 0.1 }] });
        await openedPanel();
        await press('Digit2');
        await press('Digit5');
        await tick();
        await tick();

        // Le jet reste tel que tapé : l'ordre a dit qui commence.
        expect(get(transcriptionKeyStore).dice).toEqual([2, 5]);
        await vi.waitFor(() => {
            const prompt = get(transcriptionPromptStore);
            expect(prompt?.key).toBe('transcription.rollPrompt');
            expect(prompt?.params?.player).toBe('Player 2');
        });
    });
});

// La barre nomme la sortie, jamais le salut du brouillon (ADR-0048 décision
// 12) : un seul « Terminer », et à côté ce qu'il fera. Le moteur pose le Match
// d'origine dans l'EN-TÊTE du document (`header.match_id`).
describe('la barre du brouillon nomme ses sorties', () => {
    test('un brouillon sans match propose Terminer et Abandonner, pour un nouveau match', async () => {
        transcriptionStore.set(stateFor(annotated({ expects: 'checker' })));
        render(TranscriptionPanel);
        expect(await screen.findByText('Finish')).toBeTruthy();
        expect(screen.getByText('Abandon')).toBeTruthy();
        expect(screen.getByText('new match')).toBeTruthy();
        expect(screen.queryByText('Create the match')).toBeNull();
    });

    test('un brouillon ouvert depuis un match dit qu’il le remplacera', async () => {
        const ann = /** @type {any} */ (annotated({ expects: 'checker' }));
        ann.document.header = { ...ann.document.header, match_id: 12 };
        transcriptionStore.set(stateFor(ann));
        render(TranscriptionPanel);
        expect(await screen.findByText('edits match #12')).toBeTruthy();
        expect(screen.getByText('Finish')).toBeTruthy();
    });
});
