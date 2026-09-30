// @ts-nocheck — données de test partielles : les props typées par Wails exigent des classes générées.
/**
 * Les menus contextuels de la page Direction : le contenu de chaque menu (sans DOM), puis
 * chaque vue montée, ouverte au clic droit et au clavier. Les gestes destructifs gardent leur
 * confirmation : refusée, rien n'est appelé.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

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

describe('contenu des menus', () => {
    test('une case occupée offre résultat, forfaits, changement et échange, annulation', () => {
        const l = labels(cellMenu(t, RUNNING, { openResult() {}, openMove() {}, onForfeit() {}, onCancel() {} }));
        expect(l).toEqual(['enterResult', 'forfeitOf{"name":"Alice"}', 'forfeitOf{"name":"Bob"}', 'moveTable', 'swapTable', 'cancelMatch']);
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

    test('un forfait confirmé désigne le bon vainqueur ; refusé, il ne fait rien', () => {
        const onForfeit = vi.fn();
        const confirm = vi.fn().mockReturnValue(false);
        const items = matchMenu(t, { ...RUNNING, running: true }, { onForfeit, confirm });
        items[0].onClick();
        expect(onForfeit).not.toHaveBeenCalled();
        confirm.mockReturnValue(true);
        items[0].onClick();
        expect(onForfeit).toHaveBeenCalledWith('M1', 'b', '');
    });

    test('annuler un match se confirme', () => {
        const onCancel = vi.fn();
        const confirm = vi.fn().mockReturnValue(false);
        const cancel = matchMenu(t, { ...RUNNING, running: true }, { onCancel, confirm }).find((i) => i.label.includes('cancelMatch'));
        cancel?.onClick();
        expect(onCancel).not.toHaveBeenCalled();
        confirm.mockReturnValue(true);
        cancel?.onClick();
        expect(onCancel).toHaveBeenCalledWith('M1');
    });

    test('un joueur : les entrées suivent son état', () => {
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

    test('retirer un joueur se confirme', () => {
        const onWithdraw = vi.fn();
        const confirm = vi.fn().mockReturnValue(false);
        const now = playerMenu(t, { id: 'x', name: 'Xavier', state: 'free' }, { onWithdraw, confirm }).find((i) => i.label.includes('withdrawNow'));
        now?.onClick();
        expect(onWithdraw).not.toHaveBeenCalled();
        confirm.mockReturnValue(true);
        now?.onClick();
        expect(onWithdraw).toHaveBeenCalledWith('x', false);
    });

    test('une proposition : lancer, apparier autrement (un appariement seulement), ignorer', () => {
        const h = { onLaunch() {}, onIgnore() {}, onArrange() {} };
        expect(labels(proposalMenu(t, { kind: 'start_match', a: 'a', b: 'b' }, h))).toEqual(['launch', 'arrangeOther', 'ignore']);
        expect(labels(proposalMenu(t, { kind: 'cancel_match' }, h))).toEqual(['launch', 'ignore']);
    });

    test("une ligne d'historique : corriger ou annuler, remarque, filtres", () => {
        const h = { onCorrect() {}, onCancel() {}, onNote() {}, onFilter() {} };
        const e = { aName: 'Alice', bName: 'Bob', matchId: 'M1' };
        expect(labels(historyMenu(t, { ...e, correctable: true }, h))).toEqual(['correctResult', 'addNote', 'filterOn{"name":"Alice"}', 'filterOn{"name":"Bob"}']);
        expect(labels(historyMenu(t, { ...e, cancellable: true }, h))[0]).toBe('cancelMatch');
    });
});

const menuEl = () => document.querySelector('.context-menu');
const menuLabels = () => [...document.querySelectorAll('.context-menu-item')].map((b) => b.textContent?.replace(/\s+/g, ' ').trim());

describe('les vues ouvrent leur menu', () => {
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
        const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
        const { getByTestId, queryByTestId } = render(TableGrid, { props: { cells: [RUNNING], onForfeit } });
        await fireEvent.contextMenu(getByTestId('direction-table-3'), { button: 2, clientX: 1, clientY: 1 });
        /** @type {HTMLElement[]} */ ([...document.querySelectorAll('.context-menu-item')])[1].click();
        expect(confirm).toHaveBeenCalled();
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
        expect(menuLabels()).toEqual(['Start↵', 'Pair differently', 'Ignore for now']);
    });

    test("les joueurs : le menu d'une ligne", async () => {
        const rows = [{ id: 'lb', name: 'Léa Bonnet', club: '', rating: 4, state: 'free', wins: 0, losses: 0, lives: 2, byes: 0, opponents: [] }];
        const { getByTestId } = render(PlayersView, { props: { rows, started: true, onHistory: () => {} } });
        await fireEvent.contextMenu(getByTestId('direction-player-lb'), { button: 2, clientX: 5, clientY: 5 });
        expect(menuLabels()).toContain('Mark absent');
    });
});
