import { describe, expect, it } from 'vitest';
import { OFF, alivePlays, applyStep, completedPlay, destinationsFrom, newPlay, playHop, resetPlay, selectSource, sources, undoLast } from '../services/quizPlay.js';

const BLACK = 0;
const WHITE = 1;
const NONE = -1;

/**
 * Un plateau vide, où l'on pose des piles par point.
 * @param {Record<number, [number, number]>} stacks
 */
function board(stacks) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: NONE }));
    for (const [pt, [n, color]] of Object.entries(stacks)) {
        points[Number(pt)] = { checkers: n, color };
    }
    return { points, bearoff: [0, 0] };
}

/**
 * @param {Record<number, [number, number]>} stacks
 * @param {number} [mover]
 */
function position(stacks, mover = BLACK) {
    return { board: board(stacks), player_on_roll: mover };
}

/**
 * Un coup, tel que `App.LegalMoves` le rend.
 * @param {[number, number][]} steps
 * @param {string} [notation]
 */
function play(steps, notation = 'x') {
    return { steps: steps.map(([from, to]) => ({ from, to, hit: false })), notation, result: {} };
}

describe('quizPlay — ce que le moteur offre, et rien de plus', () => {
    it("n'accepte pas un pas qu'aucun coup légal ne propose", () => {
        const pos = position({ 13: [5, BLACK] });
        const state = newPlay(pos, [play([[13, 11]])]);
        expect(playHop(state, 13, 10)).toBe(state);
        expect(playHop(state, 13, 11).steps).toHaveLength(1);
    });

    it('ne sélectionne pas un point qui ne peut rien donner', () => {
        const pos = position({ 13: [5, BLACK], 6: [2, BLACK] });
        const state = newPlay(pos, [play([[13, 11]])]);
        expect(selectSource(state, 6)).toBe(state);
        expect(selectSource(state, 13).selected).toBe(13);
    });

    it('déselectionne au second clic sur la même source', () => {
        const pos = position({ 13: [5, BLACK] });
        const state = selectSource(newPlay(pos, [play([[13, 11]])]), 13);
        expect(selectSource(state, 13).selected).toBeNull();
    });
});

describe("quizPlay — l'ordre des pas appartient au joueur", () => {
    // Le cœur de la fiche : `LegalMoves` déduplique par position résultante,
    // donc « 24/23 13/11 » n'est rendu que dans UN ordre. Le joueur qui joue
    // l'autre ne doit pas être bloqué par cet artefact.
    const pos = position({ 13: [5, BLACK], 24: [2, BLACK] });
    const plays = [
        play(
            [
                [13, 11],
                [24, 23]
            ],
            '13/11 24/23'
        )
    ];

    it("accepte l'ordre que le moteur a rendu", () => {
        let s = newPlay(pos, plays);
        s = playHop(s, 13, 11);
        s = playHop(s, 24, 23);
        expect(completedPlay(s)?.notation).toBe('13/11 24/23');
    });

    it("accepte l'ordre inverse, que le moteur n'a pas rendu", () => {
        let s = newPlay(pos, plays);
        s = playHop(s, 24, 23);
        expect(alivePlays(s)).toHaveLength(1);
        s = playHop(s, 13, 11);
        expect(completedPlay(s)?.notation).toBe('13/11 24/23');
    });

    it("garde l'ordre contraint là où le plateau le contraint", () => {
        // Un enchaînement 13/11/8 : 11/8 est impossible tant qu'aucun pion
        // n'est en 11. Rien ne l'interdit explicitement — c'est le plateau.
        const chained = [
            play(
                [
                    [13, 11],
                    [11, 8]
                ],
                '13/8'
            )
        ];
        let s = newPlay(position({ 13: [5, BLACK] }), chained);
        expect(playHop(s, 11, 8)).toBe(s);
        s = playHop(s, 13, 11);
        expect(playHop(s, 11, 8).steps).toHaveLength(2);
    });

    it('un pion à la barre entre avant tout autre pas', () => {
        const pos = { ...position({ 25: [1, BLACK], 13: [5, BLACK] }), dice: [4, 2] };
        const plays = [
            play([
                [25, 23],
                [13, 9]
            ])
        ];
        let s = newPlay(pos, plays);
        expect(playHop(s, 13, 9)).toBe(s);
        expect(sources(s)).toEqual(new Set([25]));
        s = playHop(s, 25, 23);
        s = playHop(s, 13, 9);
        expect(completedPlay(s)).not.toBeNull();
    });

    it('une sortie attend que tous les pions soient dans le jan, dans le même coup', () => {
        // 5-3 : 8/3 rentre le dernier pion, puis 3/off.
        const pos = { ...position({ 8: [1, BLACK], 3: [2, BLACK] }), dice: [5, 3] };
        const plays = [
            play([
                [8, 3],
                [3, OFF]
            ])
        ];
        let s = newPlay(pos, plays);
        expect(playHop(s, 3, OFF)).toBe(s);
        s = playHop(s, 8, 3);
        s = playHop(s, 3, OFF);
        expect(completedPlay(s)).not.toBeNull();
    });

    it('une sortie d’un dé plus fort attend qu’aucun pion ne reste plus loin', () => {
        // 6-2 : 3/off du 6 n'est légal qu'une fois le 5 vidé par 5/3.
        const pos = { ...position({ 5: [1, BLACK], 3: [1, BLACK] }), dice: [6, 2] };
        const plays = [
            play([
                [5, 3],
                [3, OFF]
            ])
        ];
        let s = newPlay(pos, plays);
        expect(playHop(s, 3, OFF)).toBe(s);
        s = playHop(s, 5, 3);
        s = playHop(s, 3, OFF);
        expect(completedPlay(s)).not.toBeNull();
    });

    it('Blanc : même règle, de l’autre côté du plateau', () => {
        const pos = { ...position({ 0: [1, WHITE], 12: [5, WHITE] }, WHITE), dice: [4, 2] };
        const plays = [
            play([
                [0, 2],
                [12, 16]
            ])
        ];
        const s = newPlay(pos, plays);
        expect(playHop(s, 12, 16)).toBe(s);
        expect(playHop(s, 0, 2).steps).toHaveLength(1);
    });
});

