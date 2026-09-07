/**
 * TranscriptView.menu.test.js — T2.5 : le clic droit sur une cellule.
 *
 * La vue ne connaît aucun geste : elle dit QUELLE cellule a été visée et OÙ,
 * en pixels client, et c'est le panneau qui ouvre le menu et corrige. Ce qui se
 * tient ici est donc ce contrat, plus la non-régression de la recette : le menu
 * natif du navigateur n'est retiré que sur les cellules du Transcript, jamais
 * ailleurs dans l'application.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import TranscriptView from '../components/TranscriptView.svelte';

const POSITION = (dice = [0, 0]) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice,
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

function annotatedOf(rows) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: rows.map((r) => r.action), cursor: 0 },
        actions: rows.map((r, index) => ({
            index,
            side: r.action.side,
            kind: r.action.kind,
            before: POSITION(r.action.dice ?? [0, 0]),
            has_position: r.action.kind !== 'opening',
            notation: r.notation ?? '',
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: []
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: 1, points_won: 2, crawford: false, finished: true, first: 0, last: rows.length - 1 }],
        next: { expects: 'checker', side: 0, position: POSITION(), crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

const ORDINARY = annotatedOf([
    { action: { side: 0, kind: 'opening', dice: [3, 1] } },
    { action: { side: 0, kind: 'checker', dice: [3, 1] }, notation: '8/5 6/5' },
    { action: { side: 1, kind: 'checker', dice: [5, 2] }, notation: '13/8 13/11' }
]);

const rightClick = (el, at = { clientX: 120, clientY: 340 }) => fireEvent.contextMenu(el, at);

afterEach(cleanup);

describe('le clic droit sur une cellule d’Action', () => {
    test('rend l’index de la cellule et l’endroit du clic', async () => {
        const onMenu = vi.fn();
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY, onSelect: () => {}, onMenu } });

        await rightClick(container.querySelector('[data-index="2"]'));
        expect(onMenu).toHaveBeenCalledExactlyOnceWith(2, { x: 120, y: 340 });
    });

    // Le Cursor n'est pas déplacé par la vue : elle ne connaît pas le geste.
    // C'est le panneau qui mène le Cursor à la cellule avant de corriger
    // (`menuCommands`), et c'est là que ça se mesure.
    test('le clic droit ne sélectionne pas : les deux boutons sont distincts', async () => {
        const onSelect = vi.fn();
        const onMenu = vi.fn();
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY, onSelect, onMenu } });

        await rightClick(container.querySelector('[data-index="1"]'));
        expect(onMenu).toHaveBeenCalledTimes(1);
        expect(onSelect).not.toHaveBeenCalled();
    });

    // Recette T2.5 : le menu natif ne disparaît que dans la zone du Transcript.
    test('le menu natif est retiré sur la cellule, et nulle part ailleurs', async () => {
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY, onSelect: () => {}, onMenu: () => {} } });

        const onCell = await rightClick(container.querySelector('[data-index="1"]'));
        expect(onCell).toBe(false); // preventDefault : pas de menu système

        // L'en-tête de la partie, le volet .mat, la vue autour : rien n'est pris.
        const elsewhere = await rightClick(container.querySelector('.game-header'));
        expect(elsewhere).toBe(true);
    });

    // Sans écouteur, la vue reste une vue : le clic droit rend le menu du
    // navigateur, comme dans un Transcript que personne ne corrige (le panneau
    // Matchs, un jour).
    test('sans onMenu, le clic droit garde le menu du navigateur', async () => {
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY, onSelect: () => {} } });
        expect(await rightClick(container.querySelector('[data-index="1"]'))).toBe(true);
    });

    // Une cellule qui n'est pas une Action — la fin de partie — n'est pas un
    // bouton et n'a rien à corriger.
    test('la cellule de résultat n’ouvre pas de menu', async () => {
        const onMenu = vi.fn();
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY, onSelect: () => {}, onMenu } });
        const result = container.querySelector('.cell.result');
        expect(result).not.toBeNull();
        await rightClick(result);
        expect(onMenu).not.toHaveBeenCalled();
    });
});
