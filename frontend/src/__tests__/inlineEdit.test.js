/**
 * inlineEdit.test.js — createInlineEdit, the one start/save/cancel state
 * machine behind every inline-editable cell of the list panels.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { createInlineEdit } from '../utils/inlineEdit.svelte.js';

function key(k, extra = {}) {
    return { key: k, stopPropagation: vi.fn(), preventDefault: vi.fn(), ...extra };
}

afterEach(() => {
    vi.useRealTimers();
    document.body.innerHTML = '';
});

describe('createInlineEdit', () => {
    test('starts idle; start() records the row and copies the value into the draft', () => {
        const edit = createInlineEdit({ onSave: vi.fn() });
        expect(edit.editingId).toBeNull();
        expect(edit.draft).toBe('');

        edit.start(7, 'Bob');
        expect(edit.editingId).toBe(7);
        expect(edit.draft).toBe('Bob');
        expect(edit.isEditing(7)).toBe(true);
        expect(edit.isEditing(8)).toBe(false);
    });

    test('an object value becomes a multi-field draft, copied so the row is not mutated', () => {
        const row = { name: 'a', location: 'b' };
        const edit = createInlineEdit({ onSave: vi.fn() });
        edit.start(1, row);
        edit.draft.name = 'z';
        expect(row.name).toBe('a');
        expect(edit.draft).toEqual({ name: 'z', location: 'b' });
    });

    test('save() hands onSave the id and a plain snapshot of the draft, then leaves edit mode', async () => {
        const onSave = vi.fn();
        const edit = createInlineEdit({ onSave });
        edit.start(3, { name: 'x' });
        edit.draft.name = 'y';

        await edit.save();

        expect(onSave).toHaveBeenCalledWith(3, { name: 'y' });
        expect(edit.editingId).toBeNull();
        expect(edit.draft).toBe('');
    });

    test('onSave returning false keeps the row in edit mode (validation failed)', async () => {
        const edit = createInlineEdit({ onSave: () => false });
        edit.start(1, '');
        expect(await edit.save()).toBe(false);
        expect(edit.editingId).toBe(1);
    });

    test('save() is a no-op when nothing is being edited', async () => {
        const onSave = vi.fn();
        const edit = createInlineEdit({ onSave });
        expect(await edit.save()).toBe(false);
        expect(onSave).not.toHaveBeenCalled();
    });

    test('a second save while the first is still awaiting does not persist twice (Enter then blur)', async () => {
        let resolve;
        const onSave = vi.fn(() => new Promise((r) => (resolve = r)));
        const edit = createInlineEdit({ onSave });
        edit.start(1, 'v');

        const first = edit.save();
        const second = edit.save();
        expect(onSave).toHaveBeenCalledTimes(1);
        expect(await second).toBe(false);

        resolve();
        expect(await first).toBe(true);
        expect(edit.editingId).toBeNull();
    });

    test('cancel() drops the draft and reports the abandoned id', () => {
        const onCancel = vi.fn();
        const onSave = vi.fn();
        const edit = createInlineEdit({ onSave, onCancel });
        edit.start(5, 'draft');
        edit.cancel();
        expect(edit.editingId).toBeNull();
        expect(edit.draft).toBe('');
        expect(onCancel).toHaveBeenCalledWith(5);
        expect(onSave).not.toHaveBeenCalled();

        edit.cancel(); // idle: nothing to report
        expect(onCancel).toHaveBeenCalledTimes(1);
    });

    test('onKeyDown: Enter saves, Escape cancels, both swallow the event; other keys pass', async () => {
        const onSave = vi.fn();
        const edit = createInlineEdit({ onSave });

        edit.start(1, 'a');
        const enter = key('Enter');
        edit.onKeyDown(enter);
        expect(enter.stopPropagation).toHaveBeenCalled();
        expect(enter.preventDefault).toHaveBeenCalled();
        await Promise.resolve();
        expect(onSave).toHaveBeenCalledWith(1, 'a');

        edit.start(2, 'b');
        const esc = key('Escape');
        edit.onKeyDown(esc);
        expect(esc.stopPropagation).toHaveBeenCalled();
        expect(edit.editingId).toBeNull();

        edit.start(3, 'c');
        const j = key('j');
        edit.onKeyDown(j);
        expect(j.stopPropagation).not.toHaveBeenCalled();
        expect(edit.editingId).toBe(3);
    });

    test('onBlur saves at once when no blurGroup is configured', async () => {
        const onSave = vi.fn();
        const edit = createInlineEdit({ onSave });
        edit.start(1, 'a');
        edit.onBlur({ target: null });
        await Promise.resolve();
        expect(onSave).toHaveBeenCalledWith(1, 'a');
    });

    test('onBlur with a blurGroup keeps editing while focus stays inside the group', async () => {
        vi.useFakeTimers();
        document.body.innerHTML = '<table><tbody><tr id="row"><td><input id="a" /></td><td><input id="b" /></td></tr></tbody></table><input id="outside" />';
        const a = document.getElementById('a');
        const onSave = vi.fn();
        const edit = createInlineEdit({ onSave, blurGroup: 'tr' });
        edit.start(1, { name: 'n' });

        // Tab from a to b: same row, still editing.
        document.getElementById('b').focus();
        edit.onBlur({ target: a });
        vi.runAllTimers();
        expect(onSave).not.toHaveBeenCalled();
        expect(edit.editingId).toBe(1);

        // Click outside the row: the edit is committed.
        document.getElementById('outside').focus();
        edit.onBlur({ target: a });
        vi.runAllTimers();
        expect(onSave).toHaveBeenCalledWith(1, { name: 'n' });
    });
});
