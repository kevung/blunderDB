/**
 * PanelTable.test.js — the table the list panels share: sortable header
 * cycling through nextSort, row selection, j/k navigation that scrolls the
 * reached row into view, the empty state, and the two pure helpers.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick, createRawSnippet } from 'svelte';
import PanelTable, { navigationDelta, stepSelection } from '../components/panels/PanelTable.svelte';

const ROWS = [
    { id: 1, name: 'Ann' },
    { id: 2, name: 'Bob' },
    { id: 3, name: 'Cid' }
];
const COLUMNS = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'n', label: '#', sortable: true, narrow: true, defaultDir: 'desc' },
    { key: 'actions', actions: true }
];
// A raw snippet renders a single element; one cell is enough to see the rows.
const cells = createRawSnippet((getRow) => ({ render: () => `<td class="name-cell">${getRow().name}</td>` }));

afterEach(cleanup);

function mount(props = {}) {
    return render(PanelTable, { props: { rows: ROWS, columns: COLUMNS, cells, ...props } });
}

describe('stepSelection', () => {
    const key = (r) => r.id;
    test('steps from the selected row and stops at both ends', () => {
        expect(stepSelection(ROWS, key, 1, 1)).toBe(ROWS[1]);
        expect(stepSelection(ROWS, key, 2, -1)).toBe(ROWS[0]);
        expect(stepSelection(ROWS, key, 3, 1)).toBeNull();
        expect(stepSelection(ROWS, key, 1, -1)).toBeNull();
    });
    test('with nothing selected, forward reaches the first row and back reaches nothing', () => {
        expect(stepSelection(ROWS, key, undefined, 1)).toBe(ROWS[0]);
        expect(stepSelection(ROWS, key, null, -1)).toBeNull();
        expect(stepSelection([], key, undefined, 1)).toBeNull();
    });
});

describe('navigationDelta', () => {
    const ev = (key, extra = {}) => ({ key, ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, ...extra });
    test('j/ArrowDown step forward, k/ArrowUp back, anything else is not navigation', () => {
        expect(navigationDelta(ev('j'))).toBe(1);
        expect(navigationDelta(ev('ArrowDown'))).toBe(1);
        expect(navigationDelta(ev('k'))).toBe(-1);
        expect(navigationDelta(ev('ArrowUp'))).toBe(-1);
        expect(navigationDelta(ev('x'))).toBe(0);
        expect(navigationDelta(ev('j', { ctrlKey: true }))).toBe(0);
    });
});

describe('PanelTable', () => {
    test('renders one header per column and one row per item through the cells snippet', () => {
        const { container } = mount();
        const ths = Array.from(container.querySelectorAll('th'));
        expect(ths.map((th) => th.textContent.trim())).toEqual(['Name', '#', '']);
        expect(ths[1].classList.contains('narrow-col')).toBe(true);
        expect(ths[2].classList.contains('actions-col')).toBe(true);
        expect(Array.from(container.querySelectorAll('tbody tr td.name-cell')).map((td) => td.textContent)).toEqual(['Ann', 'Bob', 'Cid']);
        expect(container.querySelector('.empty-state')).toBeNull();
    });

    test('shows the empty state under an empty table', () => {
        const { container } = mount({ rows: [], emptyText: 'Nothing here' });
        expect(container.querySelector('.empty-state').textContent).toBe('Nothing here');
    });

    test('clicking a sortable header cycles the sort: asc, desc, then (tristate) cleared', async () => {
        const { container } = mount({ sortOptions: { tristate: true } });
        const th = container.querySelector('th');
        const btn = th.querySelector('.sort-btn');
        const arrow = () => th.querySelector('.sort-arrow')?.textContent ?? null;

        expect(th.getAttribute('aria-sort')).toBe('none');
        await fireEvent.click(btn);
        expect(arrow()).toBe('▲');
        expect(th.getAttribute('aria-sort')).toBe('ascending');
        await fireEvent.click(btn);
        expect(arrow()).toBe('▼');
        expect(th.getAttribute('aria-sort')).toBe('descending');
        await fireEvent.click(btn);
        expect(arrow()).toBeNull();
        expect(th.getAttribute('aria-sort')).toBe('none');
    });

    test("a column's defaultDir is honoured when it is first picked", async () => {
        const { container } = mount();
        const th = container.querySelectorAll('th')[1];
        await fireEvent.click(th.querySelector('.sort-btn'));
        expect(th.querySelector('.sort-arrow').textContent).toBe('▼');
    });

    test('the sort button is reachable from the keyboard (#204)', async () => {
        const { container } = mount();
        const btn = container.querySelector('th .sort-btn');
        expect(btn.hasAttribute('tabindex')).toBe(false);
        expect(btn.tabIndex).toBeGreaterThanOrEqual(0);
    });

    test('a non-sortable header ignores clicks', async () => {
        const { container } = mount();
        const th = container.querySelectorAll('th')[2];
        expect(th.querySelector('.sort-btn')).toBeNull();
        await fireEvent.click(th);
        expect(container.querySelector('.sort-arrow')).toBeNull();
    });

    test('the selected row carries the class the keyboard service looks for; click and double-click report the row', async () => {
        const onSelect = vi.fn();
        const onActivate = vi.fn();
        const { container } = mount({ selectedKey: 2, onSelect, onActivate });
        const rows = container.querySelectorAll('tbody tr');
        expect(rows[1].classList.contains('selected')).toBe(true);
        expect(rows[0].classList.contains('selected')).toBe(false);

        await fireEvent.click(rows[2]);
        expect(onSelect).toHaveBeenCalledWith(ROWS[2], 2, expect.any(Object));
        await fireEvent.dblClick(rows[0]);
        expect(onActivate).toHaveBeenCalledWith(ROWS[0], 0, expect.any(Object));
    });

    test('rowClass and rowAttrs decorate the rows', () => {
        const { container } = mount({
            rowClass: (r) => (r.id === 1 ? 'editing-row' : ''),
            rowAttrs: (r) => ({ title: `row ${r.id}`, tabindex: 0 })
        });
        const rows = container.querySelectorAll('tbody tr');
        expect(rows[0].classList.contains('editing-row')).toBe(true);
        expect(rows[1].classList.contains('editing-row')).toBe(false);
        expect(rows[1].getAttribute('title')).toBe('row 2');
        expect(rows[1].getAttribute('tabindex')).toBe('0');
    });

    test('navigate() selects the next row and scrolls it into view once rendered', async () => {
        const onSelect = vi.fn();
        const { container, component } = mount({ selectedKey: 1, onSelect });
        const scrolled = [];
        container.querySelectorAll('tbody tr').forEach((tr) => (tr.scrollIntoView = vi.fn(() => scrolled.push(tr))));

        expect(component.navigate(1)).toBe(true);
        expect(onSelect).toHaveBeenCalledWith(ROWS[1], 1);
        await tick();
        await tick();
        expect(scrolled).toEqual([container.querySelectorAll('tbody tr')[1]]);

        expect(component.navigate(-1)).toBe(false); // the first row is selected: nothing before it
        expect(onSelect).toHaveBeenCalledTimes(1);
    });

    test('scrollToSelected() targets the selected row with the requested block', async () => {
        const { container, component } = mount({ selectedKey: 3 });
        const tr = container.querySelectorAll('tbody tr')[2];
        tr.scrollIntoView = vi.fn();
        component.scrollToSelected('center');
        await tick();
        await tick();
        expect(tr.scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth', block: 'center' });
    });
});
