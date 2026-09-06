/**
 * trainingTabService.test.js — la session de l'onglet Entraînement de bout en
 * bout, avec Wails simulé.
 *
 * Ce que ce fichier tient, et que trainingTab.test.js ne peut pas tenir : que
 * « Terminer » ÉCRIT une ligne de journal et ses nombres, que « Quitter »
 * n'écrit rien, et que le pipcount du plateau est masqué tant qu'une question
 * de Pions est ouverte. Trois critères d'acceptation, trois assertions sur ce
 * qui part vers la base ou vers le plateau — pas sur un rendu.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadPosition: vi.fn(() => Promise.resolve(null)),
    SaveTrainingSession: vi.fn(() => Promise.resolve(1)),
    LoadTrainingSessions: vi.fn(() => Promise.resolve([])),
    LoadTrainingNumberStats: vi.fn(() => Promise.resolve([]))
}));
vi.mock('../services/importService.js', () => ({ showImportedPosition: vi.fn(() => Promise.resolve()) }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), log: vi.fn() } }));

import * as db from '../../wailsjs/go/database/Database.js';
import { positionStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { trainingSessionStore, trainingPipMaskStore } from '../stores/trainingTabStore.js';
import { startTrainingSession, revealQuestion, markFault, finishTrainingSession, quitTrainingSession } from '../services/trainingTabService.js';

/** Une position où le bas a deux pions sur le point 6 et le haut deux sur le 20. */
function board() {
    return {
        player_on_roll: 0,
        board: {
            points: Array.from({ length: 26 }, (_, i) => {
                if (i === 6) return { checkers: 2, color: 0 };
                if (i === 20) return { checkers: 2, color: 1 };
                return { checkers: 0, color: -1 };
            }),
            bearoff: [0, 0]
        }
    };
}

beforeEach(() => {
    vi.clearAllMocks();
    quitTrainingSession();
    databasePathStore.set('/tmp/some.db');
    positionStore.set(board());
});

describe('une session de Scores', () => {
    test('« Terminer » écrit une ligne de journal et ses nombres', async () => {
        expect(await startTrainingSession({ exercise: 'scores', limitSeconds: 0 })).toBe(true);
        revealQuestion();
        markFault(0);
        await finishTrainingSession();

        expect(db.SaveTrainingSession).toHaveBeenCalledTimes(1);
        const row = db.SaveTrainingSession.mock.calls[0][0];
        expect(row.exercise).toBe('scores');
        expect(row.seedSource).toBe('pool');
        // Trois nombres au moins (2a-2a), quatorze au plus.
        expect(row.numbersAsked).toBeGreaterThanOrEqual(3);
        expect(row.numbersAsked).toBeLessThanOrEqual(14);
        expect(row.items).toHaveLength(row.numbersAsked);
        expect(row.faults).toBe(1);
        expect(row.items.filter((i) => i.wrong)).toHaveLength(1);
        // Aucun écart en mode déclaré : la moyenne des écarts reste vide.
        expect(row.deviations).toBe(0);
        expect(get(trainingSessionStore)).toBeNull();
    });

    test('« Quitter » n’écrit rien', async () => {
        await startTrainingSession({ exercise: 'scores' });
        revealQuestion();
        quitTrainingSession();
        expect(db.SaveTrainingSession).not.toHaveBeenCalled();
        expect(get(trainingSessionStore)).toBeNull();
    });
});

describe('une session de Pions sur le plateau', () => {
    test('demande les deux comptes, et masque le pipcount tant que la question est ouverte', async () => {
        expect(get(trainingPipMaskStore)).toBe(false);
        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'board' })).toBe(true);

        expect(get(trainingSessionStore).question.numbers.map((n) => n.type)).toEqual(['pips.bottom', 'pips.top']);
        expect(get(trainingPipMaskStore), 'le plateau porte la réponse').toBe(true);

        revealQuestion();
        expect(get(trainingPipMaskStore), '« Révéler » l’affiche').toBe(false);
    });

    test('le compte demandé est celui que le plateau affiche', async () => {
        await startTrainingSession({ exercise: 'pips', seedSource: 'board' });
        const [bottom, top] = get(trainingSessionStore).question.numbers;
        expect(bottom.value).toBe(12); // deux pions sur le point 6
        expect(top.value).toBe(10); // deux pions sur le point 20, soit 25 − 20
    });

    test('sans position sur le plateau, la session refuse plutôt que de s’ouvrir vide', async () => {
        positionStore.set({});
        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'board' })).toBe(false);
        expect(get(trainingSessionStore)).toBeNull();
    });
});
