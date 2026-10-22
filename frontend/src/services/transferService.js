// Transfer between library groups (ADR-0101, ADR-0102): a row dragged onto another group's
// header, or the basket poured into a group, is copied by the backend's one transfer
// (TransferToGroup → Database.TransferTo), in the background, attached to the group it writes.
import { get } from 'svelte/store';
import { TransferToGroup, SetLibraryGroupLocked, CopyPositionToGroup, UndoPositionCopy } from '../../wailsjs/go/gui/App.js';
import { transferBasketStore, lockedGroupsStore, transferUndoStore } from '../stores/transferBasketStore.js';
import { activeGroupId, groupIdsStore, noteGroupJob } from '../stores/groupActivityStore.js';
import { positionStore } from '../stores/positionStore.js';
import { databasePathStore, libraryChanged } from '../stores/databaseStore.js';
import { tabGroupsStore, groupName } from '../stores/libraryGroupStore.js';
import { refreshLibraryCounts } from '../stores/libraryCountsStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';

/**
 * @typedef {import('../stores/transferBasketStore.js').TransferItem} TransferItem
 * @typedef {{ group: number, positionIDs: number[], matchIDs: number[], collectionIDs: number[], lessonIDs: number[] }} TransferBatch
 */

const FIELD = /** @type {const} */ ({ position: 'positionIDs', match: 'matchIDs', collection: 'collectionIDs', lesson: 'lessonIDs' });

/**
 * The items as the backend takes them: one batch per group they come from.
 * @param {TransferItem[]} items
 * @returns {TransferBatch[]}
 */
export function batchesOf(items) {
    /** @type {Map<number, TransferBatch>} */
    const byGroup = new Map();
    for (const item of items) {
        let batch = byGroup.get(item.group);
        if (!batch) {
            batch = { group: item.group, positionIDs: [], matchIDs: [], collectionIDs: [], lessonIDs: [] };
            byGroup.set(item.group, batch);
        }
        const ids = batch[FIELD[item.kind]];
        if (!ids.includes(item.id)) ids.push(item.id);
    }
    return [...byGroup.values()];
}

/** @param {number} id */
const isLocked = (id) => get(lockedGroupsStore).has(id);

/**
 * Copies items into the library of group dst. A locked group refuses, and the status bar says
 * so; items of dst itself have nothing to copy. Resolves to whether the transfer ran to its end.
 * @param {number} dst
 * @param {TransferItem[]} items
 */
export async function transferToGroup(dst, items) {
    if (isLocked(dst)) {
        setStatusBarMessage(tMsg('transfer.locked'));
        return false;
    }
    const batches = batchesOf(items.filter((i) => i.group !== dst));
    if (batches.length === 0) {
        setStatusBarMessage(tMsg('transfer.nothing'));
        return false;
    }
    noteGroupJob(dst, 'running', 'transfer');
    if (dst === get(activeGroupId)) setStatusBarMessage(tMsg('transfer.running'));
    let summary;
    try {
        summary = await TransferToGroup(dst, /** @type {any} */ (batches));
    } catch (error) {
        logger.error('transfer between library groups failed:', error);
        const message = tMsg('transfer.failed', { error: String(error) });
        if (noteGroupJob(dst, 'done', 'transfer', message)) setStatusBarMessage(message);
        return false;
    }
    return transferDone(dst, summary).ok;
}

/**
 * Says how a transfer into dst ended: on screen, the panels read the library again; off screen,
 * its header carries the badge. Says whether it ran to its end, and what was said.
 * @param {number} dst
 * @param {import('../../wailsjs/go/models').gui.TransferSummary} summary
 */
function transferDone(dst, summary) {
    const failed = summary.errors?.length ?? 0;
    const message = summary.cancelled
        ? tMsg('transfer.cancelled')
        : tMsg(failed ? 'transfer.doneWithErrors' : 'transfer.done', {
              // -1: the backend could not count them, and says no number rather than a false one.
              positions: summary.positions < 0 ? '?' : summary.positions,
              matches: summary.matches,
              held: summary.alreadyHeld,
              collections: summary.collections,
              lessons: summary.lessons,
              failed
          });
    if (noteGroupJob(dst, 'done', 'transfer', message)) {
        setStatusBarMessage(message);
        libraryChanged();
        refreshLibraryCounts();
    }
    return { ok: !summary.cancelled && failed === 0, message };
}

/**
 * A position carried from one split-screen pane onto the other: copied straight into the
 * library of that pane (a locked group refuses), and, when the copy added it there, offered to
 * take back from the status bar.
 * @param {number} dst
 * @param {TransferItem} item
 */
export async function copyPositionToPane(dst, item) {
    if (isLocked(dst)) {
        setStatusBarMessage(tMsg('transfer.locked'));
        return false;
    }
    if (item.group === dst) {
        setStatusBarMessage(tMsg('transfer.nothing'));
        return false;
    }
    transferUndoStore.set(null);
    noteGroupJob(dst, 'running', 'transfer');
    let copy;
    try {
        copy = await CopyPositionToGroup(item.group, item.id, dst);
    } catch (error) {
        logger.error('copying a position between split panes failed:', error);
        const message = tMsg('transfer.failed', { error: String(error) });
        noteGroupJob(dst, 'done', 'transfer', message);
        setStatusBarMessage(message);
        return false;
    }
    // Both panes are in sight: the account is said now, whichever of them was written.
    const { ok, message } = transferDone(dst, copy);
    setStatusBarMessage(message);
    if (ok && copy.created) transferUndoStore.set({ dst, id: copy.created, name: groupName(pathOf(dst)) });
    return ok;
}

