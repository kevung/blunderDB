// Where a draft's video resumes when the draft is opened again: the instant the user had
// reached, kept per draft on this machine, else a little before the last Repère of the
// document, where the transcription stopped. The browser storage is a convenience: when it
// is empty or refused, the Repères still give a place to resume.

const KEY_PREFIX = 'blunderdb.transcription.videoResume.';
// Before the last Repère, so that the moment is seen coming.
export const RESUME_LEAD_MS = 3000;

/** @param {string | number} draftId */
function keyOf(draftId) {
    return KEY_PREFIX + String(draftId);
}

/**
 * @param {string | number | null | undefined} draftId
 * @param {number | null | undefined} ms
 */
export function saveVideoResume(draftId, ms) {
    if (draftId == null || draftId === '' || typeof ms !== 'number' || !Number.isFinite(ms) || ms < 0) return;
    try {
        localStorage.setItem(keyOf(draftId), String(Math.round(ms)));
    } catch (_e) {
        /* storage refused: the Repères remain */
    }
}

/**
 * @param {string | number | null | undefined} draftId
 * @returns {number | null}
 */
export function loadVideoResume(draftId) {
    if (draftId == null || draftId === '') return null;
    try {
        const raw = localStorage.getItem(keyOf(draftId));
        const ms = raw === null ? NaN : Number(raw);
        return Number.isFinite(ms) && ms > 0 ? ms : null;
    } catch (_e) {
        return null;
    }
}

/**
 * The latest Repère of the document, roll or action, null when it has none.
 *
 * @param {any[] | null | undefined} actions - `transcript.ActionInfo`s
 */
export function lastRepere(actions) {
    let last = null;
    for (const info of actions ?? []) {
        for (const ms of [info?.roll_tick_ms, info?.tick_ms]) {
            if (typeof ms === 'number' && Number.isFinite(ms) && (last === null || ms > last)) last = ms;
        }
    }
    return last;
}

/**
 * The instant a draft's video opens at: the one kept for it, else a little before the last
 * Repère, else 0 (the start, as for a video just attached).
 *
 * @param {string | number | null | undefined} draftId
 * @param {any[] | null | undefined} actions
 */
export function videoResumeMs(draftId, actions) {
    const kept = loadVideoResume(draftId);
    if (kept !== null) return kept;
    const last = lastRepere(actions);
    return last === null ? 0 : Math.max(0, last - RESUME_LEAD_MS);
}
