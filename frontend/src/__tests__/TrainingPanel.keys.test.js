/**
 * TrainingPanel.keys.test.js — une question ouverte garde ses touches (#323).
 *
 * Le défaut, hérité de #321 : après « Valider », le bouton cliqué disparaît
 * avec la question ouverte, le focus retombe sur la page, et J / K font défiler
 * la liste parcourue SOUS la question — le plateau change, la vérité affichée
 * dans le panneau ne décrit plus rien, et pour une Décision le coup armé au
 * plateau appartient à une autre position.
 *
 * Monté comme dans l'application : le répartiteur global est enregistré sur
 * `window` AVANT le panneau (App.svelte l'installe dans son `onMount`), et les
 * touches partent de l'élément qui a le focus. Un test qui enregistrerait le
 * répartiteur après le composant cacherait le piège `cancelBubble` de Svelte 5
 * (#414).
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

const nextPosition = vi.fn();
const previousPosition = vi.fn();

vi.mock('../services/clipboardService.js', () => ({ copyPosition: vi.fn(), copyBoardImage: vi.fn(), copyBoardWithAnalysisImage: vi.fn() }));
vi.mock('../services/importService.js', () => ({ pastePosition: vi.fn(), importDatabase: vi.fn(), importPosition: vi.fn(), importFolder: vi.fn() }));
vi.mock('../services/exportService.js', () => ({ exportDatabase: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ newDatabase: vi.fn(), openDatabase: vi.fn(), exitApp: vi.fn(), setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({
    deletePosition: vi.fn(),
    saveCurrentPosition: vi.fn(),
    firstPosition: vi.fn(),
    previousPosition,
    nextPosition,
    lastPosition: vi.fn(),
    updatePosition: vi.fn(),
    toggleAnalysisPanel: vi.fn(),
    toggleCommentPanel: vi.fn(),
    toggleMetadataPanel: vi.fn(),
    toggleAnkiPanel: vi.fn(),
    toggleCollectionPanelAction: vi.fn(),
    toggleMatchPanel: vi.fn(),
    toggleTournamentPanel: vi.fn(),
    toggleStatsPanel: vi.fn(),
    toggleSearchPanel: vi.fn(),
    toggleEvalMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn(),
    showDatesAndMetadata: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetTrainingSeedSources: vi.fn(() => Promise.resolve({})),
    SaveTrainingSeedSource: vi.fn(() => Promise.resolve())
}));
// Le service est simulé, mais « Valider » fait ce que fait le vrai : il révèle
// la question dans le magasin. C'est ce changement d'état qui retire le bouton.
vi.mock('../services/trainingTabService.js', async () => {
    const { get } = await import('svelte/store');
    const { trainingSessionStore } = await import('../stores/trainingTabStore.js');
    const { reveal } = await import('../services/trainingTab.js');
    return {
        startTrainingSession: vi.fn(),
        revealQuestion: vi.fn(() => {
            const session = get(trainingSessionStore);
            if (session) trainingSessionStore.set(reveal(session, 1000));
        }),
        markFault: vi.fn(),
        setTrainingAnswer: vi.fn(),
        nextTrainingQuestion: vi.fn(),
        retryTrainingQuestion: vi.fn(),
        finishTrainingSession: vi.fn(),
        quitTrainingSession: vi.fn(),
        refreshTrainingJournal: vi.fn(() => Promise.resolve()),
        answerDecisionBoard: vi.fn(),
        answerDecisionCube: vi.fn(),
        undoDecisionStep: vi.fn(),
        resetDecisionPlay: vi.fn(),
        refusalMessageKey: (/** @type {string} */ code) => (code && code !== 'noQuestion' ? `training.refusal.${code}` : 'training.noQuestion')
    };
});

const { handleKeyDown } = await import('../services/keyboardService.js');
const { default: TrainingPanel } = await import('../components/TrainingPanel.svelte');
const { trainingSessionStore } = await import('../stores/trainingTabStore.js');
const { activeTabStore, activeModal } = await import('../stores/uiStore.js');
const { newSession, askQuestion, setAnswer } = await import('../services/trainingTab.js');

/** Une question de Bearoff, champ du bas rempli. */
function bearoffSession() {
    const question = {
        kind: 'bearoff',
        key: 'pool:1',
        numbers: [
            { type: 'epc.bottom', value: 20, tolerance: 0.5, precision: 1 },
            { type: 'epc.top', value: 30, tolerance: 0.5, precision: 1 }
        ]
    };
    return setAnswer(askQuestion(newSession({ exercise: 'bearoff', seedSource: 'pool' }), question, 0), 0, '20');
}

/** Presse une touche là où est le focus, comme le clavier. @param {string} key @param {string} code */
function press(key, code) {
    const target = document.activeElement ?? document.body;
    target.dispatchEvent(new KeyboardEvent('keydown', { key, code, bubbles: true, cancelable: true }));
}

beforeEach(() => {
    vi.clearAllMocks();
    activeModal.set(null);
    activeTabStore.set('training');
    // AVANT le composant, comme dans l'application.
    window.addEventListener('keydown', handleKeyDown);
});

afterEach(() => {
    cleanup();
    window.removeEventListener('keydown', handleKeyDown);
    trainingSessionStore.set(null);
});

describe('une question dont le plateau est la surface garde ses touches', () => {
    test('après « Valider », J et K ne parcourent pas la liste sous la question', async () => {
        trainingSessionStore.set(bearoffSession());
        const panel = render(TrainingPanel);
        const validate = panel.getByTestId('training-reveal');
        validate.focus();
        validate.click();
        await tick();

        press('j', 'KeyJ');
        press('k', 'KeyK');
        press('ArrowRight', 'ArrowRight');
        expect(nextPosition).not.toHaveBeenCalled();
        expect(previousPosition).not.toHaveBeenCalled();
    });

    test('même quand le focus est sur la page, tant que la question est à l’écran', async () => {
        trainingSessionStore.set(bearoffSession());
        render(TrainingPanel);
        /** @type {HTMLElement} */ (document.activeElement)?.blur();
        press('j', 'KeyJ');
        expect(nextPosition).not.toHaveBeenCalled();
    });

    test('hors session, J parcourt la liste comme toujours', () => {
        render(TrainingPanel);
        press('j', 'KeyJ');
        expect(nextPosition).toHaveBeenCalledTimes(1);
    });

    test('une fiche de score ne tient pas le plateau : J le parcourt', () => {
        const question = { kind: 'scores', key: '3:5', numbers: [{ type: 'gv1', value: 0.5 }] };
        trainingSessionStore.set(askQuestion(newSession({ exercise: 'scores', seedSource: 'pool' }), question, 0));
        // L'onglet est ailleurs : la fiche ne se monte pas, seule la session compte.
        activeTabStore.set('analysis');
        press('j', 'KeyJ');
        expect(nextPosition).toHaveBeenCalledTimes(1);
    });
});
