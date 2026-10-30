/**
 * L'horloge des images : un rappel par image composée, avec son instant du média.
 *
 * `requestVideoFrameCallback` donne `metadata.mediaTime`, l'instant exact de l'image ; mais
 * sous WebKitGTK la fonction existe et ne rappelle jamais. La capacité se détecte donc par un
 * premier rappel reçu, jamais par `typeof`. En attendant ce rappel — pour toujours là où il
 * est muet —, une boucle `requestAnimationFrame` lit `currentTime` et retient une image à
 * chaque changement : à 1× elle désigne l'image composée à une image près et n'en perd aucune ;
 * en lecture accélérée `currentTime` devance d'une image (banc V0.1).
 */

/**
 * @param {object} options
 * @param {HTMLVideoElement} options.video
 * @param {(mediaTime: number) => void} options.onFrame appelé une fois par image retenue
 * @param {(source: 'rvfc' | 'raf') => void} [options.onSource] appelé quand l'horloge utilisée change
 * @param {(cb: FrameRequestCallback) => number} [options.requestFrame]
 * @param {(id: number) => void} [options.cancelFrame]
 */
export function createFrameClock({ video, onFrame, onSource = () => {}, requestFrame, cancelFrame }) {
    const raf = requestFrame ?? ((/** @type {FrameRequestCallback} */ cb) => globalThis.requestAnimationFrame(cb));
    const caf = cancelFrame ?? ((/** @type {number} */ id) => globalThis.cancelAnimationFrame(id));
    const hasRvfc = typeof video.requestVideoFrameCallback === 'function';

    let running = false;
    /** @type {'rvfc' | 'raf' | null} */
    let source = null;
    /** @type {number | null} */
    let rafId = null;
    /** @type {number | null} */
    let rvfcId = null;
    let lastTime = NaN;

    /** @param {'rvfc' | 'raf'} next */
    function setSource(next) {
        if (source === next) return;
        source = next;
        onSource(next);
    }

    /** @param {number} _now @param {VideoFrameCallbackMetadata} metadata */
    function onVideoFrame(_now, metadata) {
        rvfcId = null;
        if (!running) return;
        if (rafId !== null) {
            caf(rafId);
            rafId = null;
        }
        setSource('rvfc');
        lastTime = metadata.mediaTime;
        onFrame(metadata.mediaTime);
        rvfcId = video.requestVideoFrameCallback(onVideoFrame);
    }

    function onAnimationFrame() {
        rafId = null;
        if (!running || source === 'rvfc') return;
        const t = video.currentTime;
        if (t !== lastTime && Number.isFinite(t)) {
            setSource('raf');
            lastTime = t;
            onFrame(t);
        }
        rafId = raf(onAnimationFrame);
    }

    return {
        start() {
            if (running) return;
            running = true;
            lastTime = NaN;
            // The probe stays armed for the whole run: a first callback switches over even late.
            if (hasRvfc && rvfcId === null) rvfcId = video.requestVideoFrameCallback(onVideoFrame);
            if (source !== 'rvfc') rafId = raf(onAnimationFrame);
        },
        stop() {
            running = false;
            if (rafId !== null) caf(rafId);
            rafId = null;
            if (rvfcId !== null && typeof video.cancelVideoFrameCallback === 'function') video.cancelVideoFrameCallback(rvfcId);
            rvfcId = null;
        },
        /** @returns {'rvfc' | 'raf' | null} */
        get source() {
            return source;
        }
    };
}
