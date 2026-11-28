import { EventsOn } from '../../wailsjs/runtime/runtime.js';

// Les opérations longues lancées d'une boîte (export de base, archive papier,
// feuilles imprimées) tournent comme des tâches du panneau Calculs : le
// backend rend l'id de la tâche aussitôt, puis envoie sur son préfixe
// `:progress`, et une seule fois `:done`, `:cancelled` ou `:error`. Le panneau
// montre la barre et le bouton Annuler ; l'appelant attend ici l'issue pour
// l'écrire dans la barre d'état.

/**
 * @typedef {{ status: 'done', result: any } | { status: 'cancelled' } | { status: 'error', message: string }} LongOpOutcome
 */

/**
 * Lance une tâche et attend son issue.
 * @param {string} prefix le préfixe des événements (`export-db`, `paper-archive`, `print-sheet`)
 * @param {() => Promise<number>} start l'appel Start* du backend
 * @param {{ onProgress?: (done: number, total: number) => void }} [hooks]
 * @returns {Promise<LongOpOutcome>}
 */
export function runLongOp(prefix, start, hooks = {}) {
    return new Promise((resolve) => {
        /** @type {number|null} */
        let id = null;
        // Une issue peut arriver avant l'id (tâche refusée, ou très courte) :
        // gardée jusqu'à ce qu'on sache si elle est la nôtre.
        /** @type {Array<{ kind: string, p: any }>} */
        const early = [];
        /** @type {Array<() => void>} */
        const offs = [];
        let settled = false;
        const finish = (/** @type {LongOpOutcome} */ outcome) => {
            if (settled) return;
            settled = true;
            for (const off of offs) off();
            resolve(outcome);
        };
        const handle = (/** @type {string} */ kind, /** @type {any} */ p) => {
            if (kind === 'progress') hooks.onProgress?.(p?.done ?? 0, p?.total ?? 0);
            else if (kind === 'done') finish({ status: 'done', result: p?.result });
            else if (kind === 'cancelled') finish({ status: 'cancelled' });
            else finish({ status: 'error', message: String(p?.message ?? '') });
        };
        for (const kind of ['progress', 'done', 'cancelled', 'error']) {
            offs.push(
                EventsOn(`${prefix}:${kind}`, (/** @type {any} */ p) => {
                    const pid = Number(p?.id ?? 0);
                    if (id === null) early.push({ kind, p });
                    else if (pid === id || (pid === 0 && id === 0)) handle(kind, p);
                })
            );
        }
        Promise.resolve()
            .then(start)
            .then((got) => {
                id = Number(got ?? 0);
                for (const e of early) if (Number(e.p?.id ?? 0) === id) handle(e.kind, e.p);
            })
            .catch((err) => finish({ status: 'error', message: String(err) }));
    });
}
