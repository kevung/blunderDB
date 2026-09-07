/**
 * TrainingBar.test.js — la bande d'entraînement, MONTÉE.
 *
 * Le défaut que ce fichier tient : #321 a retiré `answerCurrent` du service
 * en laissant son import et son appel dans la bande. Un import nommé d'un
 * export qui n'existe pas est, sous l'ESM natif que sert Vite, une erreur de
 * LIAISON : le module entier ne charge pas, la bande ne rend rien, et
 * ``train quiz`` ouvre une session invisible. Mille neuf cents tests étaient
 * verts, parce qu'aucun ne montait ce composant.
 *
 * L'oracle est donc le montage lui-même, avant tout ce qu'il affiche. Tant
 * que la bande existe — elle disparaît avec #323, qui emmène le quiz dans
 * l'onglet — elle est montée ici.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

vi.mock('../services/trainingSessionService.js', () => ({
    answerQuiz: vi.fn(),
    answerQuizBoard: vi.fn(),
    nextQuestion: vi.fn(),
    stopTraining: vi.fn()
}));

import TrainingBar from '../components/TrainingBar.svelte';
import * as session from '../services/trainingSessionService.js';
import { trainingActiveStore, trainingIndexStore, trainingQuestionsStore, trainingVerdictStore } from '../stores/trainingStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import en from '../i18n/locales/en.json';

// Les tests tournent sur la locale par défaut : on lit les libellés dans le
// catalogue plutôt que de les recopier, sinon une traduction retouchée
// casserait un test qui ne parle pas d'elle.
const label = en.training;

// `trainingCurrentStore` est dérivé : la question courante se pose en
// remplissant la liste et l'index, jamais en écrivant le dérivé.
/** @param {string} prompt 'checker' ou 'cube' */
function openSession(prompt = 'checker') {
    trainingQuestionsStore.set([{ drill: 'quiz', positionId: 7, truth: 0, prompt }]);
    trainingIndexStore.set(0);
    trainingVerdictStore.set(null);
    trainingActiveStore.set(true);
}

beforeEach(() => {
    vi.clearAllMocks();
    quizPlayStore.set(null);
    trainingActiveStore.set(false);
    trainingVerdictStore.set(null);
    trainingQuestionsStore.set([]);
});

afterEach(cleanup);

describe('la bande sert le quiz', () => {
    // Le montage EST l'assertion : un import mort la ferait échouer ici, avant
    // toute question de rendu.
    test('une session de quiz affiche la bande', () => {
        openSession();
        const bar = render(TrainingBar);
        expect(bar.getByTestId('training-bar')).toBeTruthy();
    });

    test('au repos, elle ne rend rien', () => {
        const bar = render(TrainingBar);
        expect(bar.queryByTestId('training-bar')).toBeNull();
    });

    // La saisie au clavier est le chemin que #321 a failli couper en
    // supprimant le juge qu'elle appelait.
    test('la notation tapée part au juge du quiz, et à lui seul', async () => {
        openSession();
        const bar = render(TrainingBar);
        const field = bar.getByLabelText(label.answer);
        field.value = '13/7 8/7';
        field.dispatchEvent(new Event('input', { bubbles: true }));
        bar.getByText(label.check).click();
        expect(session.answerQuiz).toHaveBeenCalledWith('13/7 8/7');
    });

    test('une décision de videau se clique, et ne montre pas de champ', () => {
        openSession('cube');
        const bar = render(TrainingBar);
        expect(bar.queryByLabelText(label.answer)).toBeNull();
        bar.getByText(label.doublePass).click();
        expect(session.answerQuiz).toHaveBeenCalledWith('dp');
    });

    test('« Quitter » arrête la session', () => {
        openSession();
        const bar = render(TrainingBar);
        bar.getByText(label.leave).click();
        expect(session.stopTraining).toHaveBeenCalled();
    });
});
