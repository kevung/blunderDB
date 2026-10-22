// Exchanges between open libraries (ADR-0101, ADR-0102): a row dragged onto another group's
// header, copied from the « Autres bases » panel or the context menu, or the lot copied into a
// base, goes through the backend's one transfer (TransferToGroup → Database.TransferTo), in the
// background, attached to the group it writes. A copy that created something keeps a receipt:
// the session's history takes it back, newest first.
import { get } from 'svelte/store';
import { TransferToGroup, SetLibraryGroupLocked, CopyPositionToGroup, UndoPositionCopy } from '../../wailsjs/go/gui/App.js';
import { transferBasketStore, lockedGroupsStore, transferHistoryStore, protectedPaths, rememberProtection } from '../stores/transferBasketStore.js';
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
 * @typedef {{ group: number, positionIDs: number[], matchIDs: number[], tournamentIDs: number[], collectionIDs: number[], lessonIDs: number[] }} TransferBatch
 */

const FIELD = /** @type {const} */ ({ position: 'positionIDs', match: 'matchIDs', tournament: 'tournamentIDs', collection: 'collectionIDs', lesson: 'lessonIDs' });

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
            batch = { group: item.group, positionIDs: [], matchIDs: [], tournamentIDs: [], collectionIDs: [], lessonIDs: [] };
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
    const { ok } = transferDone(dst, summary);
    keepReceipt(dst, ok, summary);
    return ok;
}

/**
 * A copy that created something in dst joins the session's history, to take back from the
 * status bar or the « Autres bases » panel.
 * @param {number} dst
 * @param {boolean} ok
 * @param {{ receipt?: number }} summary
 */
function keepReceipt(dst, ok, summary) {
    if (ok && summary.receipt) transferHistoryStore.update((h) => [...h, { dst, receipt: summary.receipt, name: groupName(pathOf(dst)) }]);
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
              tournaments: summary.tournaments ?? 0,
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
 * A position copied alone into the library of group dst (a protected group refuses); when the
 * copy added it there, it joins the session's history, to take back from the status bar.
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
    noteGroupJob(dst, 'running', 'transfer');
    let copy;
    try {
        copy = await CopyPositionToGroup(item.group, item.id, dst);
    } catch (error) {
        logger.error('copying a position between libraries failed:', error);
        const message = tMsg('transfer.failed', { error: String(error) });
        noteGroupJob(dst, 'done', 'transfer', message);
        setStatusBarMessage(message);
        return false;
    }
    // The copy was asked from what is on screen: the account is said now, whichever base was written.
    const { ok, message } = transferDone(dst, copy);
    setStatusBarMessage(message);
    keepReceipt(dst, ok, copy);
    return ok;
}

/**
 * Copies one row into the base of group dst: a position alone keeps its receipt, the other rows
 * go through the transfer of a lot.
 * @param {number} dst
 * @param {TransferItem} item
 */
export function copyToGroup(dst, item) {
    return item.kind === 'position' ? copyPositionToPane(dst, item) : transferToGroup(dst, [item]);
}

/**
 * Brings position id of the base of group src into the base on screen, with the same receipt
 * as a copy the other way.
 * @param {number} src
 * @param {number} id
 */
export function fetchPositionFrom(src, id) {
    return copyPositionToPane(get(activeGroupId), { group: src, kind: 'position', id, label: `#${id}` });
}

/** Takes back the session's last copy still in the history. */
export async function undoPaneCopy() {
    const history = get(transferHistoryStore);
    const undo = history[history.length - 1];
    if (!undo) return false;
    try {
        await UndoPositionCopy(undo.receipt);
    } catch (error) {
        logger.error('taking back a copy failed:', error);
        // A copy written on since, or a base that opened another file, is gone for good; any
        // other refusal (a protection, a passing failure) keeps it to try again.
        if (String(error).includes('has changed since the copy')) transferHistoryStore.update((h) => h.filter((u) => u !== undo));
        setStatusBarMessage(tMsg('transfer.undoFailed', { error: String(error) }));
        return false;
    }
    transferHistoryStore.update((h) => h.filter((u) => u !== undo));
    setStatusBarMessage(tMsg('transfer.undone', { name: undo.name }));
    if (undo.dst === get(activeGroupId)) {
        libraryChanged();
        refreshLibraryCounts();
    }
    return true;
}

/**
 * The lot gesture, like the Pile's (Maj+B): the position on the board goes into the lot, or
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

/** Takes one row out of the lot. @param {TransferItem} item */
export function removeFromBasket(item) {
    transferBasketStore.update((items) => items.filter((i) => !sameItem(i, item)));
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
 * Protects group id from copies into it, or lifts the protection. Not quiet: the status bar says
 * so. The protection is remembered by the file, for the next session.
 * @param {number} id
 * @param {{ quiet?: boolean, locked?: boolean }} [options]
 */
export async function toggleGroupLock(id, { quiet = false, locked = !isLocked(id) } = {}) {
    try {
        await SetLibraryGroupLocked(id, locked);
    } catch (error) {
        logger.error('protecting a library group failed:', error);
        return;
    }
    lockedGroupsStore.update((s) => {
        const next = new Set(s);
        if (locked) next.add(id);
        else next.delete(id);
        return next;
    });
    if (quiet) return;
    rememberProtection(pathOf(id), locked);
    setStatusBarMessage(tMsg(locked ? 'transfer.lockOn' : 'transfer.lockOff'));
}

// A file protected in an earlier session is protected again as soon as a group opens it.
tabGroupsStore.subscribe((groups) => {
    const saved = protectedPaths();
    if (saved.length === 0) return;
    for (const g of groups) {
        if (g.path && saved.includes(g.path) && !isLocked(g.id)) toggleGroupLock(g.id, { quiet: true, locked: true });
    }
});

/**
 * Where a drag ended: a group's header (its id), a split-screen pane ({ pane: its group }) or the
 * lot. The lot itself can be dragged onto a header, which copies it there.
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
