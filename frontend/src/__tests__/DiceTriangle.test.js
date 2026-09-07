/**
 * DiceTriangle.test.js — T2.1 : les 21 jets, une case chacun.
 *
 * Ce qui est vérifié ici est la CIBLE : vingt et une cases et pas trente-six,
 * chacune rendant le jet qu'elle porte, les doubles sur la diagonale, et la
 * rangée des six dés de l'ouverture. Le budget de gestes, lui, est mesuré à
 * part (transcriptionKeys.mouse.test.js) : il se compte en clics et en touches,
 * pas en pixels.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import DiceTriangle from '../components/DiceTriangle.svelte';

const cells = () => [...document.querySelectorAll('button')];

afterEach(cleanup);

describe('le triangle des jets', () => {
    test('vingt et une cases, une par jet distinct, doubles compris', () => {
        render(DiceTriangle);
        const labels = cells().map((b) => b.textContent.trim());
        expect(labels).toHaveLength(21);
        // Aucun jet n'est offert deux fois : c'est tout l'objet du triangle
        // contre la grille de 36, où 3-1 et 1-3 sont deux cibles.
        expect(new Set(labels).size).toBe(21);
        // Les six doubles y sont, une fois chacun.
        expect(labels.filter((l) => l[0] === l[1]).sort()).toEqual(['11', '22', '33', '44', '55', '66']);
    });

    test('chaque case rend le jet qu-elle porte, dé fort d-abord', async () => {
        const onPick = vi.fn();
        render(DiceTriangle, { props: { onPick } });

        for (const cell of cells()) {
            onPick.mockClear();
            const [a, b] = cell.textContent.trim().split('').map(Number);
            await fireEvent.click(cell);
            expect(onPick).toHaveBeenCalledTimes(1);
            expect(onPick).toHaveBeenCalledWith(a, b);
            expect(a).toBeGreaterThanOrEqual(b);
        }
    });

    test('la case porte son jet en info-bulle, pour qui vise sans lire', () => {
        render(DiceTriangle);
        const [first] = cells();
        expect(first.getAttribute('title')).toContain('1');
        expect(first.getAttribute('aria-label')).toBe(first.getAttribute('title'));
    });

    // L'ouverture demande un dé par camp : une case du triangle donnerait une
    // paire, donc un second sens à la même cible. La rangée de six rend un dé.
    test('l-ouverture montre six dés, un clic rendant un seul dé', async () => {
        const onDie = vi.fn();
        const onPick = vi.fn();
        render(DiceTriangle, { props: { single: true, onDie, onPick } });

        const faces = cells();
        expect(faces.map((b) => b.textContent.trim())).toEqual(['1', '2', '3', '4', '5', '6']);

        await fireEvent.click(faces[4]);
        expect(onDie).toHaveBeenCalledWith(5);
        expect(onPick).not.toHaveBeenCalled();
    });
});
