// The calibrations of a draft's video: four clicks on the board, named, kept per draft on this
// machine like the angle and the instant the video resumes at. A convenience of the viewer,
// never written to the database. Clicks are fractions of the picture ([0,1], the unturned
// image), so they survive a resize of the pane and a rotation of the video.

const KEY_PREFIX = 'blunderdb.transcription.videoCalibrations.';

/**
 * @typedef {[number, number]} Click
 * @typedef {{ name: string, clicks: Click[] }} Calibration four clicks, in the order of the model's corners
 * @typedef {{ items: Calibration[], active: string | null }} CalibrationSet
 */

/** @param {unknown} v */
function validClick(v) {
    return Array.isArray(v) && v.length === 2 && v.every((n) => typeof n === 'number' && Number.isFinite(n) && n >= 0 && n <= 1);
}

/** @param {unknown} c @returns {c is Calibration} */
function validCalibration(c) {
    const o = /** @type {any} */ (c);
    return !!o && typeof o.name === 'string' && o.name !== '' && Array.isArray(o.clicks) && o.clicks.length === 4 && o.clicks.every(validClick);
}

/**
 * @param {string | number | null | undefined} draftId
 * @returns {CalibrationSet}
 */
export function loadCalibrations(draftId) {
    const empty = { items: [], active: null };
    if (draftId == null || draftId === '') return empty;
    try {
        const raw = JSON.parse(localStorage.getItem(KEY_PREFIX + String(draftId)) ?? 'null');
        if (!raw || !Array.isArray(raw.items)) return empty;
        /** @type {Calibration[]} */
        const items = [];
        for (const c of raw.items) if (validCalibration(c) && !items.some((i) => i.name === c.name)) items.push({ name: c.name, clicks: c.clicks.map((p) => /** @type {Click} */ ([p[0], p[1]])) });
        const active = items.some((i) => i.name === raw.active) ? raw.active : (items[0]?.name ?? null);
        return { items, active };
    } catch (_e) {
        return empty;
    }
}

/**
 * @param {string | number | null | undefined} draftId
 * @param {CalibrationSet} set
 */
export function saveCalibrations(draftId, set) {
    if (draftId == null || draftId === '') return;
    try {
        if (set.items.length === 0) localStorage.removeItem(KEY_PREFIX + String(draftId));
        else localStorage.setItem(KEY_PREFIX + String(draftId), JSON.stringify(set));
    } catch (_e) {
        /* storage refused: the calibration lasts until the pane closes */
    }
}

/**
 * A name not yet used: "base", then "base 2", "base 3"…
 * @param {Calibration[]} items
 * @param {string} base
 */
export function freeName(items, base) {
    let name = base;
    for (let n = 2; items.some((i) => i.name === name); n++) name = `${base} ${n}`;
    return name;
}

/**
 * Where the picture lies in a box that shows it whole (object-fit: contain), in box pixels.
 * @param {number} boxW
 * @param {number} boxH
 * @param {number} mediaW natural width; 0 when unknown, the picture then fills the box
 * @param {number} mediaH
 * @returns {{ x: number, y: number, w: number, h: number }}
 */
export function pictureRect(boxW, boxH, mediaW, mediaH) {
    if (!(mediaW > 0 && mediaH > 0) || !(boxW > 0 && boxH > 0)) return { x: 0, y: 0, w: boxW, h: boxH };
    const s = Math.min(boxW / mediaW, boxH / mediaH);
    const w = mediaW * s;
    const h = mediaH * s;
    return { x: (boxW - w) / 2, y: (boxH - h) / 2, w, h };
}

/**
 * A pointer position, relative to the centre of the turned element, as a fraction of the
 * unturned picture. `dx`,`dy` are measured on screen; the element turns clockwise by `rotation`.
 * @param {number} dx
 * @param {number} dy
 * @param {number} rotation 0, 90, 180 or 270
 * @param {{ x: number, y: number, w: number, h: number }} pic in element pixels
 * @param {number} boxW element width (unturned)
 * @param {number} boxH
 * @returns {Click | null} null outside the picture
 */
export function pointerToPicture(dx, dy, rotation, pic, boxW, boxH) {
    const a = (-rotation * Math.PI) / 180;
    const lx = dx * Math.cos(a) - dy * Math.sin(a) + boxW / 2;
    const ly = dx * Math.sin(a) + dy * Math.cos(a) + boxH / 2;
    const fx = (lx - pic.x) / pic.w;
    const fy = (ly - pic.y) / pic.h;
    if (!(fx >= 0 && fx <= 1 && fy >= 0 && fy <= 1)) return null;
    return [Math.round(fx * 1e6) / 1e6, Math.round(fy * 1e6) / 1e6];
}
