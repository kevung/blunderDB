/**
 * Les propriétés des tables (ADR-0058) : l'éditeur, la grille qui montre nom, repères et salles,
 * les propositions qui nomment la table, et l'Événement qui les enregistre.
 */

import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../stores/rencontreStore.js', () => ({
    listRencontres: vi.fn(),
    createRencontre: vi.fn(),
    previewAttach: vi.fn(),
    attachToRencontre: vi.fn(),
    detachFromRencontre: vi.fn(),
    trashRencontre: vi.fn(),
    setTableOutOfService: vi.fn(),
    chooseRencontreOutputDir: vi.fn(),
    forgetRencontreOutputDir: vi.fn(),
    writeRencontrePage: vi.fn(),
    setRencontreTables: vi.fn(async () => null),
    setEventRooms: vi.fn(async () => null),
    seasonRanking: vi.fn(),
    seasonCSV: vi.fn()
}));

import TableSettingsEditor from '../components/direction/TableSettingsEditor.svelte';
import RencontrePanel from '../components/direction/RencontrePanel.svelte';
import TableGrid from '../components/direction/TableGrid.svelte';
import ProposalList from '../components/direction/ProposalList.svelte';
import { listRencontres, setRencontreTables, setEventRooms } from '../stores/rencontreStore.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const flush = async () => {
    await tick();
    await Promise.resolve();
    await tick();
};

describe("l'éditeur des propriétés", () => {
    test('un lot de salle, un nom, une réservation : seules les tables touchées partent', async () => {
        const onSave = vi.fn(async () => null);
        const { getByTestId } = render(TableSettingsEditor, { props: { tables: 6, settings: [], participants: ['Alice', 'Bob'], onSave } });
        const save = /** @type {HTMLButtonElement} */ (getByTestId('table-settings-save'));
        expect(save.disabled).toBe(true);

        await fireEvent.input(getByTestId('table-settings-bulk-from'), { target: { value: '1' } });
        await fireEvent.input(getByTestId('table-settings-bulk-to'), { target: { value: '3' } });
        await fireEvent.input(getByTestId('table-settings-bulk-room'), { target: { value: 'A' } });
        await fireEvent.click(getByTestId('table-settings-bulk-apply'));
        await fireEvent.input(getByTestId('table-settings-name-2'), { target: { value: 'Stream' } });
        await fireEvent.click(getByTestId('table-settings-reserved-5'));
        await fireEvent.change(getByTestId('table-settings-assign-5'), { target: { value: 'Alice' } });
        expect(save.disabled).toBe(false);

        await fireEvent.click(save);
        await flush();
        expect(onSave).toHaveBeenCalledWith([
            { number: 1, name: '', room: 'A', reserved: false, assignedTo: [] },
            { number: 2, name: 'Stream', room: 'A', reserved: false, assignedTo: [] },
            { number: 3, name: '', room: 'A', reserved: false, assignedTo: [] },
            { number: 5, name: '', room: '', reserved: true, assignedTo: ['Alice'] }
        ]);
    });

    test('part des propriétés connues, et le refus du service se lit', async () => {
        const onSave = vi.fn(async () => {
            throw new Error('direction: table 4 porte un match');
        });
        const settings = [{ number: 4, name: 'Stream', room: 'B', reserved: false, assignedTo: [] }];
        const { getByTestId } = render(TableSettingsEditor, { props: { tables: 5, settings, participants: [], onSave } });
        expect(/** @type {HTMLInputElement} */ (getByTestId('table-settings-name-4')).value).toBe('Stream');
        await fireEvent.click(getByTestId('table-settings-reserved-4'));
        await fireEvent.click(getByTestId('table-settings-save'));
        await flush();
        expect(getByTestId('table-settings-error').textContent).toContain('table 4');
    });
});

describe("la grille et l'Événement", () => {
    /** @type {(table: number, extra?: any) => any} */
    const cell = (table, extra = {}) => ({ table, free: true, unavailable: false, reserved: false, ...extra });

    test('le nom suit le numéro, réservée et attitrée sont repérées', () => {
        const cells = [cell(1, { name: 'Stream', reserved: true }), cell(2, { assignedTo: ['Alice'] }), cell(3)];
        const { getByTestId } = render(TableGrid, { props: { cells } });
        const one = getByTestId('direction-table-1');
        expect(one.textContent).toContain('Stream');
        expect(one.querySelector('[data-testid="direction-table-reserved"]')).not.toBeNull();
        expect(getByTestId('direction-table-2').querySelector('[data-testid="direction-table-assigned"]')?.textContent).toContain('Alice');
        expect(getByTestId('direction-table-3').querySelector('.mark')).toBeNull();
    });

    test('plusieurs salles : la grille se groupe par salle ; une seule : rien ne change', () => {
        const two = [cell(1, { room: 'A' }), cell(2, { room: 'B' }), cell(3, { room: 'A' })];
        const { container, unmount } = render(TableGrid, { props: { cells: two } });
        expect([...container.querySelectorAll('.room-title')].map((h) => h.textContent)).toHaveLength(2);
        expect([...container.querySelectorAll('.cell .num')].map((n) => n.textContent)).toEqual(['1', '3', '2']);
        unmount();
        const one = render(TableGrid, { props: { cells: [cell(1, { room: 'A' }), cell(2, { room: 'A' })] } });
        expect(one.container.querySelectorAll('.room-title')).toHaveLength(0);
    });

    test('une proposition nomme la table', () => {
        /** @type {any} */
        const proposals = [{ kind: 'start_match', a: 'a', b: 'b', length: 5, table: 4, key: 'k' }];
        const { container } = render(ProposalList, { props: { proposals, tableNames: { 4: 'Stream' }, players: [] } });
        expect(container.textContent).toContain('4 · Stream');
    });

    test("le panneau Événement enregistre les tables et les salles de l'épreuve", async () => {
        /** @type {any} */
        const ev = {
            id: 9,
            name: 'Open',
            tables: 4,
            tournamentIds: [3],
            members: [{ tournamentId: 3, name: 'Principal', state: 'running' }],
            tableSettings: [
                { number: 1, name: '', room: 'A', reserved: false, assignedTo: [] },
                { number: 3, name: '', room: 'B', reserved: false, assignedTo: [] }
            ],
            eventRooms: { 3: ['A'] },
            room: { tables: 4 }
        };
        vi.mocked(listRencontres).mockResolvedValue([ev]);
        const { getByTestId } = render(RencontrePanel, { props: { tournamentId: 3, rencontreId: 9 } });
        await flush();
        expect(/** @type {HTMLInputElement} */ (getByTestId('rencontre-room-A')).checked).toBe(true);
        await fireEvent.click(getByTestId('rencontre-room-B'));
        await flush();
        expect(setEventRooms).toHaveBeenCalledWith(9, 3, ['A', 'B']);

        await fireEvent.input(getByTestId('rencontre-tables-editor-name-2'), { target: { value: 'Stream' } });
        await fireEvent.click(getByTestId('rencontre-tables-editor-save'));
        await flush();
        const sent = vi.mocked(setRencontreTables).mock.calls[0];
        expect(sent[0]).toBe(9);
        expect(sent[1]).toContainEqual({ number: 2, name: 'Stream', room: '', reserved: false, assignedTo: [] });
        expect(sent[1]).toHaveLength(3);
    });
});
