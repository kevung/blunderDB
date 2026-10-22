/**
 * The list with the item at `from` taken out and put before or after the item at `target`.
 * The same list is returned when nothing moves (unknown key, same item, or already in place).
 * @template T
 * @param {T[]} list
 * @param {(item: T) => unknown} keyOf
 * @param {unknown} key the item to move
 * @param {unknown} targetKey the item it lands beside
 * @param {boolean} after beside the target's right edge rather than its left
 * @returns {T[]}
 */
export function moveItem(list, keyOf, key, targetKey, after) {
    const from = list.findIndex((item) => keyOf(item) === key);
    const target = list.findIndex((item) => keyOf(item) === targetKey);
    if (from < 0 || target < 0 || from === target) return list;
    const rest = list.filter((_, i) => i !== from);
    let at = rest.findIndex((item) => keyOf(item) === targetKey) + (after ? 1 : 0);
    if (at === from) return list;
    at = Math.max(0, Math.min(rest.length, at));
    return [...rest.slice(0, at), list[from], ...rest.slice(at)];
}
