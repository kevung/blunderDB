/**
 * Contrôle de dérive de l'overlay du stream (ADR-0109 règle 7).
 *
 * Une zone tracée une fois est réduite, par moyenne de surface, à une grille 9×8 de gris ; chaque
 * ligne donne 8 bits (la cellule dépasse-t-elle sa voisine de droite ?), d'où un dHash de 64 bits.
 * Le dHash ne retient que le sens des gradients : un gain d'exposition ne le change pas.
 *
 * Indicateur seulement : rien ici ne lit ni n'écrit la transcription ; le résultat est un
 * événement `overlay.changed`, que l'appelant confronte à ce qu'il attend (fin de partie, passe).
 */

import { makeEvent, EVENT_TYPES, STATES } from './contract.js';
import { toGray } from './census.js';

export const GRID_W = 9;
export const GRID_H = 8;

/** Distance de Hamming au-delà de laquelle l'overlay est dit changé (sur 64 bits). */
export const DRIFT_THRESHOLD = 10;

/** Durée, en secondes de média, pendant laquelle le changement doit tenir avant d'être signalé. */
export const HOLD_SECONDS = 1;

const POPCOUNT = new Uint8Array(256);
for (let i = 1; i < 256; i++) POPCOUNT[i] = (i & 1) + POPCOUNT[i >> 1];

/**
 * dHash 64 bits d'une ROI d'une image en gris.
 * @param {Uint8Array} gray luminance de l'image entière
 * @param {number} width largeur de l'image
 * @param {{ x: number, y: number, w: number, h: number }} roi
 * @returns {Uint8Array} 8 octets, un par ligne de la grille
 */
export function dHash(gray, width, roi) {
    const { x, y, w, h } = roi;
    if (!(w >= GRID_W && h >= GRID_H)) throw new RangeError('vision: overlay ROI smaller than the 9×8 grid');
    if (x < 0 || y < 0 || (y + h) * width > gray.length || x + w > width) throw new RangeError('vision: overlay ROI outside the frame');
    const cells = new Float64Array(GRID_W * GRID_H);
    for (let gy = 0; gy < GRID_H; gy++) {
        const y0 = y + Math.floor((gy * h) / GRID_H);
        const y1 = y + Math.floor(((gy + 1) * h) / GRID_H);
        for (let gx = 0; gx < GRID_W; gx++) {
            const x0 = x + Math.floor((gx * w) / GRID_W);
            const x1 = x + Math.floor(((gx + 1) * w) / GRID_W);
            let sum = 0;
            for (let py = y0; py < y1; py++) for (let px = x0; px < x1; px++) sum += gray[py * width + px];
            cells[gy * GRID_W + gx] = sum / ((y1 - y0) * (x1 - x0));
        }
    }
    const hash = new Uint8Array(GRID_H);
    for (let gy = 0; gy < GRID_H; gy++) {
        for (let gx = 0; gx < GRID_W - 1; gx++) {
            if (cells[gy * GRID_W + gx] > cells[gy * GRID_W + gx + 1]) hash[gy] |= 1 << gx;
        }
    }
    return hash;
}

/** dHash d'une ROI d'une image RGBA. */
export function dHashRgba(rgba, width, height, roi) {
    return dHash(toGray(rgba, width, height), width, roi);
}

/** Distance de Hamming entre deux dHash. */
export function hammingDistance(a, b) {
    if (a.length !== b.length) throw new RangeError('vision: dHash length mismatch');
    let d = 0;
    for (let i = 0; i < a.length; i++) d += POPCOUNT[a[i] ^ b[i]];
    return d;
}

/**
 * Détecteur de dérive. `push` reçoit un dHash par échantillon et rend un événement
 * `overlay.changed` quand l'écart à la référence dépasse le seuil depuis plus de `holdSeconds`.
 * Après le signalement, la nouvelle apparence devient la référence : un seul événement par
 * changement. Un écart qui retombe sous le seuil avant la fin du délai est oublié.
 * @param {{ threshold?: number, holdSeconds?: number }} [options]
 */
