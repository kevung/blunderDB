// The Eval panel's evaluations, kept per position for the life of the window so that coming
// back to a position shows what was already computed instead of searching again. A cache of
// derived results only: in memory, never written, never exported (ADR-0007).
//
// An entry is keyed by everything that changes the answer — the position (every field but its
// provenance: id, flagged, individually_imported), the pruning width and the candidate count —
// and is valid only under the match equity table that produced it (metId, which every answer
// names). For a position it keeps the deepest evaluation computed: a shallower answer never
// replaces a deeper one, and each play keeps its own deepest row, so a play re-run alone at a
// greater depth keeps that depth beside the others.

export const EVAL_CACHE_LIMIT = 256;

const PROVENANCE = ['id', 'flagged', 'individually_imported'];

/**
 * @param {any} pos the position as positionStore holds it
 * @param {{ pruneK: number, candidates: number }} settings
 * @returns {string}
 */
export function evalCacheKey(pos, { pruneK, candidates }) {
    const game = { ...(pos ?? {}) };
    for (const field of PROVENANCE) delete game[field];
    return `${JSON.stringify(game)}|${pruneK}|${candidates}`;
}

/**
 * The ply of a depth label ("2-ply"); -1 when there is none.
 * @param {string | null | undefined} label
 */
export function plyOf(label) {
    const m = /^(\d+)-ply$/.exec(label ?? '');
    return m ? Number(m[1]) : -1;
}

/**
 * The plays best first, each error measured against the best, indices renumbered — as the
 * engine ranks one answer, here over rows that may come from several.
 * @param {any[]} moves
 * @returns {any[]}
 */
export function rankMoves(moves) {
    const sorted = [...moves].sort((a, b) => b.equity - a.equity);
    const best = sorted[0]?.equity;
    return sorted.map((m, i) => ({ ...m, index: i, equityError: i === 0 ? null : best - m.equity }));
}

/**
 * base with the rows of update in place of their play's; the plays of update base lacks are
 * added. With deepest, a row of update replaces one only if it is at least as deep.
 * @param {any[]} base
 * @param {any[]} update
 * @param {{ deepest?: boolean }} [opts]
 */
export function mergeMoves(base, update, { deepest = false } = {}) {
    const rows = new Map((base ?? []).map((m) => [m.move, m]));
    for (const m of update ?? []) {
        const old = rows.get(m.move);
        if (!old || !deepest || plyOf(m.analysisDepth) >= plyOf(old.analysisDepth)) rows.set(m.move, m);
    }
    return rankMoves([...rows.values()]);
}

/**
 * For each play of the last run that the previous rows had: how far its equity moved.
 * @param {any[]} previous the rows shown before the run
 * @param {any[]} ran the rows the run produced
 * @returns {Map<string, number>}
 */
export function equityDeltas(previous, ran) {
    const before = new Map((previous ?? []).map((m) => [m.move, m.equity]));
    const out = new Map();
    for (const m of ran ?? []) {
        if (before.has(m.move)) out.set(m.move, m.equity - /** @type {number} */ (before.get(m.move)));
    }
    return out;
}

/** The ply of a whole answer: its cube's depth, or its first play's. */
function answerPly(/** @type {any} */ result) {
    return plyOf(result?.cube?.analysisDepth ?? result?.moves?.[0]?.analysisDepth);
}

/**
 * @typedef {{ metId: number, ply: number, moves?: any[], cube?: any, cubeVerdict?: string, race?: any, preRoll?: any }} EvalEntry
 */

/** @param {number} [limit] */
export function createEvalCache(limit = EVAL_CACHE_LIMIT) {
    /** @type {Map<string, EvalEntry>} */
    const entries = new Map(); // insertion order is recency: the first is the oldest

    return {
        /**
         * The entry of key, if it was valued with the table metId.
         * @param {string} key @param {number} metId
         * @returns {EvalEntry | undefined}
         */
        get(key, metId) {
            const e = entries.get(key);
            if (!e || e.metId !== metId) return undefined;
            entries.delete(key);
            entries.set(key, e);
            return e;
        },

        /**
         * Folds an answer in. A partial one (the plays asked alone) only merges its rows, deepest
         * kept. A whole one at least as deep as the entry replaces its ranking, facts and cube,
         * keeping the plays a targeted run took deeper; a shallower one changes nothing. A refusal
         * is not kept. Another table's entry is dropped.
         * @param {string} key @param {any} result
         * @returns {EvalEntry | undefined} the entry after the fold
         */
        put(key, result) {
            if (!result || result.refused) return entries.get(key);
            const metId = result.metId ?? 0;
            let e = entries.get(key);
            if (e && e.metId !== metId) e = undefined;
            const next = /** @type {EvalEntry} */ (e ? { ...e } : { metId, ply: -1 });
            const ply = answerPly(result);
            if (result.partial) {
                next.moves = mergeMoves(next.moves ?? [], result.moves ?? [], { deepest: true });
            } else if (ply >= next.ply) {
                // The new ranking, plus the plays an earlier targeted run took deeper than it.
                if (result.moves) {
                    const deeper = (next.moves ?? []).filter((m) => plyOf(m.analysisDepth) > ply);
                    next.moves = deeper.length ? mergeMoves(result.moves, deeper) : result.moves;
                }
                next.ply = ply;
                if (result.cube !== undefined) {
                    next.cube = result.cube;
                    next.cubeVerdict = result.cubeVerdict ?? '';
                }
                next.race = result.race ?? null;
                next.preRoll = result.preRoll ?? next.preRoll ?? null;
            }
            entries.delete(key);
            entries.set(key, next);
            while (entries.size > limit) entries.delete(/** @type {string} */ (entries.keys().next().value));
            return next;
        },

        get size() {
            return entries.size;
        },

        clear() {
            entries.clear();
        }
    };
}

/** The panel's cache, shared by its mounts for the life of the window. */
export const evalCache = createEvalCache();
