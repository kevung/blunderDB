/**
 * positionList.js — the browsed list of positions (library, search result, collection, deck) as
 * an id list plus a bounded cache. Holding full positions cost ~45 MB of JSON across the Wails
 * bridge per reload on a 50 000-position library; ids are ~100 KB, and positions are fetched by
 * window through `loader` (LoadPositionsByIDs, or a test stub).
 *
 * Store value: `{ ids, length }`, frozen. `getPosition(i)` loads the window around `i` on a miss
 * and prefetches ahead of the browsing direction; `peek(i)` is a synchronous cache lookup.
 */
import { writable } from 'svelte/store';

/** @typedef {{ id?: number | null, [key: string]: any }} Position */
/** @typedef {(number | null)[]} IdList */

export const DEFAULT_WINDOW_SIZE = 50;
export const DEFAULT_CACHE_SIZE = 512;
export const DEFAULT_BATCH_SIZE = 500;

/** @param {IdList} ids */
function snapshot(ids) {
    return Object.freeze({ ids, length: ids.length });
}

const noop = () => {};

/**
 * @param {object} [options]
 * @param {(ids: number[]) => Promise<Position[]>} [options.loader] fetches
 *   positions by id, in any order; missing ids are simply absent.
 * @param {number} [options.windowSize] half-width of the window loaded
 *   around a missed index, and the reach of the prefetch.
 * @param {number} [options.cacheSize] most positions kept; the least
 *   recently used are evicted first.
 * @param {number} [options.batchSize] ids per loader call in bulk reads.
 */
