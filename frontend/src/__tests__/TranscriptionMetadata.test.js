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
    GetAllTournaments: vi.fn(),
    GetMatchTextValues: vi.fn()
}));

import { GetAllPlayerNames, GetAllTournaments, GetMatchTextValues } from '../../wailsjs/go/database/Database.js';
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
    const { apply } = await mountRaw(header);
    return apply;
}

/** Le même montage, mais qui rend aussi de quoi changer l'en-tête reçu. */
async function mountRaw(header = headerOf()) {
    const apply = /** @type {any} */ (vi.fn());
    const { rerender } = render(TranscriptionMetadata, { props: { header, apply } });
    await tick();
    await tick();
    return { apply, rerender };
}

beforeEach(() => {
    vi.clearAllMocks();
    // La forme que rend la liaison Go : `database.PlayerFrequency`, un nom et
    // son nombre de matchs — jamais une chaîne nue.
    /** @type {any} */ (GetAllPlayerNames).mockResolvedValue([
        { Name: 'Alice', Count: 3 },
        { Name: 'Bob', Count: 2 },
        { Name: 'Charlie', Count: 1 }
    ]);
    /** @type {any} */ (GetAllTournaments).mockResolvedValue(TOURNAMENTS);
    // Des chaînes déjà classées : le Go rend les noms, pas leurs effectifs.
    /** @type {any} */ (GetMatchTextValues).mockImplementation(async (field) => (field === 'event' ? ['Open de Paris', 'Masters de Lyon'] : ['Paris', 'Lyon']));
});

afterEach(cleanup);

