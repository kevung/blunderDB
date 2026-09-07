/**
 * Un onglet livré après coup doit apparaître à SA place, y compris pour un
 * utilisateur dont l'ordre est déjà enregistré — et sans écraser son choix.
 * C'est la couture entre la liste canonique du code et la liste persistée.
 */
import { describe, test, expect } from 'vitest';
import { applyTabOrder } from '../services/tabOrder.js';

const DEFAULTS = [{ id: 'a' }, { id: 'b' }, { id: 'c' }, { id: 'd' }];

describe('applyTabOrder', () => {
    test('sans ordre enregistré, la liste canonique', () => {
        expect(applyTabOrder(DEFAULTS, null).map((t) => t.id)).toEqual(['a', 'b', 'c', 'd']);
        expect(applyTabOrder(DEFAULTS, []).map((t) => t.id)).toEqual(['a', 'b', 'c', 'd']);
    });

    test('respecte l’ordre choisi', () => {
        expect(applyTabOrder(DEFAULTS, ['d', 'c', 'b', 'a']).map((t) => t.id)).toEqual(['d', 'c', 'b', 'a']);
    });

    test('un onglet neuf s’insère après son voisin de gauche, pas à la fin', () => {
        expect(applyTabOrder(DEFAULTS, ['a', 'b', 'd']).map((t) => t.id)).toEqual(['a', 'b', 'c', 'd']);
    });

    test('et il suit ce voisin là où l’utilisateur l’a déplacé', () => {
        expect(applyTabOrder(DEFAULTS, ['d', 'b', 'a']).map((t) => t.id)).toEqual(['d', 'b', 'c', 'a']);
    });

    test('sans aucun voisin de gauche présent, il va à la fin plutôt que de disparaître', () => {
        expect(applyTabOrder(DEFAULTS, ['d']).map((t) => t.id)).toEqual(['d', 'a', 'b', 'c']);
    });

    test('un identifiant enregistré qui n’existe plus est ignoré', () => {
        expect(applyTabOrder(DEFAULTS, ['a', 'disparu', 'b', 'c', 'd']).map((t) => t.id)).toEqual(['a', 'b', 'c', 'd']);
    });
});
