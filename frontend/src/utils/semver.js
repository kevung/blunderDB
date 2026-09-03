// Minimal semver comparison for blunderDB's own release numbering
// (scripts/release.sh always tags a bare "X.Y.Z" — no pre-release or build
// metadata suffix, see CLAUDE.md's "Changelog: major-only" note), used by
// the opt-in update check (#241) to decide whether a fetched GitHub release
// tag is newer than the running version. Not a general-purpose semver
// parser: a version string outside X.Y.Z is treated as not comparable
// (isNewerVersion returns false) rather than guessed at.

/**
 * Parses "X.Y.Z" into [X, Y, Z], or null if the string doesn't match that
 * shape.
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
 * Reports whether `candidate` is strictly newer than `current`, both
 * "X.Y.Z". Returns false (never throws) for anything that doesn't parse —
 * an unparsable version is treated as "nothing to report" rather than a
 * reason to alarm the user.
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