describe('le volet des métadonnées', () => {
    test('montre l’en-tête du brouillon', async () => {
        await mount(headerOf({ event: 'Open de Paris', location: 'Paris', round: '1/4' }));

        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Player 1')).value).toBe('Alice');
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Player 2')).value).toBe('Bob');
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Event')).value).toBe('Open de Paris');
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Location')).value).toBe('Paris');
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Round')).value).toBe('1/4');
        // Go sérialise un instant RFC 3339 ; le champ montre le jour.
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Date')).value).toBe('2026-09-07');
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Transcriber')).value).toBe('Kévin');
    });

    test('une date vide reste vide plutôt que de montrer l’an 1', async () => {
        await mount(headerOf({ date: '0001-01-01T00:00:00Z' }));
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Date')).value).toBe('');
    });

    test('écrit l’en-tête descriptif, et rien d’autre', async () => {
        const apply = await mount();

        const field = screen.getByLabelText('Event');
        await fireEvent.input(field, { target: { value: 'Open de Paris' } });
        await fireEvent.keyDown(field, { key: 'Enter' });

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

        const field = screen.getByLabelText('Location');
        await fireEvent.input(field, { target: { value: 'Paris' } });
        await fireEvent.keyDown(field, { key: 'Enter' });

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

    test.each([
        ['Event', 'event', 'Mast', 'Masters de Lyon', 'Tournoi du club'],
        ['Location', 'location', 'Ly', 'Lyon', 'Bordeaux'],
        ['Transcriber', 'transcriber', 'Char', 'Charlie', 'Zoé Martin']
    ])('%s propose les valeurs de la base, en choisir une ne remplit que ce champ', async (labelText, key, typed, proposed, free) => {
        const header = headerOf({ event: 'E0', location: 'L0', transcriber: 'T0' });
        const apply = await mount(header);
        const field = screen.getByLabelText(labelText);

        await fireEvent.focus(field);
        await fireEvent.input(field, { target: { value: typed } });
        await tick();
        await fireEvent.mouseDown(screen.getByText(proposed));
        await tick();
        const picked = apply.mock.calls.at(-1)[0];
        expect(picked.Kind).toBe('set_header');
        expect(picked.Header[key]).toBe(proposed);
        // Aucune inférence : les deux autres champs gardent leur valeur.
        for (const other of ['event', 'location', 'transcriber'].filter((k) => k !== key)) {
            expect(picked.Header[other]).toBe(header[other]);
        }

        // Un texte absent de la liste reste tel que tapé.
        await fireEvent.input(field, { target: { value: free } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header[key]).toBe(free);
    });

    test('perdre le focus écrit le texte libre, sans geste de plus', async () => {
        const apply = await mount();
        const field = screen.getByLabelText('Event');
        await fireEvent.focus(field);
        await fireEvent.input(field, { target: { value: 'Tournoi du club' } });
        await fireEvent.blur(field);
        await new Promise((resolve) => setTimeout(resolve, 200));
        expect(apply).toHaveBeenCalledTimes(1);
        expect(apply.mock.calls[0][0].Header.event).toBe('Tournoi du club');
    });

    test('quitter un champ intact n’écrit rien', async () => {
        const apply = await mount();
        for (const name of [/Événement|Event/i, /Lieu|Location/i, /Transcri/i]) {
            await fireEvent.blur(screen.getByLabelText(name));
        }
        await new Promise((resolve) => setTimeout(resolve, 200));
        expect(apply).not.toHaveBeenCalled();
    });

    test('choisir une proposition puis quitter le champ n’écrit qu’une fois', async () => {
        const apply = await mount();
        const field = screen.getByLabelText(/Lieu|Location/i);
        await fireEvent.focus(field);
        await fireEvent.input(field, { target: { value: 'Pa' } });
        await fireEvent.mouseDown(await screen.findByText('Paris'));
        await fireEvent.blur(field);
        await new Promise((resolve) => setTimeout(resolve, 200));
        expect(apply).toHaveBeenCalledTimes(1);
        expect(apply.mock.calls[0][0].Header.location).toBe('Paris');
        expect(apply.mock.calls[0][0].Header.event).toBe('');
    });

    test('le tournoi voyage par son identifiant, un nom inconnu ne détache pas', async () => {
        const apply = await mount(headerOf({ tournament_id: 3 }));
        const field = /** @type {HTMLInputElement} */ (screen.getByLabelText('Tournament'));
        expect(field.value).toBe('Open de Paris');

        await fireEvent.input(field, { target: { value: 'Championnat de Lyon' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.tournament_id).toBe(4);

        // Un nom que la bibliothèque ne connaît pas ne détache pas en silence :
        // le tournoi du brouillon reste, et le volet le dit.
        await fireEvent.input(field, { target: { value: 'Un tournoi qui n’existe pas' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.tournament_id).toBe(3);
        expect(screen.getByRole('alert').textContent).toMatch(/Un tournoi qui n’existe pas/);

        // Vider le champ détache : c'est ce que l'enregistrement lira.
        await fireEvent.input(field, { target: { value: '' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.tournament_id).toBeUndefined();
        expect(screen.queryByRole('alert')).toBeNull();
    });

    test('un nom de tournoi inconnu sur un brouillon sans tournoi n’en pose aucun, et le dit', async () => {
        const apply = await mount();
        const field = /** @type {HTMLInputElement} */ (screen.getByLabelText('Tournament'));
        await fireEvent.input(field, { target: { value: 'Inconnu' } });
        await fireEvent.keyDown(field, { key: 'Enter' });
        await tick();
        expect(apply.mock.calls.at(-1)[0].Header.tournament_id).toBeUndefined();
        expect(screen.getByRole('alert').textContent).toMatch(/Inconnu/);
    });

    test('inverser les joueurs est un geste du moteur, pas un échange de champs', async () => {
        const apply = await mount();
        await fireEvent.click(screen.getByText('Swap the players'));
        expect(apply).toHaveBeenCalledWith({ Kind: 'swap_players' });
    });
});

// T3.2 — la longueur se change en cours de transcription. Le volet n'en dérive
// rien : il envoie le geste, et le moteur rejoue le document entier.
describe('le changement de longueur', () => {
    test('montre la longueur du brouillon et l’envoie changée', async () => {
        const apply = await mount(headerOf({ match_length: 5 }));
        const field = /** @type {HTMLInputElement} */ (screen.getByLabelText('Length'));
        expect(field.value).toBe('5');

        await fireEvent.input(field, { target: { value: '7' } });
        await fireEvent.change(field);

        expect(apply).toHaveBeenCalledWith({
            Kind: 'set_length',
            HasLength: true,
            MatchLength: 7,
            HasRules: true,
            Jacoby: false,
            Beaver: false
        });
    });

    test('une longueur illisible ne part pas et le champ revient à ce qu’il était', async () => {
        const apply = await mount(headerOf({ match_length: 5 }));
        const field = /** @type {HTMLInputElement} */ (screen.getByLabelText('Length'));

        await fireEvent.input(field, { target: { value: 'sept' } });
        await fireEvent.change(field);
        expect(apply).not.toHaveBeenCalled();
        expect(field.value).toBe('5');

        // La même longueur n'est pas un changement : rien ne part.
        await fireEvent.input(field, { target: { value: '5' } });
        await fireEvent.change(field);
        expect(apply).not.toHaveBeenCalled();
    });

    test('les règles de session n’apparaissent qu’en argent', async () => {
        const { rerender } = await mountRaw(headerOf({ match_length: 7 }));
        expect(screen.queryByText('Jacoby')).toBeNull();
        expect(screen.queryByText('Beaver')).toBeNull();

        await rerender({ header: headerOf({ match_length: 0, jacoby: true, beaver: false }) });
        expect(screen.getByText('Jacoby')).toBeTruthy();
        expect(screen.getByText('Beaver')).toBeTruthy();
        expect(/** @type {HTMLInputElement} */ (screen.getByLabelText('Length')).value).toBe('0');
    });

    test('cocher le beaver garde la longueur et l’autre règle', async () => {
        const apply = await mount(headerOf({ match_length: 0, jacoby: true, beaver: false }));
        await fireEvent.click(screen.getByLabelText('Beaver'));

        expect(apply).toHaveBeenCalledWith({
            Kind: 'set_length',
            HasLength: true,
            MatchLength: 0,
            HasRules: true,
            Jacoby: true,
            Beaver: true
        });
    });
});
