/**
 * Le classement dit le bilan de chacun, pas seulement son état : un retiré garde le sien.
 */

import { test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import StandingsView from '../components/direction/StandingsView.svelte';

afterEach(() => cleanup());

test('chaque ligne porte victoires–défaites', () => {
    const view = {
        finished: false,
        pool: 0,
        retained: 0,
        payable: 0,
        entrants: 2,
        sections: [
            {
                name: '',
                rows: [
                    { rank: 1, id: 'a', name: 'Alice', note: { kind: 'alive', lives: 2 }, wins: 4, losses: 1 },
                    { rank: 2, id: 'b', name: 'Bob', note: { kind: 'withdrawn', wins: 3, losses: 0 }, wins: 3, losses: 0 }
                ]
            }
        ]
    };
    const { container } = render(StandingsView, { props: { view, running: 0 } });
    const cells = [...container.querySelectorAll('tbody td.record')].map((td) => td.textContent?.trim());
    expect(cells).toEqual(['4–1', '3–0']);
});
