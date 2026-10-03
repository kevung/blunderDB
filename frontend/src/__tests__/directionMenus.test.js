// @ts-nocheck — données de test partielles : les props typées par Wails exigent des classes générées.
/**
 * Les menus contextuels de la page Direction : le contenu de chaque menu (sans DOM), puis
 * chaque vue montée, ouverte au clic droit et au clavier. Les gestes destructifs gardent leur
 * confirmation : refusée, rien n'est appelé.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { menuRequest } from '../services/contextMenuTrigger.js';
import { get } from 'svelte/store';
import { confirmModalStore, resolveConfirm } from '../services/confirmService.js';
import SlotsView from '../components/direction/SlotsView.svelte';

import { cellMenu, matchMenu, playerMenu, proposalMenu, historyMenu } from '../services/directionMenus.js';
import TableGrid from '../components/direction/TableGrid.svelte';
import HistoryView from '../components/direction/HistoryView.svelte';
import ProposalList from '../components/direction/ProposalList.svelte';
import PlayersView from '../components/direction/PlayersView.svelte';

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

/** @type {(k: string, p?: Record<string, unknown>) => string} */
const t = (k, p) => (p ? `${k}${JSON.stringify(p)}` : k);
const labels = (/** @type {{ label: string }[]} */ items) => items.map((i) => i.label.replace(/^direction\.menu\./, ''));

