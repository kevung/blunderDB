// Minimal semver comparison for the opt-in update check: release.sh tags bare "X.Y.Z". Anything
// else is not comparable (isNewerVersion returns false), never guessed at.

/**
 * Parses "X.Y.Z" into [X, Y, Z], or null.
 * @param {string} v
 * @returns {[number, number, number] | null}
 */
export function parseVersion(v) {
    if (typeof v !== 'string') return null;
    const m = /^(\d+)\.(\d+)\.(\d+)$/.exec(v.trim());
    if (!m) return null;
    return [Number(m[1]), Number(m[2]), Number(m[3])];
}

/**
 * Whether `candidate` is strictly newer than `current`, both "X.Y.Z"; false (never throws) for
 * anything unparsable.
 * @param {string} candidate
 * @param {string} current
 * @returns {boolean}
 */
export function isNewerVersion(candidate, current) {
    const a = parseVersion(candidate);
    const b = parseVersion(current);
    if (!a || !b) return false;
    for (let i = 0; i < 3; i++) {
        if (a[i] > b[i]) return true;
        if (a[i] < b[i]) return false;
    }
    return false;
}
