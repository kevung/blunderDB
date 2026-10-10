/**
 * Narrows a value a test knows to be present (a DOM query that must match, a
 * mock call that must have happened) so the type checker follows; a missing
 * value fails the test at the point of use rather than further down.
 *
 * @template T
 * @param {T | null | undefined} value
 * @returns {T}
 */
export function must(value) {
    if (value === null || value === undefined) {
        throw new Error('must(): expected a value, got ' + value);
    }
    return value;
}
