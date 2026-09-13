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
});
