/**
 * MatchTimeHistogram.test.js
 *
 * The decision-time histogram draws the summary's counts per band, hides an
 * unticked player, and a click on a band opens its decisions one after the other.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
import MatchTimeHistogram from '../components/MatchTimeHistogram.svelte';
import { bandLabel, bandMoves, timeBand } from '../utils/decisionTime.js';

afterEach(cleanup);

const bounds = [2000, 5000];
// Players 0,1,0: 1 s, 3 s, then a 4 s play after a 1 s cube decision.
const movePositions = [{ player_on_roll: 0, decision_ms: 1000 }, { player_on_roll: 1, decision_ms: 3000 }, { player_on_roll: 0, decision_ms: 4000, cube_decision_ms: 1000 }, { player_on_roll: 1 }];
const summary = {
    histogram_bounds_ms: bounds,
    players: [{ histogram: [2, 1, 0] }, { histogram: [0, 1, 0] }]
};

describe('bands', () => {
    test('a bound opens the band above it', () => {
        expect([1999, 2000, 4999, 5000].map((ms) => timeBand(bounds, ms))).toEqual([0, 1, 1, 2]);
        expect([0, 1, 2].map((i) => bandLabel(bounds, i))).toEqual(['<2 s', '2–5 s', '>5 s']);
    });

    test('a Move enters the band of its play and of its cube decision', () => {
        expect(bandMoves(movePositions, bounds)).toEqual([
            [
                { index: 0, player: 0 },
                { index: 2, player: 0 }
            ],
            [
                { index: 1, player: 1 },
                { index: 2, player: 0 }
            ],
            []
        ]);
    });
});

describe('MatchTimeHistogram', () => {
    function mount(extra = {}) {
        const onselect = vi.fn();
        render(MatchTimeHistogram, { props: { summary, movePositions, names: ['Ann', 'Bob'], onselect, ...extra } });
        return onselect;
    }

    test('draws the counts of the summary, an empty band disabled', () => {
        mount();
        expect(screen.getByTestId('times-band-0').title).toBe('<2 s\nAnn: 2\nBob: 0');
        expect(screen.getByTestId('times-band-2').hasAttribute('disabled')).toBe(true);
        expect(screen.getByTestId('times-band-1').querySelectorAll('.bin')).toHaveLength(2);
    });

    test('a click walks the band through its decisions', async () => {
        const onselect = mount();
        const band = screen.getByTestId('times-band-0');
        await fireEvent.click(band);
        await fireEvent.click(band);
        await fireEvent.click(band);
        expect(onselect.mock.calls.map((c) => c[0])).toEqual([0, 2, 0]);
    });

    test('an unticked player leaves the bars and the walk', async () => {
        const onselect = mount({ shown: [false, true] });
        const band = screen.getByTestId('times-band-1');
        expect(band.querySelectorAll('.bin.player1')).toHaveLength(0);
        await fireEvent.click(band);
        await fireEvent.click(band);
        expect(onselect.mock.calls.map((c) => c[0])).toEqual([1, 1]);
        expect(screen.getByTestId('times-band-0').hasAttribute('disabled')).toBe(true);
    });

    test('nothing is drawn without a histogram', () => {
        render(MatchTimeHistogram, { props: { summary: { players: [{}, {}] }, movePositions, names: ['Ann', 'Bob'] } });
        expect(screen.queryByTestId('times-histogram')).toBeNull();
    });
});
