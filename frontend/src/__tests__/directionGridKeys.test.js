// @ts-nocheck — données de test partielles : les props typées par Wails exigent des classes générées.
/**
 * Le clavier de la grille des tables : chiffres = table N, flèches entre cases, M / X.
 */

import { describe, test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import TableGrid from '../components/direction/TableGrid.svelte';
import { gridKeyAction } from '../services/directionGridKeys.js';
import { openDirectionIdStore } from '../stores/directionStore.js';
import { activeTabStore } from '../stores/uiStore.js';

const cell = (/** @type {number} */ table, running = true) =>
    running
        ? { table, free: false, unavailable: false, reserved: false, matchId: `M${table}`, a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', length: 7 }
        : { table, free: true, unavailable: false, reserved: false };

beforeEach(() => {
    activeTabStore.set('tournaments');
    openDirectionIdStore.set(1);
});

afterEach(() => {
    cleanup();
    vi.useRealTimers();
    openDirectionIdStore.set(null);
});

/** Place les éléments sur une grille de `cols` colonnes (jsdom ne met rien en page). */
function layout(/** @type {HTMLElement[]} */ els, cols) {
    els.forEach((el, i) => {
        const left = (i % cols) * 100;
        const top = Math.floor(i / cols) * 50;
        el.getBoundingClientRect = () => /** @type {DOMRect} */ ({ left, top, right: left + 100, bottom: top + 50, width: 100, height: 50, x: left, y: top, toJSON() {} });
    });
}

describe('gridKeyAction', () => {
    const els = () => {
        const list = [1, 2, 3, 4, 5].map(() => document.createElement('button'));
        layout(list, 3);
        return list;
    };
    const key = (/** @type {string} */ k, init = {}) => new KeyboardEvent('keydown', { key: k, ...init });

    test("gauche / droite suivent l'ordre de lecture, début / fin les extrémités", () => {
        const c = els();
        expect(gridKeyAction(key('ArrowRight'), c, c[1], true)?.focus).toBe(c[2]);
        expect(gridKeyAction(key('ArrowLeft'), c, c[1], true)?.focus).toBe(c[0]);
        expect(gridKeyAction(key('ArrowLeft'), c, c[0], true)).toBeNull();
        expect(gridKeyAction(key('End'), c, c[0], true)?.focus).toBe(c[4]);
        expect(gridKeyAction(key('Home'), c, c[3], true)?.focus).toBe(c[0]);
    });

    test("haut / bas gardent la colonne, et s'arrêtent au bord", () => {
        const c = els();
        expect(gridKeyAction(key('ArrowDown'), c, c[1], true)?.focus).toBe(c[4]);
        expect(gridKeyAction(key('ArrowDown'), c, c[0], true)?.focus).toBe(c[3]);
        expect(gridKeyAction(key('ArrowUp'), c, c[4], true)?.focus).toBe(c[1]);
        expect(gridKeyAction(key('ArrowUp'), c, c[1], true)).toBeNull();
        expect(gridKeyAction(key('ArrowDown'), c, c[4], true)).toBeNull();
    });

    test('M et X ne valent que sur un match en cours, jamais avec un modificateur', () => {
        const c = els();
        expect(gridKeyAction(key('m'), c, c[0], true)).toEqual({ move: true });
        expect(gridKeyAction(key('X'), c, c[0], true)).toEqual({ move: true });
        expect(gridKeyAction(key('m'), c, c[0], false)).toBeNull();
        expect(gridKeyAction(key('m', { ctrlKey: true }), c, c[0], true)).toBeNull();
    });
});

describe('la grille au clavier', () => {
    test('M ouvre la fiche sur le champ de table', async () => {
        const { getByTestId, findByTestId } = render(TableGrid, { props: { cells: [cell(1), cell(2)] } });
        const c = getByTestId('direction-table-2');
        c.focus();
        await fireEvent.keyDown(c, { key: 'm' });
        const field = await findByTestId('direction-result-move-table');
        await Promise.resolve();
        expect(document.activeElement).toBe(field);
    });

    test('→ passe à la case suivante, libre comprise', async () => {
        const { getByTestId } = render(TableGrid, { props: { cells: [cell(1), cell(2, false)] } });
        const first = getByTestId('direction-table-1');
        first.focus();
        await fireEvent.keyDown(first, { key: 'ArrowRight' });
        expect(document.activeElement).toBe(getByTestId('direction-table-2'));
    });

    test('un chiffre ouvre la fiche de la table N', async () => {
        const { getByTestId, queryByTestId } = render(TableGrid, { props: { cells: [cell(1), cell(2), cell(3)] } });
        await fireEvent.keyDown(window, { code: 'Digit2', key: '2' });
        expect(document.activeElement).toBe(getByTestId('direction-table-2'));
        expect(queryByTestId('direction-result-card')).not.toBeNull();
    });

    test('deux chiffres : le second, dans le délai, désigne la table 12', async () => {
        vi.useFakeTimers();
        const cells = Array.from({ length: 12 }, (_, i) => cell(i + 1));
        const { getByTestId } = render(TableGrid, { props: { cells } });
        await fireEvent.keyDown(window, { code: 'Digit1', key: '1' });
        // Un seul chiffre est ambigu (1, 10, 11, 12) : rien n'est encore ouvert.
        expect(document.activeElement).not.toBe(getByTestId('direction-table-1'));
        await fireEvent.keyDown(window, { code: 'Digit2', key: '2' });
        expect(document.activeElement).toBe(getByTestId('direction-table-12'));
    });

    test("le premier chiffre seul l'emporte après le délai", async () => {
        vi.useFakeTimers();
        const cells = Array.from({ length: 12 }, (_, i) => cell(i + 1));
        const { getByTestId } = render(TableGrid, { props: { cells } });
        await fireEvent.keyDown(window, { code: 'Digit1', key: '1' });
        vi.advanceTimersByTime(450);
        expect(document.activeElement).toBe(getByTestId('direction-table-1'));
    });

    test('les chiffres restent au champ de saisie', async () => {
        const { container, getByTestId } = render(TableGrid, { props: { cells: [cell(1), cell(2)] } });
        const input = document.createElement('input');
        container.appendChild(input);
        input.focus();
        await fireEvent.keyDown(input, { code: 'Digit2', key: '2' });
        expect(document.activeElement).toBe(input);
        expect(getByTestId('direction-table-2')).toBeTruthy();
    });
});
