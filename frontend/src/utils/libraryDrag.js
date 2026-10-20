/**
 * Svelte action: drag a row of the library onto another library group's header, or onto the
 * basket (ADR-0101). Pointer events, as dragReorder, so as not to wake Wails' file drop.
 *
 *   <tr use:libraryDrag={() => ({ group, kind: 'match', id, label })}>
 *
 * A drop target is any element carrying `data-library-drop`: a group's id, or "basket".
 * Under the pointer it takes the class `library-drop-over`; the body takes `library-dragging`.
 */
import { createAttachmentKey } from 'svelte/attachments';

const DEAD_ZONE = 8;

/** @param {number} x @param {number} y */
export function dropTargetAt(x, y) {
    const el = document.elementFromPoint?.(x, y);
    return /** @type {HTMLElement | null} */ (el?.closest('[data-library-drop]') ?? null);
}

/** @param {HTMLElement} el */
function targetOf(el) {
    const raw = el.dataset.libraryDrop;
    return raw === 'basket' ? 'basket' : Number(raw);
}

/**
 * @param {HTMLElement} node
 * @param {() => (import('../stores/transferBasketStore.js').TransferItem | { kind: 'basket', label: string } | null)} item
 */
export function libraryDrag(node, item) {
    let current = item;
    let pid = /** @type {number | null} */ (null);
    let startX = 0;
    let startY = 0;
    let active = false;
    /** @type {HTMLElement | null} */
    let ghost = null;
    /** @type {HTMLElement | null} */
    let over = null;
    /** @type {any} */
    let payload = null;

    function setOver(/** @type {HTMLElement | null} */ el) {
        if (el === over) return;
        over?.classList.remove('library-drop-over');
        over = el;
        over?.classList.add('library-drop-over');
    }

    function end() {
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
        window.removeEventListener('pointercancel', end);
        setOver(null);
        ghost?.remove();
        ghost = null;
        document.body.classList.remove('library-dragging');
        pid = null;
        active = false;
        payload = null;
    }

    /** @param {PointerEvent} e */
    function onDown(e) {
        if (e.button !== 0) return;
        if (/** @type {Element} */ (e.target).closest('button, input, select, textarea, a, [contenteditable]')) return;
        pid = e.pointerId;
        startX = e.clientX;
        startY = e.clientY;
        window.addEventListener('pointermove', onMove);
        window.addEventListener('pointerup', onUp);
        window.addEventListener('pointercancel', end);
    }

    /** @param {PointerEvent} e */
    function onMove(e) {
        if (e.pointerId !== pid) return;
        if (!active) {
            if (Math.abs(e.clientX - startX) + Math.abs(e.clientY - startY) < DEAD_ZONE) return;
            payload = current();
            if (!payload) return end();
            active = true;
            document.body.classList.add('library-dragging');
            ghost = document.createElement('div');
            ghost.className = 'library-drag-ghost';
            ghost.dataset.testid = 'library-drag-ghost';
            ghost.textContent = payload.label;
            document.body.appendChild(ghost);
        }
        if (ghost) ghost.style.transform = `translate(${e.clientX + 12}px, ${e.clientY + 12}px)`;
        setOver(dropTargetAt(e.clientX, e.clientY));
    }

    /** @param {PointerEvent} e */
    function onUp(e) {
        if (e.pointerId !== pid) return;
        const target = active ? dropTargetAt(e.clientX, e.clientY) : null;
        const dragged = payload;
        const wasActive = active;
        end();
        if (wasActive) {
            // The click that follows a drag is not a click on the row.
            const swallow = (/** @type {Event} */ ev) => {
                ev.stopPropagation();
                ev.preventDefault();
            };
            node.addEventListener('click', swallow, { capture: true, once: true });
            setTimeout(() => node.removeEventListener('click', swallow, { capture: true }), 200);
        }
        // Loaded on the first drop: every list row carries this action, few rows are ever dropped.
        if (target && dragged) import('../services/transferService.js').then((m) => m.dropTransferItem(targetOf(target), dragged));
    }

    node.addEventListener('pointerdown', onDown);
    return {
        /** @param {typeof item} next */
        update(next) {
            current = next;
        },
        destroy() {
            end();
            node.removeEventListener('pointerdown', onDown);
        }
    };
}

/** The key under which libraryDragAttrs spreads its attachment on an element. */
const LIBRARY_DRAG = createAttachmentKey();

/**
 * libraryDrag as attributes to spread, for an element built by another component (a
 * PanelTable row through its rowAttrs).
 * @param {Parameters<typeof libraryDrag>[1]} item
 */
export function libraryDragAttrs(item) {
    return {
        [LIBRARY_DRAG]: (/** @type {HTMLElement} */ node) => {
            const action = libraryDrag(node, item);
            return () => action.destroy();
        }
    };
}