/** Takes back the last position carried between split panes. */
export async function undoPaneCopy() {
    const undo = get(transferUndoStore);
    if (!undo) return false;
    transferUndoStore.set(null);
    try {
        await UndoPositionCopy(undo.dst, undo.id);
    } catch (error) {
        logger.error('taking back a position copy failed:', error);
        setStatusBarMessage(tMsg('transfer.undoFailed', { error: String(error) }));
        return false;
    }
    setStatusBarMessage(tMsg('transfer.undone', { name: undo.name }));
    if (undo.dst === get(activeGroupId)) {
        libraryChanged();
        refreshLibraryCounts();
    }
    return true;
}

/**
 * The basket gesture, like the Pile's: the position on the board goes into the basket, or
 * leaves it. A draft is not a row of the library yet: it is saved first.
 */
export function toggleBasket() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDbOpenedFirst'));
        return;
    }
    const position = get(positionStore);
    if (!position?.id) {
        setStatusBarMessage(tMsg('transfer.basketDraft'));
        return;
    }
    const item = stamped({ group: get(activeGroupId), kind: 'position', id: position.id, label: `#${position.id}` });
    const had = get(transferBasketStore).some((i) => sameItem(i, item));
    if (had) transferBasketStore.update((items) => items.filter((i) => !sameItem(i, item)));
    else transferBasketStore.update((items) => [...items, item]);
    setStatusBarMessage(tMsg(had ? 'transfer.basketOut' : 'transfer.basketIn', { n: get(transferBasketStore).length }));
}

/** @param {TransferItem} a @param {TransferItem} b */
const sameItem = (a, b) => a.group === b.group && a.kind === b.kind && a.id === b.id;

/** @param {number} group */
const pathOf = (group) => get(tabGroupsStore).find((g) => g.id === group)?.path ?? '';

/** The item with the file its group holds now. @param {TransferItem} item @returns {TransferItem} */
const stamped = (item) => ({ ...item, path: pathOf(item.group) });

/** @param {TransferItem} item */
export function addToBasket(item) {
    item = stamped(item);
    if (!get(transferBasketStore).some((i) => sameItem(i, item))) transferBasketStore.update((items) => [...items, item]);
    setStatusBarMessage(tMsg('transfer.basketIn', { n: get(transferBasketStore).length }));
}

export function emptyBasket() {
    transferBasketStore.set([]);
}

/**
 * Pours the whole basket into group dst at once. What it copied leaves the basket; on a
 * refusal or a failure the basket stays as it was, to pour again.
 * @param {number} dst
 */
export async function pourBasket(dst) {
    // A row whose group has since opened another file names nothing there any more.
    const items = get(transferBasketStore).filter((i) => i.path === pathOf(i.group));
    if (items.length !== get(transferBasketStore).length) transferBasketStore.set(items);
    if (items.length === 0) {
        setStatusBarMessage(tMsg('transfer.basketEmpty'));
        return false;
    }
    const ok = await transferToGroup(dst, items);
    if (ok) transferBasketStore.update((now) => now.filter((i) => !items.includes(i) || i.group === dst));
    return ok;
}

/**
 * Locks or unlocks group id against transfers into it.
 * @param {number} id
 */
export async function toggleGroupLock(id) {
    const locked = !isLocked(id);
    try {
        await SetLibraryGroupLocked(id, locked);
    } catch (error) {
        logger.error('locking a library group failed:', error);
        return;
    }
    lockedGroupsStore.update((s) => {
        const next = new Set(s);
        if (locked) next.add(id);
        else next.delete(id);
        return next;
    });
    setStatusBarMessage(tMsg(locked ? 'transfer.lockOn' : 'transfer.lockOff'));
}

/**
 * Where a drag ended: a group's header (its id), a split-screen pane ({ pane: its group }) or the
 * basket. The basket itself can be dragged onto a header, which pours it there.
 * @param {number | 'basket' | { pane: number }} target
 * @param {TransferItem | { kind: 'basket' }} dragged
 */
export function dropTransferItem(target, dragged) {
    if (typeof target === 'object') {
        if (dragged.kind === 'position') copyPositionToPane(target.pane, /** @type {TransferItem} */ (dragged));
        else if (dragged.kind !== 'basket') transferToGroup(target.pane, [/** @type {TransferItem} */ (dragged)]);
        else pourBasket(target.pane);
        return;
    }
    if (dragged.kind === 'basket') {
        if (target !== 'basket') pourBasket(target);
        return;
    }
    const item = /** @type {TransferItem} */ (dragged);
    if (target === 'basket') addToBasket(item);
    else if (target !== item.group && get(groupIdsStore).includes(target)) transferToGroup(target, [item]);
}
