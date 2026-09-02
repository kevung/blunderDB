/**
 * Svelte action: drag a handle to resize the panel next to it.
 *
 *   <div class="resize-handle" use:resizable={{ side, size, onResize, onCommit }}></div>
 *
 * The panel sits to the right of the board (side mode: drag horizontally,
 * dragging left grows it) or below it (bottom mode: drag vertically, dragging
 * up grows it). The mode and the starting size are read at mousedown, so a
 * layout flip mid-drag cannot mix axes; both callbacks receive the mode the
 * drag started in.
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
    let cleanupDrag = null;

    function clamp(side, size) {
        return side ? Math.min(Math.max(150, size), window.innerWidth - 200) : Math.min(Math.max(80, size), window.innerHeight - 160);
    }

    function onMouseDown(e) {
        e.preventDefault();
        const { side, size: startSize, onResize, onCommit } = current;
        document.body.style.cursor = side ? 'ew-resize' : 'ns-resize';
        document.body.style.userSelect = 'none';
        const start = side ? e.clientX : e.clientY;
        let size = startSize;
        let moved = false;

        function onMouseMove(e) {
            moved = true;
            size = clamp(side, startSize + (start - (side ? e.clientX : e.clientY)));
            onResize(size, side);
            // Let two.js re-measure the board box as the panel grows/shrinks.
            window.dispatchEvent(new Event('resize'));
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
            if (moved) onCommit(size, side);
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
