/**
 * sendActions.js — the handlers of actionCatalog.js's « Envoyer » group, on any object a menu is
 * opened on: copy it to another open base, add it to the lot, add a position to a collection.
 * The copies go through transferService.js, with the same history to take them back.
 */

import { get } from 'svelte/store';
import { activeGroupId } from '../stores/groupActivityStore.js';
import { tabGroupsStore } from '../stores/libraryGroupStore.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { copyToGroup, addToBasket } from './transferService.js';
import { setStatusBarMessage } from './databaseService.js';
import { AddPositionToCollection } from '../../wailsjs/go/database/Database.js';
import { logger } from '../utils/logger.js';
import { t, tMsg } from '../i18n';

/** The open bases other than the one on screen, by name. @returns {{ id: number, name: string }[]} */
export function otherBases() {
    const active = get(activeGroupId);
    return get(tabGroupsStore)
        .filter((g) => g.id !== active)
        .map((g) => ({ id: g.id, name: g.name || get(t)('libraryGroups.unnamed') }));
}

/**
 * The context and handlers of the send group for one row of the base on screen.
 * @param {import('../stores/transferBasketStore.js').TransferKind} kind
 * @param {number} id
 * @param {string} label
 * @returns {{ otherBases: { id: number, name: string }[], collections?: { id: number, name: string }[], do: Record<string, (arg?: any) => any> }}
 */
export function sendContext(kind, id, label) {
    const item = () => ({ group: get(activeGroupId), kind, id, label });
    /** @type {Record<string, (arg?: any) => any>} */
    const handlers = {
        copyTo: (/** @type {number} */ dst) => copyToGroup(dst, item()),
        addToLot: () => addToBasket(item())
    };
    if (kind !== 'position') return { otherBases: otherBases(), do: handlers };
    handlers.addToCollection = async (/** @type {{ id: number, name: string }} */ c) => {
        try {
            await AddPositionToCollection(c.id, id);
            setStatusBarMessage(tMsg('collection.positionAdded'));
        } catch (error) {
            logger.error('adding the position to a collection failed:', error);
            setStatusBarMessage(tMsg('transfer.failed', { error: String(error) }));
        }
    };
    const collections = get(collectionsStore).map((c) => ({ id: c.id, name: c.name }));
    return { otherBases: otherBases(), collections, do: handlers };
}
