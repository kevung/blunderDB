/**
 * positionList.js — the browsed list of positions (library, search result, collection, deck) as
 * an id list plus a bounded cache. Holding full positions cost ~45 MB of JSON across the Wails
 * bridge per reload on a 50 000-position library; ids are ~100 KB, and positions are fetched by
 * window through `loader` (LoadPositionsByIDs, or a test stub).
 *
 * The id list itself is held one of two ways. A *local* list (`setIds`, `set`) keeps every id: a
 * search result, a collection, a deck. A *paged* list (`setSource`) keeps only its length and the
 * id pages it has been asked for, fetched through a source (`count`, `window`, `indexOf`): the
 * library, whose size has no bound. `idAt`/`indexOf` answer synchronously from what is held;
 * `resolveIdAt`/`findIndex` ask the source when it is not.
 *
 * Store value: `{ ids, length, paged }`, frozen; `ids` is null for a paged list. `getPosition(i)`
 * loads the window around `i` on a miss and prefetches ahead of the browsing direction;
 * `peek(i)` is a synchronous cache lookup.
 */
import { writable } from 'svelte/store';

/** @typedef {{ id?: number | null, [key: string]: any }} Position */
/** @typedef {(number | null)[]} IdList */
/**
 * @typedef {object} IdSource
 * @property {() => Promise<number>} count how many ids the list holds
 * @property {(offset: number, limit: number) => Promise<number[]>} window ids [offset, offset+limit)
 * @property {(id: number) => Promise<number>} indexOf the rank of id, or -1
 */
/** @typedef {{ ids: IdList } | { source: IdSource, length: number }} ListSnapshot */

export const DEFAULT_WINDOW_SIZE = 50;
export const DEFAULT_CACHE_SIZE = 512;
export const DEFAULT_BATCH_SIZE = 500;
export const DEFAULT_ID_PAGE_SIZE = 1000;
export const DEFAULT_ID_PAGES = 32;

/**
 * @param {IdList | null} ids
 * @param {number} length
 */
function snapshot(ids, length) {
    return Object.freeze({ ids, length, paged: ids === null });
}

/**
 * The rank of `id` in a list snapshot (`snapshotList`), asking a paged list's source; -1 if absent.
 * @param {ListSnapshot | null | undefined} list
 * @param {number} id
 */
export async function indexInList(list, id) {
    if (!list) return -1;
    if ('source' in list) {
        const index = await list.source.indexOf(id);
        return Number.isInteger(index) ? index : -1;
    }
    return list.ids.indexOf(id);
}