const RUNNING = { table: 3, free: false, unavailable: false, reserved: false, matchId: 'M1', a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', length: 7 };
const FREE = { table: 4, free: true, unavailable: false, reserved: false };

describe('contenu des menus', async () => {
    test('une case occupée offre résultat, forfaits, changement de table (M / X), annulation', async () => {
        const l = labels(cellMenu(t, RUNNING, { openResult() {}, openMove() {}, onForfeit() {}, onCancel() {} }));
        expect(l).toEqual(['enterResult', 'forfeitOf{"name":"Alice"}', 'forfeitOf{"name":"Bob"}', 'moveTable', 'cancelMatch']);
    });

    test("une case libre n'offre que la mise hors service, une case hors service la remise", () => {
        const h = { onOutOfService: vi.fn() };
        const free = cellMenu(t, FREE, h);
        expect(labels(free)).toEqual(['outOfService']);
        free[0].onClick();
        expect(h.onOutOfService).toHaveBeenCalledWith(4, true);
        const out = cellMenu(t, { ...FREE, unavailable: true }, h);
        expect(labels(out)).toEqual(['backInService']);
        expect(cellMenu(t, { ...FREE, elsewhere: 'B' }, h)).toEqual([]);
    });

    test('un forfait confirmé désigne le bon vainqueur ; refusé, il ne fait rien', async () => {
        const onForfeit = vi.fn();
        const confirm = vi.fn().mockReturnValue(false);
        const items = matchMenu(t, { ...RUNNING, running: true }, { onForfeit, confirm });
        await items[0].onClick();
        expect(onForfeit).not.toHaveBeenCalled();
        confirm.mockReturnValue(true);
        await items[0].onClick();
        expect(onForfeit).toHaveBeenCalledWith('M1', 'b', '');
    });

    test('annuler un match se confirme', async () => {
        const onCancel = vi.fn();
        const confirm = vi.fn().mockReturnValue(false);
        const cancel = matchMenu(t, { ...RUNNING, running: true }, { onCancel, confirm }).find((i) => i.label.includes('cancelMatch'));
        await cancel?.onClick();
        expect(onCancel).not.toHaveBeenCalled();
        confirm.mockReturnValue(true);
        await cancel?.onClick();
        expect(onCancel).toHaveBeenCalledWith('M1');
    });

    test('un joueur : les entrées suivent son état', async () => {
        const h = { onGoTable() {}, onHistory() {}, onAbsent() {}, onReturn() {}, onWithdraw() {}, onReinstate() {}, onEdit() {} };
        const p = { id: 'x', name: 'Xavier' };
        expect(labels(playerMenu(t, { ...p, state: 'playing', table: 2 }, h))).toEqual([
            'enterResult',
            'goToTable{"n":2}',
            'historyOf{"name":"Xavier"}',
            'withdrawNow',
            'withdrawLater',
            'correctPlayer'
        ]);
        expect(labels(playerMenu(t, { ...p, state: 'free' }, h))).toContain('absent');
        expect(labels(playerMenu(t, { ...p, state: 'absent' }, h))).toContain('present');
        const out = labels(playerMenu(t, { ...p, state: 'withdrawn' }, h));
        expect(out).toContain('reinstate');
        expect(out).not.toContain('withdrawNow');
    });

    test('retirer un joueur se confirme', async () => {
        const onWithdraw = vi.fn();
        const confirm = vi.fn().mockReturnValue(false);
        const now = playerMenu(t, { id: 'x', name: 'Xavier', state: 'free' }, { onWithdraw, confirm }).find((i) => i.label.includes('withdrawNow'));
        await now?.onClick();
        expect(onWithdraw).not.toHaveBeenCalled();
        confirm.mockReturnValue(true);
        await now?.onClick();
        expect(onWithdraw).toHaveBeenCalledWith('x', false);
    });

    test('une proposition : lancer, apparier autrement (un appariement seulement), ignorer', async () => {
        const h = { onLaunch() {}, onIgnore() {}, onArrange() {} };
        expect(labels(proposalMenu(t, { kind: 'start_match', a: 'a', b: 'b' }, h))).toEqual(['launch', 'arrangeOther', 'ignore']);
        const full = { onLaunch() {}, onIgnore() {}, onArrange() {}, onLaunchAtTable() {}, onChangeLength() {}, onPrintSheet() {} };
        expect(labels(proposalMenu(t, { kind: 'start_match', a: 'a', b: 'b' }, full))).toEqual(['launch', 'launchAtTable', 'changeLength', 'arrangeOther', 'ignore', 'printSheet']);
        expect(labels(proposalMenu(t, { kind: 'cancel_match' }, h))).toEqual(['launch', 'ignore']);
    });

    test("une ligne d'historique : corriger ou annuler, remarque, filtres", () => {
        const h = { onCorrect() {}, onCancel() {}, onNote() {}, onFilter() {} };
        const e = { aName: 'Alice', bName: 'Bob', matchId: 'M1' };
        expect(labels(historyMenu(t, { ...e, correctable: true }, h))).toEqual(['correctResult', 'addNote', 'filterOn{"name":"Alice"}', 'filterOn{"name":"Bob"}']);
        expect(labels(historyMenu(t, { ...e, cancellable: true }, h))[0]).toBe('cancelMatch');
    });
});

describe('un menu est grisé pendant une action en cours', async () => {
    const disabled = (/** @type {{ label: string, disabled?: boolean }[]} */ items) => items.filter((i) => i.disabled).map((i) => i.label.replace(/^direction\.menu\./, '').replace(/\{.*/, ''));

    test('forfaits et annulation du match, mise hors service', async () => {
        const busy = matchMenu(t, { ...RUNNING, running: true }, { busy: true, openResult() {}, openMove() {}, onForfeit() {}, onCancel() {} });
        expect(disabled(busy)).toEqual(['forfeitOf', 'forfeitOf', 'cancelMatch']);
        expect(disabled(cellMenu(t, FREE, { busy: true, onOutOfService() {}, onLaunchHere() {} }))).toEqual(['launchHere', 'outOfService']);
        expect(disabled(cellMenu(t, { ...FREE, unavailable: true }, { busy: true, onOutOfService() {} }))).toEqual(['backInService']);
        expect(disabled(cellMenu(t, FREE, { onOutOfService() {} }))).toEqual([]);
    });

    test('retrait, réintégration, retour, détacher, transcrire', async () => {
        const h = { busy: true, onWithdraw() {}, onReinstate() {}, onReturn() {}, onManual() {} };
        expect(disabled(playerMenu(t, { id: 'x', name: 'X', state: 'playing', table: 1 }, h))).toEqual(['withdrawNow', 'withdrawLater']);
        expect(disabled(playerMenu(t, { id: 'x', name: 'X', state: 'withdrawn' }, h))).toEqual(['reinstate']);
        expect(disabled(playerMenu(t, { id: 'x', name: 'X', state: 'absent' }, h))).toEqual(['present', 'withdrawNow']);
        expect(disabled(playerMenu(t, { id: 'x', name: 'X', state: 'free' }, h))).toEqual(['pairManually', 'withdrawNow']);
        expect(disabled(matchMenu(t, { matchId: 'M', aName: 'a', bName: 'b' }, { busy: true, onDetach() {}, onTranscribe() {} }))).toEqual(['detach', 'transcribe']);
    });

    test("l'historique : annuler", () => {
        expect(disabled(historyMenu(t, { cancellable: true, matchId: 'M', aName: 'a', bName: 'b' }, { busy: true, onCancel() {} }))).toEqual(['cancelMatch']);
    });
});

describe('entrées du plan : table libre, joueur, rattacher', async () => {
    test('une table réservée ne propose rien ; libre : lancer ici puis hors service', async () => {
        const h = { onOutOfService() {}, onLaunchHere: vi.fn() };
        expect(cellMenu(t, { ...FREE, free: false, reserved: true }, h)).toEqual([]);
        const items = cellMenu(t, FREE, h);
        expect(labels(items)).toEqual(['launchHere', 'outOfService']);
        await items[0].onClick();
        expect(h.onLaunchHere).toHaveBeenCalledWith(4);
    });

    test('un joueur libre : apparier à la main ; joue aussi ailleurs : aller là-bas', async () => {
        const onManual = vi.fn();
        const onGoElsewhere = vi.fn();
        const p = { id: 'x', name: 'Xavier', state: 'free', elsewhere: { event: 'Consolante', table: 3 } };
        const items = playerMenu(t, p, { onManual, onGoElsewhere });
        expect(labels(items)).toEqual(['pairManually', 'playsElsewhere{"event":"Consolante"}']);
        await items[0].onClick();
        items[1].onClick();
        expect(onManual).toHaveBeenCalledWith('x');
        expect(onGoElsewhere).toHaveBeenCalledWith('Consolante');
        expect(labels(playerMenu(t, { ...p, state: 'playing', table: 2 }, { onManual }))).not.toContain('pairManually');
    });

    test('une place sans match : rattacher chaque match attendu ; avec un match : rien', async () => {
        const onAttach = vi.fn();
        const attachables = [{ matchId: 'M9', label: 'Alice – Bob' }];
        const items = matchMenu(t, { aName: 'Alice', bName: 'Bob' }, { attachables, onAttach });
        expect(labels(items)).toContain('attach{"name":"Alice – Bob"}');
        items.find((i) => i.label.includes('attach'))?.onClick();
        expect(onAttach).toHaveBeenCalledWith('M9');
        expect(labels(matchMenu(t, { matchId: 'M1' }, { attachables, onAttach }))).not.toContain('attach{"name":"Alice – Bob"}');
    });
});

describe("menuRequest : un appui venu d'un enfant n'est pas celui de la ligne", () => {
    test('la touche Menu sur un bouton de la ligne ouvre rien ; sur la ligne, un menu', async () => {
        const row = document.createElement('div');
        const btn = document.createElement('button');
        row.appendChild(btn);
        let seen = null;
        row.addEventListener('keydown', (e) => (seen = menuRequest(e, () => [{ label: 'x', onClick() {} }])));
        btn.dispatchEvent(new KeyboardEvent('keydown', { key: 'ContextMenu', bubbles: true }));
        expect(seen).toBeNull();
        row.dispatchEvent(new KeyboardEvent('keydown', { key: 'ContextMenu', bubbles: true }));
        expect(seen).not.toBeNull();
    });
});

const menuEl = () => document.querySelector('.context-menu');
const menuLabels = () => [...document.querySelectorAll('.context-menu-item')].map((b) => b.textContent?.replace(/\s+/g, ' ').trim());

describe('les vues ouvrent leur menu', async () => {
    test('la grille : clic droit sur une case occupée, menu sur le pointeur', async () => {
        const { getByTestId } = render(TableGrid, { props: { cells: [RUNNING, FREE], onOutOfService: () => {} } });
        await fireEvent.contextMenu(getByTestId('direction-table-3'), { button: 2, clientX: 40, clientY: 50 });
        expect(menuEl()).not.toBeNull();
        expect(menuLabels().length).toBeGreaterThanOrEqual(5);
        expect(/** @type {HTMLElement} */ (menuEl()).style.left).toBe('40px');
    });

    test('la grille : Maj+F10 et la touche Menu ouvrent le menu de la case focalisée, libre comprise', async () => {
        const { getByTestId } = render(TableGrid, { props: { cells: [RUNNING, FREE], onOutOfService: () => {} } });
        const free = getByTestId('direction-table-4');
        // Une case libre se focalise : elle n'est plus désactivée.
        expect(/** @type {HTMLButtonElement} */ (free).disabled).toBe(false);
        free.focus();
        await fireEvent.keyDown(free, { key: 'F10', shiftKey: true });
        expect(menuLabels()).toEqual(['Take the table out of service']);
        cleanup();
        const r = render(TableGrid, { props: { cells: [RUNNING] } });
        await fireEvent.keyDown(r.getByTestId('direction-table-3'), { key: 'ContextMenu' });
        expect(menuEl()).not.toBeNull();
    });

    test("le menu rend le focus à la case qui l'a ouvert", async () => {
        const { getByTestId } = render(TableGrid, { props: { cells: [RUNNING] } });
        const cell = getByTestId('direction-table-3');
        cell.focus();
        await fireEvent.keyDown(cell, { key: 'ContextMenu' });
        expect(document.activeElement?.classList.contains('context-menu-item')).toBe(true);
        await fireEvent.keyDown(window, { key: 'ArrowDown' });
        await fireEvent.keyDown(document.activeElement ?? window, { key: 'Escape' });
        await Promise.resolve();
        expect(menuEl()).toBeNull();
        expect(document.activeElement).toBe(cell);
    });

    test('la grille : « Saisir le résultat » ouvre la fiche, « Forfait » demande confirmation', async () => {
        const onForfeit = vi.fn();
        const { getByTestId, queryByTestId } = render(TableGrid, { props: { cells: [RUNNING], onForfeit } });
        await fireEvent.contextMenu(getByTestId('direction-table-3'), { button: 2, clientX: 1, clientY: 1 });
        /** @type {HTMLElement[]} */ ([...document.querySelectorAll('.context-menu-item')])[1].click();
        await Promise.resolve();
        expect(get(confirmModalStore)).not.toBeNull();
        resolveConfirm(false);
        await Promise.resolve();
        expect(onForfeit).not.toHaveBeenCalled();
        await fireEvent.contextMenu(getByTestId('direction-table-3'), { button: 2, clientX: 1, clientY: 1 });
        /** @type {HTMLElement[]} */ ([...document.querySelectorAll('.context-menu-item')])[0].click();
        await Promise.resolve();
        expect(queryByTestId('direction-result-card')).not.toBeNull();
    });

    test("l'historique : le menu d'une ligne corrigeable", async () => {
        const entries = [{ seq: 1, kind: 'result', time: '', matchId: 'M1', a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', correctable: true }];
        const { getByTestId } = render(HistoryView, { props: { entries } });
        await fireEvent.contextMenu(getByTestId('direction-history-1'), { button: 2, clientX: 5, clientY: 5 });
        expect(menuLabels()[0]).toBe('Correct the result');
    });

    test("la file : le menu d'une proposition", async () => {
        const proposals = [{ kind: 'start_match', phase: 0, a: 'a', b: 'b', length: 7, table: 1 }];
        const { container } = render(ProposalList, {
            props: {
                proposals,
                players: [
                    { id: 'a', name: 'Alice' },
                    { id: 'b', name: 'Bob' }
                ]
            }
        });
        const row = /** @type {Element} */ (container.querySelector('ul.queue li'));
        await fireEvent.keyDown(row, { key: 'ContextMenu' });
        expect(menuLabels()).toEqual(['Start↵', 'Start at a table…', 'Change the length…', 'Pair differently', 'Ignore for now']);
    });

    test("les joueurs : le menu d'une ligne", async () => {
        const rows = [{ id: 'lb', name: 'Léa Bonnet', club: '', rating: 4, state: 'free', wins: 0, losses: 0, lives: 2, byes: 0, opponents: [] }];
        const { getByTestId } = render(PlayersView, { props: { rows, started: true, onHistory: () => {} } });
        await fireEvent.contextMenu(getByTestId('direction-player-lb'), { button: 2, clientX: 5, clientY: 5 });
        expect(menuLabels()).toContain('Mark absent');
    });

    test('les emplacements : le menu rattache un match attendu et se grise quand busy', async () => {
        const slots = [{ slotId: 's1', label: { kind: 'swiss_group', losses: 0, match: 1 }, aName: 'Alice', bName: 'Bob' }];
        const unattached = [{ matchId: 'M9', player1: 'Alice', player2: 'Bob', suggestSlot: 's1' }];
        const onAttach = vi.fn();
        const { container } = render(SlotsView, { props: { slots, unattached, busy: true, onAttach } });
        const row = /** @type {Element} */ (container.querySelector('tbody tr'));
        await fireEvent.keyDown(row, { key: 'ContextMenu' });
        const items = /** @type {HTMLButtonElement[]} */ ([...document.querySelectorAll('.context-menu-item')]);
        expect(items.map((b) => b.textContent?.trim())).toEqual(['Transcribe the match', 'Attach Alice – Bob']);
        expect(items.every((b) => b.disabled)).toBe(true);
    });

    test("la touche Menu sur un bouton d'une ligne n'ouvre pas le menu de la ligne", async () => {
        const slots = [{ slotId: 's1', label: { kind: 'swiss_group', losses: 0, match: 1 }, aName: 'Alice', bName: 'Bob' }];
        const { container } = render(SlotsView, { props: { slots, unattached: [] } });
        const btn = /** @type {Element} */ (container.querySelector('tbody tr button'));
        await fireEvent.keyDown(btn, { key: 'ContextMenu' });
        expect(menuEl()).toBeNull();
    });

    test("les joueurs : apparier à la main et aller dans l'autre épreuve", async () => {
        const rows = [{ id: 'lb', name: 'Léa Bonnet', club: '', rating: 4, state: 'free', wins: 0, losses: 0, lives: 2, byes: 0, opponents: [] }];
        const onManual = vi.fn();
        const onGoEpreuve = vi.fn();
        const { getByTestId } = render(PlayersView, { props: { rows, started: true, onManual, onGoEpreuve, elsewhere: { lb: { event: 'Consolante', table: 2 } } } });
        await fireEvent.contextMenu(getByTestId('direction-player-lb'), { button: 2, clientX: 5, clientY: 5 });
        expect(menuLabels()).toContain('Pair by hand with…');
        expect(menuLabels()).toContain('Also plays in Consolante, go there');
        /** @type {HTMLElement} */ (document.querySelector('.context-menu-item')).click();
        expect(onManual).toHaveBeenCalledWith('lb');
    });

    test("la file : lancer à la table et changer la longueur ouvrent l'appariement à la main sur le bon champ", async () => {
        const proposals = [{ kind: 'start_match', phase: 0, a: 'a', b: 'b', length: 7, table: 4 }];
        const players = [
            { id: 'a', name: 'Alice' },
            { id: 'b', name: 'Bob' }
        ];
        const { container } = render(ProposalList, { props: { proposals, players, onPrintSheet: () => {} } });
        const row = /** @type {Element} */ (container.querySelector('ul.queue li'));
        await fireEvent.keyDown(row, { key: 'ContextMenu' });
        expect(menuLabels()).toEqual(['Start↵', 'Start at a table…', 'Change the length…', 'Pair differently', 'Ignore for now', 'Print the round sheet']);
        /** @type {HTMLElement[]} */ ([...document.querySelectorAll('.context-menu-item')])[2].click();
        await new Promise((r) => setTimeout(r));
        const inputs = /** @type {HTMLInputElement[]} */ ([...container.querySelectorAll('.manual input')]);
        expect(document.activeElement).toBe(inputs[0]);
        expect(inputs[0].value).toBe('7');
        expect(inputs[1].value).toBe('4');
    });

    test('la file : une demande « lancer ici » lance la proposition sélectionnée sur la table', async () => {
        const proposals = [{ kind: 'start_match', phase: 0, a: 'a', b: 'b', length: 7, table: 0 }];
        const onManual = vi.fn();
        const { rerender } = render(ProposalList, { props: { proposals, onManual, request: null } });
        await rerender({ proposals, onManual, request: { kind: 'launchHere', table: 5, seq: 1 } });
        expect(onManual).toHaveBeenCalledWith('a', 'b', 7, 5);
    });
});

describe('la grille : accessibilité et ouverture sur le champ de table', async () => {
    test('grille ARIA, un seul arrêt de Tab, aria-haspopup seulement avec des entrées', async () => {
        const { container, getByTestId } = render(TableGrid, { props: { cells: [RUNNING, FREE, { ...FREE, table: 5, free: false, reserved: true }], onOutOfService: () => {} } });
        expect(container.querySelector('[role="grid"]')).not.toBeNull();
        const cells = /** @type {HTMLElement[]} */ ([...container.querySelectorAll('[role="gridcell"].cell')]);
        expect(cells).toHaveLength(3);
        expect(cells.map((c) => c.tabIndex)).toEqual([0, -1, -1]);
        cells[1].focus();
        await Promise.resolve();
        expect(cells.map((c) => c.tabIndex)).toEqual([-1, 0, -1]);
        expect(getByTestId('direction-table-3').getAttribute('aria-haspopup')).toBe('menu');
        expect(getByTestId('direction-table-5').getAttribute('aria-haspopup')).toBeNull();
    });

    test("une table réservée n'offre pas « hors service »", async () => {
        const { getByTestId } = render(TableGrid, { props: { cells: [{ ...FREE, free: false, reserved: true }], onOutOfService: () => {} } });
        await fireEvent.keyDown(getByTestId('direction-table-4'), { key: 'ContextMenu' });
        expect(menuEl()).toBeNull();
    });

    test('M sur une fiche déjà ouverte met le champ de table devant le curseur', async () => {
        const { getByTestId, findByTestId } = render(TableGrid, { props: { cells: [RUNNING] } });
        const cell = getByTestId('direction-table-3');
        await fireEvent.click(cell);
        cell.focus();
        await fireEvent.keyDown(cell, { key: 'm' });
        expect(document.activeElement).toBe(await findByTestId('direction-result-move-table'));
        // Seconde demande, le focus ayant quitté le champ : elle se rejoue.
        cell.focus();
        await fireEvent.keyDown(cell, { key: 'x' });
        await Promise.resolve();
        await Promise.resolve();
        expect(document.activeElement).toBe(await findByTestId('direction-result-move-table'));
    });

    test('un menu ouvert ne coexiste pas avec un second', async () => {
        const { getByTestId } = render(TableGrid, { props: { cells: [RUNNING, { ...RUNNING, table: 6, matchId: 'M6' }] } });
        await fireEvent.contextMenu(getByTestId('direction-table-3'), { button: 2, clientX: 1, clientY: 1 });
        await fireEvent.contextMenu(getByTestId('direction-table-6'), { button: 2, clientX: 9, clientY: 9 });
        expect(document.querySelectorAll('.context-menu')).toHaveLength(1);
        expect(/** @type {HTMLElement} */ (menuEl()).style.left).toBe('9px');
    });
});
