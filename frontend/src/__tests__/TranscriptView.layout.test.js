/**
 * TranscriptView.layout.test.js — T1.6 : le Transcript en deux colonnes.
 *
 * Ce qui est tenu ici est la DISPOSITION, au sens du glossaire : une cellule
 * par Action, dans la colonne du camp qui a agi, une ligne par tour, l'action
 * de videau et la fin de partie dans la colonne de celui qui agit — la
 * disposition de `testdata/test.mat` et de toute feuille de match. Plus les
 * quatre autres promesses de la recette (ux.md §5) : la cellule du Cursor
 * encadrée, chaque Incohérence décorée et NOMMÉE, la partie courante ouverte
 * et les autres repliées. Le texte `.mat` a quitté ce composant : c'est une
 * modale du panneau de transcription (ADR-0048 décision 6), parce qu'un `.mat`
 * est de l'ASCII aligné en colonnes que 320 px désalignent, et parce qu'un volet
 * propre au brouillon n'avait rien à faire dans un composant que le panneau
 * Match doit pouvoir monter sur un match stocké.
 *
 * La vue ne dérive rien : tout ce qu'elle affiche lui est donné par le Replay
 * annoté du paquet Go. Les documents montés ici sont donc des `Annotated`
 * écrits à la main, dans la forme exacte que la liaison Wails rend.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

// La langue par défaut des tests est l'anglais : les info-bulles sont
// comparées au catalogue lui-même, jamais à une phrase recopiée à la main.
import en from '../i18n/locales/en.json';
import TranscriptView, { transcriptRows } from '../components/TranscriptView.svelte';

const POSITION = (dice = [0, 0], cubeValue = 0) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: cubeValue },
    dice,
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

/**
 * Un document annoté : les Actions telles que le document les porte, les
 * ActionInfo telles que le Replay les rend, et les parties qu'il dérive.
 */
