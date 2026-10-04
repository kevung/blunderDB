import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import ProposalList from '../components/direction/ProposalList.svelte';
import TableGrid from '../components/direction/TableGrid.svelte';

afterEach(() => {
    cleanup();
    document.body.innerHTML = '';
});

const flush = () => new Promise((r) => setTimeout(r, 0));
const PROPOSALS = [
    { kind: 'start_match', phase: 0, a: 'ha', b: 'lb', length: 7 },
    { kind: 'start_match', phase: 0, a: 'mc', b: 'nd', length: 7 }
];

describe('le focus ne retombe pas sur body', () => {
    test('après « Lancer », le focus reste dans la file des propositions', async () => {
        const { container, rerender } = render(ProposalList, { props: { proposals: PROPOSALS, onConfirm: () => {} } });
        await tick();
        const go = /** @type {HTMLElement} */ (container.querySelector('.queue li .go'));
        go.focus();
        expect(document.activeElement).toBe(go);

        await rerender({ proposals: PROPOSALS.slice(1), onConfirm: () => {} });
        await flush();

        expect(document.activeElement).not.toBe(document.body);
        expect(container.querySelector('.proposals')?.contains(document.activeElement)).toBe(true);
    });

    test('après la fermeture de la fiche de résultat, le focus est sur une case de table', async () => {
        const cells = [{ table: 1, free: false, unavailable: false, reserved: false, matchId: 'M1', a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', length: 7 }];
        const { container, getByTestId } = render(TableGrid, { props: { cells, onResult: vi.fn() } });
        await fireEvent.click(getByTestId('direction-table-1'));
        await tick();
        const close = /** @type {HTMLElement} */ (container.querySelector('.card .close'));
        close.focus();
        await fireEvent.click(close);
        await flush();

        expect(document.activeElement).not.toBe(document.body);
        expect(container.querySelector('.grid .cell')).toBe(document.activeElement);
    });
});