export function createOverlayWatch({ threshold = DRIFT_THRESHOLD, holdSeconds = HOLD_SECONDS } = {}) {
    /** @type {Uint8Array | null} */
    let reference = null;
    /** @type {{ since: number, hash: Uint8Array } | null} */
    let pending = null;
    return {
        /**
         * @param {number} mediaTime
         * @param {Uint8Array} hash
         * @param {string} [state] état de la machine, copié dans l'événement
         */
        push(mediaTime, hash, state = STATES.CHECK) {
            if (!reference) {
                reference = hash;
                return null;
            }
            if (hammingDistance(reference, hash) <= threshold) {
                pending = null;
                return null;
            }
            // Le candidat est comparé à son propre point de départ : un overlay qui bouge encore
            // (fondu, animation) repart de zéro, seule une apparence stable est signalée.
            if (!pending || hammingDistance(pending.hash, hash) > threshold) {
                pending = { since: mediaTime, hash };
                return null;
            }
            if (mediaTime - pending.since <= holdSeconds) return null;
            const distance = hammingDistance(reference, hash);
            const since = pending.since;
            reference = hash;
            pending = null;
            // Daté du début du changement : la confrontation tolère le retard de l'overlay.
            return makeEvent(EVENT_TYPES.OVERLAY_CHANGED, since, state, { distance });
        }
    };
}

/** Période minimale, en secondes de média, entre deux échantillons de l'overlay (ADR-0109 règle 7 : 0,5–1 s). */
export const SAMPLE_PERIOD = 0.5;

/** Retard toléré de l'overlay sur l'action qui le fait changer, en secondes de média. */
export const LATE_SECONDS = 4;

/** Avance tolérée : le début d'un changement est daté à un échantillon, l'action à une image voisine. */
export const EARLY_SECONDS = 1;

/**
 * Délai de décision : un événement attendu n'est émis par les automates qu'après l'épisode qu'il
 * clôt (une fin de partie à la remise à zéro du plateau), donc bien après le changement d'overlay
 * qu'il explique. Un changement n'est tranché qu'une fois ce délai écoulé.
 */
export const DECISION_SECONDS = 10;

/** Événements après lesquels l'overlay change normalement : fin et début de partie, passage de la main. */
export const EXPECTED_ACTIONS = Object.freeze([EVENT_TYPES.GAME_END, EVENT_TYPES.GAME_START, EVENT_TYPES.DICE_PICKUP]);

/** @param {Uint8Array} hash */
export function hashToHex(hash) {
    return Array.from(hash, (b) => b.toString(16).padStart(2, '0')).join('');
}

/** @param {string} hex */
export function hexToHash(hex) {
    if (!/^(?:[0-9a-f]{2})+$/.test(hex)) throw new TypeError('vision: invalid dHash hex');
    return Uint8Array.from(hex.match(/../g) ?? [], (b) => parseInt(b, 16));
}

/**
 * Confronte les changements d'overlay aux actions attendues. `push` reçoit, par relevé, l'instant
 * du média, le dHash éventuel (hex) et les événements que les automates viennent d'émettre ; il rend
 * les `overlay.changed` inattendus, datés du début du changement. Un changement proche d'une action
 * attendue se tait ; sans overlay (jamais de dHash), rien n'est émis.
 * @param {{ threshold?: number, holdSeconds?: number }} [options]
 */
export function createOverlayDrift(options) {
    const watch = createOverlayWatch(options);
    /** @type {number[]} */
    let actions = [];
    /** @type {import('./contract.js').VisionEvent[]} */
    let candidates = [];
    return {
        /**
         * @param {number} mediaTime
         * @param {string | undefined} hex
         * @param {import('./contract.js').VisionEvent[]} events
         * @returns {import('./contract.js').VisionEvent[]}
         */
        push(mediaTime, hex, events) {
            for (const e of events) if (EXPECTED_ACTIONS.includes(e.type)) actions.push(e.mediaTime);
            if (hex !== undefined) {
                const changed = watch.push(mediaTime, hexToHash(hex));
                if (changed) candidates.push(changed);
            }
            const out = [];
            const waiting = [];
            for (const c of candidates) {
                if (mediaTime - c.mediaTime < DECISION_SECONDS) waiting.push(c);
                else if (!actions.some((a) => c.mediaTime >= a - EARLY_SECONDS && c.mediaTime <= a + LATE_SECONDS)) out.push(c);
            }
            candidates = waiting;
            // An action older than any candidate can still settle can be forgotten.
            actions = actions.filter((a) => mediaTime - a < DECISION_SECONDS + LATE_SECONDS);
            return out;
        }
    };
}
