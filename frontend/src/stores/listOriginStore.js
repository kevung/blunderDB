import { writable } from 'svelte/store';

// Where the list on screen comes from, in a form small enough to persist and
// replay: the library, or the search payload that produced it. A session and
// its views keep this, never the ids themselves (the list may hold millions).
export const LIBRARY_ORIGIN = Object.freeze({ kind: 'library' });

// A search payload larger than this is not worth persisting: it carries a
// pasted id list, and the view falls back to the library.
export const MAX_ORIGIN_PAYLOAD_BYTES = 32 * 1024;

export const listOriginStore = writable(LIBRARY_ORIGIN);

export function searchOrigin(payload) {
    try {
        const copy = JSON.parse(JSON.stringify(payload));
        if (JSON.stringify(copy).length > MAX_ORIGIN_PAYLOAD_BYTES) return LIBRARY_ORIGIN;
        return { kind: 'search', payload: copy };
    } catch {
        return LIBRARY_ORIGIN;
    }
}
