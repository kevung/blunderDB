/**
 * dragReorder.js is a plain Svelte action (no component): pointer-based drag
 * reorder for a list of rows, deliberately built on pointer events instead of
 * the HTML5 drag API (which conflicts with Wails' OnFileDrop on Linux). These
 * tests drive it directly against a jsdom <tbody>/<tr> tree.
 */
import { describe, test, expect, afterEach, vi } from 'vitest';
import { dragReorder } from '../utils/dragReorder.js';

let container;

afterEach(() => {
    if (container) {
        container.remove();
        container = null;
    }
    vi.restoreAllMocks();
});

// jsdom doesn't implement layout, so getBoundingClientRect() always returns
// zeros. dragReorder's rowAtY() needs distinct rects per row to tell them
// apart, so each test stubs a fixed-height layout: row i occupies
// [i*ROW_HEIGHT, (i+1)*ROW_HEIGHT).
const ROW_HEIGHT = 20;

function layoutRows(rows) {
    rows.forEach((row, i) => {
        row.getBoundingClientRect = () => ({
            top: i * ROW_HEIGHT,
            bottom: (i + 1) * ROW_HEIGHT,
            left: 0,
            right: 100,
            height: ROW_HEIGHT,
            width: 100
        });
    });
}

function buildTable(rowLabels) {
    container = document.createElement('tbody');
    for (const label of rowLabels) {
        const tr = document.createElement('tr');
        tr.textContent = label;
        container.appendChild(tr);
    }
    document.body.appendChild(container);
    layoutRows(Array.from(container.children));
    return container;
}

function pointerEvent(type, opts = {}) {
    return new PointerEvent(type, { bubbles: true, cancelable: true, pointerId: 1, clientX: 0, clientY: 0, button: 0, ...opts });
}

// Drags row `fromIdx` to the vertical position of row `toIdx` and releases.
function drag(node, fromIdx, toIdx) {
    const rows = Array.from(node.children);
    const fromRow = rows[fromIdx];
    fromRow.dispatchEvent(pointerEvent('pointerdown', { clientX: 0, clientY: fromIdx * ROW_HEIGHT + 5 }));
    // Move far enough to clear the dead zone (default 5px), landing over toIdx.
    window.dispatchEvent(pointerEvent('pointermove', { clientY: toIdx * ROW_HEIGHT + 5, clientX: 20 }));
    window.dispatchEvent(pointerEvent('pointerup', { clientY: toIdx * ROW_HEIGHT + 5 }));
}

