/**
 * Les gestes qui retirent quelque chose de la salle se confirment ; Ctrl+Z dans un champ reste
 * l'annulation de la frappe ; une table partagée par deux matchs les montre tous les deux.
 */
import { test, expect, vi, afterEach } from 'vitest';
import { answerConfirm } from './confirmHelper.js';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import PlayersView from '../components/direction/PlayersView.svelte';
import LastDecision from '../components/direction/LastDecision.svelte';
import TableGrid from '../components/direction/TableGrid.svelte';

afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
    vi.restoreAllMocks();
});

test('retirer un joueur se confirme', async () => {
    const onWithdraw = vi.fn();
    const rows = [{ id: 'p1', name: 'Alice', state: 'free', wins: 0, losses: 0 }];
    render(PlayersView, { props: { rows, started: true, onWithdraw } });
    await fireEvent.click(screen.getByTestId('direction-player-withdraw-now'));
    await answerConfirm(false);
    expect(onWithdraw).not.toHaveBeenCalled();
    await fireEvent.click(screen.getByTestId('direction-player-withdraw-now'));
    await answerConfirm(true);
    expect(onWithdraw).toHaveBeenCalledWith('p1', false);
});

const started = { kind: 'match_started', seq: 3, matchId: 'm1', a: 'pa', b: 'pb', aName: 'Alice', bName: 'Bruno', cancellable: true, correctable: false };

test('annuler le dernier match lancé se confirme', async () => {
    const onCancelMatch = vi.fn();
    render(LastDecision, { props: { last: started, onCancelMatch } });
    await fireEvent.click(screen.getByTestId('direction-last-cancel'));
    await answerConfirm(false);
    expect(onCancelMatch).not.toHaveBeenCalled();
    await fireEvent.click(screen.getByTestId('direction-last-cancel'));
    await answerConfirm(true);
    expect(onCancelMatch).toHaveBeenCalledWith('m1');
});

test('Ctrl+Z dans un champ texte n’est pas intercepté', async () => {
    render(LastDecision, { props: { last: { ...started, correctable: true, cancellable: false, winner: 'pa', winnerName: 'Alice' } } });
    const input = document.createElement('input');
    document.body.appendChild(input);
    input.focus();
    const ev = new KeyboardEvent('keydown', { key: 'z', code: 'KeyZ', ctrlKey: true, bubbles: true, cancelable: true });
    input.dispatchEvent(ev);
    await tick();
    expect(ev.defaultPrevented).toBe(false);
});

test('deux matchs sur une table : les deux cases restent, signalées', async () => {
    const cells = [
        { table: 1, matchId: 'm1', a: 'pa', b: 'pb', aName: 'Alice', bName: 'Bruno', length: 7, shared: true },
        { table: 2, free: true },
        { table: 1, matchId: 'm2', a: 'pc', b: 'pd', aName: 'Chloé', bName: 'Denis', length: 7, shared: true }
    ];
    render(TableGrid, { props: { cells: /** @type {any} */ (cells) } });
    expect(screen.getByTestId('direction-table-shared-m1')).toBeTruthy();
    expect(screen.getByTestId('direction-table-shared-m2')).toBeTruthy();
    expect(screen.getAllByTestId('direction-table-conflict')).toHaveLength(2);
});
