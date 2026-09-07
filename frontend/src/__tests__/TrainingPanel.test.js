/**
 * Le panneau Entraînement : ce qu'il montre, et surtout ce qu'il ne doit pas
 * escamoter.
 *
 * Le défaut que ce fichier tient : quand la question suivante ne peut pas être
 * bâtie (la position tirée a été supprimée entre-temps), le panneau
 * rebasculait sur le lanceur. « Terminer » disparaissait, les nombres déjà
 * répondus devenaient inatteignables, et « Démarrer » les écrasait — tout un
 * journal de session partait sans un mot. L'oracle est donc la présence des
 * boutons de la session, pas la valeur d'un store.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

vi.mock('../services/trainingTabService.js', () => ({
    startTrainingSession: vi.fn(),
    revealQuestion: vi.fn(),
    markFault: vi.fn(),
    nextTrainingQuestion: vi.fn(),
    retryTrainingQuestion: vi.fn(),
    finishTrainingSession: vi.fn(),
    quitTrainingSession: vi.fn(),
    refreshTrainingJournal: vi.fn(() => Promise.resolve())
}));

import TrainingPanel from '../components/TrainingPanel.svelte';
import { trainingSessionStore, trainingJournalStore } from '../stores/trainingTabStore.js';
import { newSession, askQuestion, reveal, recordQuestion, failNextQuestion } from '../services/trainingTab.js';

function pipsQuestion() {
    return {
        kind: 'pips',
        key: '7',
        positionId: 7,
        numbers: [
            { type: 'pips.bottom', value: 12 },
            { type: 'pips.top', value: 10 }
        ]
    };
}

/** Une session dont une question a été répondue, et dont la suivante a échoué. */
function sessionWithAFailedNextQuestion() {
    let s = askQuestion(newSession({ exercise: 'pips', seedSource: 'library' }), pipsQuestion(), 0);
    s = recordQuestion(reveal(s, 2000));
    return failNextQuestion(s, 'noQuestion');
}

beforeEach(() => {
    trainingSessionStore.set(null);
    trainingJournalStore.set({});
});

afterEach(() => {
    cleanup();
    trainingSessionStore.set(null);
});

describe('au repos', () => {
    test('montre le lanceur et le bilan', () => {
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-start"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-summary-scores"]')).not.toBeNull();
    });
});

describe('pendant une question', () => {
    test('montre « Révéler », « Terminer » et « Quitter », et pas le lanceur', () => {
        trainingSessionStore.set(askQuestion(newSession({ exercise: 'pips', seedSource: 'board' }), pipsQuestion(), 0));
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-reveal"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-finish"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-quit"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-start"]')).toBeNull();
    });

    test('la consigne de cochage est à l’écran une fois révélée, pas seulement dans une infobulle', () => {
        const asked = askQuestion(newSession({ exercise: 'pips', seedSource: 'board' }), pipsQuestion(), 0);
        trainingSessionStore.set(reveal(asked, 1000));
        const { container } = render(TrainingPanel);
        expect(container.querySelector('.hint')).not.toBeNull();
    });
});

describe('quand la question suivante a échoué', () => {
    test('la session reste à l’écran : l’échec se dit, « Terminer » et « Réessayer » sont là', () => {
        trainingSessionStore.set(sessionWithAFailedNextQuestion());
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-question-failed"]'), "l'échec doit se dire").not.toBeNull();
        expect(container.querySelector('[data-testid="training-finish"]'), 'les nombres répondus doivent rester enregistrables').not.toBeNull();
        expect(container.querySelector('[data-testid="training-retry"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-start"]'), 'le lanceur écraserait la session').toBeNull();
    });

    test('le chronomètre ne compte plus rien et disparaît', () => {
        trainingSessionStore.set(sessionWithAFailedNextQuestion());
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-clock"]')).toBeNull();
    });
});
