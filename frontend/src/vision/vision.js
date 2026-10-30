/**
 * La façade de Z-vision sur le fil principal : la seule chose que le panneau importe.
 *
 * Elle branche l'horloge des images (`clock.js`), l'échantillonneur (`sampler.js`) et le
 * Worker (`worker.js`), et expose le contrat de `contract.js` : `subscribe` pour les
 * événements, `calibrate` / `start` / `stop` / `clearJournal` pour les commandes. Une seule
 * image est en vol vers le Worker : si l'analyse prend du retard, les images suivantes sont
 * sautées plutôt qu'empilées.
 *
 * Elle ne lance ni n'attend jamais `video.play()` (la promesse ne se résout pas sous WebKitGTK
 * quand `playbackRate ≠ 1`) et n'appelle jamais WebCodecs (`VideoDecoder.isConfigSupported`
 * gèle la page) : elle lit ce que la lecture compose.
 */

import { createFrameClock } from './clock.js';
import { EVENT_TYPES, makeEvent, normalizeCalibration, STATES } from './contract.js';
import { createFrameGrabber, createRateGate, TaintedFrameError } from './sampler.js';
import { STATE_RATE } from './workerCore.js';

/** @returns {Worker} */
function defaultWorker() {
    return new Worker(new URL('./worker.js', import.meta.url), { type: 'module' });
}

/**
 * @param {object} options
 * @param {HTMLVideoElement} options.video le `<video>` du volet, `crossorigin="anonymous"` posé avant `src`
 * @param {() => Pick<Worker, 'postMessage' | 'terminate'> & { onmessage: any, onerror: any }} [options.createWorker]
 * @param {(video: HTMLVideoElement, width: number, height: number) => { grab(): ImageData }} [options.createGrabber]
 * @param {(cb: FrameRequestCallback) => number} [options.requestFrame]
 * @param {(id: number) => void} [options.cancelFrame]
 */
export function createVision({ video, createWorker = defaultWorker, createGrabber, requestFrame, cancelFrame }) {
    const makeGrabber = createGrabber ?? ((v, width, height) => createFrameGrabber({ video: v, width, height }));
    /** @type {Set<(event: import('./contract.js').VisionEvent) => void>} */
    const listeners = new Set();
    /** @type {Map<number, { resolve: () => void, reject: (err: Error) => void }>} */
    const pendingClears = new Map();
    let clearId = 0;

    const gate = createRateGate(STATE_RATE);
    /** @type {{ grab(): ImageData } | null} */
    let grabber = null;
    let size = { width: 0, height: 0 };
    let running = false;
    let inFlight = false;
    let halted = false;

    const now = () => (Number.isFinite(video.currentTime) && video.currentTime > 0 ? video.currentTime : 0);

    /** @param {import('./contract.js').VisionEvent} event */
    function emit(event) {
        for (const listener of listeners) {
            try {
                listener(event);
            } catch (err) {
                // A failing listener must not starve the others: report it out of band.
                queueMicrotask(() => {
                    throw err;
                });
            }
        }
    }

    /** @param {string} reason */
    function halt(reason) {
        if (halted) return;
        halted = true;
        // A halt ends the run as stop() would: the worker flushes its journal, and a later
        // start() — after the source is reloaded — begins a fresh run.
        running = false;
        clock.stop();
        worker.postMessage({ kind: 'stop' });
        emit(makeEvent(EVENT_TYPES.ERROR, now(), STATES.UNCERTAIN, { reason }));
    }

    const worker = createWorker();
    worker.onmessage = (/** @type {MessageEvent} */ e) => {
        const msg = e.data;
        switch (msg?.kind) {
            case 'frameDone':
                inFlight = false;
                return;
            case 'rate':
                gate.setRate(msg.fps);
                return;
            case 'event': {
                const ev = msg.event;
                emit(makeEvent(ev.type, ev.mediaTime, ev.state, ev.payload));
                return;
            }
            case 'journalCleared': {
                const p = pendingClears.get(msg.id);
                pendingClears.delete(msg.id);
                if (p) msg.error ? p.reject(new Error(msg.error)) : p.resolve();
                return;
            }
            case 'error': {
                const err = new Error(`vision: worker ${msg.message}`);
                queueMicrotask(() => {
                    throw err;
                });
                return;
            }
        }
    };
    worker.onerror = () => halt('worker');

    /** @param {number} mediaTime */
    function onFrame(mediaTime) {
        if (!running || halted || inFlight || !grabber) return;
        if (!gate.accept(mediaTime)) return;
        /** @type {ImageData} */
        let frame;
        try {
            frame = grabber.grab();
        } catch (err) {
            if (err instanceof TaintedFrameError) halt('tainted');
            else throw err;
            return;
        }
        inFlight = true;
        const buffer = frame.data.buffer;
        worker.postMessage({ kind: 'frame', mediaTime, width: frame.width, height: frame.height, buffer }, [buffer]);
    }

    const clock = createFrameClock({
        video,
        onFrame,
        onSource: (source) => emit(makeEvent(EVENT_TYPES.CLOCK, now(), STATES.SURE, { source })),
        requestFrame,
        cancelFrame
    });

    return {
        /** @param {(event: import('./contract.js').VisionEvent) => void} listener */
        subscribe(listener) {
            listeners.add(listener);
            return () => void listeners.delete(listener);
        },
        /** @param {Parameters<typeof normalizeCalibration>[0]} calibration */
        calibrate(calibration) {
            const cal = normalizeCalibration(calibration);
            if (!grabber || cal.width !== size.width || cal.height !== size.height) {
                grabber = makeGrabber(video, cal.width, cal.height);
                size = { width: cal.width, height: cal.height };
            }
            worker.postMessage({ kind: 'calibrate', calibration: cal });
        },
        /** @param {{ session: string }} options */
        start({ session }) {
            if (running) return;
            if (typeof session !== 'string' || session === '') throw new TypeError('vision: start needs a session');
            running = true;
            halted = false;
            inFlight = false;
            gate.reset();
            worker.postMessage({ kind: 'start', session });
            clock.start();
        },
        stop() {
            if (!running) return;
            running = false;
            clock.stop();
            worker.postMessage({ kind: 'stop' });
        },
        /**
         * @param {string} [session]
         * @returns {Promise<void>}
         */
        clearJournal(session) {
            const id = ++clearId;
            return new Promise((resolve, reject) => {
                pendingClears.set(id, { resolve, reject });
                worker.postMessage({ kind: 'clearJournal', session, id });
            });
        },
        dispose() {
            this.stop();
            listeners.clear();
            for (const p of pendingClears.values()) p.reject(new Error('vision: disposed'));
            pendingClears.clear();
            worker.terminate();
        }
    };
}