describe('quizPlay — ce que le plateau montre', () => {
    it('déplace le pion, et vide le point qu’il quitte', () => {
        const b = board({ 13: [1, BLACK] });
        const next = applyStep(b, { from: 13, to: 11 }, BLACK);
        expect(next.points[13]).toEqual({ checkers: 0, color: NONE });
        expect(next.points[11]).toEqual({ checkers: 1, color: BLACK });
        expect(b.points[13].checkers).toBe(1); // l'original n'a pas bougé
    });

    it('envoie le blot frappé sur la barre de son propriétaire', () => {
        const b = board({ 13: [1, BLACK], 11: [1, WHITE] });
        const next = applyStep(b, { from: 13, to: 11 }, BLACK);
        expect(next.points[11]).toEqual({ checkers: 1, color: BLACK });
        expect(next.points[0]).toEqual({ checkers: 1, color: WHITE });
    });

    it('compte le pion sorti dans le bon plateau de sortie', () => {
        const b = board({ 3: [1, BLACK] });
        const next = applyStep(b, { from: 3, to: OFF }, BLACK);
        expect(next.bearoff[BLACK]).toBe(1);
        expect(next.points[3].checkers).toBe(0);
    });
});

describe('quizPlay — revenir en arrière', () => {
    const pos = position({ 13: [5, BLACK], 24: [2, BLACK] });
    const plays = [
        play([
            [13, 11],
            [24, 23]
        ])
    ];

    it('annule le dernier pas et rend le plateau qui allait avec', () => {
        let s = newPlay(pos, plays);
        s = playHop(s, 13, 11);
        s = playHop(s, 24, 23);
        s = undoLast(s, pos);
        expect(s.steps).toEqual([{ from: 13, to: 11 }]);
        expect(s.board.points[24].checkers).toBe(2);
        expect(s.board.points[11].checkers).toBe(1);
    });

    it('remet tout à la question posée', () => {
        let s = newPlay(pos, plays);
        s = playHop(s, 13, 11);
        s = resetPlay(s, pos);
        expect(s.steps).toEqual([]);
        expect(s.board.points[13].checkers).toBe(5);
        expect(s.selected).toBeNull();
    });
});

describe('quizPlay — ce que l’interface met en avant', () => {
    it('offre les sources, puis les destinations de la source choisie', () => {
        const pos = position({ 13: [5, BLACK], 24: [2, BLACK] });
        const s = newPlay(pos, [
            play([
                [13, 11],
                [24, 23]
            ]),
            play([
                [13, 11],
                [13, 12]
            ])
        ]);
        expect([...sources(s)].sort()).toEqual([13, 24]);
        expect([...destinationsFrom(s, 13)].sort()).toEqual([11, 12]);
        expect([...destinationsFrom(s, 6)]).toEqual([]);
    });

    it("n'a plus rien à offrir quand le coup est complet", () => {
        const pos = position({ 13: [5, BLACK] });
        let s = newPlay(pos, [play([[13, 11]])]);
        s = playHop(s, 13, 11);
        expect([...sources(s)]).toEqual([]);
        expect(completedPlay(s)).not.toBeNull();
    });

    it("ne dit pas complet tant qu'il manque un pas", () => {
        const pos = position({ 13: [5, BLACK], 24: [2, BLACK] });
        let s = newPlay(pos, [
            play([
                [13, 11],
                [24, 23]
            ])
        ]);
        s = playHop(s, 13, 11);
        expect(completedPlay(s)).toBeNull();
    });
});
