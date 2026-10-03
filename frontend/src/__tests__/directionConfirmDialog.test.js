/**
 * Les gestes destructifs de l'historique et de la Rencontre se confirment par le dialogue
 * thémé (confirmAction) ; refusés, ils ne font rien.
 */

import { test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { answerConfirm } from './confirmHelper.js';

vi.mock('../stores/rencontreStore.js', () => ({
    listRencontres: vi.fn(),
    createRencontre: vi.fn(),
    previewAttach: vi.fn(),
    attachToRencontre: vi.fn(),
    detachFromRencontre: vi.fn(),
    trashRencontre: vi.fn(async () => null),
    setTableOutOfService: vi.fn(),
    chooseRencontreOutputDir: vi.fn(),
    forgetRencontreOutputDir: vi.fn(),
    writeRencontrePage: vi.fn(),
    setRencontreTables: vi.fn(async () => null),
    setEventRooms: vi.fn(async () => null)
}));

import HistoryView from '../components/direction/HistoryView.svelte';
import RencontrePanel from '../components/direction/RencontrePanel.svelte';
import { listRencontres, trashRencontre } from '../stores/rencontreStore.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

test("annuler un match depuis l'historique se confirme", async () => {
    const onCancel = vi.fn();
    const entries = [{ seq: 4, kind: 'match_started', matchId: 'M1', a: 'ha', b: 'lb', aName: 'Hugo', bName: 'Léa', cancellable: true, correctable: false }];
    const { container } = render(HistoryView, { props: { entries, onCancel } });
    const btn = /** @type {Element} */ (container.querySelector('[data-testid="direction-history-4"] button'));
    await fireEvent.click(btn);
    await answerConfirm(false);
    expect(onCancel).not.toHaveBeenCalled();
    await fireEvent.click(btn);
    await answerConfirm(true);
    expect(onCancel).toHaveBeenCalledWith('M1');
});

test('mettre une Rencontre à la corbeille se confirme', async () => {
    /** @type {any} */
    const ev = { id: 9, name: 'Open', tables: 4, tournamentIds: [3], members: [{ tournamentId: 3, name: 'Principal', state: 'running' }], tableSettings: [], eventRooms: {}, room: { tables: 4 } };
    vi.mocked(listRencontres).mockResolvedValue([ev]);
    const { getByTestId } = render(RencontrePanel, { props: { tournamentId: 3, rencontreId: 9 } });
    await tick();
    await vi.waitFor(() => expect(getByTestId('rencontre-trash')).toBeTruthy());
    await fireEvent.click(getByTestId('rencontre-trash'));
    await answerConfirm(false);
    expect(trashRencontre).not.toHaveBeenCalled();
    await fireEvent.click(getByTestId('rencontre-trash'));
    await answerConfirm(true);
    await vi.waitFor(() => expect(trashRencontre).toHaveBeenCalledWith(9));
});
