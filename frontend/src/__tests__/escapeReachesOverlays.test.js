/**
 * escapeReachesOverlays.test.js — #414
 *
 * Ce qui est ouvert et a quelque chose à fermer (menu contextuel, fiche de résultat, reprise
 * de la direction) se ferme sur un appui d'Échap, avec le répartiteur global actif — monté
 * comme App.svelte le monte : `window.addEventListener('keydown', handleKeyDown)` dans
 * `onMount`, donc AVANT tout composant ouvert plus tard (un menu, la page Direction).
 *
 * Chaque test enregistre le répartiteur d'abord, puis monte le composant : c'est l'ordre de
 * l'application, celui que les tests de #410 inversaient sans le savoir. Dans cet ordre, le
 * `stopPropagation()` du répartiteur rendait muet tout `<svelte:window onkeydown>` : Svelte 5
 * n'appelle pas un gestionnaire déclaratif quand `event.cancelBubble` est vrai.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../services/importService.js', () => ({
    importDatabase: vi.fn(),
    importPosition: vi.fn(),
    importFolder: vi.fn(),
    pastePosition: vi.fn()
}));

vi.mock('../services/positionService.js', async (importOriginal) => ({
    ...(await importOriginal()),
    leaveSubSearchResults: vi.fn()
}));

const { viewStoreMock } = vi.hoisted(() => ({ viewStoreMock: /** @type {any} */ ({}) }));
vi.mock('../stores/viewStore', async () => {
    const { writable } = await import('svelte/store');
    Object.assign(viewStoreMock, {
        views: writable([]),
        activeViewId: writable(1),
        switchTo: vi.fn(),
        addView: vi.fn(),
        closeView: vi.fn(),
        renameView: vi.fn(),
        selectPreviousView: vi.fn(),
        selectNextView: vi.fn()
    });
    return { viewStore: viewStoreMock };
});

import { handleKeyDown } from '../services/keyboardService.js';
import { leaveSubSearchResults } from '../services/positionService.js';
import ContextMenu from '../components/ContextMenu.svelte';
import LastDecision from '../components/direction/LastDecision.svelte';
import ResultCard from '../components/direction/ResultCard.svelte';
import ProposalList from '../components/direction/ProposalList.svelte';
import ViewTabs from '../components/ViewTabs.svelte';
import ModalFixture from './fixtures/ModalFixture.svelte';

/** @param {Element | null} from @param {string} key @param {KeyboardEventInit} [extra] */
async function press(from, key, extra = {}) {
    const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...extra });
    (from ?? document.body).dispatchEvent(event);
    await tick();
    return event;
}

/** @param {() => void} onClose */
function menu(onClose) {
    return render(ContextMenu, { props: { x: 0, y: 0, items: [{ label: 'Masquer', onClick: vi.fn() }], onClose } });
}

const LAST = { matchId: 7, a: 'ha', b: 'lb', aName: 'Hugo Andrieu', bName: 'Léa Bonnet', winner: 'ha', winnerName: 'Hugo Andrieu', correctable: true, cancellable: false };
const CELL = { table: 1, matchId: 7, a: 'ha', b: 'lb', aName: 'Hugo Andrieu', bName: 'Léa Bonnet' };
const PROPOSALS = [
    { kind: 'start_match', phase: 0, a: 'ha', b: 'lb', length: 7 },
    { kind: 'start_match', phase: 0, a: 'mc', b: 'nd', length: 7 }
];

/** Ce que voit un écouteur posé sur window après le répartiteur : ce qui a traversé. */
const later = vi.fn();

beforeEach(() => {
    // L'ordre de l'application : le répartiteur est là avant tout ce qui s'ouvre ensuite.
    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keydown', later);
    viewStoreMock.views.set([
        { id: 1, name: '#1' },
        { id: 2, name: '#2' }
    ]);
});

afterEach(() => {
    window.removeEventListener('keydown', handleKeyDown);
    window.removeEventListener('keydown', later);
    cleanup();
    document.body.innerHTML = '';
    vi.clearAllMocks();
});

