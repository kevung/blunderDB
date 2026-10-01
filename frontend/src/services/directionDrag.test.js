import { describe, it, expect } from 'vitest';
import { pastThreshold, canGrab, canDrop, dropAction, DRAG_THRESHOLD_PX } from './directionDrag.js';

describe('directionDrag', () => {
    it('un appui qui bouge à peine reste un clic', () => {
        expect(pastThreshold({ x: 0, y: 0 }, { x: 2, y: 2 })).toBe(false);
        expect(pastThreshold({ x: 0, y: 0 }, { x: DRAG_THRESHOLD_PX, y: 0 })).toBe(true);
    });

    it('on saisit une case occupée, jamais une libre', () => {
        expect(canGrab({ matchId: 'm1' })).toBe(true);
        expect(canGrab({})).toBe(false);
    });

    it('un match sans table ou une table partagée ne reçoit pas', () => {
        expect(canDrop({ table: 3 })).toBe(true);
        expect(canDrop({ table: 0, noTable: true })).toBe(false);
        expect(canDrop({ table: 3, shared: true })).toBe(false);
    });

    it('déposé sur une case libre, le match se déplace', () => {
        expect(dropAction({ matchId: 'm1', table: 1 }, { table: 4 })).toEqual({ kind: 'move', matchId: 'm1', table: 4 });
    });

    it("déposé sur une case occupée, les deux matchs s'échangent", () => {
        expect(dropAction({ matchId: 'm1', table: 1 }, { matchId: 'm2', table: 2 })).toEqual({ kind: 'swap', matchId: 'm1', table: 2, from: 1, to: 2 });
    });

    it("déposé sur lui-même, ou hors d'une case, rien ne se passe", () => {
        expect(dropAction({ matchId: 'm1', table: 1 }, { matchId: 'm1', table: 1 })).toEqual({ kind: 'none' });
        expect(dropAction({ matchId: 'm1', table: 1 }, null)).toEqual({ kind: 'none' });
        expect(dropAction({ matchId: 'm1', table: 1 }, { table: 0, noTable: true, matchId: 'm9' })).toEqual({ kind: 'none' });
    });

    it("la table hors service n'est pas refusée ici : la règle est celle du service", () => {
        expect(dropAction({ matchId: 'm1', table: 1 }, { table: 3 }).kind).toBe('move');
    });
});
