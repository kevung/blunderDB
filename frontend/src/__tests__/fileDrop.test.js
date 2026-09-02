/**
 * fileDrop.js is the window-wide file drag & drop App.svelte used to wire in
 * onMount/onDestroy. The action owns the Wails OnFileDrop registration and the
 * drag-over highlight counter; the business logic (classifying and importing
 * the dropped paths) stays in importService and is only a callback here.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';

const runtime = vi.hoisted(() => ({ dropCallback: null, OnFileDrop: vi.fn(), OnFileDropOff: vi.fn() }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    OnFileDrop: (cb, useDropTarget) => {
        runtime.OnFileDrop(cb, useDropTarget);
        runtime.dropCallback = cb;
    },
    OnFileDropOff: () => runtime.OnFileDropOff()
}));

import { fileDrop } from '../utils/fileDrop.js';

let node;
let action;

beforeEach(() => {
    runtime.dropCallback = null;
    runtime.OnFileDrop.mockClear();
    runtime.OnFileDropOff.mockClear();
    node = document.createElement('main');
    document.body.appendChild(node);
});

afterEach(() => {
    action?.destroy();
    action = null;
    node.remove();
});

function drag(type) {
    const ev = new Event(type, { bubbles: true, cancelable: true });
    window.dispatchEvent(ev);
    return ev;
}

describe('fileDrop', () => {
    test('registers with Wails without a drop target and forwards the paths', () => {
        const onDrop = vi.fn();
        action = fileDrop(node, { onDrop, onOverlayChange: vi.fn() });
        expect(runtime.OnFileDrop).toHaveBeenCalledTimes(1);
        expect(runtime.OnFileDrop.mock.calls[0][1]).toBe(false);
        runtime.dropCallback(10, 20, ['/tmp/a.xg']);
        expect(onDrop).toHaveBeenCalledWith(10, 20, ['/tmp/a.xg']);
    });

    test('shows the overlay on dragover (preventing default) and hides it on dragleave', () => {
        const onOverlayChange = vi.fn();
        action = fileDrop(node, { onDrop: vi.fn(), onOverlayChange });
        const over = drag('dragover');
        expect(over.defaultPrevented).toBe(true);
        expect(onOverlayChange).toHaveBeenCalledWith(true);
        drag('dragover');
        expect(onOverlayChange).toHaveBeenCalledTimes(1);
        drag('dragleave');
        expect(onOverlayChange).toHaveBeenLastCalledWith(false);
        expect(onOverlayChange).toHaveBeenCalledTimes(2);
    });

    test('a stray dragleave never drives the counter negative', () => {
        const onOverlayChange = vi.fn();
        action = fileDrop(node, { onDrop: vi.fn(), onOverlayChange });
        drag('dragleave');
        drag('dragleave');
        expect(onOverlayChange).not.toHaveBeenCalled();
        drag('dragover');
        expect(onOverlayChange).toHaveBeenCalledWith(true);
        drag('dragleave');
        expect(onOverlayChange).toHaveBeenLastCalledWith(false);
    });

    test('drop hides the overlay and resets the counter', () => {
        const onOverlayChange = vi.fn();
        action = fileDrop(node, { onDrop: vi.fn(), onOverlayChange });
        drag('dragover');
        drag('drop');
        expect(onOverlayChange).toHaveBeenLastCalledWith(false);
        drag('dragover');
        expect(onOverlayChange).toHaveBeenLastCalledWith(true);
    });

    test('update() swaps the callbacks; destroy() unregisters everything', () => {
        const first = vi.fn();
        const second = vi.fn();
        const onOverlayChange = vi.fn();
        action = fileDrop(node, { onDrop: first, onOverlayChange });
        action.update({ onDrop: second, onOverlayChange });
        runtime.dropCallback(0, 0, ['/x.db']);
        expect(first).not.toHaveBeenCalled();
        expect(second).toHaveBeenCalledWith(0, 0, ['/x.db']);

        action.destroy();
        action = null;
        expect(runtime.OnFileDropOff).toHaveBeenCalledTimes(1);
        drag('dragover');
        expect(onOverlayChange).not.toHaveBeenCalled();
    });
});
