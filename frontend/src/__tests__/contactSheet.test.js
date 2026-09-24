/**
 * contactSheet.test.js — la planche-contact de la liste parcourue (#287).
 *
 * Deux choses se vérifient ici : le module (pages, clavier, garde
 * d'ouverture) et le composant MONTÉ — une grille qui ne se monte pas ne se
 * voit dans aucun test de logique.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadPositionsByIDs: vi.fn(async () => [])
}));

import { PAGE_SIZE, pageOf, pageCount, pageBounds, columnsFor, targetIndex, openContactSheet } from '../services/contactSheet.js';
import ContactSheetModal from '../components/ContactSheetModal.svelte';
import { positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { currentPositionIndexStore, statusBarModeStore, activeModal, MODAL } from '../stores/uiStore.js';

function position(id) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    points[6] = { checkers: 5, color: 0 };
    points[19] = { checkers: 5, color: 1 };
    return { id, board: { points, bearoff: [10, 10] }, cube: { owner: -1, value: 0 }, dice: [3, 1], score: [-1, -1], player_on_roll: 0, decision_type: 0 };
}

const key = (k, extra = {}) => ({ key: k, ...extra });

describe('pages et clavier', () => {
    test('découpe la liste en pages', () => {
        expect(pageOf(0)).toBe(0);
        expect(pageOf(PAGE_SIZE)).toBe(1);
        expect(pageCount(0)).toBe(0);
        expect(pageCount(PAGE_SIZE + 1)).toBe(2);
        expect(pageBounds(1, PAGE_SIZE + 5)).toEqual({ from: PAGE_SIZE, to: PAGE_SIZE + 5 });
    });

    test('compte les colonnes, quatre quand la largeur est inconnue', () => {
        expect(columnsFor(0, 150, 8)).toBe(4);
        expect(columnsFor(632, 150, 8)).toBe(4);
        expect(columnsFor(100, 150, 8)).toBe(1);
    });

    test('se déplace dans la grille sans jamais en sortir', () => {
        const grid = { columns: 4, total: 30 };
        expect(targetIndex(0, key('ArrowRight'), grid)).toBe(1);
        expect(targetIndex(0, key('ArrowLeft'), grid)).toBe(0);
        expect(targetIndex(1, key('ArrowDown'), grid)).toBe(5);
        expect(targetIndex(28, key('ArrowDown'), grid)).toBe(28);
        expect(targetIndex(5, key('ArrowUp'), grid)).toBe(1);
        expect(targetIndex(2, key('j'), grid)).toBe(3);
        expect(targetIndex(2, key('k'), grid)).toBe(1);
        expect(targetIndex(5, key('End'), grid)).toBe(PAGE_SIZE - 1);
        expect(targetIndex(PAGE_SIZE + 2, key('Home'), grid)).toBe(PAGE_SIZE);
        expect(targetIndex(2, key('PageDown'), grid)).toBe(PAGE_SIZE + 2);
        expect(targetIndex(29, key('PageDown'), grid)).toBe(29);
        expect(targetIndex(2, key('PageUp'), grid)).toBe(0);
    });

    test('laisse passer ce qui ne la concerne pas', () => {
        const grid = { columns: 4, total: 30 };
        expect(targetIndex(2, key('Enter'), grid)).toBe(null);
        expect(targetIndex(2, key('j', { ctrlKey: true }), grid)).toBe(null);
        expect(targetIndex(2, key('J', { shiftKey: true }), grid)).toBe(null);
        expect(targetIndex(0, key('ArrowRight'), { columns: 4, total: 0 })).toBe(null);
    });
});

describe("garde d'ouverture", () => {
    beforeEach(() => {
        activeModal.set(null);
        databasePathStore.set('/tmp/x.db');
        statusBarModeStore.set('NORMAL');
        positionsStore.set([position(1)]);
    });

    test('ouvre la planche sur une liste', () => {
        expect(openContactSheet()).toBe(true);
        expect(get(activeModal)).toBe(MODAL.CONTACT_SHEET);
    });

    test.each(['EDIT', 'MATCH'])('refuse en mode %s', (mode) => {
        statusBarModeStore.set(mode);
        expect(openContactSheet()).toBe(false);
        expect(get(activeModal)).toBe(null);
    });

    test('refuse une liste vide', () => {
        positionsStore.set([]);
        expect(openContactSheet()).toBe(false);
    });

    test('refuse sans base', () => {
        databasePathStore.set('');
        expect(openContactSheet()).toBe(false);
    });
});

describe('la planche montée', () => {
    beforeEach(() => {
        positionsStore.set(Array.from({ length: 30 }, (_, i) => position(i + 1)));
        currentPositionIndexStore.set(2);
    });
    afterEach(() => cleanup());

    test('montre la page de la position courante, et la dessine', async () => {
        const { container } = render(ContactSheetModal, { visible: true, onClose: () => {} });
        const tiles = container.querySelectorAll('.tile');
        expect(tiles.length).toBe(PAGE_SIZE);
        expect(container.querySelector('.tile.current')?.getAttribute('data-index')).toBe('2');
        await waitFor(() => expect(container.querySelectorAll('.tile img').length).toBe(PAGE_SIZE), { timeout: 5000 });
        expect(container.querySelector('.tile img')?.getAttribute('src')).toMatch(/^data:image\/svg\+xml/);
    });

    test('se parcourt au clavier et ouvre la vignette choisie', async () => {
        const onClose = vi.fn();
        const { container } = render(ContactSheetModal, { visible: true, onClose });
        await waitFor(() => expect(document.activeElement?.getAttribute('data-index')).toBe('2'));
        await fireEvent.keyDown(document.activeElement, { key: 'ArrowRight' });
        await waitFor(() => expect(document.activeElement?.getAttribute('data-index')).toBe('3'));
        // Une seule vignette dans l'ordre de tabulation : celle qui a le focus.
        expect(container.querySelectorAll('.tile[tabindex="0"]').length).toBe(1);

        await fireEvent.keyDown(document.activeElement, { key: 'PageDown' });
        await waitFor(() => expect(document.activeElement?.getAttribute('data-index')).toBe(String(PAGE_SIZE + 3)));
        expect(container.querySelectorAll('.tile').length).toBe(30 - PAGE_SIZE);

        await fireEvent.click(document.activeElement);
        expect(get(currentPositionIndexStore)).toBe(PAGE_SIZE + 3);
        expect(onClose).toHaveBeenCalled();
    });
});
