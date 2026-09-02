/**
 * positionList.js
 *
 * The browsed list of positions (the library, a search result, a collection,
 * a deck) as an id list plus a bounded cache of loaded positions.
 *
 * The board shows one position at a time, yet the list used to be the full
 * array of positions: on a 50 000-position library that is ~45 MB of JSON
 * across the Wails bridge on every reload. The store now keeps only the ids
 * (~100 KB) and fetches positions in windows around the index being shown,
 * through the `loader` it was built with (LoadPositionsByIDs in the app, a
 * stub in tests).
 *
 * Store value: `{ ids, length }`, frozen, replaced whenever the list changes.
 * Positions are reached through the async `getPosition(i)`, which loads the
 * window around `i` on a miss and prefetches ahead of the browsing
 * direction, or `peek(i)` for a synchronous cache lookup of a position that
 * is already shown.
 */
import { writable } from 'svelte/store';

export const DEFAULT_WINDOW_SIZE = 50;
export const DEFAULT_CACHE_SIZE = 512;
export const DEFAULT_BATCH_SIZE = 500;

function snapshot(ids) {
    return Object.freeze({ ids, length: ids.length });
}

const noop = () => {};

/**
 * @param {object} [options]
 * @param {(ids: number[]) => Promise<object[]>} [options.loader] fetches
 *   positions by id, in any order; missing ids are simply absent.
 * @param {number} [options.windowSize] half-width of the window loaded
 *   around a missed index, and the reach of the prefetch.
 * @param {number} [options.cacheSize] most positions kept; the least
 *   recently used are evicted first.
 * @param {number} [options.batchSize] ids per loader call in bulk reads.
 */
export function createPositionList({ loader = async () => [], windowSize = DEFAULT_WINDOW_SIZE, cacheSize = DEFAULT_CACHE_SIZE, batchSize = DEFAULT_BATCH_SIZE } = {}) {
    const { subscribe, set: publish } = writable(snapshot([]));

    /** @type {number[]} */
    let ids = [];
    /** @type {Map<number, number> | null} id → first index, built on demand */
    let indexById = null;
    /** @type {Map<number, object>} id → position, insertion order = LRU order */
    const cache = new Map();
    /** @type {Map<number, Promise<void>>} id → the fetch that will bring it */
    const pending = new Map();
    /** @type {Set<number>} ids a fetch asked for and did not get back (deleted) */
    const absent = new Set();
    let loadFn = loader;
    let loaderCalls = 0;

    // ── Cache ────────────────────────────────────────────────────────────

    function remember(position) {
        const id = position?.id;
        if (id == null) return;
        absent.delete(id);
        cache.delete(id);
        cache.set(id, position);
        while (cache.size > cacheSize) {
            cache.delete(cache.keys().next().value);
        }
    }

    function hit(id) {
        const position = cache.get(id);
        if (position !== undefined) {
            cache.delete(id);
            cache.set(id, position);
        }
        return position;
    }

    const covered = (id) => cache.has(id) || pending.has(id) || absent.has(id);

    // ── List ─────────────────────────────────────────────────────────────

    function replaceIds(next) {
        ids = next;
        indexById = null;
        publish(snapshot(ids));
    }

    function inBounds(i) {
        return Number.isInteger(i) && i >= 0 && i < ids.length;
    }

    function range(from, to) {
        const out = [];
        for (let i = Math.max(0, from); i <= Math.min(ids.length - 1, to); i++) out.push(i);
        return out;
    }

    // ── Loading ──────────────────────────────────────────────────────────

    /**
     * One loader call for the ids of `indices` that are neither cached nor
     * already on their way. Returns null when nothing is missing.
     */
    function fetchMissing(indices) {
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
                // An id that did not come back no longer exists: remember it
                // so browsing past it does not ask again at every step.
                for (const id of batch) if (!cache.has(id)) absent.add(id);
            })
            .finally(() => {
                for (const id of batch) if (pending.get(id) === request) pending.delete(id);
            });
        for (const id of batch) pending.set(id, request);
        return request;
    }

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
     * The position at index `i`, loading the window around it on a miss.
     * Resolves to null when `i` is out of bounds or the position no longer
     * exists. Browsing sequentially costs one loader call per half-window.
     */
    async function getPosition(i) {
        if (!inBounds(i)) return null;
        const id = ids[i];
        if (!cache.has(id)) {
            // Already on its way (a neighbour's window): wait for that fetch
            // rather than open another; a real miss loads the whole window.
            if (pending.has(id)) await pending.get(id);
            else if (!absent.has(id)) await fetchMissing(range(i - windowSize, i + windowSize));
        }
        prefetchAround(i);
        return hit(id) ?? null;
    }

    /**
     * The positions of indices [from, to), in list order, missing ones
     * skipped. Bulk reads go through the loader in batches and do not touch
     * the cache: an export of 50 000 positions must not evict the window
     * being browsed.
     */
    async function getPositions(from, to) {
        const indices = range(from, to - 1);
        const byId = new Map();
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
        const out = [];
        for (const i of indices) {
            const position = byId.get(ids[i]);
            if (position !== undefined) out.push(position);
        }
        return out;
    }

    // ── Public surface ───────────────────────────────────────────────────

    return {
        subscribe,

        /**
         * Replace the list by ids. `reset` drops the cache too: use it when
         * stored positions may have changed (a reload after an edit).
         */
        setIds(next, { reset = false } = {}) {
            if (reset) {
                cache.clear();
                absent.clear();
            }
            replaceIds(Array.isArray(next) ? [...next] : []);
        },

        /**
         * Replace the list by full positions, which also seed the cache (the
         * first `cacheSize` of them). Search results, collections and decks
         * still arrive whole from the backend; this keeps their first window
         * free of a round trip.
         */
        set(positions) {
            const list = Array.isArray(positions) ? positions : [];
            // Seeded back to front so index 0 — where a fresh list opens — is
            // the most recently used and the last to be evicted.
            const seeded = list.slice(0, cacheSize);
            for (let i = seeded.length - 1; i >= 0; i--) remember(seeded[i]);
            replaceIds(list.map((p) => (p && p.id != null ? p.id : null)));
        },

        /** The id at index `i`, or undefined out of bounds. */
        idAt(i) {
            return inBounds(i) ? ids[i] : undefined;
        },

        /** The first index holding `id`, or -1. */
        indexOf(id) {
            if (!indexById) {
                indexById = new Map();
                ids.forEach((v, i) => {
                    if (v != null && !indexById.has(v)) indexById.set(v, i);
                });
            }
            return indexById.get(id) ?? -1;
        },

        /** Synchronous cache lookup; undefined when not loaded. */
        peek(i) {
            return inBounds(i) ? cache.get(ids[i]) : undefined;
        },

        getPosition,
        getPositions,

        /** Every position of the list, in order, fetched in batches. */
        getAllPositions() {
            return getPositions(0, ids.length);
        },

        /** Put a freshly saved or edited position in the cache. */
        upsert(position) {
            remember(position);
        },

        /** Forget one cached position (or all of them). */
        invalidate(id) {
            if (id === undefined) {
                cache.clear();
                absent.clear();
            } else {
                cache.delete(id);
                absent.delete(id);
            }
        },

        /** Swap the loader (tests). */
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
