import { describe, test, expect } from 'vitest';
import { dropAction } from '../services/directionDrag.js';

describe('dropAction', () => {
    test('deux matchs à leur table : un échange à confirmer', () => {
        expect(dropAction({ matchId: 'm1', table: 3 }, { matchId: 'm2', table: 7 })).toEqual({ kind: 'swap', matchId: 'm1', table: 7, from: 3, to: 7 });
    });

    test("un match sans table sur une table occupée n'est pas un échange « Table 0 ↔ N »", () => {
        expect(dropAction({ matchId: 'm1', table: 0, noTable: true }, { matchId: 'm2', table: 7 })).toEqual({ kind: 'move', matchId: 'm1', table: 7 });
    });
});
