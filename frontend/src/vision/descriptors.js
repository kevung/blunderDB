/**
 * D'une image à son relevé de descripteurs, et d'une suite de relevés aux événements.
 *
 * La coupure est voulue : tout ce qui suit le relevé (`createDetector`) n'est fonction que des
 * relevés, jamais des pixels. Le journal (`journal.js`) garde les relevés ; les rejouer dans
 * un détecteur neuf redonne les mêmes événements sans la vidéo, et c'est sur ces journaux que
 * les seuils des automates se règlent (ADR-0109 règle 8).
 *
 * Un relevé (`DescriptorRecord`) est un objet JSON compact :
 * `{ t, u, w, r: { [nom]: [moyenne, dPrec, dFond] } }` — `t` l'instant du média (s), `u` 1 si
 * l'image est incertaine (témoin au-delà du seuil), `w` la fraction des pixels du témoin dont
 * la signature census a changé depuis l'image précédente, et pour chaque ROI sa moyenne normalisée, sa distance census à l'image
 * analysée précédente et au fond ; `null` quand la référence manque. Valeurs arrondies à 1e-4.
 * `o`, présent quand la calibration trace un overlay, est son dHash en hexadécimal, pris au plus
 * toutes les `SAMPLE_PERIOD` secondes et jamais sur une image incertaine.
 */

import { censusChangedFraction, censusDistance, censusRoi, DEFAULT_TAU, toGray } from './census.js';
import { createAutomata } from './automata.js';
import { EVENT_TYPES, makeEvent, normalizeCalibration, STATES } from './contract.js';
import { createOverlayDrift, dHash, hashToHex, SAMPLE_PERIOD } from './overlay.js';

/** Fraction des pixels du témoin au-delà de laquelle l'image est incertaine (ADR-0109 règle 4). */
export const WITNESS_THRESHOLD = 0.6;

/**
 * L'image est incertaine quand le témoin change **strictement** au-delà du seuil.
 * @param {number | null} w fraction des pixels du témoin changés, `null` sans image précédente
 * @param {number} [threshold]
 */
export function witnessIsUncertain(w, threshold = WITNESS_THRESHOLD) {
    return w !== null && w > threshold;
}

/** @typedef {{ t: number, u: 0 | 1, w: number | null, r: Record<string, [number, number | null, number | null]>, o?: string }} DescriptorRecord */

/** @param {number | null} v */
const round = (v) => (v === null ? null : Math.round(v * 1e4) / 1e4);

/**
 * @param {Parameters<typeof normalizeCalibration>[0]} calibration
 * @param {{ tau?: number, witnessThreshold?: number }} [options]
 */
export function createDescriber(calibration, { tau = DEFAULT_TAU, witnessThreshold = WITNESS_THRESHOLD } = {}) {
    const cal = normalizeCalibration(calibration);
    /** @type {Map<string, Uint8Array>} */
    const prev = new Map();
    /** @type {Map<string, Uint8Array>} */
    const background = new Map();
    /** @type {Uint8Array | null} */
    let prevWitness = null;
    /** @type {number | null} */
    let lastOverlay = null;

    return {
        calibration: cal,
        /** Oublie les fonds : la prochaine image les reprend. */
        resetBackgrounds() {
            background.clear();
        },
        /**
         * @param {{ mediaTime: number, data: Uint8ClampedArray | Uint8Array, width?: number, height?: number }} frame
         * @returns {DescriptorRecord}
         */
        describe(frame) {
            const width = frame.width ?? cal.width;
            const height = frame.height ?? cal.height;
            if (width !== cal.width || height !== cal.height) throw new RangeError(`vision: frame ${width}×${height}, calibration ${cal.width}×${cal.height}`);
            const gray = toGray(frame.data, width, height);

            const witness = censusRoi(gray, width, cal.witness, tau).bits;
            const w = prevWitness ? censusChangedFraction(prevWitness, witness) : null;
            prevWitness = witness;
            const uncertain = witnessIsUncertain(w, witnessThreshold);
            // A jump of the witness means the whole picture moved (cut, exposure, camera):
            // every background is stale, and this frame becomes the new reference.
            if (uncertain) background.clear();

            /** @type {DescriptorRecord['r']} */
            const r = {};
            for (const roi of cal.rois) {
                const { mean, bits } = censusRoi(gray, width, roi, tau);
                const p = prev.get(roi.name);
                const bg = background.get(roi.name);
                r[roi.name] = [round(mean), p ? round(censusDistance(p, bits)) : null, bg ? round(censusDistance(bg, bits)) : null];
                prev.set(roi.name, bits);
                if (!bg) background.set(roi.name, bits);
            }
            const record = { t: frame.mediaTime, u: uncertain ? 1 : 0, w: round(w), r };
            // The overlay is sampled apart from the dice ROIs: a slower cadence, and never on a
            // frame whose picture jumped (the uncertain frame already says so).
            if (cal.overlay && !uncertain && (lastOverlay === null || frame.mediaTime < lastOverlay || frame.mediaTime - lastOverlay >= SAMPLE_PERIOD)) {
                lastOverlay = frame.mediaTime;
                record.o = hashToHex(dHash(gray, width, cal.overlay));
            }
            return record;
        }
    };
}

/**
 * Les événements tirés des relevés : un épisode d'images incertaines ne donne qu'un
 * `frame.uncertain`, à l'instant de sa première image ; les automates mains / dés / partie et
 * la cadence voulue viennent de `automata.js`, sur les mêmes relevés.
 * @param {Parameters<typeof createAutomata>[0]} [thresholds] pour rejouer sous d'autres seuils
 */
export function createDetector(thresholds) {
    const automata = createAutomata(thresholds);
    const drift = createOverlayDrift();
    let inUncertain = false;
    return {
        /**
         * @param {DescriptorRecord} record
         * @returns {import('./contract.js').VisionEvent[]}
         */
        push(record) {
            /** @type {import('./contract.js').VisionEvent[]} */
            const events = [];
            if (record.u === 1 && !inUncertain) events.push(makeEvent(EVENT_TYPES.FRAME_UNCERTAIN, record.t, STATES.UNCERTAIN, { witness: record.w }));
            inUncertain = record.u === 1;
            const fromAutomata = automata.push(record);
            events.push(...fromAutomata);
            events.push(...drift.push(record.t, record.o, fromAutomata));
            return events;
        },
        /** Cadence d'échantillonnage voulue après le dernier relevé, en images du média par seconde. */
        get rate() {
            return automata.rate;
        }
    };
}

/**
 * Rejoue un journal dans un détecteur neuf et rend tous ses événements, dans l'ordre.
 * @param {Iterable<DescriptorRecord>} records
 * @param {{ push: (record: DescriptorRecord) => import('./contract.js').VisionEvent[] }} [detector]
 */
export function replay(records, detector = createDetector()) {
    /** @type {import('./contract.js').VisionEvent[]} */
    const events = [];
    for (const record of records) events.push(...detector.push(record));
    return events;
}
