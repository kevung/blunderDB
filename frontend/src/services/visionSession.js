// The panel's side of the vision core for one video: the calibration in force becomes the
// ROIs of the core, the core's events flow into the panel's store, and the presence of the
// calibrated board is judged here, at about 2 Hz, against a thumbnail taken at calibration
// (ADR-0109 rule 3). The reference thumbnail lives in localStorage with the calibration, per
// draft; nothing of it is written to the database.

import { createVision } from '../vision/vision.js';
import { createFrameGrabber } from '../vision/sampler.js';
import { EVENT_TYPES, makeEvent, STATES } from '../vision/contract.js';
import { BOARD_RECT, boardRois, createPresenceTracker, presenceScore, rectifyZone } from '../vision/geometry/index.js';
import { overlayInFrame, scaleHomography } from './videoCalibration.js';

/** Presence is read twice a second. */
export const PRESENCE_PERIOD_MS = 500;
/** The reference thumbnail: the rectified board, in pixels. */
export const THUMB = Object.freeze({ width: 64, height: 32 });
/** The frame the thumbnail is sampled from. */
const FRAME = Object.freeze({ width: 320, height: 180 });
const KEY_PREFIX = 'blunderdb.transcription.videoPresence.';

/**
 * @typedef {{ name: string, clicks: [number, number][], modelToImage: number[], overlay?: { x: number, y: number, w: number, h: number } }} ActiveCalibration
 */

/** @param {string} key @param {ActiveCalibration} c */
function loadReference(key, c) {
    try {
        const raw = JSON.parse(localStorage.getItem(KEY_PREFIX + key + '.' + c.name) ?? 'null');
        if (raw && JSON.stringify(raw.clicks) === JSON.stringify(c.clicks) && Array.isArray(raw.data) && raw.data.length === THUMB.width * THUMB.height) {
            return { width: THUMB.width, height: THUMB.height, data: Float32Array.from(raw.data, (/** @type {number | null} */ v) => v ?? NaN) };
        }
    } catch (_e) {
        /* unreadable: the reference is taken again */
    }
    return null;
}

/** @param {string} key @param {ActiveCalibration} c @param {{ data: Float32Array }} thumb */
function saveReference(key, c, thumb) {
    try {
        localStorage.setItem(KEY_PREFIX + key + '.' + c.name, JSON.stringify({ clicks: c.clicks, data: Array.from(thumb.data, (v) => (Number.isNaN(v) ? null : Math.round(v))) }));
    } catch (_e) {
        /* storage refused: the reference lasts until the pane closes */
    }
}

/**
 * @param {object} options
 * @param {HTMLVideoElement} options.video
 * @param {string | number} options.key the draft
 * @param {(event: import('../vision/contract.js').VisionEvent) => void} options.push where the events go
 * @param {ReturnType<typeof createVision>} [options.vision]
 * @param {() => { grab(): { width: number, height: number, data: ArrayLike<number> } }} [options.createGrabber]
 */
export function createVisionSession({ video, key, push, vision = createVision({ video }), createGrabber = () => createFrameGrabber({ video, ...FRAME }) }) {
    const unsubscribe = vision.subscribe(push);
    /** @type {ActiveCalibration | null} */
    let active = null;
    let session = '';
    /** @type {ReturnType<typeof createPresenceTracker> | null} */
    let tracker = null;
    /** @type {{ width: number, height: number, data: ArrayLike<number> } | null} */
    let reference = null;
    /** @type {ReturnType<typeof setInterval> | null} */
    let timer = null;
    /** @type {{ grab(): { width: number, height: number, data: ArrayLike<number> } } | null} */
    let grabber = null;
    let running = false;

    function thumbnail() {
        if (!active) return null;
        grabber ??= createGrabber();
        const image = grabber.grab();
        return rectifyZone(image, scaleHomography(active.modelToImage, image.width, image.height), BOARD_RECT, THUMB.width, THUMB.height);
    }

    function tick() {
        if (!active || !tracker || video.readyState < 2 || video.seeking) return;
        let thumb;
        try {
            thumb = thumbnail();
        } catch (_e) {
            return; // an unreadable frame says nothing about the board
        }
        if (!thumb) return;
        if (!reference) {
            reference = thumb;
            saveReference(String(key), active, thumb);
            return;
        }
        const score = presenceScore(reference, thumb);
        const { present, changed } = tracker.update(score);
        if (!changed) return;
        push(makeEvent(present ? EVENT_TYPES.VIEW_FOUND : EVENT_TYPES.VIEW_LOST, Math.max(0, video.currentTime), STATES.SURE, { score }));
        // While the board is out of the picture the analysis keeps silent.
        if (present) startAnalysis();
        else stopAnalysis();
    }

    function startAnalysis() {
        if (running) return;
        running = true;
        vision.start({ session });
    }
    function stopAnalysis() {
        if (!running) return;
        running = false;
        vision.stop();
    }

    function halt() {
        stopAnalysis();
        if (timer !== null) clearInterval(timer);
        timer = null;
    }

    /** @returns {boolean} whether the calibration was handed to the core */
    function begin() {
        if (!active || !(video.videoWidth > 0) || !(video.videoHeight > 0)) return false;
        const res = boardRois({ modelToImage: scaleHomography(active.modelToImage, video.videoWidth, video.videoHeight), sourceWidth: video.videoWidth, sourceHeight: video.videoHeight });
        if (!res.ok) return false;
        const overlay = overlayInFrame(active.overlay, FRAME.width, FRAME.height);
        vision.calibrate(overlay ? { ...res.calibration, overlay } : res.calibration);
        session = `${key}:${active.name}`;
        reference = loadReference(String(key), active);
        tracker = createPresenceTracker();
        startAnalysis();
        timer ??= setInterval(tick, PRESENCE_PERIOD_MS);
        return true;
    }

    const onMeta = () => {
        video.removeEventListener('loadedmetadata', onMeta);
        begin();
    };

    return {
        /** @param {ActiveCalibration | null} c the calibration in force, null when there is none */
        calibrate(c) {
            halt();
            active = c;
            tracker = null;
            reference = null;
            video.removeEventListener('loadedmetadata', onMeta);
            if (!c) return;
            // The size of the picture is known once its metadata is.
            if (!begin()) video.addEventListener('loadedmetadata', onMeta);
        },
        /** Erases the journal of the vision core for this draft's calibrations. */
        clearJournal() {
            return vision.clearJournal();
        },
        dispose() {
            halt();
            video.removeEventListener('loadedmetadata', onMeta);
            unsubscribe();
            vision.dispose();
        }
    };
}
