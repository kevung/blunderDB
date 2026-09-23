/**
 * TranscriptView.pending.test.js — l'Action en cours de saisie, DESSINÉE.
 *
 * Ce que ce fichier tient : le tableau des décisions montre ce que l'utilisateur
 * tape, à l'endroit où cela sera écrit, avant toute validation. Sans cela une
 * correction laissait la cellule afficher l'Action enregistrée — on lisait une
 * chose en en tapant une autre — et une insertion n'apparaissait nulle part
 * avant d'être validée.
 *
 * Rien n'est dérivé ici : la vue lit `annotated.entry`, que le moteur remplit
 * (transcript.EntryInfo — le rang, s'il remplace, le camp, les dés, la notation
 * du coup choisi, la sorte d'Action que la validation écrirait).
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import TranscriptView from '../components/TranscriptView.svelte';

const POSITION = () => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

const ROWS = [
    { action: { side: 0, kind: 'opening', dice: [3, 1] }, notation: '' },
    { action: { side: 0, kind: 'checker', dice: [3, 1] }, notation: '8/5 6/5' },
    { action: { side: 1, kind: 'checker', dice: [5, 2] }, notation: '13/8 13/11' },
    { action: { side: 0, kind: 'double' }, notation: '' },
    { action: { side: 1, kind: 'pass' }, notation: '' }
];

/**
 * @param {any} entry - `annotated.entry`, ou null
 * @param {number} cursor
 * @param {any} gameOver
 */
function annotated(entry, cursor = 0, gameOver = {}) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: ROWS.map((r) => r.action), cursor },
        actions: ROWS.map((r, index) => ({
            index,
            side: r.action.side,
            kind: r.action.kind,
            before: POSITION(),
            has_position: r.action.kind !== 'opening',
            notation: r.notation,
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: []
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: 4, ...gameOver }],
        next: { expects: 'checker', side: 0, position: POSITION(), crawford: false },
        entry,
        score: [0, 0],
        cursor
    };
}

const ENTRY = (/** @type {any} */ over) => ({ at: 0, replacing: true, side: 0, dice: [0, 0], selected: false, review: false, kind: 'checker', ...over });

const cells = (/** @type {any} */ container) => [...container.querySelectorAll('.cell')].filter((el) => !el.classList.contains('empty-cell'));
const texts = (/** @type {any} */ container) => cells(container).map((el) => el.textContent.trim());
const pending = (/** @type {any} */ container) => container.querySelector('.cell.pending');
const columnOf = (/** @type {any} */ cell) => [...cell.closest('tr').children].indexOf(cell.closest('td'));

afterEach(() => cleanup());

describe('la correction en place', () => {
    test('les dés retapés remplacent le texte de la cellule, avant toute validation', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 1, replacing: true, side: 0, dice: [4, 2], selected: true, notation: '24/20 13/11' }), 1),
            onSelect: () => {}
        });

        const cell = pending(container);
        expect(cell).not.toBeNull();
        expect(cell?.textContent.trim()).toBe('42: 24/20 13/11');
        // L'Action enregistrée ne se lit plus à côté : une cellule, un coup.
        expect(texts(container)).not.toContain('31: 8/5 6/5');
        // Et rien n'a été ajouté : la correction tient la place de l'Action.
        expect(cells(container)).toHaveLength(5);
    });

    test('un dé seul se lit déjà, l’autre reste en attente', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 1, replacing: true, dice: [4, 0] }), 1),
            onSelect: () => {}
        });
        expect(pending(container)?.textContent.trim()).toBe('4·');
    });

    test('le Cursor encadre la cellule en cours de saisie, et elle seule', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 1, replacing: true, dice: [4, 2], notation: '24/20 13/11' }), 1),
            onSelect: () => {}
        });
        const framed = [...container.querySelectorAll('.cell.cursor')];
        expect(framed).toHaveLength(1);
        expect(framed[0].classList.contains('pending')).toBe(true);
    });

    test('le Cursor posé sans rien taper montre l’Action telle qu’elle est écrite', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 1, replacing: true, dice: [3, 1], selected: true, notation: '8/5 6/5' }), 1),
            onSelect: () => {}
        });
        // Les dés de l'Action sont ceux chargés par le moteur : la cellule dit
        // la même chose que le document, et n'a pas à se dessiner en pointillés
        // tant que rien n'a changé… mais elle le fait dès qu'un dé est tapé.
        // Ici on tient le texte, qui est celui de l'Action.
        expect(texts(container)).toContain('31: 8/5 6/5');
    });

    test('le camp suit l’ordre du moteur : la cellule change de colonne aussitôt', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 1, replacing: true, side: 1, dice: [4, 2], notation: '24/20 13/11' }), 1),
            onSelect: () => {}
        });
        expect(columnOf(pending(container))).toBe(2); // la colonne du joueur 2
    });
});

describe('l’insertion', () => {
    test('le trou ouvert par une insertion se voit avant que rien n’y soit tapé', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 2, replacing: false, side: 1, dice: [0, 0] }), 2),
            onSelect: () => {}
        });
        const cell = pending(container);
        expect(cell).not.toBeNull();
        expect(cell?.textContent.trim()).toBe('··');
        // Rien n'a disparu : l'Action repoussée est toujours là.
        expect(texts(container)).toContain('52: 13/8 13/11');
        expect(cells(container)).toHaveLength(6);
    });

    test('une saisie neuve en bout de document apparaît dans le tableau', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 5, replacing: false, side: 0, dice: [6, 5], selected: true, notation: '24/13' }), 5),
            onSelect: () => {}
        });
        expect(pending(container)?.textContent.trim()).toBe('65: 24/13');
    });

    test('une partie finie n’accueille pas la saisie de la suivante', () => {
        const { container } = render(TranscriptView, {
            annotated: annotated(ENTRY({ at: 5, replacing: false, side: 0, dice: [6, 0] }), 5, { winner: 0, points_won: 1, finished: true }),
            onSelect: () => {}
        });
        // L'ouverture tapée ouvrira la partie 2, qui n'existe pas encore : il
        // n'y a aucun tableau où la poser, et rien n'est inventé.
        expect(pending(container)).toBeNull();
    });
});
