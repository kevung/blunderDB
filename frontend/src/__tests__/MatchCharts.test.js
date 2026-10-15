/**
 * MatchCharts.test.js
 *
 * The legend of the match charts ticks each player: one setting shared by the three
 * charts, the marks of an unticked player hidden, and every pointer and key target
 * falling on the decisions of the player left.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
import MatchCharts from '../components/MatchCharts.svelte';

afterEach(cleanup);

// Six decisions of 10 px: players 0,1,0,1,1,0.
const movePositions = [0, 1, 0, 1, 1, 0].map((p, i) => ({
    move_id: 100 + i,
    game_number: 1,
    move_number: i + 1,
    move_type: 'checker',
    player_on_roll: p,
    decision_ms: 1000 * (i + 1)
}));
const losses = movePositions.map((mp) => ({ move_id: mp.move_id, mwc_loss: 0.01, difficulty: 0, avoidable: false }));
const times = { has_cadence: false, players: [1, 2].map(() => ({ checker_count: 3, cube_count: 0, total_ms: 1, checker_total_ms: 1, cube_total_ms: 0 })) };

function mount(extra = {}) {
    const onselect = vi.fn();
    const onhover = vi.fn();
    const utils = render(MatchCharts, {
        props: { matchId: 1, times, losses, movePositions, player1: 'Ann', player2: 'Bob', hovered: null, onhover, onselect, ...extra }
    });
    return { ...utils, onselect, onhover };
}
const plots = () => ['times-plot', 'loss-plot-per', 'loss-plot-cum'].map((id) => screen.getByTestId(id));
function rect(plot) {
    plot.getBoundingClientRect = () => ({ left: 0, top: 0, width: 60, height: 100, right: 60, bottom: 100 });
}
const toggle = (name, nth = 0) => screen.getAllByRole('button', { name })[nth];
const bars = (plot) => plot.querySelectorAll('rect.bar').length;

describe('legend toggles', () => {
    test('each chart offers a toggle per player, pressed at first', () => {
        mount();
        for (const name of ['Ann', 'Bob']) {
            const btns = screen.getAllByRole('button', { name });
            expect(btns).toHaveLength(3);
            for (const b of btns) expect(b.getAttribute('aria-pressed')).toBe('true');
        }
    });

    test('unticking a player hides his marks on every chart', async () => {
        mount();
        const [t, per, cum] = plots();
        expect(bars(t)).toBe(6);
        expect(bars(per)).toBe(6);
        expect(cum.querySelectorAll('polyline.line.player2')).toHaveLength(1);
        await fireEvent.click(toggle('Bob', 1));
        expect(bars(t)).toBe(3);
        expect(bars(per)).toBe(3);
        expect(t.querySelectorAll('rect.player2')).toHaveLength(0);
        expect(per.querySelectorAll('rect.player2')).toHaveLength(0);
        expect(cum.querySelectorAll('polyline.line.player2')).toHaveLength(0);
        // The setting is shared: the other charts' toggles follow.
        for (const b of screen.getAllByRole('button', { name: 'Bob' })) expect(b.getAttribute('aria-pressed')).toBe('false');
    });

    test('the last player ticked cannot be unticked', async () => {
        mount();
        await fireEvent.click(toggle('Bob'));
        await fireEvent.click(toggle('Ann'));
        expect(toggle('Ann').getAttribute('aria-pressed')).toBe('true');
        expect(toggle('Ann').getAttribute('aria-disabled')).toBe('true');
        expect(bars(plots()[0])).toBe(3);
    });

    test('a toggle works from the keyboard (a button)', () => {
        mount();
        expect(toggle('Ann').tagName).toBe('BUTTON');
    });

    test('opening another match ticks both players again', async () => {
        const { rerender } = mount();
        await fireEvent.click(toggle('Bob'));
        expect(bars(plots()[0])).toBe(3);
        await rerender({ matchId: 2 });
        expect(bars(plots()[0])).toBe(6);
        expect(toggle('Bob').getAttribute('aria-pressed')).toBe('true');
    });
});

describe('only the decisions of the player left answer', () => {
    test('a click over a hidden decision selects the closest visible one', async () => {
        const { onselect } = mount();
        await fireEvent.click(toggle('Bob'));
        const [t] = plots();
        rect(t);
        // x=15 is decision 1 (Bob's); Ann's closest are 0 and 2, earlier one wins.
        await fireEvent.click(t, { clientX: 15 });
        expect(onselect).toHaveBeenLastCalledWith(0);
        await fireEvent.click(t, { clientX: 45 });
        expect(onselect).toHaveBeenLastCalledWith(5);
    });

    test('hovering reports a visible decision', async () => {
        const { onhover } = mount();
        await fireEvent.click(toggle('Ann'));
        const [t] = plots();
        rect(t);
        await fireEvent.mouseMove(t, { clientX: 5 });
        // Decision 0 is Ann's, hidden: the closest of Bob's is 1.
        expect(onhover).toHaveBeenLastCalledWith(101);
    });

    test('the arrows walk the visible decisions only', async () => {
        const { onselect, onhover, rerender } = mount();
        await fireEvent.click(toggle('Bob'));
        const [t] = plots();
        await fireEvent.keyDown(t, { key: 'ArrowRight' });
        expect(onhover).toHaveBeenLastCalledWith(100);
        // The parent feeds the focus back: Bob is hidden, so the next stop skips decision 1.
        await rerender({ hovered: 100 });
        await fireEvent.keyDown(t, { key: 'ArrowRight' });
        expect(onhover).toHaveBeenLastCalledWith(102);
        await fireEvent.keyDown(t, { key: 'End' });
        expect(onhover).toHaveBeenLastCalledWith(105);
        await rerender({ hovered: 102 });
        await fireEvent.keyDown(t, { key: 'Enter' });
        expect(onselect).toHaveBeenCalledTimes(1);
        expect(onselect).toHaveBeenCalledWith(2);
    });
});
