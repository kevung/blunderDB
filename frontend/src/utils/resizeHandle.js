import { tick } from 'svelte';
import { toCssPx } from './uiZoom.js';

/**
 * The room a panel may take, the board keeping the rest.
 *
 * @param {boolean} side   - true: a width (side panel); false: a height (bottom panel)
 * @param {number}  size   - the wanted size, in CSS pixels of the zoomed UI
 * @param {number}  extent - the UI's width (side) or height, in the same pixels
 * @returns {number}
 */
export function clampPanelSize(side, size, extent) {
    return Math.max(0, side ? Math.min(Math.max(150, size), extent - 200) : Math.min(Math.max(80, size), extent - 160));
}

/**
 * Svelte action: drag a handle to resize the panel next to it.
 *
 *   <div class="resize-handle" use:resizable={{ side, size, onResize, onCommit }}></div>
 *
 * Side mode drags horizontally (left grows), bottom mode vertically (up grows). Mode and starting
 * size are read at mousedown, so a layout flip mid-drag cannot mix axes.
 *
 * @param {HTMLElement} node
 * @param {Object} params
 * @param {boolean}  params.side      - true: resize a width (side panel); false: a height (bottom panel)
 * @param {number}   params.size      - current size of the panel in px (width in side mode, height otherwise)
 * @param {Function} params.onResize  - (size, side) on every mousemove, after clamping
 * @param {Function} params.onCommit  - (size, side) on mouseup, only when the drag moved the handle at all
 */
export function resizable(node, params) {
    let current = params;
    /** @type {(() => void) | null} */
    let cleanupDrag = null;

    function clamp(side, /** @type {number} */ size) {
        // Sizes are CSS pixels of the zoomed UI, the window and the pointer are window pixels.
        return clampPanelSize(side, size, toCssPx(side ? window.innerWidth : window.innerHeight));
    }

    // The new size reaches the DOM only once Svelte has flushed; measuring before that makes the
    // board fit the previous step.
    function notifyResize() {
        tick().then(() => window.dispatchEvent(new Event('resize')));
    }

    function onMouseDown(/** @type {MouseEvent} */ e) {
        e.preventDefault();
        const { side, size: startSize, onResize, onCommit } = current;
        document.body.style.cursor = side ? 'ew-resize' : 'ns-resize';
        document.body.style.userSelect = 'none';
        const start = side ? e.clientX : e.clientY;
        let size = startSize;
        let moved = false;

        function onMouseMove(/** @type {MouseEvent} */ e) {
            moved = true;
            size = clamp(side, startSize + toCssPx(start - (side ? e.clientX : e.clientY)));
            onResize(size, side);
            notifyResize();
        }
        function endDrag() {
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
            window.removeEventListener('mousemove', onMouseMove);
            window.removeEventListener('mouseup', onMouseUp);
            cleanupDrag = null;
        }
        function onMouseUp() {
            endDrag();
            // Persist only when the drag moved the handle (a plain click is a no-op).
            if (moved) {
                onCommit(size, side);
                notifyResize();
            }
        }
        window.addEventListener('mousemove', onMouseMove);
        window.addEventListener('mouseup', onMouseUp);
        cleanupDrag = endDrag;
    }

    node.addEventListener('mousedown', onMouseDown);

    return {
        update(next) {
            current = next;
        },
        destroy() {
            node.removeEventListener('mousedown', onMouseDown);
            if (cleanupDrag) cleanupDrag();
        }
    };
}
