/**
 * La fiche de score (ADR-0040 règle 4) : un score non ordonné, ses deux faces,
 * et RIEN d'autre que les cases que les tables de référence définissent.
 *
 * Ce qui se teste ici est une géométrie, pas un rendu : combien de cases un
 * score donne, lesquelles, et avec quelle valeur. C'est la couture de
 * l'exercice Scores — le composant qui l'affiche n'a pas d'autre décision à
 * prendre — et elle se vérifie sans plateau, sans Wails et sans navigateur.
 */
import { describe, test, expect } from 'vitest';
import { SCORE_CARD_ROWS, UNORDERED_SCORES, buildScoreCard, scoreCardNumbers } from '../services/scoreCard.js';
import { takePoint2LiveTable } from '../stores/takePoint2LiveTable.js';
import { gammonValue4Table } from '../stores/gammonValue4Table.js';

describe('les sept lignes', () => {
    test('sont celles de la règle 4, dans son ordre', () => {
        expect(SCORE_CARD_ROWS).toEqual(['tp2.live', 'tp2.last', 'tp4.live', 'tp4.last', 'gv1', 'gv2', 'gv4']);
    });
});

describe('le vivier', () => {
    test('compte les 36 scores non ordonnés de 2 à 9 away', () => {
        expect(UNORDERED_SCORES).toHaveLength(36);
        for (const [a, b] of UNORDERED_SCORES) {
            expect(a).toBeGreaterThanOrEqual(2);
            expect(b).toBeLessThanOrEqual(9);
            expect(a).toBeLessThanOrEqual(b);
        }
        // Non ordonnés : aucun doublon, et 3:5 n'apparaît pas aussi comme 5:3.
        expect(new Set(UNORDERED_SCORES.map(([a, b]) => `${a}:${b}`)).size).toBe(36);
    });
});

describe('buildScoreCard', () => {
    test('à 2a-2a : une seule colonne et trois nombres', () => {
        const card = buildScoreCard(2, 2);
        expect(card.faces).toHaveLength(1);
        const numbers = scoreCardNumbers(card);
        expect(numbers.map((n) => n.type)).toEqual(['tp2.live', 'tp2.last', 'gv1']);
    });

    test('à 4a-4a : une seule colonne, six nombres — gv4 n’a pas d’objet à 4 away', () => {
        const card = buildScoreCard(4, 4);
        expect(card.faces).toHaveLength(1);
        expect(scoreCardNumbers(card).map((n) => n.type)).toEqual(['tp2.live', 'tp2.last', 'tp4.live', 'tp4.last', 'gv1', 'gv2']);
    });

    test('à 3a-5a : deux colonnes qui diffèrent, gv4 absent du côté 3 away', () => {
        const card = buildScoreCard(3, 5);
        expect(card.faces.map((f) => f.away)).toEqual([3, 5]);
        const numbers = scoreCardNumbers(card);
        const gv4 = numbers.filter((n) => n.type === 'gv4');
        expect(gv4).toHaveLength(1);
        expect(gv4[0].away).toBe(5);
        expect(numbers).toHaveLength(13);
    });

    test('quatorze nombres au maximum', () => {
        const counts = UNORDERED_SCORES.map(([a, b]) => scoreCardNumbers(buildScoreCard(a, b)).length);
        expect(Math.max(...counts)).toBe(14);
        expect(Math.min(...counts)).toBe(3);
    });

    test('aucune case « sans objet » : toute case rendue porte un nombre', () => {
        for (const [a, b] of UNORDERED_SCORES) {
            for (const number of scoreCardNumbers(buildScoreCard(a, b))) {
                expect(Number.isFinite(number.value), `${a}:${b} ${number.type}`).toBe(true);
            }
        }
    });

    test('une ligne dont aucune face n’a de case ne figure pas sur la fiche', () => {
        const card = buildScoreCard(2, 2);
        expect(card.rows.map((r) => r.type)).toEqual(['tp2.live', 'tp2.last', 'gv1']);
        for (const row of card.rows) expect(row.cells.some((c) => c !== null)).toBe(true);
    });

    test('les valeurs sont celles des tables de référence, lues à la bonne case', () => {
        // tp2 course longue : ligne = l’away de celui dont c’est le point de
        // prise, colonne = l’away de l’adversaire ; les deux tables commencent
        // à 2 away.
        const card = buildScoreCard(3, 7);
        const numbers = scoreCardNumbers(card);
        const mine = numbers.find((n) => n.type === 'tp2.live' && n.away === 3);
        expect(mine.value).toBe(takePoint2LiveTable[3 - 2][7 - 2]);
        // gv4 ne commence qu’à 5 away : la case du 7-away se lit à la ligne 2.
        const gv4 = numbers.find((n) => n.type === 'gv4');
        expect(gv4.away).toBe(7);
        expect(gv4.value).toBe(gammonValue4Table[7 - 5][3 - 2]);
    });

    test('le score est non ordonné : 3a-5a et 5a-3a donnent la même fiche', () => {
        const a = scoreCardNumbers(buildScoreCard(3, 5)).map((n) => `${n.type}/${n.away}=${n.value}`);
        const b = scoreCardNumbers(buildScoreCard(5, 3)).map((n) => `${n.type}/${n.away}=${n.value}`);
        expect(new Set(a)).toEqual(new Set(b));
    });
});
