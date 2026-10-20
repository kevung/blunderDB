import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get, writable } from 'svelte/store';

vi.mock('../../wailsjs/go/gui/App.js', () => ({ TransferToGroup: vi.fn(), SetLibraryGroupLocked: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../stores/libraryGroupStore.js', () => ({
    tabGroupsStore: writable([
        { id: 1, path: '/libs/mine.db' },
        { id: 2, path: '/libs/club.db' }
    ])
}));
vi.mock('../stores/libraryCountsStore.js', () => ({ refreshLibraryCounts: vi.fn() }));

import { TransferToGroup, SetLibraryGroupLocked } from '../../wailsjs/go/gui/App.js';
import { setStatusBarMessage } from '../services/databaseService.js';
import { tabGroupsStore } from '../stores/libraryGroupStore.js';
import { transferBasketStore, lockedGroupsStore, forgetTransferGroup } from '../stores/transferBasketStore.js';
import { groupIdsStore, activeGroupId, groupActivityStore } from '../stores/groupActivityStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { batchesOf, transferToGroup, toggleBasket, pourBasket, toggleGroupLock, dropTransferItem, addToBasket } from '../services/transferService.js';

const done = { group: 2, positions: 1, matches: 1, alreadyHeld: 0, collections: 0, lessons: 0, errors: [], cancelled: false };
const lastMessage = () => vi.mocked(setStatusBarMessage).mock.calls.at(-1)?.[0];

describe('transferService', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        transferBasketStore.set([]);
        lockedGroupsStore.set(new Set());
        groupIdsStore.set([1, 2]);
        activeGroupId.set(1);
        groupActivityStore.set(new Map());
        databasePathStore.set('/libs/mine.db');
        positionStore.set({ ...emptyPosition(), id: 7 });
        vi.mocked(TransferToGroup).mockResolvedValue(done);
    });

    it('sends one batch per source group, each id once', () => {
        expect(
            batchesOf([
                { group: 1, kind: 'position', id: 7, label: '' },
                { group: 1, kind: 'position', id: 7, label: '' },
                { group: 1, kind: 'match', id: 3, label: '' },
                { group: 2, kind: 'lesson', id: 4, label: '' }
            ])
        ).toEqual([
            { group: 1, positionIDs: [7], matchIDs: [3], collectionIDs: [], lessonIDs: [] },
            { group: 2, positionIDs: [], matchIDs: [], collectionIDs: [], lessonIDs: [4] }
        ]);
    });

    it('copies a row dropped on another group, and badges that group off screen', async () => {
        dropTransferItem(2, { group: 1, kind: 'match', id: 3, label: 'A – B' });
        await vi.waitFor(() => expect(get(groupActivityStore).get(2)?.state).toBe('done'));
        expect(TransferToGroup).toHaveBeenCalledWith(2, [{ group: 1, positionIDs: [], matchIDs: [3], collectionIDs: [], lessonIDs: [] }]);
    });

    it('drops nothing on the group the row comes from', () => {
        dropTransferItem(1, { group: 1, kind: 'match', id: 3, label: '' });
        expect(TransferToGroup).not.toHaveBeenCalled();
    });

    it('refuses a locked group and says so, without calling the backend', async () => {
        vi.mocked(SetLibraryGroupLocked).mockResolvedValue(undefined);
        await toggleGroupLock(2);
        expect(SetLibraryGroupLocked).toHaveBeenCalledWith(2, true);
        expect(await transferToGroup(2, [{ group: 1, kind: 'position', id: 7, label: '' }])).toBe(false);
        expect(TransferToGroup).not.toHaveBeenCalled();
        expect(lastMessage()).toEqual(expect.objectContaining({ i18nKey: 'transfer.locked' }));
    });

    it('fills the basket with the position on the board, and the same gesture takes it out', () => {
        toggleBasket();
        expect(get(transferBasketStore)).toEqual([{ group: 1, kind: 'position', id: 7, label: '#7', path: '/libs/mine.db' }]);
        toggleBasket();
        expect(get(transferBasketStore)).toEqual([]);
    });

    it('keeps a draft out of the basket', () => {
        positionStore.set(emptyPosition());
        toggleBasket();
        expect(get(transferBasketStore)).toEqual([]);
        expect(lastMessage()).toEqual(expect.objectContaining({ i18nKey: 'transfer.basketDraft' }));
    });

    it('pours the basket at once and empties what it copied', async () => {
        addToBasket({ group: 1, kind: 'position', id: 7, label: '' });
        addToBasket({ group: 1, kind: 'collection', id: 2, label: '' });
        expect(await pourBasket(2)).toBe(true);
        expect(TransferToGroup).toHaveBeenCalledTimes(1);
        expect(TransferToGroup).toHaveBeenCalledWith(2, [{ group: 1, positionIDs: [7], matchIDs: [], collectionIDs: [2], lessonIDs: [] }]);
        expect(get(transferBasketStore)).toEqual([]);
    });

    it('keeps the basket when the transfer fails', async () => {
        vi.mocked(TransferToGroup).mockResolvedValue({ ...done, errors: ['boom'] });
        addToBasket({ group: 1, kind: 'position', id: 7, label: '' });
        expect(await pourBasket(2)).toBe(false);
        expect(get(transferBasketStore)).toHaveLength(1);
    });

    it('drops the rows of a group that has since opened another file', async () => {
        addToBasket({ group: 1, kind: 'position', id: 7, label: '' });
        /** @type {import("svelte/store").Writable<any>} */ (tabGroupsStore).set([
            { id: 1, path: '/libs/other.db' },
            { id: 2, path: '/libs/club.db' }
        ]);
        expect(await pourBasket(2)).toBe(false);
        expect(TransferToGroup).not.toHaveBeenCalled();
        expect(get(transferBasketStore)).toEqual([]);
        /** @type {import("svelte/store").Writable<any>} */ (tabGroupsStore).set([
            { id: 1, path: '/libs/mine.db' },
            { id: 2, path: '/libs/club.db' }
        ]);
    });

    it('forgets the rows and the lock of a closed group', () => {
        addToBasket({ group: 2, kind: 'match', id: 1, label: '' });
        lockedGroupsStore.set(new Set([2]));
        forgetTransferGroup(2);
        expect(get(transferBasketStore)).toEqual([]);
        expect(get(lockedGroupsStore).has(2)).toBe(false);
    });
});
