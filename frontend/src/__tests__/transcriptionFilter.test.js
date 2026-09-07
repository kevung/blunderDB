/**
 * transcriptionFilter.test.js — T2.2 : le filtre par point de départ.
 *
 * Le filtre est pur, et le budget d'ux.md §4.1 se mesure ici, en gestes : le
 * rang douze coûte treize touches au clavier (3,64 s) ; un clic sur le point de
 * départ, puis la fin au clavier, doit rester sous les trois secondes. Les deux
 * derniers tests COMPTENT les gestes et échouent au-delà.
 *
 * Ce que la mesure corrige au passage : un seul clic ne ramène pas toujours le
 * coup dans les trois premiers — sur le jet type, le rang douze descend au
 * quatrième — mais le budget tient quand même, et un second point suffit
 * lorsqu'un coup part de deux points différents.
 */

import { describe, test, expect } from 'vitest';
import { sourcesOf, filterByPoints, nextFilter } from '../services/transcriptionFilter.js';

const play = (notation, froms) => ({ move: { move: notation }, steps: froms.map((from) => ({ from, to: from - 1 })) });

// Un jet 3-1 vu du camp au trait : dix-sept coups légaux, dont le douzième est
// celui qui a été joué. Seuls les points de départ comptent ici.
const CANDIDATES = [
    play('8/5 6/5', [8, 6]),
    play('13/10 24/23', [13, 24]),
    play('24/21 6/5', [24, 6]),
    play('13/10 6/5', [13, 6]),
    play('8/5 24/23', [8, 24]),
    play('13/10 13/12', [13, 13]),
    play('24/21 24/23', [24, 24]),
    play('8/7 8/5', [8, 8]),
    play('13/12 13/10', [13, 13]),
    play('6/3 6/5', [6, 6]),
    play('24/23 13/10', [24, 13]),
    play('8/5 8/7', [8, 8]),
    play('6/3 8/7', [6, 8]),
    play('13/10 8/7', [13, 8]),
    play('24/21 13/12', [24, 13]),
    play('6/5 13/10', [6, 13]),
    play('8/7 13/10', [8, 13])
];

describe('le filtre par point de départ', () => {
    test('sans point, la liste entière : un filtre vide est l’absence de filtre', () => {
        expect(filterByPoints(CANDIDATES, [])).toHaveLength(CANDIDATES.length);
        expect(filterByPoints(CANDIDATES, null)).toHaveLength(CANDIDATES.length);
    });

    test('un point ne garde que les coups qui en partent', () => {
        const kept = filterByPoints(CANDIDATES, [24]);
        expect(kept.length).toBeGreaterThan(0);
        expect(kept.length).toBeLessThan(CANDIDATES.length);
        for (const candidate of kept) expect(candidate.steps.some((s) => s.from === 24)).toBe(true);
    });

    // « Un second clic sur un autre point la réduit encore » : ET, jamais OU.
    test('un second point réduit encore, il n’élargit pas', () => {
        const one = filterByPoints(CANDIDATES, [24]);
        const two = filterByPoints(CANDIDATES, [24, 13]);
        expect(two.length).toBeLessThanOrEqual(one.length);
        for (const candidate of two) {
            expect(candidate.steps.some((s) => s.from === 24)).toBe(true);
            expect(candidate.steps.some((s) => s.from === 13)).toBe(true);
        }
    });

    test('un point d’où rien ne part rend une liste vide, sans erreur', () => {
        expect(filterByPoints(CANDIDATES, [17])).toEqual([]);
        expect(filterByPoints([], [8])).toEqual([]);
        expect(filterByPoints(undefined, [8])).toEqual([]);
    });

    test('les points de départ offerts sont ceux des candidats', () => {
        expect(sourcesOf(CANDIDATES)).toEqual(new Set([8, 6, 13, 24]));
        expect(sourcesOf([])).toEqual(new Set());
    });

    describe('le clic suivant', () => {
        test('ajoute un point d’où part au moins un candidat', () => {
            expect(nextFilter([], 24, CANDIDATES)).toEqual([24]);
            expect(nextFilter([24], 13, CANDIDATES)).toEqual([24, 13]);
        });

        test('enlève un point déjà filtré — l’annulation sur la cible même', () => {
            expect(nextFilter([24, 13], 24, CANDIDATES)).toEqual([13]);
            expect(nextFilter([24], 24, CANDIDATES)).toEqual([]);
        });

        // Le geste reste disponible pour le déplacement libre de pions (T2.4).
        test('ne fait rien quand il ne resterait aucun candidat', () => {
            expect(nextFilter([], 17, CANDIDATES)).toBeNull();
            expect(nextFilter([6], 24, CANDIDATES.slice(0, 1))).toBeNull();
        });
    });
});

describe('le budget d’ux.md §4.1, ligne « coup loin dans la liste »', () => {
    // ux.md §1 : K = 0,28 s, B = 0,10 s, H = 0,40 s. P d'un point du damier,
    // mesuré à la loi de Fitts (b = 0,15 s/bit) sur une cible de 45 px à 300 px.
    const K = 0.28;
    const B = 0.1;
    const H = 0.4;
    const P_POINT = 0.15 * Math.log2(300 / 45 + 1);

    /** Le rang du coup joué dans une liste, ou -1. */
    const rankOf = (list, notation) => list.findIndex((c) => c.move.move === notation);

    test('le rang douze au clavier coûte treize touches', () => {
        const rank = rankOf(CANDIDATES, '8/5 8/7');
        expect(rank).toBe(11);
        const keystrokes = 2 + rank; // les deux dés, puis onze `j`
        expect(keystrokes).toBe(13);
        expect(keystrokes * K).toBeCloseTo(3.64, 2);
    });

    // Le geste que la fiche demande : cliquer le point de départ du premier pas
    // connu, puis finir au clavier. La main reste sur la souris entre deux
    // clics, donc un seul H, et un P par clic.
    const cost = (clicks, ranks) => 2 * K + H + clicks * (P_POINT + 2 * B) + ranks * K;

    /** Les points de départ distincts d'un coup, dans l'ordre de ses pas. */
    const departures = (candidate) => [...new Set(candidate.steps.map((s) => s.from))];

    test('le même coup, filtré par son point de départ, tient sous les trois secondes', () => {
        const filtered = filterByPoints(CANDIDATES, [8]);
        const rank = rankOf(filtered, '8/5 8/7');
        // Le rang tombe de douze à quatre : c'est le gain, et il est réel même
        // quand il ne descend pas jusqu'au premier de la liste.
        expect(rank).toBeLessThan(rankOf(CANDIDATES, '8/5 8/7'));
        expect(cost(1, rank)).toBeLessThanOrEqual(3);
        expect(cost(1, rank)).toBeLessThan(13 * K);
    });

    // Le budget est une propriété du dessin, pas d'un coup choisi : aucun
    // candidat, filtré par les points d'où il part, ne dépasse les trois
    // secondes — et le test COMPTE les gestes, il ne les suppose pas.
    test('aucun coup ne dépasse son budget une fois filtré', () => {
        for (const candidate of CANDIDATES) {
            const points = [];
            let ranks = 0;
            let clicks = 0;
            for (const point of departures(candidate)) {
                if (clicks && ranks <= 2) break; // assez réduit, on finit au clavier
                points.push(point);
                clicks += 1;
                ranks = rankOf(filterByPoints(CANDIDATES, points), candidate.move.move);
                expect(ranks).toBeGreaterThanOrEqual(0);
            }
            expect(clicks).toBeLessThanOrEqual(2);
            expect(cost(clicks, ranks)).toBeLessThanOrEqual(3);
        }
    });
});