describe('Échap ferme ce qui est ouvert, répartiteur global actif', () => {
    test('menu contextuel : un appui le ferme, et rien d’autre ne le voit', async () => {
        const onClose = vi.fn();
        menu(onClose);
        await tick();
        const item = document.querySelector('.context-menu-item');
        expect(document.activeElement).toBe(item);
        await press(item, 'Escape');
        expect(onClose).toHaveBeenCalledTimes(1);
        expect(leaveSubSearchResults).not.toHaveBeenCalled();
        expect(later).not.toHaveBeenCalled();
    });

    test('reprise de la direction : un appui la ferme', async () => {
        const { container } = render(LastDecision, { props: { last: LAST } });
        await fireEvent.click(/** @type {Element} */ (container.querySelector('.last button')));
        expect(container.querySelector('.correct')).not.toBeNull();
        await press(document.body, 'Escape');
        expect(container.querySelector('.correct')).toBeNull();
        expect(leaveSubSearchResults).not.toHaveBeenCalled();
        // Fermée, elle ne réclame plus rien : l'Échap suivant va au répartiteur.
        await press(document.body, 'Escape');
        expect(leaveSubSearchResults).toHaveBeenCalledTimes(1);
    });

    test('fiche de résultat : un appui la ferme, même quand le focus l’a quittée', async () => {
        const onClose = vi.fn();
        render(ResultCard, { props: { cell: CELL, onClose } });
        await tick();
        await press(document.body, 'Escape');
        expect(onClose).toHaveBeenCalledTimes(1);
        expect(later).not.toHaveBeenCalled();
    });

    test('renommage d’une vue : un appui l’abandonne', async () => {
        const { container } = render(ViewTabs);
        await fireEvent.dblClick(/** @type {Element} */ (container.querySelector('[role="tab"]')));
        const input = /** @type {HTMLInputElement} */ (container.querySelector('.rename-input'));
        expect(input).not.toBeNull();
        input.focus();
        await press(input, 'Escape');
        expect(container.querySelector('.rename-input')).toBeNull();
        expect(viewStoreMock.renameView).not.toHaveBeenCalled();
    });

    test('sans rien d’ouvert, l’Échap arrive au répartiteur', async () => {
        await press(document.body, 'Escape');
        expect(leaveSubSearchResults).toHaveBeenCalledTimes(1);
    });
});

describe('l’ordre de priorité', () => {
    test('une surcouche ouverte passe avant le palier d’un panneau qui écoute document', async () => {
        // Comme MatchPanel, TournamentPanel, CollectionPanel : un écouteur document qui consomme Échap.
        const panelTier = vi.fn((/** @type {KeyboardEvent} */ e) => e.key === 'Escape' && e.stopPropagation());
        document.addEventListener('keydown', panelTier);
        try {
            const onClose = vi.fn();
            menu(onClose);
            await tick();
            await press(document.querySelector('.context-menu-item'), 'Escape');
            expect(onClose).toHaveBeenCalledTimes(1);
            expect(panelTier).not.toHaveBeenCalled();
        } finally {
            document.removeEventListener('keydown', panelTier);
        }
    });

    test('la dernière ouverte se ferme la première', async () => {
        const first = vi.fn();
        const second = vi.fn();
        menu(first);
        const top = menu(second);
        await tick();
        await press(document.body, 'Escape');
        expect(second).toHaveBeenCalledTimes(1);
        expect(first).not.toHaveBeenCalled();
        top.unmount();
        await press(document.body, 'Escape');
        expect(first).toHaveBeenCalledTimes(1);
    });

    test('une modale ouverte par-dessus garde son Échap', async () => {
        const menuClose = vi.fn();
        const modalClose = vi.fn();
        menu(menuClose);
        const { container } = render(ModalFixture, { props: { open: true, onclose: modalClose, onkeydown: undefined } });
        await tick();
        await press(container.querySelector('#first'), 'Escape');
        expect(modalClose).toHaveBeenCalledTimes(1);
        expect(menuClose).not.toHaveBeenCalled();
    });
});

describe('un écouteur `<svelte:window>` monté après le répartiteur reçoit ses touches', () => {
    test('CTRL-Z ouvre la reprise de la direction', async () => {
        const { container } = render(LastDecision, { props: { last: LAST } });
        await press(document.body, 'z', { ctrlKey: true });
        expect(container.querySelector('.correct')).not.toBeNull();
    });

    test('CTRL-2 passe à la deuxième vue', async () => {
        render(ViewTabs);
        await press(document.body, 'é', { code: 'Digit2', ctrlKey: true });
        expect(viewStoreMock.switchTo).toHaveBeenCalledWith(2);
    });

    test('la file des propositions, qui les reçoit désormais, ignore les combinaisons', async () => {
        const onConfirm = vi.fn();
        const { container } = render(ProposalList, { props: { proposals: PROPOSALS, onConfirm } });
        await tick();
        const selected = () => [...container.querySelectorAll('.queue li')].findIndex((li) => li.classList.contains('selected'));
        await press(document.body, 'j', { ctrlKey: true });
        await press(document.body, 'Enter', { ctrlKey: true });
        expect(selected()).toBe(0);
        expect(onConfirm).not.toHaveBeenCalled();
    });
});
