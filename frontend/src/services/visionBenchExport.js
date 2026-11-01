// Development only: the events the vision core reported, written to a file the bench
// (`tools/vision/mesure.py`) compares with the Repères. Nothing goes to the database, and the
// descriptor journal stays local (ADR-0108 rule 6).
import { SaveBoardImageDialog, SaveBoardSVG } from '../../wailsjs/go/gui/App.js';
import { listJournalSessions, readJournal } from '../vision/journal.js';
import { loadCalibrations } from './videoCalibration.js';

/**
 * The bench file: events in arrival order, `mediaTime` in seconds as in the contract.
 * @param {readonly import('../vision/contract.js').VisionEvent[]} events
 * @param {string} calibration
 * @param {string} source
 */
export function benchFile(events, calibration, source) {
    return {
        events: events.map((e) => ({ type: e.type, mediaTime: e.mediaTime, state: e.state })),
        calibration,
        source
    };
}

/**
 * @param {readonly import('../vision/contract.js').VisionEvent[]} events
 * @param {string} calibration
 * @param {string} source
 * @returns {Promise<string | null>} the path written, or null when the dialog was cancelled
 */
export async function saveBenchEvents(events, calibration, source) {
    const path = await SaveBoardImageDialog('json', 'events.json');
    if (!path) return null;
    await SaveBoardSVG(path, JSON.stringify(benchFile(events, calibration, source), null, 2));
    return path;
}

/**
 * The descriptor journal of a draft for `tools/vision/replay.mjs`: every session named
 * `<draftId>:<calibration>`, the records in writing order, and the calibrations kept for that
 * draft (clicks, homography, overlay) so the place of the witness can be recomputed.
 * Read-only: nothing is erased.
 * @param {string | number} draftId
 * @param {{ indexedDB?: IDBFactory }} [options]
 */
export async function journalFile(draftId, { indexedDB = globalThis.indexedDB } = {}) {
    const prefix = `${draftId}:`;
    const names = (await listJournalSessions({ indexedDB })).filter((s) => s.startsWith(prefix));
    /** @type {Record<string, import('../vision/descriptors.js').DescriptorRecord[]>} */
    const sessions = {};
    for (const name of names) sessions[name] = await readJournal({ session: name, indexedDB });
    const set = loadCalibrations(draftId);
    /** @type {Record<string, import('./videoCalibration.js').Calibration>} */
    const calibrations = {};
    for (const c of set.items) if (names.includes(prefix + c.name)) calibrations[prefix + c.name] = c;
    return { draftId: String(draftId), sessions, calibrations };
}

/**
 * @param {string | number} draftId
 * @returns {Promise<string | null>} the path written, or null when the dialog was cancelled
 */
export async function saveBenchJournal(draftId) {
    const path = await SaveBoardImageDialog('json', 'journal.json');
    if (!path) return null;
    await SaveBoardSVG(path, JSON.stringify(await journalFile(draftId)));
    return path;
}
