/**
 * L'échéance d'une micro-ronde, côté interface (issue #388).
 *
 * Elle tombe SANS QU'AUCUN ÉVÉNEMENT NE SOIT ÉCRIT : à 14 h 20 les joueurs libres deviennent
 * appariables, et rien dans la base ne l'a dit. Le rafraîchissement est donc programmé sur
 * l'échéance que porte la file — d'où ce test, qui tient le calcul du délai.
 */
import { describe, test, expect } from 'vitest';
import { nextBatchDelay } from '../stores/directionStore.js';

const now = Date.UTC(2026, 8, 12, 14, 0, 0);

describe('le délai avant la prochaine micro-ronde', () => {
    test('vaut le temps qui reste jusqu’à l’échéance la plus proche', () => {
        const view = {
            proposals: [
                { kind: 'wait', reason: 'waiting_batch', until: '2026-09-12T14:20:00Z' },
                { kind: 'wait', reason: 'waiting_batch', until: '2026-09-12T14:05:00Z' }
            ]
        };
        expect(nextBatchDelay(view, now)).toBe(5 * 60 * 1000);
    });

    test('est nul quand aucune proposition ne porte d’échéance', () => {
        expect(nextBatchDelay({ proposals: [{ kind: 'start_match' }] }, now)).toBeNull();
        expect(nextBatchDelay({ proposals: [] }, now)).toBeNull();
        expect(nextBatchDelay(null, now)).toBeNull();
    });

    /* Un `time.Time` nul côté Go arrive en l'an 1 : ce n'est pas une échéance, et la prendre
       pour telle programmerait un rafraîchissement immédiat en boucle. */
    test('ignore une date zéro venue du moteur', () => {
        const view = { proposals: [{ kind: 'wait', until: '0001-01-01T00:00:00Z' }] };
        expect(nextBatchDelay(view, now)).toBeNull();
    });

    test('vaut zéro, jamais un négatif, quand l’échéance est passée', () => {
        const view = { proposals: [{ kind: 'wait', until: '2026-09-12T13:50:00Z' }] };
        expect(nextBatchDelay(view, now)).toBe(0);
    });
});
