/**
 * Un onglet livré après coup doit apparaître à SA place, y compris pour un
 * utilisateur dont l'ordre est déjà enregistré — et sans écraser son choix.
 * C'est la couture entre la liste canonique du code et la liste persistée.
 */
import { describe, test, expect } from 'vitest';
import { applyTabOrder, normalizeTabId } from '../services/tabOrder.js';

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

// #401 : l'onglet Eval s'appelait `epc`. Un ordre enregistré avant le
// renommage garde la place qu'on lui avait donnée, sous son nom actuel.
describe('normalizeTabId', () => {
    test('`epc` enregistré rouvre `eval`', () => {
        expect(normalizeTabId('epc')).toBe('eval');
    });

    test('tout autre identifiant revient tel quel', () => {
        expect(normalizeTabId('eval')).toBe('eval');
        expect(normalizeTabId('stats')).toBe('stats');
        expect(normalizeTabId(undefined)).toBe(undefined);
    });

    test('un ordre enregistré avec `epc` place Eval là où il était', () => {
        const defaults = [{ id: 'matches' }, { id: 'eval' }, { id: 'training' }, { id: 'stats' }];
        expect(applyTabOrder(defaults, ['stats', 'epc', 'matches', 'training']).map((t) => t.id)).toEqual(['stats', 'eval', 'matches', 'training']);
    });
});