function annotatedOf(rows, games) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: rows.map((r) => r.action), cursor: 0 },
        actions: rows.map((r, index) => ({
            index,
            side: r.action.side,
            kind: r.action.kind,
            before: r.before ?? POSITION(r.action.dice ?? [0, 0]),
            has_position: r.action.kind !== 'opening',
            notation: r.notation ?? '',
            game_index: r.game ?? 0,
            game_number: (r.game ?? 0) + 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: r.flaws ?? []
        })),
        games,
        next: { expects: 'checker', side: 0, position: POSITION(), crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

const GAME = (over = {}) => ({ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: 0, ...over });

/** Une partie ordinaire : ouverture, quatre coups, un double pris. */
const ORDINARY = annotatedOf(
    [
        { action: { side: 0, kind: 'opening', dice: [3, 1] } },
        { action: { side: 0, kind: 'checker', dice: [3, 1] }, notation: '8/5 6/5' },
        { action: { side: 1, kind: 'checker', dice: [5, 2] }, notation: '13/8 13/11' },
        { action: { side: 0, kind: 'checker', dice: [6, 4] }, notation: '24/14' },
        { action: { side: 1, kind: 'double' }, before: POSITION([0, 0], 0) },
        { action: { side: 0, kind: 'take' }, before: POSITION([0, 0], 1) },
        { action: { side: 1, kind: 'checker', dice: [4, 1] }, notation: '13/9 6/5' }
    ],
    [GAME({ last: 6 })]
);

const cells = (container) => [...container.querySelectorAll('.cell')].filter((el) => !el.classList.contains('empty-cell'));
const rowsOf = (container) => [...container.querySelectorAll('tbody tr')];
const columnOf = (cell) => [...cell.closest('tr').children].indexOf(cell.closest('td'));

beforeEach(() => vi.clearAllMocks());
afterEach(() => cleanup());

describe('la disposition en deux colonnes', () => {
    test('une cellule par Action, dans la colonne de son camp', () => {
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY } });

        // Sept Actions, sept cellules — l'ouverture comprise, plus rien.
        expect(cells(container).filter((c) => c.dataset.index !== undefined).length).toBe(7);

        const byIndex = (i) => container.querySelector(`[data-index="${i}"]`);
        // Colonne 0 = le numéro de tour, 1 = joueur 1, 2 = joueur 2.
        expect(columnOf(byIndex(1))).toBe(1); // le coup de Kévin, à gauche
        expect(columnOf(byIndex(2))).toBe(2); // celui d'Alice, à droite
        expect(columnOf(byIndex(4))).toBe(2); // le double d'Alice : dans SA colonne
        expect(columnOf(byIndex(5))).toBe(1); // la prise de Kévin, dans la sienne
    });

    test('une ligne par tour : les deux camps du même tour partagent la ligne', () => {
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY } });
        const line = (i) => rowsOf(container).indexOf(container.querySelector(`[data-index="${i}"]`).closest('tr'));

        expect(line(1)).toBe(line(2)); // 31: 8/5 6/5 | 52: 13/8 13/11
        expect(line(3)).toBe(line(4)); // 64: 24/14   | Double
        expect(line(3)).toBe(line(1) + 1);
        // L'ouverture appartient aux deux camps : elle prend sa ligne entière.
        expect(container.querySelector('[data-index="0"]').closest('td').getAttribute('colspan')).toBe('2');
    });

    test('les dés précèdent la notation, comme sur une feuille de match', () => {
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY } });
        expect(container.querySelector('[data-index="1"]').textContent.trim()).toBe('31: 8/5 6/5');
    });

    test('le videau porte la valeur atteinte, la prise et la passe leur mot', () => {
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY } });
        expect(container.querySelector('[data-index="4"]').textContent).toContain('2');
        expect(container.querySelector('[data-index="5"]').textContent.trim()).not.toBe('');
    });

    test('la fin de partie est une cellule de la colonne du vainqueur', () => {
        const won = annotatedOf(
            [{ action: { side: 0, kind: 'opening', dice: [3, 1] } }, { action: { side: 0, kind: 'checker', dice: [3, 1] }, notation: '8/5 6/5' }],
            [GAME({ winner: 1, points_won: 2, finished: true, last: 1 })]
        );
        const { container } = render(TranscriptView, { props: { annotated: won } });

        const result = cells(container).find((c) => c.dataset.index === undefined);
        expect(result).toBeTruthy();
        expect(columnOf(result)).toBe(2); // Alice a gagné : sa colonne
        expect(result.textContent).toContain('2');
        // Une ligne sans tour ne porte pas de numéro (comme le .mat).
        expect(result.closest('tr').querySelector('.num').textContent.trim()).toBe('');
    });

    test('transcriptRows range chaque Action dans sa partie', () => {
        const twoGames = annotatedOf(
            [
                { action: { side: 0, kind: 'opening', dice: [3, 1] }, game: 0 },
                { action: { side: 0, kind: 'checker', dice: [3, 1] }, notation: '8/5 6/5', game: 0 },
                { action: { side: 1, kind: 'opening', dice: [5, 2] }, game: 1 },
                { action: { side: 1, kind: 'checker', dice: [5, 2] }, notation: '13/8 13/11', game: 1 }
            ],
            [GAME({ number: 1, last: 1 }), GAME({ number: 2, initial_score: [1, 0], first: 2, last: 3 })]
        );
        const layout = transcriptRows(twoGames);
        expect(layout.length).toBe(2);
        expect(layout[1].rows.some((r) => r.right?.index === 3)).toBe(true);
        expect(layout[0].rows.some((r) => r.left?.index === 1)).toBe(true);
    });
});

describe('le Cursor est une cellule encadrée', () => {
    test('la cellule visée porte la marque, et elle seule', () => {
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY, cursor: 3 } });
        const framed = [...container.querySelectorAll('.cell.cursor')];
        expect(framed.length).toBe(1);
        expect(framed[0].dataset.index).toBe('3');
        expect(framed[0].getAttribute('aria-current')).toBe('true');
    });

    test('un clic sur une cellule rend son index à l’appelant', async () => {
        const onSelect = vi.fn();
        const { container } = render(TranscriptView, { props: { annotated: ORDINARY, cursor: 1, onSelect } });
        await fireEvent.click(container.querySelector('[data-index="4"]'));
        expect(onSelect).toHaveBeenCalledWith(4);
    });
});

