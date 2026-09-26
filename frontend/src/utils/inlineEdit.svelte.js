// Inline "click a cell, type, Enter/Escape/blur" editing shared by the list panels, so Escape,
// blur-saves and single-flight saves behave the same everywhere. A `.svelte.js` module, so the
// factory holds its state in runes that components track like any `$state`:
//
//     const nameEdit = createInlineEdit({ onSave: (id, draft) => rename(id, draft) });
//     {#if nameEdit.editingId === row.id}
//         <input bind:value={nameEdit.draft} onkeydown={nameEdit.onKeyDown} onblur={nameEdit.onBlur} />
//     {/if}
//
// `draft` is what `start(id, value)` was given (a string, or an object for a multi-field row).
// `onSave` receives a plain snapshot; returning `false` keeps the row in edit mode.

import { untrack } from 'svelte';

/**
 * @template T
 * @param {object} opts
 * @param {(id: any, draft: T) => (void | boolean | Promise<void | boolean>)} opts.onSave
 *   Persist the draft. Return `false` to stay in edit mode.
 * @param {(id: any) => void} [opts.onCancel] Called after an edit is abandoned.
 * @param {string} [opts.blurGroup] CSS selector of the element grouping the
 *   inputs of one edit (e.g. `'tr'`). When set, blurring one input saves only
 *   if focus has left that group — moving between the fields of the same row
 *   keeps editing. Without it, blur saves at once.
 */
export function createInlineEdit({ onSave, onCancel, blurGroup } = {}) {
    let editingId = $state(null);
    let draft = $state('');
    let saving = false;

    function start(id, value = '') {
        editingId = id;
        draft = typeof value === 'object' && value !== null ? { ...value } : value;
    }

    function snapshot() {
        return typeof draft === 'object' && draft !== null ? $state.snapshot(draft) : draft;
    }

    async function save() {
        // `saving` guards the Enter-then-blur pair: both fire on the same edit,
        // and the second must not persist (or clear) anything.
        if (editingId === null || saving) return false;
        saving = true;
        const id = editingId;
        try {
            const keep = await onSave?.(id, snapshot());
            if (keep === false) return false;
            // The edit may have been cancelled or restarted while awaiting.
            if (untrack(() => editingId) === id) {
                editingId = null;
                draft = '';
            }
            return true;
        } finally {
            saving = false;
        }
    }

    function cancel() {
        if (editingId === null) return;
        const id = editingId;
        editingId = null;
        draft = '';
        onCancel?.(id);
    }

    function onKeyDown(event) {
        if (event.key === 'Enter') {
            event.stopPropagation();
            event.preventDefault();
            save();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            cancel();
        }
    }

    function onBlur(event) {
        if (!blurGroup) {
            save();
            return;
        }
        const group = event?.target?.closest?.(blurGroup) ?? null;
        // Focus lands on the next field after this blur; look once it has.
        setTimeout(() => {
            if (group && group.contains(document.activeElement)) return;
            save();
        }, 0);
    }

    return {
        get editingId() {
            return editingId;
        },
        get draft() {
            return draft;
        },
        set draft(value) {
            draft = value;
        },
        /** True while `id` is the row being edited. */
        isEditing(id) {
            return editingId !== null && editingId === id;
        },
        start,
        save,
        cancel,
        onKeyDown,
        onBlur
    };
}
