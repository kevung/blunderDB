// The plays chosen in a candidate moves table, shared by the Analysis and Eval panels: a plain
// click selects one play (or drops it), Ctrl+click adds or removes one, Shift+click takes the
// range from the last play clicked, in the order on screen. `selected` is the play the board
// draws (selectedMoveStore); `picked` the plays of a multiple selection, empty when there is
// none; `anchor` where a Shift+click range starts.

/**
 * @typedef {{ picked: string[], anchor: string | null, selected: string | null }} MoveSelection
 */

/** @returns {MoveSelection} */
export function emptySelection() {
    return { picked: [], anchor: null, selected: null };
}

/**
 * The plays the selection names: the picked ones, else the one selected, else none.
 * @param {MoveSelection} sel
 * @returns {string[]}
 */
export function selectedPlays(sel) {
    if (sel.picked.length) return sel.picked;
    return sel.selected ? [sel.selected] : [];
}

/**
 * The selection after a click on the row of move.
 * @param {MoveSelection} sel
 * @param {string} move
 * @param {{ ctrlKey?: boolean, metaKey?: boolean, shiftKey?: boolean } | undefined} event
 * @param {string[]} order the plays in the order on screen
 * @returns {MoveSelection}
 */
export function clickSelection(sel, move, event, order) {
    if (event?.ctrlKey || event?.metaKey) {
        const base = selectedPlays(sel);
        const picked = base.includes(move) ? base.filter((m) => m !== move) : [...base, move];
        return { picked, anchor: move, selected: picked.includes(move) ? move : (picked.at(-1) ?? null) };
    }
    const from = sel.anchor ?? sel.selected;
    if (event?.shiftKey && from) {
        const a = order.indexOf(from);
        const b = order.indexOf(move);
        if (a >= 0 && b >= 0) return { picked: order.slice(Math.min(a, b), Math.max(a, b) + 1), anchor: sel.anchor, selected: move };
    }
    return { picked: [], anchor: move, selected: sel.selected === move ? null : move };
}

/**
 * The selection when the row of move is right-clicked: a play outside the selection becomes it.
 * @param {MoveSelection} sel
 * @param {string} move
 * @returns {MoveSelection}
 */
export function contextSelection(sel, move) {
    return selectedPlays(sel).includes(move) ? sel : { picked: [], anchor: move, selected: move };
}
