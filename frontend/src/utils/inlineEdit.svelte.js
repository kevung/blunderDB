// Inline "click a cell, type, Enter/Escape/blur" editing, factored out of the
// eight start/save/cancel triplets the list panels had each hand-written
// (MatchPanel ×3, TournamentPanel ×3, CollectionPanel, AnkiPanel). Every copy
// held the same three pieces of state — which row is being edited, the draft
// value, and a keydown dispatcher — and the copies had drifted on details
// (Escape with or without preventDefault, whether blur saved or cancelled,
// whether a second save could fire while the first was still awaiting).
//
// This module is a `.svelte.js` file so the factory can hold its state in
// runes: a component reads `edit.editingId` / binds `edit.draft` and Svelte
// tracks them like any other `$state`. A plain `.js` module cannot do that.
//
// Usage in a component:
//
//     const nameEdit = createInlineEdit({ onSave: (id, draft) => rename(id, draft) });
//     …
//     {#if nameEdit.editingId === row.id}
//         <input bind:value={nameEdit.draft} onkeydown={nameEdit.onKeyDown} onblur={nameEdit.onBlur} />
//     {/if}
//
// `draft` is whatever `start(id, value)` was given: a string for a single
// field, or an object for a multi-field row (`bind:value={edit.draft.name}`).
// `onSave` receives a plain snapshot of it; returning `false` keeps the row in
// edit mode (validation failed), anything else ends the edit.

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
