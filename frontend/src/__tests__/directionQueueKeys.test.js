/**
 * directionQueueKeys.test.js — #415
 *
 * Tant que la page Direction est affichée, J / K / ↓ / ↑ / ENTRÉE vont à la file des
 * propositions ; le panneau Tournois ne les reprend que lorsque le focus est dans le panneau.
 *
 * Monté dans l'ordre de l'application, celui que #414 a appris à respecter : le répartiteur
 * global (posé sur window dans `onMount` d'App.svelte) et le panneau Tournois (écouteur
 * `document`) sont là AVANT la page Direction, qui s'ouvre plus tard. Dans cet ordre, sur main,
 * le panneau arrêtait toute touche nue, et la file ne recevait rien.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllTournaments: vi.fn().mockResolvedValue([]),
    GetTournamentMatches: vi.fn().mockResolvedValue([]),
    GetAllMatches: vi.fn().mockResolvedValue([]),
    ListDirections: vi.fn().mockResolvedValue([])
}));

vi.mock('../services/importService.js', () => ({
    importDatabase: vi.fn(),
    importPosition: vi.fn(),
    importFolder: vi.fn(),
    pastePosition: vi.fn()
}));

vi.mock('../services/positionService.js', async (importOriginal) => ({
    ...(await importOriginal()),
    nextPosition: vi.fn(),
    previousPosition: vi.fn(),
    leaveSubSearchResults: vi.fn()
}));

import { GetAllTournaments } from '../../wailsjs/go/database/Database.js';
import { handleKeyDown } from '../services/keyboardService.js';
import { nextPosition, previousPosition } from '../services/positionService.js';
import TournamentPanel from '../components/TournamentPanel.svelte';
import ProposalList from '../components/direction/ProposalList.svelte';
import ContextMenu from '../components/ContextMenu.svelte';
import { openPanels, PANEL, activeTabStore } from '../stores/uiStore.js';
import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore } from '../stores/tournamentStore.js';
import { openDirectionIdStore } from '../stores/directionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

const TOURNAMENTS = [
    { id: 1, name: 'Open de Lyon', matchCount: 0, date: '2026-09-12', location: 'Lyon', pr: 0, mwc_loss: 0 },
    { id: 2, name: 'Open de Paris', matchCount: 0, date: '2026-10-12', location: 'Paris', pr: 0, mwc_loss: 0 }
];

const PROPOSALS = [
    { kind: 'start_match', phase: 0, a: 'ha', b: 'lb', length: 7 },
    { kind: 'start_match', phase: 0, a: 'mc', b: 'nd', length: 7 },
    { kind: 'start_match', phase: 0, a: 'oe', b: 'pf', length: 7 }
];

/** @param {Element | null} from @param {string} key @param {KeyboardEventInit} [extra] */
async function press(from, key, extra = {}) {
    const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...extra });
    (from ?? document.body).dispatchEvent(event);
    await tick();
    return event;
}

/** L'index de la proposition choisie dans la file. */
function queueIndex() {
    return [...document.querySelectorAll('.proposals .queue li')].findIndex((li) => li.classList.contains('selected'));
}

const onConfirm = vi.fn();

/** Le panneau Tournois ouvert sur l'onglet Tournois, la liste chargée. */
async function openTournamentPanel() {
    openPanels.set(new Set([PANEL.TOURNAMENT]));
    activeTabStore.set('tournaments');
    render(TournamentPanel, { props: {} });
    await screen.findByText('Open de Lyon');
}

/** La page Direction affichée, sa file des propositions devant soi. */
async function showDirectionQueue() {
    openDirectionIdStore.set(1);
    render(ProposalList, { props: { proposals: PROPOSALS, onConfirm } });
    await tick();
}

/** Au-delà de la prise de focus différée du panneau (100 ms). */
const pastFocusTimer = () => new Promise((resolve) => setTimeout(resolve, 150));

beforeEach(() => {
    // L'ordre de l'application : le répartiteur est là avant tout le reste.
    window.addEventListener('keydown', handleKeyDown);
    vi.mocked(GetAllTournaments).mockResolvedValue(/** @type {any} */ (TOURNAMENTS));
    tournamentsStore.set([]);
    selectedTournamentStore.set(null);
    tournamentMatchesStore.set([]);
    openDirectionIdStore.set(null);
    databasePathStore.set('/fake/db.sqlite');
});

afterEach(() => {
    window.removeEventListener('keydown', handleKeyDown);
    cleanup();
    document.body.innerHTML = '';
    openPanels.set(new Set());
    activeTabStore.set('');
    openDirectionIdStore.set(null);
    vi.clearAllMocks();
});

