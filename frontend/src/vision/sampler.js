/**
 * L'échantillonneur : choisit les images à analyser et les réduit à 320×180.
 *
 * Le coût d'une image est la lecture de l'image décodée par `drawImage` (≈ 5 ms en 1080p sous
 * WebKitGTK), pas la taille de la cible ; le canvas est créé avec `willReadFrequently`, qui
 * supprime les pics de `getImageData` sans rien coûter (banc V0.1).
 */

export const FRAME_WIDTH = 320;
export const FRAME_HEIGHT = 180;

/**
 * Retient au plus `fps` images par seconde **du média** : une par créneau de 1/fps s. Compter
 * en instant du média plutôt qu'en horloge murale garde la même densité d'analyse en lecture
 * accélérée. Un recul (seek arrière) rouvre le créneau. `fps <= 0` retient tout.
 * @param {number} fps
 */
export function createRateGate(fps) {
    let rate = fps;
    let lastSlot = NaN;
    let lastTime = NaN;
    return {
        /** @param {number} mediaTime */
        accept(mediaTime) {
            if (!(rate > 0)) {
                lastTime = mediaTime;
                return true;
            }
            // The epsilon absorbs float error on frame times that fall exactly on a slot edge.
            const slot = Math.floor(mediaTime * rate + 1e-6);
            if (slot === lastSlot && !(mediaTime < lastTime)) return false;
            lastSlot = slot;
            lastTime = mediaTime;
            return true;
        },
        /** @param {number} next */
        setRate(next) {
            rate = next;
            lastSlot = NaN;
        },
        get rate() {
            return rate;
        },
        reset() {
            lastSlot = NaN;
            lastTime = NaN;
        }
    };
}

/**
 * @param {number} width
 * @param {number} height
 * @returns {HTMLCanvasElement | OffscreenCanvas}
 */
function defaultCanvas(width, height) {
    if (typeof OffscreenCanvas === 'function') return new OffscreenCanvas(width, height);
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;
    return canvas;
}

/** Le canvas a été teinté : la source n'est plus lisible (CORS manquant, ou après `ended`). */
export class TaintedFrameError extends Error {
    constructor() {
        super('vision: video frame is not readable (tainted canvas)');
        this.name = 'TaintedFrameError';
    }
}

/**
 * @param {object} options
 * @param {CanvasImageSource} options.video
 * @param {number} [options.width]
 * @param {number} [options.height]
 * @param {(width: number, height: number) => HTMLCanvasElement | OffscreenCanvas} [options.createCanvas]
 */
export function createFrameGrabber({ video, width = FRAME_WIDTH, height = FRAME_HEIGHT, createCanvas = defaultCanvas }) {
    const canvas = createCanvas(width, height);
    const ctx = /** @type {CanvasRenderingContext2D | OffscreenCanvasRenderingContext2D | null} */ (canvas.getContext('2d', { willReadFrequently: true }));
    if (!ctx) throw new Error('vision: no 2d context');
    return {
        width,
        height,
        /**
         * Rend les pixels RGBA de l'image courante réduite ; lève `TaintedFrameError` si le
         * canvas est illisible.
         * @returns {ImageData}
         */
        grab() {
            ctx.drawImage(video, 0, 0, width, height);
            try {
                return ctx.getImageData(0, 0, width, height);
            } catch (err) {
                if (err && /** @type {Error} */ (err).name === 'SecurityError') throw new TaintedFrameError();
                throw err;
            }
        }
    };
}
