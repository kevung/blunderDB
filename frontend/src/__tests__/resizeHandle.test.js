/**
 * resizeHandle.js is the resize-handle drag App.svelte used to own inline. It
 * is a plain Svelte action, so these tests drive it against a jsdom element:
 * body cursor / user-select during the drag and restored after, clamping on
 * both axes, and onCommit only when the handle actually moved.
 */
import { describe, test, expect, afterEach, vi } from 'vitest';
import { resizable } from '../utils/resizeHandle.js';

let handle;
let action;

afterEach(() => {
    action?.destroy();
    action = null;
    handle?.remove();
    handle = null;
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
});

function mount(params) {
    handle = document.createElement('div');
    document.body.appendChild(handle);
    action = resizable(handle, params);
    return action;
}

function mouse(target, type, coords = {}) {
    const ev = new MouseEvent(type, { bubbles: true, cancelable: true, clientX: 0, clientY: 0, ...coords });
    target.dispatchEvent(ev);
    return ev;
}

describe('resizable — bottom panel (height)', () => {
    test('dragging up grows the height, mouseup restores body and commits', () => {
        const onResize = vi.fn();
        const onCommit = vi.fn();
        mount({ side: false, size: 200, onResize, onCommit });

        const down = mouse(handle, 'mousedown', { clientY: 500 });
        expect(down.defaultPrevented).toBe(true);
        expect(document.body.style.cursor).toBe('ns-resize');
        expect(document.body.style.userSelect).toBe('none');

        mouse(window, 'mousemove', { clientY: 450 });
        expect(onResize).toHaveBeenLastCalledWith(250, false);
        mouse(window, 'mousemove', { clientY: 520 });
        expect(onResize).toHaveBeenLastCalledWith(180, false);

        mouse(window, 'mouseup');
        expect(document.body.style.cursor).toBe('');
        expect(document.body.style.userSelect).toBe('');
        expect(onCommit).toHaveBeenCalledTimes(1);
        expect(onCommit).toHaveBeenCalledWith(180, false);
    });

    test('a plain click commits nothing', () => {
        const onResize = vi.fn();
        const onCommit = vi.fn();
        mount({ side: false, size: 200, onResize, onCommit });
        mouse(handle, 'mousedown', { clientY: 500 });
        mouse(window, 'mouseup');
        expect(onResize).not.toHaveBeenCalled();
        expect(onCommit).not.toHaveBeenCalled();
        expect(document.body.style.cursor).toBe('');
    });

    test('clamps to [80, innerHeight - 160]', () => {
        const onResize = vi.fn();
        mount({ side: false, size: 200, onResize, onCommit: vi.fn() });
        mouse(handle, 'mousedown', { clientY: 500 });
        mouse(window, 'mousemove', { clientY: 5000 });
        expect(onResize).toHaveBeenLastCalledWith(80, false);
        mouse(window, 'mousemove', { clientY: -5000 });
        expect(onResize).toHaveBeenLastCalledWith(window.innerHeight - 160, false);
        mouse(window, 'mouseup');
    });

    test('each mousemove dispatches a window resize so the board re-fits', () => {
        const onWindowResize = vi.fn();
        window.addEventListener('resize', onWindowResize);
        mount({ side: false, size: 200, onResize: vi.fn(), onCommit: vi.fn() });
        mouse(handle, 'mousedown', { clientY: 500 });
        mouse(window, 'mousemove', { clientY: 490 });
        mouse(window, 'mouseup');
        window.removeEventListener('resize', onWindowResize);
        expect(onWindowResize).toHaveBeenCalledTimes(1);
    });
});

describe('resizable — side panel (width)', () => {
    test('dragging left grows the width, clamped to [150, innerWidth - 200]', () => {
        const onResize = vi.fn();
        const onCommit = vi.fn();
        mount({ side: true, size: 300, onResize, onCommit });

        mouse(handle, 'mousedown', { clientX: 800 });
        expect(document.body.style.cursor).toBe('ew-resize');

        mouse(window, 'mousemove', { clientX: 700 });
        expect(onResize).toHaveBeenLastCalledWith(400, true);
        mouse(window, 'mousemove', { clientX: 5000 });
        expect(onResize).toHaveBeenLastCalledWith(150, true);
        mouse(window, 'mousemove', { clientX: -5000 });
        expect(onResize).toHaveBeenLastCalledWith(window.innerWidth - 200, true);

        mouse(window, 'mouseup');
        expect(onCommit).toHaveBeenCalledWith(window.innerWidth - 200, true);
        expect(document.body.style.cursor).toBe('');
    });
});

describe('resizable — parameters and lifecycle', () => {
    test('update() feeds the next drag the current size and mode', () => {
        const onResize = vi.fn();
        const a = mount({ side: false, size: 200, onResize, onCommit: vi.fn() });
        a.update({ side: true, size: 320, onResize, onCommit: vi.fn() });
        mouse(handle, 'mousedown', { clientX: 600 });
        mouse(window, 'mousemove', { clientX: 590 });
        mouse(window, 'mouseup');
        expect(onResize).toHaveBeenLastCalledWith(330, true);
    });

    test('destroy() mid-drag restores the body and stops listening', () => {
        const onResize = vi.fn();
        const a = mount({ side: false, size: 200, onResize, onCommit: vi.fn() });
        mouse(handle, 'mousedown', { clientY: 500 });
        a.destroy();
        action = null;
        expect(document.body.style.cursor).toBe('');
        mouse(window, 'mousemove', { clientY: 400 });
        expect(onResize).not.toHaveBeenCalled();
        mouse(handle, 'mousedown', { clientY: 500 });
        expect(document.body.style.cursor).toBe('');
    });
});
