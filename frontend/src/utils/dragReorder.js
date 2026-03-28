/**
 * Svelte action for pointer-based drag reorder of list items.
 * Uses pointer events instead of HTML5 drag API to avoid conflict
 * with Wails file drag-and-drop (OnFileDrop).
 *
 * Usage:
 *   <tbody use:dragReorder={{ onReorder, itemSelector: 'tr' }}>
 *
 * @param {HTMLElement} node - Container element (e.g. <tbody>)
 * @param {Object} params
 * @param {Function} params.onReorder - Called with (fromIndex, toIndex) on drop
 * @param {string}  [params.itemSelector='tr'] - CSS selector for draggable items
 * @param {string}  [params.dragOverClass='drag-over'] - Class added to the drop target row
 * @param {string}  [params.draggingClass='dragging'] - Class added to the dragged row
 * @param {number}  [params.deadZone=5] - Pixels of movement before drag activates
 * @param {boolean} [params.enabled=true] - Whether drag is enabled
 */
export function dragReorder(node, params) {
    let onReorder, itemSelector, dragOverClass, draggingClass, deadZone, enabled;

    function updateParams(p) {
        onReorder = p.onReorder;
        itemSelector = p.itemSelector || 'tr';
        dragOverClass = p.dragOverClass || 'drag-over';
        draggingClass = p.draggingClass || 'dragging';
        deadZone = p.deadZone ?? 5;
        enabled = p.enabled !== false;
    }
    updateParams(params);

    let dragIdx = -1;
    let overIdx = -1;
    let active = false;
    let startX = 0;
    let startY = 0;
    let pid = null;

    function getRows() {
        return Array.from(node.querySelectorAll(`:scope > ${itemSelector}`));
    }

    /** Find row index at vertical position y */
    function rowAtY(y) {
        const rows = getRows();
        for (let i = 0; i < rows.length; i++) {
            const rect = rows[i].getBoundingClientRect();
            if (y >= rect.top && y <= rect.bottom) return i;
        }
        if (rows.length === 0) return -1;
        if (y < rows[0].getBoundingClientRect().top) return 0;
        return rows.length - 1;
    }

    function updateIndicator(newOver) {
        if (newOver === overIdx) return;
        const rows = getRows();
        if (overIdx >= 0 && rows[overIdx]) rows[overIdx].classList.remove(dragOverClass);
        overIdx = newOver;
        if (overIdx >= 0 && overIdx !== dragIdx && rows[overIdx]) {
            rows[overIdx].classList.add(dragOverClass);
        }
    }

    function clearClasses() {
        const rows = getRows();
        rows.forEach(r => {
            r.classList.remove(dragOverClass);
            r.classList.remove(draggingClass);
        });
    }

    function onDown(e) {
        if (!enabled || e.button !== 0) return;
        if (e.target.closest('button, input, select, textarea, a, [contenteditable]')) return;

        const rows = getRows();
        const row = e.target.closest(itemSelector);
        if (!row || !node.contains(row)) return;
        const idx = rows.indexOf(row);
        if (idx < 0) return;

        dragIdx = idx;
        startX = e.clientX;
        startY = e.clientY;
        active = false;
        pid = e.pointerId;

        window.addEventListener('pointermove', onMove);
        window.addEventListener('pointerup', onUp);
    }

    function onMove(e) {
        if (e.pointerId !== pid) return;

        if (!active) {
            if (Math.abs(e.clientX - startX) + Math.abs(e.clientY - startY) < deadZone) return;
            active = true;
            const rows = getRows();
            if (rows[dragIdx]) rows[dragIdx].classList.add(draggingClass);
        }

        e.preventDefault();
        updateIndicator(rowAtY(e.clientY));
    }

    function onUp(e) {
        if (e.pointerId !== pid) return;

        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);

        const wasActive = active;
        const from = dragIdx;
        const to = overIdx;

        clearClasses();
        dragIdx = -1;
        overIdx = -1;
        active = false;
        pid = null;

        if (wasActive) {
            // Block the click event that follows a completed drag
            node.addEventListener('click', suppressClick, { capture: true, once: true });
            // Safety: remove the blocker after a short delay in case click never fires
            setTimeout(() => node.removeEventListener('click', suppressClick, { capture: true }), 200);

            if (from >= 0 && to >= 0 && from !== to) {
                onReorder(from, to);
            }
        }
    }

    function suppressClick(e) {
        e.stopPropagation();
        e.preventDefault();
    }

    node.addEventListener('pointerdown', onDown);

    return {
        update(newParams) {
            updateParams(newParams);
        },
        destroy() {
            window.removeEventListener('pointermove', onMove);
            window.removeEventListener('pointerup', onUp);
            node.removeEventListener('pointerdown', onDown);
            clearClasses();
        }
    };
}