/** @param {ListSnapshot | null | undefined} list */
export function listLength(list) {
    if (!list) return 0;
    return 'source' in list ? list.length : list.ids.length;
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
 * @param {number} [options.idPageSize] ids per source call of a paged list.
 * @param {number} [options.idPages] most id pages a paged list keeps.
 */
export function createPositionList({
    loader = async () => [],
    windowSize = DEFAULT_WINDOW_SIZE,
    cacheSize = DEFAULT_CACHE_SIZE,
    batchSize = DEFAULT_BATCH_SIZE,
    idPageSize = DEFAULT_ID_PAGE_SIZE,
    idPages = DEFAULT_ID_PAGES
} = {}) {
    const { subscribe, set: publish } = writable(snapshot([], 0));

    /** @type {IdList} the local list; empty while paged */
    let ids = [];
    /** @type {IdSource | null} set while the list is paged */
    let source = null;
    let pagedLength = 0;
    /** @type {Map<number, number[]>} page number → its ids, insertion order = LRU order */
    const idPageCache = new Map();
    /** @type {Map<number, Promise<void>>} page number → the fetch that will bring it */
    const idPagePending = new Map();
    // Bumped whenever the list is replaced: a page fetched for an older list is dropped.
    let listEpoch = 0;
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

    const length = () => (source ? pagedLength : ids.length);

    function dropIdPages() {
        listEpoch++;
        idPageCache.clear();
        idPagePending.clear();
    }

    /** @param {IdList} next */
    function replaceIds(next) {
        source = null;
        pagedLength = 0;
        dropIdPages();
        ids = next;
        indexById = null;
        publish(snapshot(ids, ids.length));
    }

    /**
     * @param {IdSource} next
     * @param {number} total
     */
    function replaceSource(next, total) {
        dropIdPages();
        ids = [];
        indexById = null;
        source = next;
        pagedLength = Math.max(0, Number.isInteger(total) ? total : 0);
        publish(snapshot(null, pagedLength));
    }

    /** @param {number} i */
    function inBounds(i) {
        return Number.isInteger(i) && i >= 0 && i < length();
    }

    /**
     * The id at `i` from what is held, or undefined (out of bounds, or a page not loaded).
     * @param {number} i
     */
    function idAt(i) {
        if (!inBounds(i)) return undefined;
        if (!source) return ids[i];
        const page = idPageCache.get(Math.floor(i / idPageSize));
        return page ? page[i % idPageSize] : undefined;
    }

    /** @param {number} page */
    function fetchIdPage(page) {
        const cached = idPageCache.get(page);
        if (cached) {
            idPageCache.delete(page);
            idPageCache.set(page, cached);
            return null;
        }
        const inFlight = idPagePending.get(page);
        if (inFlight) return inFlight;
        const from = source;
        const epoch = listEpoch;
        const request = Promise.resolve()
            .then(() => /** @type {IdSource} */ (from).window(page * idPageSize, idPageSize))
            .then((rows) => {
                if (epoch !== listEpoch) return;
                idPageCache.set(page, Array.isArray(rows) ? rows : []);
                while (idPageCache.size > idPages) {
                    const oldest = idPageCache.keys().next();
                    if (oldest.done) break;
                    idPageCache.delete(oldest.value);
                }
            })
            .finally(() => {
                if (idPagePending.get(page) === request) idPagePending.delete(page);
            });
        idPagePending.set(page, request);
        return request;
    }

    /**
     * Make the ids of [from, to] available to `idAt` (a no-op for a local list).
     * @param {number} from
     * @param {number} to
     */
    async function ensureIds(from, to) {
        if (!source) return;
        const lo = Math.max(0, from);
        const hi = Math.min(pagedLength - 1, to);
        if (hi < lo) return;
        const requests = [];
        for (let page = Math.floor(lo / idPageSize); page <= Math.floor(hi / idPageSize); page++) {
            const request = fetchIdPage(page);
            if (request) requests.push(request);
        }
        await Promise.all(requests);
    }

    /**
     * The ids of [from, to), in order. A paged list reads past its page cache straight from the
     * source, so a bulk read does not evict the pages being browsed.
     * @param {number} from
     * @param {number} to
     */
    async function idsBetween(from, to) {
        const lo = Math.max(0, from);
        const hi = Math.min(length(), to);
        if (hi <= lo) return [];
        if (!source) return ids.slice(lo, hi);
        /** @type {IdList} */
        const out = [];
        for (let i = lo; i < hi;) {
            const page = Math.floor(i / idPageSize);
            const end = Math.min(hi, (page + 1) * idPageSize);
            const held = idPageCache.get(page);
            const rows = held ? held.slice(i - page * idPageSize, end - page * idPageSize) : await source.window(i, end - i);
            for (const id of rows || []) out.push(id);
            i = end;
        }
        return out;
    }

    /**
     * @param {number} from
     * @param {number} to
     */
    function range(from, to) {
        const out = [];
        for (let i = Math.max(0, from); i <= Math.min(length() - 1, to); i++) out.push(i);
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
            const id = idAt(i);
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
        const ahead = async () => {
            await ensureIds(i + 1, i + windowSize);
            if (i + reach < length() && !covered(idAt(i + reach))) await fetchMissing(range(i + 1, i + windowSize));
        };
        const behind = async () => {
            await ensureIds(i - windowSize, i - 1);
            if (i - reach >= 0 && !covered(idAt(i - reach))) await fetchMissing(range(i - windowSize, i - 1));
        };
        if (!source) {
            if (i + reach < ids.length && !covered(ids[i + reach])) fetchMissing(range(i + 1, i + windowSize))?.catch(noop);
            if (i - reach >= 0 && !covered(ids[i - reach])) fetchMissing(range(i - windowSize, i - 1))?.catch(noop);
            return;
        }
        ahead().catch(noop);
        behind().catch(noop);
    }

    /**
     * The position at index `i`, loading the window around it on a miss; null out of bounds or
     * gone. Sequential browsing costs one loader call per half-window.
     * @param {number} i
     */
    async function getPosition(i) {
        if (!inBounds(i)) return null;
        if (source && idAt(i) === undefined) await ensureIds(i - windowSize, i + windowSize);
        const id = idAt(i);
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
        if (source) {
            /** @type {Position[]} */
            const out = [];
            for (let start = Math.max(0, from); start < Math.min(length(), to); start += batchSize) {
                out.push(...(await positionsOf(await idsBetween(start, Math.min(to, start + batchSize)))));
            }
            return out;
        }
        return positionsOf(range(from, to - 1).map((i) => ids[i]));
    }

    /**
     * The positions of `list`, in order, missing ones skipped; cached ones are not refetched.
     * @param {IdList} list
     */
    async function positionsOf(list) {
        /** @type {Map<number, Position>} */
        const byId = new Map();
        /** @type {number[]} */
        const missing = [];
        for (const id of list) {
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
        for (const id of list) {
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
         * Replace the list by a paged one read through `next`; resolves to its length. `reset`
         * also drops the position cache.
         * @param {IdSource} next
         * @param {{ reset?: boolean }} [options]
         */
        async setSource(next, { reset = false } = {}) {
            const total = await next.count();
            if (reset) {
                cache.clear();
                absent.clear();
            }
            replaceSource(next, total);
            return pagedLength;
        },

        /**
         * Re-read a paged list's length after positions were added (they come last, ids ascending)
         * or removed (every page is refetched). A local list is left as it is.
         */
        async recount() {
            if (!source) return ids.length;
            const current = source;
            const total = await current.count();
            if (source !== current || total === pagedLength) return length();
            if (total < pagedLength) {
                replaceSource(current, total);
            } else {
                // Grown at the end (ids ascend): full pages still hold, a partial one does not.
                for (const [page, rows] of idPageCache) if (rows.length < idPageSize) idPageCache.delete(page);
                listEpoch++;
                idPagePending.clear();
                pagedLength = total;
                publish(snapshot(null, pagedLength));
            }
            return length();
        },

        /** Whether the list is paged (read through a source) rather than held whole. */
        isPaged() {
            return source !== null;
        },

        /**
         * Whether the list is the paged one `of` reads.
         * @param {IdSource} of
         */
        isSource(of) {
            return source !== null && source === of;
        },

        /**
         * What the list is, to put it back later with `restoreList` (a view, a mode left and
         * re-entered): the ids of a local list, the source and length of a paged one.
         * @returns {ListSnapshot}
         */
        snapshotList() {
            return source ? { source, length: pagedLength } : { ids: [...ids] };
        },

        /**
         * Put back a list taken by `snapshotList`. A paged list keeps its length until `recount`.
         * @param {ListSnapshot | null | undefined} list
         */
        restoreList(list) {
            if (list && 'source' in list) {
                if (source !== list.source || pagedLength !== list.length) replaceSource(list.source, list.length);
            } else {
                replaceIds(list && Array.isArray(list.ids) ? [...list.ids] : []);
            }
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
         * The id at index `i`, or undefined out of bounds — or, for a paged list, when its page is
         * not loaded (the page of a position just shown always is).
         * @param {number} i
         */
        idAt,

        /**
         * The id at index `i`, fetching its page when the list is paged.
         * @param {number} i
         */
        async resolveIdAt(i) {
            if (!inBounds(i)) return undefined;
            await ensureIds(i, i);
            return idAt(i);
        },

        /** Make the ids of [from, to] available to `idAt`. */
        ensureIds,

        /** The ids of [from, to), in order. */
        idsBetween,

        /**
         * The first index holding `id` among the ids held, or -1. A paged list only looks in its
         * loaded pages: `findIndex` asks the source.
         * @param {number} id
         */
        indexOf(id) {
            if (source) {
                for (const [page, rows] of idPageCache) {
                    const at = rows.indexOf(id);
                    if (at >= 0) return page * idPageSize + at;
                }
                return -1;
            }
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
         * The index of `id`, or -1; a paged list asks its source when no loaded page holds it.
         * @param {number} id
         */
        async findIndex(id) {
            const held = this.indexOf(id);
            if (held >= 0 || !source) return held;
            const index = await source.indexOf(id);
            return Number.isInteger(index) && index >= 0 && index < pagedLength ? index : -1;
        },

        /**
         * Synchronous cache lookup; undefined when not loaded.
         * @param {number} i
         */
        peek(i) {
            const id = idAt(i);
            return id == null ? undefined : cache.get(id);
        },

        getPosition,
        getPositions,

        /** Every position of the list, in order, fetched in batches. */
        getAllPositions() {
            return getPositions(0, length());
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
        },
        /** Instrumentation for tests: ids a paged list holds in its pages. */
        get heldIds() {
            let n = 0;
            for (const rows of idPageCache.values()) n += rows.length;
            return n;
        }
    };
}
