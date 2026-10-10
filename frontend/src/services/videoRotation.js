// The angle a draft's video is turned to, kept per draft on this machine like the instant it
// resumes at: a convenience of the viewer, never written to the database. A video filmed
// sideways stays upright when the draft is opened again.

const KEY_PREFIX = 'blunderdb.transcription.videoRotation.';
export const ROTATIONS = Object.freeze([0, 90, 180, 270]);

/** @param {number} angle */
export function nextRotation(angle) {
    const at = ROTATIONS.indexOf(angle);
    return ROTATIONS[(at + 1) % ROTATIONS.length];
}

/** @param {string | number | null | undefined} draftId */
export function loadVideoRotation(draftId) {
    if (draftId == null || draftId === '') return 0;
    try {
        const angle = Number(localStorage.getItem(KEY_PREFIX + String(draftId)));
        return ROTATIONS.includes(angle) ? angle : 0;
    } catch (_e) {
        return 0;
    }
}

/**
 * @param {string | number | null | undefined} draftId
 * @param {number} angle
 */
export function saveVideoRotation(draftId, angle) {
    if (draftId == null || draftId === '' || !ROTATIONS.includes(angle)) return;
    try {
        if (angle === 0) localStorage.removeItem(KEY_PREFIX + String(draftId));
        else localStorage.setItem(KEY_PREFIX + String(draftId), String(angle));
    } catch (_e) {
        /* storage refused: the video shows upright */
    }
}
