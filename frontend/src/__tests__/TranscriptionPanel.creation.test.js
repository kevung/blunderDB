/**
 * TranscriptionPanel.creation.test.js — T1.2 : créer un brouillon, l'ouvrir.
 *
 * Ce qui est vérifié, c'est fonctionnel.md §1.1 et §6 flux 1-2 : la LONGUEUR est
 * le seul champ exigé, elle est proposée à celle du dernier brouillon (sinon 7),
 * `0` est une partie d'argent et c'est ce qui déplie Jacoby et le beaver — les
 * deux règles de session, jamais des Actions (ADR-0028, ADR-0044). Puis
 * l'ouverture : deux dés, le plus fort commence, une égalité relance.
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

import { ListTranscriptions, CreateTranscription, OpenTranscription, ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, transcriptionKeyStore, clearTranscription } from '../stores/transcriptionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

// Un document annoté minimal, tel que le moteur Go le renvoie : ce que le
// panneau lit, et rien de plus (il ne dérive ni score, ni Crawford, ni trait).
function annotated({ expects = 'opening', side = 0, length = 7, actions = [], cursor = 0, score = [0, 0] } = {}) {
    return {
        document: { header: { match_length: length, player1: '', player2: '' }, actions, cursor },
        actions: [],
        games: [],
        next: { expects, side, position: { board: { points: [], bearoff: [0, 0] }, dice: [0, 0] }, crawford: false },
        score,
        cursor
    };
}

function stateFor(ann) {
    return { id: 1, annotated: ann };
}

beforeEach(() => {
    vi.clearAllMocks();
    ListTranscriptions.mockResolvedValue([]);
    CreateTranscription.mockResolvedValue(stateFor(annotated()));
    OpenTranscription.mockResolvedValue(stateFor(annotated()));
    ApplyTranscriptionGesture.mockResolvedValue(stateFor(annotated()));
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
        expect(screen.getByLabelText('Length').value).toBe('7');
    });

    // « défaut : celle du dernier brouillon, sinon 7 » — la liste arrive triée
    // du plus récemment modifié au plus ancien.
    test('propose la longueur du dernier brouillon', async () => {
        ListTranscriptions.mockResolvedValue([{ id: 2, match_length: 11, action_count: 3 }]);
        render(TranscriptionPanel);
        await screen.findByText('11 pts');
        await fireEvent.click(await screen.findByText('New transcription'));
        expect(screen.getByLabelText('Length').value).toBe('11');
    });

    test('une longueur non renseignée bloque la création, et rien d’autre ne la bloque', async () => {
        render(TranscriptionPanel);
        await fireEvent.click(await screen.findByText('New transcription'));
        const input = screen.getByLabelText('Length');
        const create = screen.getByText('Create');

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
        await screen.findByText('Opening dice');
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

describe("l'ouverture", () => {
    async function openedPanel() {
        transcriptionStore.set(stateFor(annotated()));
        const rendered = render(TranscriptionPanel);
        await tick();
        document.getElementById('transcriptionPanel')?.focus();
        return rendered;
    }

    function press(code) {
        const digit = /^Digit([1-9])$/.exec(code);
        return fireEvent.keyDown(document, { code, key: digit ? digit[1] : code });
    }

    test('deux dés suffisent : le second valide, sans troisième touche', async () => {
        await openedPanel();
        await press('Digit6');
        await press('Digit3');
        await tick();

        expect(ApplyTranscriptionGesture.mock.calls.map((c) => c[1])).toEqual([{ Kind: 'enter_die', Die: 6 }, { Kind: 'enter_die', Die: 3 }, { Kind: 'validate' }]);
    });

    test('une égalité affiche « relance » et n’attend rien d’autre', async () => {
        // Le moteur répond qu'une ouverture est de nouveau attendue.
        ApplyTranscriptionGesture.mockResolvedValue(stateFor(annotated({ expects: 'opening' })));
        await openedPanel();
        await press('Digit4');
        await press('Digit4');
        await tick();

        expect(get(transcriptionKeyStore).tie).toBe(true);
        expect(await screen.findByText('Tie: roll again.')).toBeTruthy();
    });

    test('le gagnant a le trait, avec les deux dés de l’ouverture', async () => {
        // Le moteur ne change ce qu'il attend qu'une fois l'ouverture validée :
        // saisir un dé ne décide de rien.
        ApplyTranscriptionGesture.mockImplementation((_id, gesture) => Promise.resolve(stateFor(gesture.Kind === 'validate' ? annotated({ expects: 'checker', side: 1 }) : annotated())));
        await openedPanel();
        await press('Digit2');
        await press('Digit5');
        await tick();
        await tick();

        // Le jet reste affiché, plus fort d'abord : il n'est pas ressaisi.
        expect(get(transcriptionKeyStore).dice).toEqual([5, 2]);
        expect(await screen.findByText("Player 2's dice")).toBeTruthy();
    });
});
