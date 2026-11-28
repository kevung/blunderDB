/**
 * Multiple selection of a list's rows: Ctrl/Cmd+click adds or removes a row, Shift+click selects
 * the range from the last clicked row. A click with neither modifier is not handled here.
 */

/** The most matches one print or one paper archive takes: printsheet.MaxMatches. */
export const MAX_SELECTION = 100;

/**
 * @template T
 * @param {T[]} rows the rows in display order
 * @param {(row: T) => any} keyOf
 * @param {Set<any>} current the keys already selected
 * @param {any} anchor the key last clicked, or null
 * @param {T} row the row clicked
 * @param {{ ctrlKey?: boolean, metaKey?: boolean, shiftKey?: boolean }} event
 * @param {any} [seed] the key of the single row already selected, which a first Ctrl/Shift+click keeps
 * @returns {{ keys: Set<any>, anchor: any } | null} null for a plain click
 */
export function modifierSelect(rows, keyOf, current, anchor, row, event, seed = null) {
    const key = keyOf(row);
    if (event.shiftKey) {
        const from = anchor ?? seed ?? key;
        let a = rows.findIndex((r) => keyOf(r) === from);
        let b = rows.findIndex((r) => keyOf(r) === key);
        if (a < 0) a = b;
        if (a > b) [a, b] = [b, a];
        const keys = new Set(event.ctrlKey || event.metaKey ? current : []);
        for (const r of rows.slice(a, b + 1)) keys.add(keyOf(r));
        return { keys, anchor: from };
    }
    if (event.ctrlKey || event.metaKey) {
        const keys = new Set(current);
        if (keys.size === 0 && seed != null) keys.add(seed);
        if (keys.has(key)) keys.delete(key);
        else keys.add(key);
        return { keys, anchor: key };
    }
    return null;
}

/**
 * The ids a command applies to when invoked on `id`: the whole selection, in display order, when
 * `id` belongs to a selection of several rows; `id` alone otherwise.
 * @template T
 * @param {T[]} rows
 * @param {(row: T) => number} idOf
 * @param {Set<number>} selection
 * @param {number} id
 * @returns {number[]}
 */
export function idsFor(rows, idOf, selection, id) {
    if (selection.size > 1 && selection.has(id)) return rows.map(idOf).filter((x) => selection.has(x));
    return [id];
}
