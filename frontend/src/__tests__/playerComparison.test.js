import { describe, expect, it } from 'vitest';
import { blunderRate, compareRows } from '../services/playerComparison.js';

/** Une ligne de la table Joueurs, avec ce qu'on veut dessus. */
function row(over = {}) {
    return {
        name: 'X',
        matches: 10,
        wins: 6,
        losses: 4,
        decisions: 200,
        checker_decisions: 180,
        cube_decisions: 20,
        pr: 5,
        pr_checker: 4.5,
        pr_cube: 8,
        snowie_er: 6,
        blunders: 10,
        luck_known: true,
        luck_rate_mp: 1.2,
        ...over
    };
}

const lineOf = (lines, key) => lines.find((l) => l.key === key);

describe('playerComparison — ce qui reçoit un verdict', () => {
    it('donne la ligne au plus petit PR', () => {
        const lines = compareRows(row({ pr: 4 }), row({ pr: 7 }));
        expect(lineOf(lines, 'pr').better).toBe('a');
        expect(lineOf(lines, 'pr').a).toBe('4.00');
    });

    it("ne départage pas une égalité : ce n'est pas une victoire", () => {
        const lines = compareRows(row({ pr: 5 }), row({ pr: 5 }));
        expect(lineOf(lines, 'pr').better).toBeNull();
    });

    it('ne départage pas quand un des deux taux n’est pas mesuré', () => {
        const lines = compareRows(row({ cube_decisions: 0 }), row());
        const l = lineOf(lines, 'pr_cube');
        expect(l.better).toBeNull();
        expect(l.a).toBe('—');
    });
});

describe('playerComparison — ce qui n’en reçoit jamais', () => {
    it('ne fait pas gagner le plus chanceux', () => {
        const lines = compareRows(row({ luck_rate_mp: 9 }), row({ luck_rate_mp: -9 }));
        const l = lineOf(lines, 'luck');
        expect(l.better).toBeNull();
        expect(l.kind).toBe('context');
        expect(l.a).toBe('+9.0');
        expect(l.b).toBe('−9.0');
    });

    it('ne fait pas gagner celui qui a le moins joué', () => {
        for (const key of ['matches', 'record', 'decisions']) {
            const lines = compareRows(row({ matches: 1, decisions: 10, wins: 1, losses: 0 }), row());
            expect(lineOf(lines, key).better, key).toBeNull();
        }
    });

    it('ne compare pas des comptes de blunders bruts', () => {
        // Douze blunders sur mille décisions valent mieux que dix sur cent ;
        // le compte brut dirait l'inverse.
        const many = row({ blunders: 12, decisions: 1000 });
        const few = row({ blunders: 10, decisions: 100 });
        expect(lineOf(compareRows(many, few), 'blunders').better).toBeNull();
        expect(lineOf(compareRows(many, few), 'blunder_rate').better).toBe('a');
    });
});

describe('playerComparison — le taux de blunders', () => {
    it('compte pour cent décisions', () => {
        expect(blunderRate(row({ blunders: 10, decisions: 200 }))).toBe(5);
    });

    it("n'existe pas sans décision comptée", () => {
        expect(blunderRate(row({ decisions: 0 }))).toBeNull();
        expect(blunderRate(null)).toBeNull();
    });
});

describe('playerComparison — la forme rendue', () => {
    it('rend les mêmes lignes, dans le même ordre, quels que soient les joueurs', () => {
        const keys = compareRows(row(), row()).map((l) => l.key);
        expect(keys).toEqual(['matches', 'record', 'decisions', 'pr', 'pr_checker', 'pr_cube', 'snowie_er', 'blunder_rate', 'blunders', 'luck']);
        expect(compareRows(null, null).map((l) => l.key)).toEqual(keys);
    });

    it('survit à deux lignes vides sans inventer de verdict', () => {
        for (const l of compareRows(null, null)) expect(l.better).toBeNull();
    });
});
