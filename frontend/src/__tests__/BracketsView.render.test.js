/**
 * BracketsView.render.test.js
 *
 * L'arbre est dessiné en SVG avec ses traits, les poules en croisé, un tableau non tiré en
 * squelette ; cliquer une place ouvre la fiche de résultat (match en cours) ou la correction
 * (match fini). Le composant est monté : c'est là que se perdent les props et les runes.
 */

import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import BracketsView from '../components/direction/BracketsView.svelte';
import { layoutSection, skeletonSection } from '../components/direction/bracketLayout.js';

afterEach(cleanup);

/** @param {string} key @param {number} round @param {Record<string, any>} [extra] */
const place = (key, round, extra = {}) => ({ key, label: { kind: 'round' }, round, length: 5, done: false, running: false, ...extra });

const main = {
    name: 'Principal',
    kind: 'main',
    rounds: 2,
    matches: [
        place('m0', 0, { a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', matchId: 'x1', done: true, winner: 'a', scoreA: 5, scoreB: 2 }),
        place('m1', 0, { a: 'c', b: 'd', aName: 'Carl', bName: 'Dora', matchId: 'x2', running: true }),
        place('m2', 1, {
            feeds: [
                { side: 0, key: 'm0' },
                { side: 1, key: 'm1' }
            ]
        })
    ]
};

/** @param {Record<string, any>} [over] */
const phase = (over = {}) => ({ index: 0, kind: 'bracket', current: true, drawn: true, sections: [main], config: {}, ...over });

describe('l’arbre des rencontres', () => {
    test('dessine les places et un trait par source', () => {
        const { container } = render(BracketsView, { props: { phases: [phase()] } });
        expect(container.querySelector('svg.graph')).toBeTruthy();
        expect(container.querySelectorAll('[data-testid="bracket-place"]').length).toBe(3);
        expect(container.querySelectorAll('path.edge').length).toBe(2);
    });

    test('une place en cours ouvre la fiche de résultat, qui valide', async () => {
        const onResult = vi.fn();
        const { container, getByTestId } = render(BracketsView, { props: { phases: [phase()], cells: [{ table: 3, matchId: 'x2' }], onResult } });
        const places = container.querySelectorAll('[data-testid="bracket-place"]');
        await fireEvent.click(places[1]);
        expect(getByTestId('direction-result-card')).toBeTruthy();
        await fireEvent.click(getByTestId('direction-result-winner-b'));
        expect(onResult).toHaveBeenCalledWith('x2', 'd', 0, 0, '');
    });

    test('Entrée sur une place finie ouvre la correction', async () => {
        const onCorrect = vi.fn();
        const { container, getByTestId } = render(BracketsView, { props: { phases: [phase()], onCorrect } });
        const places = container.querySelectorAll('[data-testid="bracket-place"]');
        await fireEvent.keyDown(places[0], { key: 'Enter' });
        await fireEvent.click(getByTestId('bracket-correct-winner-b'));
        expect(onCorrect).toHaveBeenCalledWith('x1', 'b', 0, 0);
    });

    test('une place sans match ne s’ouvre pas', async () => {
        const { container, queryByTestId } = render(BracketsView, { props: { phases: [phase()] } });
        await fireEvent.click(container.querySelectorAll('[data-testid="bracket-place"]')[2]);
        expect(queryByTestId('bracket-card')).toBeNull();
    });

    test('une poule est un tableau croisé', () => {
        const pool = {
            name: 'Poule A',
            kind: 'poule',
            rounds: 1,
            matches: [place('p0', 0, { a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', matchId: 'y', done: true, winner: 'a', scoreA: 5, scoreB: 3 })]
        };
        const { getByTestId } = render(BracketsView, { props: { phases: [phase({ kind: 'round_robin', sections: [pool] })] } });
        const t = getByTestId('bracket-pool');
        expect(t.textContent).toContain('5–3');
        expect(t.textContent).toContain('3–5');
    });

    test('un tableau non tiré montre son squelette', () => {
        const { getByTestId } = render(BracketsView, { props: { phases: [phase({ drawn: false, sections: [] })], entrants: 8 } });
        expect(getByTestId('bracket-skeleton').querySelectorAll('rect.box').length).toBe(7);
    });
});

describe('bracketLayout', () => {
    test('une place se centre sur ses deux sources', () => {
        const g = layoutSection(skeletonSection(4));
        const final = g.nodes.find((n) => n.m.round === 1);
        const first = g.nodes.filter((n) => n.m.round === 0);
        expect(final?.y).toBe((first[0].y + first[1].y) / 2);
        expect(g.edges.length).toBe(2);
    });
});
