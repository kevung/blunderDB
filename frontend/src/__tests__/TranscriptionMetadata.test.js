/**
 * TranscriptionMetadata.test.js — T3.1 : le volet des métadonnées.
 *
 * Ce qui est tenu ici : les champs de `fonctionnel.md` §1.1 marqués « plus
 * tard » sont saisissables à tout moment, aucun n'est exigé, et ce qui part au
 * moteur est UN geste — `set_header` pour la partie descriptive de l'en-tête,
 * `swap_players` pour l'inversion. Ni la longueur, ni les règles de session, ni
 * le `match_id` ne voyagent dans ce geste : ils ont le leur, et un formulaire
 * qui les emporterait transformerait un match en partie d'argent ou ferait
 * classer un second Match au prochain enregistrement.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllPlayerNames: vi.fn(),
    GetAllTournaments: vi.fn()
}));

import { GetAllPlayerNames, GetAllTournaments } from '../../wailsjs/go/database/Database.js';
import TranscriptionMetadata from '../components/TranscriptionMetadata.svelte';

const TOURNAMENTS = [
    { id: 3, name: 'Open de Paris' },
    { id: 4, name: 'Championnat de Lyon' }
];

function headerOf(extra = {}) {
    return {
        match_length: 7,
        jacoby: false,
        beaver: false,
        player1: 'Alice',
        player2: 'Bob',
        event: '',
        location: '',
        round: '',
        date: '2026-09-07T00:00:00Z',
        transcriber: 'Kévin',
        match_id: 12,
        ...extra
    };
}

/** Monte le volet et attend que les deux listes soient chargées. */
async function mount(header = headerOf()) {
    const apply = vi.fn();
    render(TranscriptionMetadata, { props: { header, apply } });
    await tick();
    await tick();
    return apply;
}

beforeEach(() => {
    vi.clearAllMocks();
    GetAllPlayerNames.mockResolvedValue(['Alice', 'Bob', 'Charlie']);
    GetAllTournaments.mockResolvedValue(TOURNAMENTS);
});

afterEach(cleanup);

describe('le volet des métadonnées', () => {
    test('montre l’en-tête du brouillon', async () => {
        await mount(headerOf({ event: 'Open de Paris', location: 'Paris', round: '1/4' }));

        expect(screen.getByLabelText('Player 1').value).toBe('Alice');
        expect(screen.getByLabelText('Player 2').value).toBe('Bob');
        expect(screen.getByLabelText('Event').value).toBe('Open de Paris');
        expect(screen.getByLabelText('Location').value).toBe('Paris');
        expect(screen.getByLabelText('Round').value).toBe('1/4');
        // Go sérialise un instant RFC 3339 ; le champ montre le jour.
        expect(screen.getByLabelText('Date').value).toBe('2026-09-07');
        expect(screen.getByLabelText('Transcriber').value).toBe('Kévin');
    });

    test('une date vide reste vide plutôt que de montrer l’an 1', async () => {
        await mount(headerOf({ date: '0001-01-01T00:00:00Z' }));
        expect(screen.getByLabelText('Date').value).toBe('');
    });

    test('écrit l’en-tête descriptif, et rien d’autre', async () => {
        const apply = await mount();

        await fireEvent.input(screen.getByLabelText('Event'), { target: { value: 'Open de Paris' } });
        await fireEvent.change(screen.getByLabelText('Event'));

        expect(apply).toHaveBeenCalledTimes(1);
        const gesture = apply.mock.calls[0][0];
        expect(gesture.Kind).toBe('set_header');
        expect(gesture.Header.event).toBe('Open de Paris');
        expect(gesture.Header.player1).toBe('Alice');
        expect(gesture.Header.date).toBe('2026-09-07T00:00:00Z');
        // La longueur, les règles et le match id ont leurs propres gestes.
        expect(gesture.Header.match_length).toBeUndefined();
        expect(gesture.Header.jacoby).toBeUndefined();
        expect(gesture.Header.match_id).toBeUndefined();
    });

    test('rien n’est exigé : des noms vides s’écrivent comme le reste', async () => {
        const apply = await mount(headerOf({ player1: '', player2: '', transcriber: '' }));

        await fireEvent.input(screen.getByLabelText('Location'), { target: { value: 'Paris' } });
        await fireEvent.change(screen.getByLabelText('Location'));

        const gesture = apply.mock.calls[0][0];
        expect(gesture.Header.player1).toBe('');
        expect(gesture.Header.player2).toBe('');
        expect(gesture.Header.location).toBe('Paris');
    });

    test('les joueurs s’autocomplètent depuis la base, et un nom absent reste tel quel', async () => {
        const apply = await mount(headerOf({ player1: '' }));
        const field = screen.getByLabelText('Player 1');

        // Un nom déjà présent est proposé…
        await fireEvent.focus(field);
        await fireEvent.input(field, { target: { value: 'Char' } });
        await tick();
        const option = screen.getByText('Charlie');
        await fireEvent.mouseDown(option);
        await tick();
        expect(apply.mock.calls[0][0].Header.player1).toBe('Charlie');

        // … et un nom que la base ne connaît pas est gardé tel qu'il a été tapé.
        await fireEvent.input(field, { target: { value: 'Zoé Martin' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.player1).toBe('Zoé Martin');
    });

    test('le tournoi voyage par son identifiant, un nom inconnu n’en pose aucun', async () => {
        const apply = await mount(headerOf({ tournament_id: 3 }));
        const field = screen.getByLabelText('Tournament');
        expect(field.value).toBe('Open de Paris');

        await fireEvent.input(field, { target: { value: 'Championnat de Lyon' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.tournament_id).toBe(4);

        await fireEvent.input(field, { target: { value: 'Un tournoi qui n’existe pas' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.tournament_id).toBeUndefined();

        // Vider le champ détache : c'est ce que l'enregistrement lira.
        await fireEvent.input(field, { target: { value: '' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.tournament_id).toBeUndefined();
    });

    test('inverser les joueurs est un geste du moteur, pas un échange de champs', async () => {
        const apply = await mount();
        await fireEvent.click(screen.getByText('Swap the players'));
        expect(apply).toHaveBeenCalledWith({ Kind: 'swap_players' });
    });
});