export function createPositionList({ loader = async () => [], windowSize = DEFAULT_WINDOW_SIZE, cacheSize = DEFAULT_CACHE_SIZE, batchSize = DEFAULT_BATCH_SIZE } = {}) {
    const { subscribe, set: publish } = writable(snapshot([]));

    /** @type {IdList} */
    let ids = [];
    /** @type {Map<number, number> | null} id → first index, built on demand */
    let indexById = null;
    /** @type {Map<number, Position>} id → position, insertion order = LRU order */
    const cache = new Map();
    /** @type {Map<number, Promise<void>>} id → the fetch that will bring it */
    const pending = new Map();
    /** @type {Set<number>} ids a fetch asked for and did not get back (deleted) */
    const absent = new Set();
    let loadFn = loader;
    let loaderCalls = 0;

    // ── Cache ────────────────────────────────────────────────────────────

    /** @param {Position} position */
    function remember(position) {
        const id = position?.id;
        if (id == null) return;
        absent.delete(id);
        cache.delete(id);
        cache.set(id, position);
        while (cache.size > cacheSize) {
            const oldest = cache.keys().next();
            if (oldest.done) break;
            cache.delete(oldest.value);
        }
    }

    /** @param {number} id */
    function hit(id) {
        const position = cache.get(id);
        if (position !== undefined) {
            cache.delete(id);
            cache.set(id, position);
        }
        return position;
    }

    /** @param {number | null | undefined} id */
    const covered = (id) => id != null && (cache.has(id) || pending.has(id) || absent.has(id));

    // ── List ─────────────────────────────────────────────────────────────

    /** @param {IdList} next */
    function replaceIds(next) {
        ids = next;
        indexById = null;
        publish(snapshot(ids));
    }

    /** @param {number} i */
    function inBounds(i) {
        return Number.isInteger(i) && i >= 0 && i < ids.length;
    }

    /**
     * @param {number} from
     * @param {number} to
     */
    function range(from, to) {
        const out = [];
        for (let i = Math.max(0, from); i <= Math.min(ids.length - 1, to); i++) out.push(i);
        return out;
    }

    // ── Loading ──────────────────────────────────────────────────────────

    /**
     * One loader call for the ids of `indices` neither cached nor in flight; null if none.
     * @param {number[]} indices
     */
    function fetchMissing(indices) {
        /** @type {Set<number>} */
        const missing = new Set();
        for (const i of indices) {
            const id = ids[i];
            if (id != null && !covered(id)) missing.add(id);
        }
        if (missing.size === 0) return null;
        const batch = [...missing];
        loaderCalls++;
        const request = Promise.resolve()
            .then(() => loadFn(batch))
            .then((rows) => {
                for (const row of rows || []) remember(row);
                // A missing id no longer exists: remember it so browsing does not ask again.
                for (const id of batch) if (!cache.has(id)) absent.add(id);
            })
            .finally(() => {
                for (const id of batch) if (pending.get(id) === request) pending.delete(id);
            });
        for (const id of batch) pending.set(id, request);
        return request;
    }

    /** @param {number} i */
    function prefetchAround(i) {
        const reach = Math.max(1, Math.floor(windowSize / 2));
        if (i + reach < ids.length && !covered(ids[i + reach])) {
            fetchMissing(range(i + 1, i + windowSize))?.catch(noop);
        }
        if (i - reach >= 0 && !covered(ids[i - reach])) {
            fetchMissing(range(i - windowSize, i - 1))?.catch(noop);
        }
    }

    /**
     * The position at index `i`, loading the window around it on a miss; null out of bounds or
     * gone. Sequential browsing costs one loader call per half-window.
     * @param {number} i
     */
    async function getPosition(i) {
        if (!inBounds(i)) return null;
        const id = ids[i];
        if (id == null) return null;
        if (!cache.has(id)) {
            // Already in flight (a neighbour's window): wait for it; a real miss loads the window.
            if (pending.has(id)) await pending.get(id);
            else if (!absent.has(id)) await fetchMissing(range(i - windowSize, i + windowSize));
        }
        prefetchAround(i);
        return hit(id) ?? null;
    }

    /**
     * The positions of [from, to), in order, missing ones skipped. Bulk reads bypass the cache:
     * an export of 50 000 positions must not evict the browsed window.
     * @param {number} from
     * @param {number} to
     */
    async function getPositions(from, to) {
        const indices = range(from, to - 1);
        /** @type {Map<number, Position>} */
        const byId = new Map();
        /** @type {number[]} */
        const missing = [];
        for (const i of indices) {
            const id = ids[i];
            if (id == null) continue;
            const cached = cache.get(id);
            if (cached !== undefined) byId.set(id, cached);
            else if (!byId.has(id)) missing.push(id);
        }
        for (let start = 0; start < missing.length; start += batchSize) {
            loaderCalls++;
            const rows = await loadFn(missing.slice(start, start + batchSize));
            for (const row of rows || []) if (row?.id != null) byId.set(row.id, row);
        }
        /** @type {Position[]} */
        const out = [];
        for (const i of indices) {
            const id = ids[i];
            const position = id == null ? undefined : byId.get(id);
            if (position !== undefined) out.push(position);
        }
        return out;
    }

    // ── Public surface ───────────────────────────────────────────────────

    return {
        subscribe,

        /**
         * Replace the list by ids. `reset` also drops the cache (stored positions may have changed).
         * @param {IdList} next
         * @param {{ reset?: boolean }} [options]
         */
        setIds(next, { reset = false } = {}) {
            if (reset) {
                cache.clear();
                absent.clear();
            }
            replaceIds(Array.isArray(next) ? [...next] : []);
        },

        /**
         * Replace the list by full positions, which also seed the cache (first `cacheSize`) so
         * search results, collections and decks, still returned whole, skip a round trip.
         * @param {Position[]} positions
         */
        set(positions) {
            const list = Array.isArray(positions) ? positions : [];
            // Seeded back to front, so index 0 (where a list opens) is evicted last.
            const seeded = list.slice(0, cacheSize);
            for (let i = seeded.length - 1; i >= 0; i--) remember(seeded[i]);
            replaceIds(list.map((p) => (p && p.id != null ? p.id : null)));
        },

        /**
         * The id at index `i`, or undefined out of bounds.
         * @param {number} i
         */
        idAt(i) {
            return inBounds(i) ? ids[i] : undefined;
        },

        /**
         * The first index holding `id`, or -1.
         * @param {number} id
         */
        indexOf(id) {
            if (!indexById) {
                /** @type {Map<number, number>} */
                const built = new Map();
                ids.forEach((v, i) => {
                    if (v != null && !built.has(v)) built.set(v, i);
                });
                indexById = built;
            }
            return indexById.get(id) ?? -1;
        },

        /**
         * Synchronous cache lookup; undefined when not loaded.
         * @param {number} i
         */
        peek(i) {
            const id = inBounds(i) ? ids[i] : null;
            return id == null ? undefined : cache.get(id);
        },

        getPosition,
        getPositions,

        /** Every position of the list, in order, fetched in batches. */
        getAllPositions() {
            return getPositions(0, ids.length);
        },

        /**
         * Put a freshly saved or edited position in the cache.
         * @param {Position} position
         */
        upsert(position) {
            remember(position);
        },

        /**
         * Forget one cached position (or all of them).
         * @param {number} [id]
         */
        invalidate(id) {
            if (id === undefined) {
                cache.clear();
                absent.clear();
            } else {
                cache.delete(id);
                absent.delete(id);
            }
        },

        /**
         * Swap the loader (tests).
         * @param {(ids: number[]) => Promise<Position[]>} fn
         */
        setLoader(fn) {
            loadFn = fn;
        },

        /** Instrumentation for tests: loader calls made so far. */
        get loaderCalls() {
            return loaderCalls;
        },
        get cacheSize() {
            return cache.size;
        }
    };
}
