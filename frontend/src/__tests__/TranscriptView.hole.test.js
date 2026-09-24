/**
 * TranscriptView.hole.test.js — ADR-0054 : le trou d'un double trait est une
 * case du Transcript.
 *
 * Une suppression laisse un camp jouer deux fois de suite ; le tour de l'autre
 * manque entre les deux. Cette case est dessinée dans la colonne du camp qui
 * n'a pas joué, marquée, et un clic y mène le Cursor — c'est là que la décision
 * supprimée se tape à nouveau. Le rang des arrêts du Cursor (`cursorStop`) la
 * compte, sans quoi un clic plus loin s'arrêterait un pas trop tôt.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import TranscriptView from '../components/TranscriptView.svelte';
import { cursorStop, currentStop, cursorCommands, COMMAND } from '../services/transcriptionKeys.js';

const POSITION = () => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

const DOUBLE_TURN = [{ kind: 'double_turn', detail: '' }];

/**
 * Ouverture, puis Kévin (0), Alice (1), et `double` le camp qui rejoue à
 * l'index 3.
 *
 * @param {number} double
 * @param {any} [entry]
 */
function holed(double, entry = null, cursor = 0) {
    const rows = [
        { side: 0, kind: 'opening', dice: [3, 1] },
        { side: 0, kind: 'checker', dice: [3, 1] },
        { side: 1, kind: 'checker', dice: [5, 2] },
        { side: double, kind: 'checker', dice: [6, 4] }
    ];
    if (double === 0) rows.splice(2, 1); // Kévin joue deux fois : 1 puis 2
    return {
        document: { header: { match_length: 7 }, actions: rows, cursor },
        actions: rows.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            before: POSITION(),
            has_position: a.kind !== 'opening',
            notation: '',
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: index > 1 && rows[index - 1].side === a.side ? DOUBLE_TURN : []
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: rows.length - 1 }],
        next: { expects: 'checker', side: 0, position: POSITION(), crawford: false },
        score: [0, 0],
        cursor,
        entry
    };
}

const columnOf = (/** @type {any} */ cell) => [...cell.closest('tr').children].indexOf(cell.closest('td'));

afterEach(() => cleanup());

describe('le trou est dessiné dans la colonne du camp qui n’a pas joué', () => {
    test('Kévin joue deux fois : le trou est à droite, sur sa première ligne', () => {
        const { container } = render(TranscriptView, { props: { annotated: holed(0), onSelect: () => {}, onHole: () => {} } });
        const hole = /** @type {any} */ (container.querySelector('[data-hole="2"]'));
        expect(hole).toBeTruthy();
        expect(columnOf(hole)).toBe(2);
        expect(hole.closest('tr')).toBe(container.querySelector('[data-index="1"]')?.closest('tr'));
    });

    test('Alice joue deux fois : le trou est à gauche, sur la ligne de son second coup', () => {
        const { container } = render(TranscriptView, { props: { annotated: holed(1), onSelect: () => {}, onHole: () => {} } });
        const hole = /** @type {any} */ (container.querySelector('[data-hole="3"]'));
        expect(hole).toBeTruthy();
        expect(columnOf(hole)).toBe(1);
        expect(hole.closest('tr')).toBe(container.querySelector('[data-index="3"]')?.closest('tr'));
    });

    test('un clic sur le trou rend l’index de l’Action qu’il précède', async () => {
        /** @type {number[]} */
        const got = [];
        const { container } = render(TranscriptView, { props: { annotated: holed(1), onSelect: () => {}, onHole: (/** @type {number} */ i) => got.push(i) } });
        await fireEvent.click(/** @type {any} */ (container.querySelector('[data-hole="3"]')));
        expect(got).toEqual([3]);
    });

    test('l’insertion qui le remplit prend sa place, sans second trou', () => {
        const entry = { at: 3, replacing: false, side: 0, dice: [0, 0], notation: '', kind: 'checker' };
        const { container } = render(TranscriptView, { props: { annotated: holed(1, entry, 3), onSelect: () => {}, onHole: () => {} } });
        expect(container.querySelector('[data-hole]')).toBeNull();
        const pending = /** @type {any} */ (container.querySelector('[data-pending="true"]'));
        expect(columnOf(pending)).toBe(1);
        expect(pending.closest('tr')).toBe(container.querySelector('[data-index="3"]')?.closest('tr'));
    });
});

describe('le rang des arrêts compte le trou', () => {
    test('l’Action après le trou est un pas plus loin que son index', () => {
        const ann = holed(1);
        expect(cursorStop(ann, 2)).toBe(2);
        expect(cursorStop(ann, 3, true)).toBe(3);
        expect(cursorStop(ann, 3)).toBe(4);
    });

    test('un clic par-dessus le trou le traverse : trois pas de 1 à 3', () => {
        const ann = { ...holed(1), cursor: 1 };
        expect(cursorCommands(currentStop(ann), cursorStop(ann, 3))).toEqual([{ kind: COMMAND.CURSOR_FORWARD }, { kind: COMMAND.CURSOR_FORWARD }, { kind: COMMAND.CURSOR_FORWARD }]);
    });

    test('le Cursor sur le trou est à son rang, et l’Action après lui à un pas', () => {
        const ann = holed(1, { at: 3, replacing: false, side: 0, dice: [0, 0] }, 3);
        expect(currentStop(ann)).toBe(3);
        expect(cursorCommands(currentStop(ann), cursorStop(ann, 3))).toEqual([{ kind: COMMAND.CURSOR_FORWARD }]);
    });

    test('sans trou, le rang est l’index', () => {
        const ann = { ...holed(1), actions: holed(1).actions.map((/** @type {any} */ a) => ({ ...a, inconsistencies: [] })) };
        for (let i = 0; i < 4; i++) expect(cursorStop(ann, i)).toBe(i);
    });
});
