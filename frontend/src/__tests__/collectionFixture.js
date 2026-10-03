// Open a collection the way the panel does: by its id source, the positions pre-loaded in the list's
// cache so the first one is shown without a Wails round trip.
import { positionsStore } from '../stores/positionStore.js';

/**
 * An id source over a fixed list of positions.
 * @param {{ id: number }[]} positions
 * @returns {import('../stores/positionList.js').IdSource}
 */
export function collectionSourceOf(positions) {
    const ids = positions.map((p) => p.id);
    return {
        count: async () => ids.length,
        window: async (offset, limit) => (limit > 0 ? ids.slice(offset, offset + limit) : ids.slice(offset)),
        indexOf: async (id) => ids.indexOf(id)
    };
}

/**
 * @param {(collection: any, source: import('../stores/positionList.js').IdSource) => Promise<void>} handleOpenCollection
 * @param {any} collection
 * @param {{ id: number }[]} positions
 */
export async function openCollectionOf(handleOpenCollection, collection, positions) {
    for (const position of positions) positionsStore.upsert(position);
    await handleOpenCollection(collection, collectionSourceOf(positions));
}
