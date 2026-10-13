/**
 * captureService.js — a screen capture of a board, read onto the scratch board.
 *
 * The reading is a proposal: it lands on the EDIT board with what it doubted ringed there, and
 * nothing is stored until the user saves the corrected position by the usual gesture.
 */
import { get } from 'svelte/store';
import { ReadCaptureData, ReadCaptureFile } from '../../wailsjs/go/database/Database.js';
import { statusBarModeStore } from '../stores/uiStore.js';
import { duelHoldsBoardStore } from '../stores/duelStore.js';
import { captureDoubtsStore } from '../stores/captureStore.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';
import { setStatusBarMessage } from './databaseService.js';
import { applyPositionToBoard } from './importService.js';
import { enterEditMode } from './modeMachine.js';

export const CAPTURE_EXTENSIONS = ['png', 'jpg', 'jpeg', 'gif'];

/** @param {string} path */
export function isCaptureFile(path) {
    const ext = String(path).toLowerCase().split('.').pop() ?? '';
    return CAPTURE_EXTENSIONS.includes(ext);
}

// The doubts Board.svelte rings on the EDIT board; the others are named in the status line.
const RINGED_FIELDS = new Set(['point', 'bar', 'cube']);

/** @param {string[]} fields */
const fieldNames = (fields) => fields.map((f) => tMsg(`status.captureField.${f}`));

/**
 * Puts a reading on the EDIT board (entering EDIT unless a scratch board is already up).
 * @param {{ position: any, software?: string, doubts?: import('../stores/captureStore.js').CaptureDoubt[], unread?: string[] }} reading
 * @param {number} [ignored] further images dropped with this one and not read
 */
export async function showCaptureOnBoard(reading, ignored = 0) {
    // EDIT, not EVAL: only EDIT draws the doubts the status message announces.
    if (get(statusBarModeStore) !== 'EDIT') await enterEditMode();
    applyPositionToBoard(reading.position);
    const doubts = reading.doubts ?? [];
    captureDoubtsStore.set(doubts);
    const ringed = doubts.filter((d) => RINGED_FIELDS.has(d.field)).length;
    const unsure = [...new Set(doubts.filter((d) => !RINGED_FIELDS.has(d.field)).map((d) => d.field))];
    const key = ringed > 0 ? 'status.captureOnBoardDoubts' : 'status.captureOnBoard';
    setStatusBarMessage(
        tMsg(key, {
            software: reading.software ?? '',
            count: ringed,
            unread: fieldNames(reading.unread ?? []),
            unsure: unsure.length > 0 ? tMsg('status.captureUnsure', { fields: fieldNames(unsure) }) : '',
            more: ignored > 0 ? tMsg('status.captureOnlyFirst', { count: ignored + 1 }) : ''
        })
    );
}

/** @param {() => Promise<any>} read @param {number} [ignored] */
async function readOntoBoard(read, ignored = 0) {
    if (get(duelHoldsBoardStore)) return false;
    try {
        await showCaptureOnBoard(await read(), ignored);
        return true;
    } catch (error) {
        logger.error('could not read the capture:', error);
        setStatusBarMessage(tMsg('status.captureFailed', { error }));
        return false;
    }
}

/**
 * @param {string} path an image file on disk
 * @param {number} [ignored] further images dropped with it, named in the same status line
 */
export function importCaptureFile(path, ignored = 0) {
    return readOntoBoard(() => ReadCaptureFile(path), ignored);
}

/** @param {Blob} blob an image from the clipboard */
export function importCaptureBlob(blob) {
    return readOntoBoard(async () => ReadCaptureData(await blobToBase64(blob)));
}

/** @param {Blob} blob */
async function blobToBase64(blob) {
    const bytes = new Uint8Array(await blob.arrayBuffer());
    let binary = '';
    for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
    return btoa(binary);
}

/**
 * The image a paste event carries, if any; null for a text paste.
 * @param {ClipboardEvent} event
 * @returns {File | null}
 */
export function imageOfPaste(event) {
    for (const item of event.clipboardData?.items ?? []) {
        if (item.kind === 'file' && item.type.startsWith('image/')) return item.getAsFile();
    }
    return null;
}

/**
 * Window-level paste handler: an image on the clipboard is read as a capture, in any mode.
 * Text is left to the Ctrl+V keyboard path.
 * @param {ClipboardEvent} event
 */
export function handlePasteEvent(event) {
    const target = /** @type {HTMLElement | null} */ (event.target);
    if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) return;
    const image = imageOfPaste(event);
    if (!image) return;
    event.preventDefault();
    importCaptureBlob(image);
}