describe('les Incohérences sont décorées et nommées', () => {
    const KINDS = ['illegal_move', 'double_turn', 'impossible_cube', 'past_end', 'inconsistent_dice'];

    test.each(KINDS)('%s a sa décoration et son info-bulle', (kind) => {
        const flawed = annotatedOf(
            [
                { action: { side: 0, kind: 'opening', dice: [3, 1] } },
                { action: { side: 0, kind: 'checker', dice: [3, 1] }, notation: '8/5 6/5', flaws: [{ kind, detail: 'whatever the engine says' }] }
            ],
            [GAME({ last: 1 })]
        );
        const { container } = render(TranscriptView, { props: { annotated: flawed } });

        const cell = container.querySelector('[data-index="1"]');
        expect(cell.classList.contains('flawed')).toBe(true);
        expect(cell.dataset.inconsistency).toBe(kind);
        expect(cell.getAttribute('title')).toContain(en.transcript.inconsistency[kind]);
    });

    test('deux Incohérences sur la même cellule sont toutes deux nommées', () => {
        const flawed = annotatedOf(
            [
                { action: { side: 0, kind: 'opening', dice: [3, 1] } },
                {
                    action: { side: 0, kind: 'checker', dice: [3, 1] },
                    notation: '8/5 6/5',
                    flaws: [
                        { kind: 'double_turn', detail: '' },
                        { kind: 'inconsistent_dice', detail: '' }
                    ]
                }
            ],
            [GAME({ last: 1 })]
        );
        const { container } = render(TranscriptView, { props: { annotated: flawed } });
        const title = container.querySelector('[data-index="1"]').getAttribute('title');
        expect(title).toContain(en.transcript.inconsistency.double_turn);
        expect(title).toContain(en.transcript.inconsistency.inconsistent_dice);
    });
});

describe('les parties sont repliables', () => {
    const twoGames = annotatedOf(
        [
            { action: { side: 0, kind: 'opening', dice: [3, 1] }, game: 0 },
            { action: { side: 0, kind: 'checker', dice: [3, 1] }, notation: '8/5 6/5', game: 0 },
            { action: { side: 1, kind: 'opening', dice: [5, 2] }, game: 1 },
            { action: { side: 1, kind: 'checker', dice: [5, 2] }, notation: '13/8 13/11', game: 1 }
        ],
        [GAME({ number: 1, last: 1 }), GAME({ number: 2, initial_score: [1, 0], first: 2, last: 3 })]
    );

    test('la partie du Cursor est ouverte, les autres repliées', () => {
        const { container } = render(TranscriptView, { props: { annotated: twoGames, cursor: 3 } });
        const sections = [...container.querySelectorAll('details.game')];
        expect(sections.map((s) => s.open)).toEqual([false, true]);
    });

    test('replier une partie ne déplace pas le Cursor', async () => {
        const onSelect = vi.fn();
        const { container } = render(TranscriptView, { props: { annotated: twoGames, cursor: 3, onSelect } });
        const second = [...container.querySelectorAll('details.game')][1];

        second.open = false;
        await fireEvent(second, new Event('toggle'));
        for (let i = 0; i < 4; i++) await tick();

        expect([...container.querySelectorAll('details.game')][1].open).toBe(false);
        // Le repli est un geste d'affichage : rien n'a été demandé à l'appelant,
        // et la cellule encadrée reste celle qu'il a désignée.
        expect(onSelect).not.toHaveBeenCalled();
        const { container: reopened } = render(TranscriptView, { props: { annotated: twoGames, cursor: 3 } });
        expect(reopened.querySelector('.cell.cursor').dataset.index).toBe('3');
    });

    test('déplier une partie repliée montre ses cellules', async () => {
        const { container } = render(TranscriptView, { props: { annotated: twoGames, cursor: 3 } });
        expect(container.querySelector('[data-index="1"]')).toBeNull();

        const first = [...container.querySelectorAll('details.game')][0];
        first.open = true;
        await fireEvent(first, new Event('toggle'));
        for (let i = 0; i < 4; i++) await tick();

        expect(container.querySelector('[data-index="1"]')).not.toBeNull();
    });
});