describe('page Direction affichée : la file des propositions a J / K / ENTRÉE', () => {
    test('J descend dans la file et ne change ni le tournoi, ni la position', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        expect(queueIndex()).toBe(0);

        await press(document.body, 'j');

        expect(queueIndex()).toBe(1);
        expect(get(selectedTournamentStore)).toBeNull();
        expect(nextPosition).not.toHaveBeenCalled();
    });

    test('K remonte ; ↓ et ↑ font de même', async () => {
        await openTournamentPanel();
        await showDirectionQueue();

        await press(document.body, 'j');
        await press(document.body, 'j');
        await press(document.body, 'k');
        expect(queueIndex()).toBe(1);
        await press(document.body, 'ArrowDown');
        expect(queueIndex()).toBe(2);
        await press(document.body, 'ArrowUp');
        expect(queueIndex()).toBe(1);

        expect(get(selectedTournamentStore)).toBeNull();
        expect(previousPosition).not.toHaveBeenCalled();
    });

    test('ENTRÉE confirme la proposition choisie', async () => {
        await openTournamentPanel();
        await showDirectionQueue();

        await press(document.body, 'j');
        await press(document.body, 'Enter');

        expect(onConfirm).toHaveBeenCalledTimes(1);
        expect(onConfirm).toHaveBeenCalledWith(PROPOSALS[1]);
    });

    test('ENTRÉE sur un bouton de la page (« Tout lancer ») active le bouton et ne confirme rien', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        const all = /** @type {HTMLElement} */ (document.querySelector('.proposals .all'));
        all.focus();

        const event = await press(all, 'Enter');

        expect(onConfirm).not.toHaveBeenCalled();
        // Rien n'empêche le navigateur d'activer le bouton qui a le focus.
        expect(event.defaultPrevented).toBe(false);
    });

    test('ENTRÉE sur le bouton « Lancer » d’une autre ligne ne confirme pas la proposition choisie', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        const third = /** @type {HTMLElement} */ (document.querySelectorAll('.proposals .queue li .go')[2]);
        third.focus();

        const event = await press(third, 'Enter');

        expect(onConfirm).not.toHaveBeenCalled();
        expect(event.defaultPrevented).toBe(false);
    });

    test('focus sur l’onglet cliqué, J place le focus dans la file, et ENTRÉE confirme la 2ᵉ proposition', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        const tab = document.createElement('button');
        tab.setAttribute('role', 'tab');
        document.body.appendChild(tab);
        tab.focus();

        await press(tab, 'j');
        expect(document.activeElement?.closest('.proposals .queue')).not.toBeNull();
        await press(document.activeElement, 'Enter');

        expect(onConfirm).toHaveBeenCalledTimes(1);
        expect(onConfirm).toHaveBeenCalledWith(PROPOSALS[1]);
    });

    test('la prise de focus différée du panneau ne vole pas les touches de la file', async () => {
        await showDirectionQueue();
        await openTournamentPanel();

        await pastFocusTimer();
        expect(document.activeElement?.id).not.toBe('tournamentPanel');

        await press(document.activeElement, 'j');
        expect(queueIndex()).toBe(1);
        expect(get(selectedTournamentStore)).toBeNull();
    });

    test('dans un champ de saisie, J reste une lettre', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        const input = document.createElement('input');
        document.body.appendChild(input);
        input.focus();

        await press(input, 'j');
        await press(input, 'Enter');

        expect(queueIndex()).toBe(0);
        expect(onConfirm).not.toHaveBeenCalled();
    });

    test('sur un autre onglet de la Direction, sans file, J ne change pas de tournoi', async () => {
        await openTournamentPanel();
        openDirectionIdStore.set(1);
        await tick();

        await press(document.body, 'j');

        expect(get(selectedTournamentStore)).toBeNull();
        expect(nextPosition).not.toHaveBeenCalled();
    });
});

describe('focus dans le panneau Tournois : J / K lui reviennent', () => {
    test('un clic dans le panneau lui rend J / K, et ENTRÉE ne confirme rien', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        const panel = /** @type {HTMLElement} */ (document.getElementById('tournamentPanel'));
        panel.focus();

        await press(panel, 'j');
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));
        await press(document.activeElement, 'Enter');

        expect(queueIndex()).toBe(0);
        expect(onConfirm).not.toHaveBeenCalled();
    });
});

describe('sans page Direction : rien ne change', () => {
    test('Direction fermée, J / K parcourent les tournois', async () => {
        await openTournamentPanel();

        await press(document.body, 'j');
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));
        await press(document.body, 'j');
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 2 }));
        await press(document.body, 'k');
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));
    });

    test('Direction ouverte mais onglet Tournois quitté : la file ne prend rien', async () => {
        await showDirectionQueue();
        activeTabStore.set('matches');
        await tick();

        await press(document.body, 'j');

        expect(queueIndex()).toBe(0);
        expect(nextPosition).toHaveBeenCalledTimes(1);
    });
});

describe('Échap garde sa règle (#414)', () => {
    test('page Direction affichée, Échap descend encore les paliers du panneau', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        await fireEvent.click(/** @type {Element} */ (screen.getByText('Open de Lyon').closest('tr')));
        await vi.waitFor(() => expect(get(selectedTournamentStore)).toMatchObject({ id: 1 }));

        await press(document.body, 'Escape');

        expect(get(selectedTournamentStore)).toBeNull();
    });

    test('une surcouche ouverte garde Échap, et la file ne lui prend pas ses flèches', async () => {
        await openTournamentPanel();
        await showDirectionQueue();
        const onClose = vi.fn();
        render(ContextMenu, { props: { x: 0, y: 0, items: [{ label: 'Masquer', onClick: vi.fn() }], onClose } });
        await tick();

        await press(document.activeElement, 'ArrowDown');
        expect(queueIndex()).toBe(0);
        await press(document.activeElement, 'Escape');
        expect(onClose).toHaveBeenCalledTimes(1);
        expect(get(selectedTournamentStore)).toBeNull();
    });
});
