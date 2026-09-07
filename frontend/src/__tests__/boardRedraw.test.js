/**
 * Écrire un store ne peint pas.
 *
 * `drawBoard()` lit ses stores impérativement dans une frame d'animation :
 * une valeur qui change sans que personne ait planifié un repaint ne change
 * rien à l'écran. C'est exactement ce qui est arrivé au masque du pipcount de
 * l'exercice Pions (#320) — il se calculait, et le plateau continuait
 * d'afficher la réponse. La preuve que le piège est réel était déjà dans le
 * dépôt : `togglePipcount` pousse `positionStore` EXPRÈS pour forcer le
 * repaint.
 *
 * L'oracle ici est donc un COMPTE de repaints demandés, pas une valeur de
 * store : il bouge quand le défaut est là.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

import { BOARD_REDRAW_TRIGGERS, subscribeBoardRedrawTriggers } from '../services/boardRedraw.js';
import { showPipcountStore, pipcountVisibleStore } from '../stores/uiStore.js';
import { trainingSessionStore } from '../stores/trainingTabStore.js';
import { newSession, askQuestion, reveal } from '../services/trainingTab.js';

/** Une question de Pions déjà bâtie, sans plateau ni Wails. */
function pipsQuestion() {
    return {
        kind: 'pips',
        key: 'board',
        numbers: [
            { type: 'pips.bottom', value: 12 },
            { type: 'pips.top', value: 10 }
        ]
    };
}

beforeEach(() => {
    trainingSessionStore.set(null);
    showPipcountStore.set(true);
});

describe('les déclencheurs de repaint', () => {
    test('la visibilité du pipcount en fait partie', () => {
        expect(BOARD_REDRAW_TRIGGERS.map((t) => t.name)).toContain('pipcountVisible');
    });

    test('chaque déclencheur demande un repaint, et le désabonnement les arrête tous', () => {
        const schedule = vi.fn();
        const unsubscribe = subscribeBoardRedrawTriggers(schedule);
        // Un abonnement rend sa valeur courante : autant d'appels que de stores.
        expect(schedule).toHaveBeenCalledTimes(BOARD_REDRAW_TRIGGERS.length);
        schedule.mockClear();

        showPipcountStore.set(false);
        expect(schedule).toHaveBeenCalled();

        unsubscribe();
        schedule.mockClear();
        showPipcountStore.set(true);
        expect(schedule).not.toHaveBeenCalled();
    });
});

describe('le masque du pipcount pendant une question de Pions', () => {
    test('poser la question demande un repaint, et « Révéler » un autre', () => {
        const schedule = vi.fn();
        const unsubscribe = subscribeBoardRedrawTriggers(schedule);
        schedule.mockClear();

        trainingSessionStore.set(askQuestion(newSession({ exercise: 'pips', seedSource: 'board' }), pipsQuestion(), 0));
        expect(get(pipcountVisibleStore), 'le plateau porte la réponse').toBe(false);
        expect(schedule, 'le masque doit REPEINDRE, pas seulement se calculer').toHaveBeenCalled();

        schedule.mockClear();
        trainingSessionStore.set(reveal(get(trainingSessionStore), 3000));
        expect(get(pipcountVisibleStore)).toBe(true);
        expect(schedule, '« Révéler » doit repeindre').toHaveBeenCalled();

        unsubscribe();
    });

    test('révéler affiche le pipcount même à qui l’a masqué, et ne touche pas à sa préférence', () => {
        showPipcountStore.set(false);
        trainingSessionStore.set(askQuestion(newSession({ exercise: 'pips', seedSource: 'board' }), pipsQuestion(), 0));
        expect(get(pipcountVisibleStore)).toBe(false);

        trainingSessionStore.set(reveal(get(trainingSessionStore), 1000));
        expect(get(pipcountVisibleStore), 'une vérité qui ne s’affiche pas ne se vérifie pas').toBe(true);
        expect(get(showPipcountStore), 'un masque, pas un réglage').toBe(false);

        // Fin de session : la préférence reprend la main, telle qu'elle était.
        trainingSessionStore.set(null);
        expect(get(pipcountVisibleStore)).toBe(false);
    });

    test('hors question de Pions, la préférence décide seule', () => {
        showPipcountStore.set(true);
        trainingSessionStore.set(askQuestion(newSession({ exercise: 'scores', seedSource: 'pool' }), { kind: 'scores', key: '3:5', numbers: [] }, 0));
        expect(get(pipcountVisibleStore)).toBe(true);
        showPipcountStore.set(false);
        expect(get(pipcountVisibleStore)).toBe(false);
    });
});
