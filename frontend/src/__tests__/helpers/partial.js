/**
 * A test fixture that names only the fields the test reads. Bind it to a
 * model with a cast of the identity function, so the object literal is checked
 * against the model's field types rather than against its full shape:
 *
 *   const statsResult = /** @type {(v: DeepPartial<StatsResult>) => StatsResult} *\/ (partial);
 *
 * @template T
 * @typedef {T extends (infer U)[] ? DeepPartial<U>[] : T extends object ? { [K in keyof T]?: DeepPartial<T[K]> } : T} DeepPartial
 */

/**
 * @param {any} value
 * @returns {any}
 */
export function partial(value) {
    return value;
}
