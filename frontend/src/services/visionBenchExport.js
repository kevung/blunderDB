// Development only: the events the vision core reported, written to a file the bench
// (`tools/vision/mesure.py`) compares with the Repères. Nothing goes to the database, and the
// descriptor journal stays local (ADR-0108 rule 6).
import { SaveBoardImageDialog, SaveBoardSVG } from '../../wailsjs/go/gui/App.js';

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
