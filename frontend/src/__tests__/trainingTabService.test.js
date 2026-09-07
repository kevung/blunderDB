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
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { trainingSessionStore } from '../stores/trainingTabStore.js';
import { pipcountVisibleStore } from '../stores/uiStore.js';
import { subscribeBoardRedrawTriggers } from '../services/boardRedraw.js';
import { startTrainingSession, revealQuestion, markFault, nextTrainingQuestion, retryTrainingQuestion, finishTrainingSession, quitTrainingSession } from '../services/trainingTabService.js';

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
    positionsStore.setIds([]);
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
    // L'oracle est le COMPTE de repaints demandés, et pas seulement la valeur
    // du store : la première version de ce test n'assérait que la valeur, elle
    // était verte alors que le plateau continuait d'afficher la réponse
    // pendant toute la question (le masque se calculait sans jamais repeindre).
    test('demande les deux comptes, masque le pipcount, et REPEINT le plateau', async () => {
        const schedule = vi.fn();
        const unsubscribe = subscribeBoardRedrawTriggers(schedule);
        expect(get(pipcountVisibleStore)).toBe(true);
        schedule.mockClear();

        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'board' })).toBe(true);
        expect(get(trainingSessionStore).question.numbers.map((n) => n.type)).toEqual(['pips.bottom', 'pips.top']);
        expect(get(pipcountVisibleStore), 'le plateau porte la réponse').toBe(false);
        expect(schedule, 'masquer sans repeindre ne masque rien').toHaveBeenCalled();

        schedule.mockClear();
        revealQuestion();
        expect(get(pipcountVisibleStore), '« Révéler » l’affiche').toBe(true);
        expect(schedule, 'révéler sans repeindre n’affiche rien').toHaveBeenCalled();

        unsubscribe();
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

describe('quand la question suivante ne peut pas être posée', () => {
    // La position tirée a disparu entre-temps. La session rebasculait sur le
    // lanceur : « Terminer » disparaissait, et le journal de la session partait
    // sans un mot.
    test('la session reste ouverte, le dit, et « Terminer » enregistre ce qui a été répondu', async () => {
        positionsStore.setIds([7]);
        db.LoadPosition.mockResolvedValueOnce(board());
        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'library' })).toBe(true);
        revealQuestion();
        markFault(0);

        // La position suivante n'existe plus.
        db.LoadPosition.mockResolvedValueOnce(null);
        await nextTrainingQuestion();

        const session = get(trainingSessionStore);
        expect(session, 'une session ne se perd pas sans que l’utilisateur l’ait décidé').not.toBeNull();
        expect(session.question).toBeNull();
        expect(session.questionError).toBe('noQuestion');
        expect(session.items, 'les nombres déjà répondus sont toujours là').toHaveLength(2);

        await finishTrainingSession();
        expect(db.SaveTrainingSession).toHaveBeenCalledTimes(1);
        const row = db.SaveTrainingSession.mock.calls[0][0];
        expect(row.numbersAsked).toBe(2);
        expect(row.faults).toBe(1);
    });

    test('« Réessayer » repose une question sans rien enregistrer de plus', async () => {
        positionsStore.setIds([7]);
        db.LoadPosition.mockResolvedValueOnce(board());
        await startTrainingSession({ exercise: 'pips', seedSource: 'library' });
        revealQuestion();
        db.LoadPosition.mockResolvedValueOnce(null);
        await nextTrainingQuestion();
        expect(get(trainingSessionStore).question).toBeNull();

        db.LoadPosition.mockResolvedValueOnce(board());
        await retryTrainingQuestion();
        const session = get(trainingSessionStore);
        expect(session.question).not.toBeNull();
        expect(session.questionError).toBe('');
        expect(session.items, 'la question ratée n’a rien ajouté').toHaveLength(2);
    });
});
