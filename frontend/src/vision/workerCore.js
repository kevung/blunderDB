/**
 * Le cœur du Worker, sans `self` : reçoit les messages du fil principal, décrit chaque image,
 * consigne le relevé au journal et renvoie les événements. `worker.js` ne fait que le brancher
 * sur `self`, ce qui le rend testable sans Worker.
 *
 * Protocole interne à Z-vision (le panneau ne le voit pas, il voit `contract.js`) :
 *
 * fil principal → Worker
 * - `{ kind: 'calibrate', calibration }`
 * - `{ kind: 'start', session }`
 * - `{ kind: 'frame', mediaTime, width, height, buffer }` — `buffer` transféré, RGBA
 * - `{ kind: 'stop' }`
 * - `{ kind: 'clearJournal', session?, id }`
 *
 * Worker → fil principal
 * - `{ kind: 'rate', fps }` — la cadence d'échantillonnage voulue, en images du média par seconde
 * - `{ kind: 'frameDone' }` — l'image est traitée, la suivante peut partir
 * - `{ kind: 'event', event }` — un événement du contrat
 * - `{ kind: 'journalCleared', id }`
 * - `{ kind: 'error', message }`
 */

import { createDescriber, createDetector } from './descriptors.js';
import { clearJournal as clearStoredJournal, openJournal as openStoredJournal } from './journal.js';

/** Cadence de la comparaison d'états hors fenêtre (ADR-0109 règle 1 : 1–2 img/s). */
export const STATE_RATE = 2;

/**
 * @param {object} options
 * @param {(message: object) => void} options.post
 * @param {typeof openStoredJournal} [options.openJournal]
 * @param {typeof clearStoredJournal} [options.clearJournal]
 */
export function createWorkerCore({ post, openJournal = openStoredJournal, clearJournal = clearStoredJournal }) {
    /** @type {ReturnType<typeof createDescriber> | null} */
    let describer = null;
    /** @type {ReturnType<typeof createDetector>} */
    let detector = createDetector();
    /** @type {Promise<Awaited<ReturnType<typeof openStoredJournal>>> | null} */
    let journal = null;
    let running = false;

    /** @param {unknown} err */
    const fail = (err) => post({ kind: 'error', message: err instanceof Error ? err.message : String(err) });

    async function closeJournal() {
        const j = journal;
        journal = null;
        if (j) await (await j).close();
    }

    /** @param {any} msg */
    async function handle(msg) {
        switch (msg?.kind) {
            case 'calibrate':
                describer = createDescriber(msg.calibration);
                detector = createDetector();
                return;
            case 'start':
                await closeJournal();
                journal = openJournal({ session: msg.session });
                journal.catch(fail);
                detector = createDetector();
                running = true;
                post({ kind: 'rate', fps: STATE_RATE });
                return;
            case 'frame': {
                try {
                    if (!running || !describer) return;
                    const record = describer.describe({ mediaTime: msg.mediaTime, width: msg.width, height: msg.height, data: new Uint8ClampedArray(msg.buffer) });
                    if (journal)
                        void journal.then(
                            (j) => j.append(record),
                            () => {}
                        );
                    for (const event of detector.push(record)) post({ kind: 'event', event });
                } finally {
                    post({ kind: 'frameDone' });
                }
                return;
            }
            case 'stop':
                running = false;
                await closeJournal();
                return;
            case 'clearJournal': {
                try {
                    // The open journal drops its pending batch first, or its next flush would
                    // write back what was just erased.
                    const j = journal ? await journal : null;
                    if (j && (msg.session === undefined || msg.session === j.session)) await j.clear();
                    await clearJournal({ session: msg.session });
                    post({ kind: 'journalCleared', id: msg.id });
                } catch (err) {
                    post({ kind: 'journalCleared', id: msg.id, error: err instanceof Error ? err.message : String(err) });
                }
                return;
            }
            default:
                throw new Error(`vision: unknown worker message ${msg?.kind}`);
        }
    }

    return {
        /** @param {any} msg */
        handle(msg) {
            return handle(msg).catch(fail);
        }
    };
}
