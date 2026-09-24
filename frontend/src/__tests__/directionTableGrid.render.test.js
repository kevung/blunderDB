/**
 * Le temps écoulé d'un match qui vient de commencer (issue #408).
 *
 * Le backend encode `elapsedSeconds` avec `omitempty` : une table dont le match vient d'être
 * lancé arrive SANS ce champ. La case calculait alors `Math.floor(undefined / 3600)` et
 * affichait « NaN min » jusqu'au rafraîchissement suivant — la minute où le directeur regarde
 * justement si le match est bien parti.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import TableGrid from '../components/direction/TableGrid.svelte';

afterEach(cleanup);

describe('la grille des tables', () => {
    test('un match qui vient de commencer affiche zéro minute, pas NaN', () => {
        const cells = [{ table: 1, free: false, unavailable: false, reserved: false, matchId: 'M1', a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', length: 7 }];
        const { container } = render(TableGrid, { props: { cells } });
        const meta = /** @type {HTMLElement} */ (container.querySelector('.cell .meta'));
        expect(meta.textContent).not.toContain('NaN');
        expect(meta.textContent).toContain('0 min');
    });

    // #437 : un match apparié à la main dans une salle pleine n'a pas de table. Il doit
    // quand même occuper une case, sous « sans table », et ne pas compter comme une table.
    test('un match sans table a sa case « sans table », hors du compte des tables', () => {
        const cells = [
            { table: 1, free: false, unavailable: false, reserved: false, matchId: 'M1', a: 'a', b: 'b', aName: 'Alice', bName: 'Bob', length: 7 },
            { table: 0, free: false, unavailable: false, reserved: false, noTable: true, matchId: 'M2', a: 'c', b: 'd', aName: 'Chloé', bName: 'David', length: 5 },
            { table: 0, free: false, unavailable: false, reserved: false, noTable: true, matchId: 'M3', a: 'e', b: 'f', aName: 'Émile', bName: 'Fanny', length: 5 }
        ];
        const { container, getByTestId } = render(TableGrid, { props: { cells } });
        expect(container.querySelectorAll('.cell')).toHaveLength(3);
        const none = getByTestId('direction-table-none-M2');
        expect(none.textContent).toContain('Chloé');
        expect(none.querySelector('.num')?.textContent).not.toBe('0');
        expect(getByTestId('direction-table-none-M3').textContent).toContain('Fanny');
        expect(container.querySelector('h3')?.textContent).toContain('1');
    });
});
