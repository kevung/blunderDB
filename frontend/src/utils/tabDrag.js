/**
 * Svelte action: reorder the tab bar by pointer (pointer events, as dragReorder, so as not to
 * wake Wails' file drop). The bar is one flat row: a library group's header, then its tabs.
 *
 *   - a tab (`[role="tab"]`) can only land beside another tab of its own group;
 *   - a header (`.group-header`) can land beside any other group, headers and tabs alike, and
 *     takes its tabs with it, since they are drawn from the group.
 *
 * Every element carries `data-group-id`; a tab also `data-view-id`. A drop anywhere else does
 * nothing. The landing side shows as `tab-drop-before` / `tab-drop-after` on the target.
 */

const DEAD_ZONE = 6;
const BEFORE = 'tab-drop-before';
const AFTER = 'tab-drop-after';

/**
 * @typedef {{
 *   onMoveView: (groupId: number, viewId: number, targetViewId: number, after: boolean) => void,
 *   onMoveGroup: (groupId: number, targetGroupId: number, after: boolean) => void
 * }} TabDragParams
 */

/**
 * @param {HTMLElement} node
 * @param {TabDragParams} params
 */
export function tabDrag(node, params) {
    let current = params;
    /** @type {{ kind: 'view' | 'group', group: number, view: number, pointer: number, x: number, y: number, active: boolean } | null} */
    let drag = null;
    /** @type {{ el: HTMLElement, after: boolean } | null} */
    let landing = null;

    function clearLanding() {
        landing?.el.classList.remove(BEFORE, AFTER);
        landing = null;
    }

    /**
     * Where the pointer would drop the dragged item, null when the drop is refused.
     * @param {number} x
     * @param {number} y
     */
    function landingAt(x, y) {
        if (!drag) return null;
        const hit = /** @type {HTMLElement | null} */ (document.elementFromPoint?.(x, y)?.closest('[data-group-id]') ?? null);
        if (!hit || !node.contains(hit)) return null;
        const group = Number(hit.dataset.groupId);
        const rect = hit.getBoundingClientRect();
        const after = x > rect.left + rect.width / 2;
        if (drag.kind === 'view') {
            // A view belongs to its library: never into another group, nor onto a header.
            if (hit.getAttribute('role') !== 'tab' || group !== drag.group || Number(hit.dataset.viewId) === drag.view) return null;
            return { el: hit, after, group, view: Number(hit.dataset.viewId) };
        }
        if (group === drag.group) return null;
        // A group takes the place of the one it is dropped on: dragged right, it lands after that
        // group's last tab; dragged left, before its header.
        const all = /** @type {HTMLElement[]} */ ([...node.querySelectorAll(`[data-group-id="${group}"]`)]);
        const mine = /** @type {HTMLElement} */ (node.querySelector(`[data-group-id="${drag.group}"]`));
        const toRight = !!(mine.compareDocumentPosition(all[0]) & Node.DOCUMENT_POSITION_FOLLOWING);
        return { el: toRight ? all[all.length - 1] : all[0], after: toRight, group, view: 0 };
    }

    /** @param {PointerEvent} e */
    function onDown(e) {
        if (e.button !== 0) return;
        const target = /** @type {Element} */ (e.target);
        if (target.closest('button, input, select, textarea, a')) return;
        const item = /** @type {HTMLElement | null} */ (target.closest('[role="tab"], .group-header'));
        if (!item || !node.contains(item)) return;
        const isTab = item.getAttribute('role') === 'tab';
        drag = {
            kind: isTab ? 'view' : 'group',
            group: Number(item.dataset.groupId),
            view: isTab ? Number(item.dataset.viewId) : 0,
            pointer: e.pointerId,
            x: e.clientX,
            y: e.clientY,
            active: false
        };
        window.addEventListener('pointermove', onMove);
        window.addEventListener('pointerup', onUp);
    }

    /** @param {PointerEvent} e */
    function onMove(e) {
        if (!drag || e.pointerId !== drag.pointer) return;
        if (!drag.active) {
            if (Math.abs(e.clientX - drag.x) + Math.abs(e.clientY - drag.y) < DEAD_ZONE) return;
            drag.active = true;
        }
        e.preventDefault();
        clearLanding();
        const at = landingAt(e.clientX, e.clientY);
        if (at) {
            landing = { el: at.el, after: at.after };
            at.el.classList.add(at.after ? AFTER : BEFORE);
        }
    }

    /** @param {PointerEvent} e */
    function onUp(e) {
        if (!drag || e.pointerId !== drag.pointer) return;
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        const done = drag;
        drag = null;
        clearLanding();
        if (!done.active) return;
        // The click that follows a drag is not a selection.
        node.addEventListener('click', swallow, { capture: true, once: true });
        setTimeout(() => node.removeEventListener('click', swallow, { capture: true }), 200);
        drag = done;
        const at = landingAt(e.clientX, e.clientY);
        drag = null;
        if (!at) return;
        if (done.kind === 'view') current.onMoveView(done.group, done.view, at.view, at.after);
        else current.onMoveGroup(done.group, at.group, at.after);
    }

    /** @param {Event} e */
    function swallow(e) {
        e.stopPropagation();
        e.preventDefault();
    }

    node.addEventListener('pointerdown', onDown);
    return {
        /** @param {TabDragParams} next */
        update(next) {
            current = next;
        },
        destroy() {
            node.removeEventListener('pointerdown', onDown);
            window.removeEventListener('pointermove', onMove);
            window.removeEventListener('pointerup', onUp);
            clearLanding();
        }
    };
}