describe('dragReorder', () => {
    test('calls onReorder with the from/to indices on a completed drag', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder });

        drag(node, 0, 2);

        expect(onReorder).toHaveBeenCalledTimes(1);
        expect(onReorder).toHaveBeenCalledWith(0, 2);
    });

    test('does not call onReorder when the drop target is the same row', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder });

        drag(node, 1, 1);

        expect(onReorder).not.toHaveBeenCalled();
    });

    test('does not call onReorder for a small movement under the dead zone (a click, not a drag)', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder });

        const row = node.children[0];
        row.dispatchEvent(pointerEvent('pointerdown', { clientX: 0, clientY: 0 }));
        window.dispatchEvent(pointerEvent('pointermove', { clientX: 1, clientY: 1 })); // well under deadZone=5
        window.dispatchEvent(pointerEvent('pointerup', { clientX: 1, clientY: 1 }));

        expect(onReorder).not.toHaveBeenCalled();
    });

    test('respects a custom deadZone', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder, deadZone: 50 });

        const row = node.children[0];
        row.dispatchEvent(pointerEvent('pointerdown', { clientX: 0, clientY: 0 }));
        // Movement of 40px total is under the custom 50px dead zone.
        window.dispatchEvent(pointerEvent('pointermove', { clientX: 0, clientY: 40 }));
        window.dispatchEvent(pointerEvent('pointerup', { clientX: 0, clientY: 40 }));

        expect(onReorder).not.toHaveBeenCalled();
    });

    test('ignores non-primary (right-click) pointer down', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder });

        const row = node.children[0];
        row.dispatchEvent(pointerEvent('pointerdown', { clientX: 0, clientY: 0, button: 2 }));
        window.dispatchEvent(pointerEvent('pointermove', { clientX: 0, clientY: 40 }));
        window.dispatchEvent(pointerEvent('pointerup', { clientX: 0, clientY: 40 }));

        expect(onReorder).not.toHaveBeenCalled();
    });

    test('ignores pointerdown starting on an interactive child (button/input/etc.)', () => {
        container = document.createElement('tbody');
        const tr = document.createElement('tr');
        const btn = document.createElement('button');
        tr.appendChild(btn);
        container.appendChild(tr);
        document.body.appendChild(container);
        layoutRows(Array.from(container.children));

        const onReorder = vi.fn();
        dragReorder(container, { onReorder });

        btn.dispatchEvent(pointerEvent('pointerdown', { clientX: 0, clientY: 0 }));
        window.dispatchEvent(pointerEvent('pointermove', { clientX: 0, clientY: 40 }));
        window.dispatchEvent(pointerEvent('pointerup', { clientX: 0, clientY: 40 }));

        expect(onReorder).not.toHaveBeenCalled();
    });

    test('disabled: enabled=false suppresses drag entirely', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder, enabled: false });

        drag(node, 0, 2);

        expect(onReorder).not.toHaveBeenCalled();
    });

    test('adds and removes the dragging/drag-over classes during a drag', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder });
        const rows = Array.from(node.children);

        rows[0].dispatchEvent(pointerEvent('pointerdown', { clientX: 0, clientY: 5 }));
        window.dispatchEvent(pointerEvent('pointermove', { clientX: 0, clientY: 45 })); // over row 2

        expect(rows[0].classList.contains('dragging')).toBe(true);
        expect(rows[2].classList.contains('drag-over')).toBe(true);

        window.dispatchEvent(pointerEvent('pointerup', { clientX: 0, clientY: 45 }));

        // Classes are cleared once the drag completes.
        expect(rows[0].classList.contains('dragging')).toBe(false);
        expect(rows[2].classList.contains('drag-over')).toBe(false);
    });

    test('honours custom class names', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder, dragOverClass: 'my-over', draggingClass: 'my-dragging' });
        const rows = Array.from(node.children);

        rows[0].dispatchEvent(pointerEvent('pointerdown', { clientX: 0, clientY: 5 }));
        window.dispatchEvent(pointerEvent('pointermove', { clientX: 0, clientY: 45 }));

        expect(rows[0].classList.contains('my-dragging')).toBe(true);
        expect(rows[2].classList.contains('my-over')).toBe(true);

        window.dispatchEvent(pointerEvent('pointerup', { clientX: 0, clientY: 45 }));
    });

    test('update() swaps in a new onReorder callback', () => {
        const node = buildTable(['a', 'b', 'c']);
        const first = vi.fn();
        const second = vi.fn();
        const action = dragReorder(node, { onReorder: first });

        action.update({ onReorder: second });
        drag(node, 0, 2);

        expect(first).not.toHaveBeenCalled();
        expect(second).toHaveBeenCalledWith(0, 2);
    });

    test('destroy() removes listeners so a subsequent drag is a no-op', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        const action = dragReorder(node, { onReorder });

        action.destroy();
        drag(node, 0, 2);

        expect(onReorder).not.toHaveBeenCalled();
    });

    test('a completed drag suppresses the click that follows it', () => {
        const node = buildTable(['a', 'b', 'c']);
        const onReorder = vi.fn();
        dragReorder(node, { onReorder });

        drag(node, 0, 2);

        const clickHandler = vi.fn();
        node.addEventListener('click', clickHandler);
        const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
        node.dispatchEvent(clickEvent);

        expect(clickEvent.defaultPrevented).toBe(true);
    });

    test('itemSelector customises which children are treated as rows', () => {
        container = document.createElement('div');
        for (const label of ['a', 'b', 'c']) {
            const row = document.createElement('div');
            row.className = 'row';
            row.textContent = label;
            container.appendChild(row);
        }
        document.body.appendChild(container);
        layoutRows(Array.from(container.children));

        const onReorder = vi.fn();
        dragReorder(container, { onReorder, itemSelector: '.row' });

        drag(container, 0, 2);

        expect(onReorder).toHaveBeenCalledWith(0, 2);
    });
});
