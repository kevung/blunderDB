/**
 * Descripteurs census d'une ROI (ADR-0109 règle 4).
 *
 * La ROI est normalisée par sa moyenne, puis chaque pixel intérieur est comparé à ses huit
 * voisins : le bit k vaut 1 si le voisin k dépasse le centre de plus de `tau` (en fraction de
 * la moyenne). Le résultat ne dépend que de l'ordre relatif des pixels : une auto-exposition
 * qui multiplie toute la ROI par un gain ne change aucun bit. `tau` écarte le bruit du capteur
 * sur les aplats, où sans lui le moindre grain retourne des bits.
 */

export const DEFAULT_TAU = 0.03;

// Neighbour offsets, clockwise from top-left: bit k of a census byte is neighbour k.
const NEIGHBOURS = [
    [-1, -1],
    [0, -1],
    [1, -1],
    [1, 0],
    [1, 1],
    [0, 1],
    [-1, 1],
    [-1, 0]
];

const POPCOUNT = new Uint8Array(256);
for (let i = 1; i < 256; i++) POPCOUNT[i] = (i & 1) + POPCOUNT[i >> 1];

/**
 * Luminance entière (Rec. 601) d'une image RGBA.
 * @param {Uint8ClampedArray | Uint8Array} rgba
 * @param {number} width
 * @param {number} height
 */
export function toGray(rgba, width, height) {
    const n = width * height;
    if (rgba.length < n * 4) throw new RangeError('vision: frame buffer smaller than width × height × 4');
    const gray = new Uint8Array(n);
    for (let i = 0, j = 0; i < n; i++, j += 4) gray[i] = (77 * rgba[j] + 150 * rgba[j + 1] + 29 * rgba[j + 2]) >> 8;
    return gray;
}

/**
 * @param {Uint8Array} gray luminance de l'image entière
 * @param {number} width largeur de l'image
 * @param {{ x: number, y: number, w: number, h: number }} roi
 * @param {number} [tau]
 * @returns {{ mean: number, bits: Uint8Array }} `mean` en [0 ; 1], `bits` un octet par pixel intérieur
 */
export function censusRoi(gray, width, roi, tau = DEFAULT_TAU) {
    const { x, y, w, h } = roi;
    let sum = 0;
    for (let r = y; r < y + h; r++) for (let c = x; c < x + w; c++) sum += gray[r * width + c];
    const mean = sum / (w * h);
    // A black ROI has no scale to normalise by: every comparison then falls to "not brighter".
    const margin = tau * (mean > 0 ? mean : 1);
    const bits = new Uint8Array((w - 2) * (h - 2));
    let k = 0;
    for (let r = y + 1; r < y + h - 1; r++) {
        for (let c = x + 1; c < x + w - 1; c++) {
            const centre = gray[r * width + c] + margin;
            let byte = 0;
            for (let b = 0; b < 8; b++) {
                const [dx, dy] = NEIGHBOURS[b];
                if (gray[(r + dy) * width + c + dx] > centre) byte |= 1 << b;
            }
            bits[k++] = byte;
        }
    }
    return { mean: mean / 255, bits };
}

/**
 * Fraction des bits census qui diffèrent entre deux relevés de la même ROI, en [0 ; 1].
 * @param {Uint8Array} a
 * @param {Uint8Array} b
 */
export function censusDistance(a, b) {
    if (a.length !== b.length) throw new RangeError('vision: census of different ROIs');
    if (a.length === 0) return 0;
    let diff = 0;
    for (let i = 0; i < a.length; i++) diff += POPCOUNT[a[i] ^ b[i]];
    return diff / (a.length * 8);
}

/**
 * Fraction des pixels dont la signature census a changé (au moins un bit), en [0 ; 1]. C'est
 * la mesure du témoin : deux images sans rapport ne diffèrent que sur la moitié des bits
 * environ, mais sur presque tous les pixels, si bien qu'un seuil absolu (60 %) n'a de sens
 * qu'en pixels.
 * @param {Uint8Array} a
 * @param {Uint8Array} b
 */
export function censusChangedFraction(a, b) {
    if (a.length !== b.length) throw new RangeError('vision: census of different ROIs');
    if (a.length === 0) return 0;
    let changed = 0;
    for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) changed++;
    return changed / a.length;
}
