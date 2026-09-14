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
    LoadTrainingNumberStats: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    GradeQuizChecker: vi.fn(),
    GradeQuizCheckerMove: vi.fn(),
    GradeQuizCube: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ GenerateBearoffQuestion: vi.fn(), GenerateEvaluationQuestion: vi.fn(), LegalMoves: vi.fn(() => Promise.resolve([])) }));
// Le simulacre fait ce que fait le vrai : il bascule sur l'onglet Analyse. Sans
// cet effet, un appel à `showImportedPosition` depuis une session ne se verrait
// pas — et c'est lui qui cachait l'onglet Entraînement sous la question posée.
vi.mock('../services/importService.js', async () => {
    const { activeTabStore } = await import('../stores/uiStore.js');
    return {
        showImportedPosition: vi.fn(async () => {
            activeTabStore.set('analysis');
        })
    };
});
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), log: vi.fn() } }));

import * as dbModule from '../../wailsjs/go/database/Database.js';
import * as appModule from '../../wailsjs/go/gui/App.js';
import * as importServiceModule from '../services/importService.js';
import * as databaseServiceModule from '../services/databaseService.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { playHop } from '../services/quizPlay.js';
import fr from '../i18n/locales/fr.json';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { trainingSessionStore, trainingRefusalStore, trainingAnalysisHiddenStore } from '../stores/trainingTabStore.js';
import { pipcountVisibleStore, activeTabStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { subscribeBoardRedrawTriggers } from '../services/boardRedraw.js';
import {
    startTrainingSession,
    revealQuestion,
    markFault,
    setTrainingAnswer,
    nextTrainingQuestion,
    retryTrainingQuestion,
    finishTrainingSession,
    quitTrainingSession,
    answerDecisionBoard,
    answerDecisionCube,
    undoDecisionStep,
    resetDecisionPlay
} from '../services/trainingTabService.js';

// Les modules simulés, typés comme tels : `vi.mocked` dit au vérificateur que
// `mockResolvedValue` existe là où le module réel n'a qu'une fonction.
const db = vi.mocked(dbModule);
const app = vi.mocked(appModule);
const importService = vi.mocked(importServiceModule);
const statusBar = vi.mocked(databaseServiceModule);

/** La session en cours — le test l'a démarrée, elle n'est pas nulle.
 *  @returns {any} */
function current() {
    return get(trainingSessionStore);
}

/** Une position où le bas a deux pions sur le point 6 et le haut deux sur le 20.
 *  @returns {any} */
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

/** Les index où la liste parcourue a été pointée, dans l'ordre. -1 (« aucun ») ne
 *  compte pas : c'est le cran qui force le rechargement d'un même index. */
function recordIndexMoves() {
    /** @type {number[]} */
    const moves = [];
    const unsubscribe = currentPositionIndexStore.subscribe((value) => {
        if (value >= 0) moves.push(value);
    });
    return { moves, unsubscribe };
}

/** Laisse tourner les micro-tâches en attente, sans horloge simulée. */
function flush() {
    return new Promise((resolve) => setTimeout(resolve, 0));
}

/** Une question de Bearoff telle que le moteur la rend.
 *  @returns {any} */
function generated(bottom = 87.4, top = 91.2, extra = {}) {
    return {
        generated: true,
        refusal: '',
        source: 'pool',
        plies: 3,
        position: board(),
        epc: { bottom: { epc: { epc: bottom } }, top: { epc: { epc: top } } },
        ...extra
    };
}

beforeEach(() => {
    // `clearAllMocks` n'efface QUE les appels : une valeur `once` qu'un test
    // n'a pas consommée fuirait dans le suivant et le ferait échouer là où il
    // n'y a rien. Le préchargement en consomme une de plus qu'avant, donc la
    // fuite est devenue certaine plutôt que rare.
    db.LoadPosition.mockReset();
    db.LoadPosition.mockResolvedValue(null);
    app.GenerateBearoffQuestion.mockReset();
    app.GenerateEvaluationQuestion.mockReset();
    db.LoadPositionIDsByFilters.mockReset();
    db.LoadPositionIDsByFilters.mockResolvedValue([]);
    vi.clearAllMocks();
    activeTabStore.set('training');
    currentPositionIndexStore.set(-1);
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
        expect(row.items.filter((/** @type {any} */ i) => i.wrong)).toHaveLength(1);
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
        expect(current().question.numbers.map((n) => n.type)).toEqual(['pips.bottom', 'pips.top']);
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
        const [bottom, top] = current().question.numbers;
        expect(bottom.value).toBe(12); // deux pions sur le point 6
        expect(top.value).toBe(10); // deux pions sur le point 20, soit 25 − 20
    });

    test('sans position sur le plateau, la session refuse plutôt que de s’ouvrir vide', async () => {
        positionStore.set(/** @type {any} */ ({}));
        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'board' })).toBe(false);
        expect(get(trainingSessionStore)).toBeNull();
    });
});

describe('quand la question suivante ne peut pas être posée', () => {
    // La position tirée a disparu entre-temps. La session rebasculait sur le
    // lanceur : « Terminer » disparaissait, et le journal de la session partait
    // sans un mot.
    //
    // Le préchargement déplace le MOMENT de la découverte, pas la règle : la
    // question qu'on regarde a été fabriquée avant, donc c'est celle d'après
    // qui échoue. D'où les deux « Suivante » ci-dessous.
    test('la session reste ouverte, le dit, et « Terminer » enregistre ce qui a été répondu', async () => {
        positionsStore.setIds([7]);
        db.LoadPosition.mockResolvedValue(board());
        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'library' })).toBe(true);
        revealQuestion();
        markFault(0);

        // Plus aucune position ne se charge.
        db.LoadPosition.mockResolvedValue(null);
        await nextTrainingQuestion();
        revealQuestion();
        await nextTrainingQuestion();

        const session = current();
        expect(session, 'une session ne se perd pas sans que l’utilisateur l’ait décidé').not.toBeNull();
        expect(session.question).toBeNull();
        expect(session.questionError).toBe('noQuestion');
        expect(session.items, 'les nombres déjà répondus sont toujours là').toHaveLength(4);

        await finishTrainingSession();
        expect(db.SaveTrainingSession).toHaveBeenCalledTimes(1);
        const row = db.SaveTrainingSession.mock.calls[0][0];
        expect(row.numbersAsked).toBe(4);
        expect(row.faults).toBe(1);
    });

    test('« Réessayer » repose une question sans rien enregistrer de plus', async () => {
        positionsStore.setIds([7]);
        db.LoadPosition.mockResolvedValue(board());
        await startTrainingSession({ exercise: 'pips', seedSource: 'library' });
        revealQuestion();
        db.LoadPosition.mockResolvedValue(null);
        await nextTrainingQuestion();
        revealQuestion();
        await nextTrainingQuestion();
        expect(current().question).toBeNull();

        db.LoadPosition.mockResolvedValue(board());
        await retryTrainingQuestion();
        const session = current();
        expect(session.question).not.toBeNull();
        expect(session.questionError).toBe('');
        expect(session.items, 'la question ratée n’a rien ajouté').toHaveLength(4);
    });
});

describe('le préchargement (ADR-0041 règle 5)', () => {
    // Le critère : « le chrono d'une question ne contient pas sa génération ».
    // On le tient par le MOMENT de la fabrication : la question n+1 se fait
    // pendant qu'on répond à la n.
    test('la question suivante est déjà en fabrication pendant qu’on répond à celle-ci', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue(generated());
        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'pool' })).toBe(true);
        await flush();
        expect(app.GenerateBearoffQuestion, 'une question à l’écran, deux fabriquées').toHaveBeenCalledTimes(2);
    });

    // L'oracle qui rougit vraiment : le générateur ne rend plus rien À PARTIR
    // DE « Valider ». Seule une fabrication lancée AVANT la réponse peut donc
    // aboutir — c'est exactement ce que le critère demande, et sans
    // préchargement « Suivante » attendrait pour toujours.
    test('« Suivante » affiche une question fabriquée avant la réponse, pas après', async () => {
        const ready = generated();
        app.GenerateBearoffQuestion.mockImplementation(() => (get(trainingSessionStore)?.revealed ? new Promise(() => {}) : Promise.resolve(ready)));
        await startTrainingSession({ exercise: 'bearoff', seedSource: 'pool' });
        await flush();
        revealQuestion();

        nextTrainingQuestion();
        await flush();
        expect(current().question, 'la question suivante était prête').not.toBeNull();
        expect(current().askedQuestions).toBe(1);
    });

    // Sur la source « plateau » de Pions il n'y a qu'une question à poser :
    // précharger reposerait la même, et fabriquer pour rien est du travail
    // qu'on facture à la machine sans rien donner à personne.
    test('rien n’est préchargé là où il n’y a qu’une question', async () => {
        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'board' })).toBe(true);
        await flush();
        expect(db.LoadPosition).not.toHaveBeenCalled();
    });
});

describe('fabriquer n’est pas montrer', () => {
    // Le défaut : le préchargement appelait la fabrication ENTIÈRE, effet
    // d'affichage compris. Le plateau sautait sur la position de la question
    // n+1 pendant qu'on répondait à la n, et `showImportedPosition` refermait
    // l'onglet Entraînement au passage (il force l'onglet Analyse).
    //
    // L'oracle est un COMPTE d'appels — celui-là même que les tests de cette
    // suite avaient cessé de faire quand ils sont passés à `mockResolvedValue`
    // pour absorber l'appel supplémentaire au lieu de le contester.
    test('Pions / base : deux questions fabriquées, une seule montrée', async () => {
        positionsStore.setIds([7, 8, 9, 10]);
        db.LoadPosition.mockResolvedValue(board());
        const { moves, unsubscribe } = recordIndexMoves();
        expect(await startTrainingSession({ exercise: 'pips', seedSource: 'library' })).toBe(true);
        await flush();
        unsubscribe();

        expect(db.LoadPosition, 'la question suivante se prépare bien').toHaveBeenCalledTimes(2);
        expect(moves, 'une seule question est à l’écran').toHaveLength(1);
    });

    test('Bearoff / base : le préchargement ne touche pas au plateau', async () => {
        positionsStore.setIds([7, 8, 9, 10]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateBearoffQuestion.mockResolvedValue(generated());
        const { moves, unsubscribe } = recordIndexMoves();
        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'library' })).toBe(true);
        await flush();
        unsubscribe();

        expect(app.GenerateBearoffQuestion).toHaveBeenCalledTimes(2);
        expect(moves).toHaveLength(1);
    });

    test('Bearoff / vivier : le plateau garde la question posée, pas la suivante', async () => {
        // Deux positions distinctes : celle de la question à l'écran, et celle
        // que le préchargement fabrique derrière.
        const shown = generated(87.4, 91.2);
        const next = generated(50, 50);
        next.position = { ...board(), player_on_roll: 1 };
        app.GenerateBearoffQuestion.mockResolvedValueOnce(shown).mockResolvedValue(next);

        await startTrainingSession({ exercise: 'bearoff', seedSource: 'pool' });
        await flush();
        expect(app.GenerateBearoffQuestion).toHaveBeenCalledTimes(2);
        expect(get(positionStore).player_on_roll, 'le plateau a sauté sur la question suivante').toBe(shown.position.player_on_roll);
    });

    // Une question tirée de la base a un identifiant : c'est par lui qu'elle
    // arrive sur le plateau. Deux écrivains — l'un asynchrone, l'autre non —
    // auraient laissé l'identifiant final dépendre de l'ordre d'arrivée.
    test('une question tirée de la base arrive par son identifiant, pas par sa copie', async () => {
        positionsStore.setIds([7]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateBearoffQuestion.mockResolvedValue(generated());
        positionStore.set(/** @type {any} */ ({ id: 42 }));
        await startTrainingSession({ exercise: 'bearoff', seedSource: 'library' });

        expect(get(currentPositionIndexStore), 'la liste parcourue pointe la position tirée').toBe(positionsStore.indexOf(7));
        expect(get(positionStore).id, 'la copie engendrée a écrasé la position de la base').toBe(42);
    });
});

describe('une question tirée de la base laisse l’onglet ouvert (ADR-0040 règle 1)', () => {
    // Le défaut : la question tirée de la base arrivait par `showImportedPosition`,
    // qui bascule sur l'onglet Analyse — le geste d'un IMPORT, qui veut montrer
    // l'analyse de ce qu'on vient d'apporter. Sous une session, il cachait
    // l'onglet où l'on répond : les champs, « Révéler », « Terminer ». La
    // position est déjà dans la liste parcourue, puisqu'elle y a été tirée ; il
    // suffit de la pointer.
    test.each([
        ['pips', () => db.LoadPosition.mockResolvedValue(board())],
        [
            'bearoff',
            () => {
                db.LoadPosition.mockResolvedValue(board());
                app.GenerateBearoffQuestion.mockResolvedValue(generated());
            }
        ]
    ])('%s / base : la question est sur le plateau, et l’onglet Entraînement reste celui qu’on voit', async (exercise, arrange) => {
        positionsStore.setIds([7, 8, 9, 10]);
        arrange();
        expect(await startTrainingSession({ exercise, seedSource: 'library' })).toBe(true);
        await flush();

        expect(get(activeTabStore), 'la session a fermé son propre onglet').toBe('training');
        const { positionId } = current().question;
        expect(get(currentPositionIndexStore)).toBe(positionsStore.indexOf(positionId));
        expect(importService.showImportedPosition).not.toHaveBeenCalled();

        revealQuestion();
        await nextTrainingQuestion();
        expect(get(activeTabStore), '« Suivante » aussi').toBe('training');
    });
});

describe('la source « base » de Bearoff tire parmi les bearoffs de la liste (ADR-0041 règle 2)', () => {
    // Un tirage à l'aveugle dans la liste parcourue refusait une base qui a des
    // bearoffs : sur la base de démonstration, 30 positions du domaine sur 757,
    // et trente tirages sans remise n'en trouvent aucune une fois sur trois —
    // l'utilisateur lisait alors « cet exercice demande un bearoff » devant une
    // base qui en a. La liste est d'abord restreinte à la phase `bearoff` que la
    // base a déjà calculée ; le moteur reste seul juge du domaine (4 à 15 pions).
    test('le tirage ne regarde que les positions de la liste en phase bearoff', async () => {
        positionsStore.setIds([7, 8, 9, 10]);
        // 99 est un bearoff de la base, mais pas de la liste parcourue.
        db.LoadPositionIDsByFilters.mockResolvedValue([99, 9]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateBearoffQuestion.mockResolvedValue(generated());

        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'library' })).toBe(true);
        expect(db.LoadPositionIDsByFilters.mock.calls[0][0].gamePhaseFilter).toBe('bearoff');
        // Le préchargement tire aussi : toutes les positions chargées sont la seule candidate.
        expect(new Set(db.LoadPosition.mock.calls.map((call) => call[0]))).toEqual(new Set([9]));
        expect(current().question.positionId).toBe(9);
    });

    test('la restriction se calcule une fois par session, pas à chaque question', async () => {
        positionsStore.setIds([7, 8, 9, 10]);
        db.LoadPositionIDsByFilters.mockResolvedValue([8, 9]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateBearoffQuestion.mockResolvedValue(generated());

        await startTrainingSession({ exercise: 'bearoff', seedSource: 'library' });
        await flush();
        revealQuestion();
        await nextTrainingQuestion();
        await flush();
        expect(db.LoadPositionIDsByFilters).toHaveBeenCalledTimes(1);
    });

    // Une base dont les phases n'ont jamais été calculées (lignes d'avant 2.19.0,
    // jamais passées par `blunderdb repair`) ne répond rien à la phase : le
    // tirage retombe alors sur toute la liste, comme avant, plutôt que de
    // refuser une base qui a peut-être des bearoffs.
    test('sans aucune position classée bearoff dans la liste, le tirage retombe sur la liste entière', async () => {
        positionsStore.setIds([7, 8, 9, 10]);
        db.LoadPositionIDsByFilters.mockResolvedValue([]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateBearoffQuestion.mockResolvedValue(generated());

        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'library' })).toBe(true);
        expect([7, 8, 9, 10]).toContain(db.LoadPosition.mock.calls[0][0]);
    });
});

describe('la source « plateau » ne dérive pas (ADR-0041 règle 2)', () => {
    // « La position telle qu'elle est AU DÉMARRAGE ». Le défaut : la question
    // était écrite dans `positionStore` avant que le préchargement n'y relise
    // la graine, donc chaque question devenait la graine de la suivante — et
    // dès qu'un camp touchait quatre pions, le générateur s'arrêtait à zéro pli
    // et reservait indéfiniment la position qu'on venait de répondre.
    test('toutes les questions partent de la MÊME graine', async () => {
        const seed = board();
        positionStore.set(seed);
        app.GenerateBearoffQuestion.mockImplementation(() =>
            // Le moteur rend une position DIFFÉRENTE de la graine, comme il le
            // fait après un à quatre plis.
            Promise.resolve(generated(70, 70, { position: { ...board(), player_on_roll: 1 } }))
        );

        await startTrainingSession({ exercise: 'bearoff', seedSource: 'board' });
        await flush();
        revealQuestion();
        await nextTrainingQuestion();
        await flush();

        const seeds = app.GenerateBearoffQuestion.mock.calls.map((call) => call[0].seed);
        expect(seeds.length).toBeGreaterThanOrEqual(3);
        for (const sent of seeds) {
            expect(sent, 'une question est devenue la graine de la suivante').toEqual(seed);
        }
    });
});

describe('une session de Bearoff', () => {
    test('le moteur est appelé avec la source, et les deux EPC deviennent les nombres', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue(generated(87.4, 91.2));
        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'pool' })).toBe(true);
        expect(app.GenerateBearoffQuestion.mock.calls[0][0]).toEqual({ source: 'pool' });

        const question = current().question;
        expect(question.numbers).toEqual([
            { type: 'epc.bottom', value: 87.4, tolerance: 0.5, precision: 1 },
            { type: 'epc.top', value: 91.2, tolerance: 0.5, precision: 1 }
        ]);
    });

    test('la source « plateau » envoie la position telle qu’elle est comme graine', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue(generated());
        await startTrainingSession({ exercise: 'bearoff', seedSource: 'board' });
        const request = app.GenerateBearoffQuestion.mock.calls[0][0];
        expect(request.source).toBe('board');
        expect(request.seed).toEqual(board());
    });

    test('la position engendrée arrive SUR LE PLATEAU : elle n’est dans aucune base', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue(generated());
        positionStore.set(/** @type {any} */ ({ id: 42 }));
        await startTrainingSession({ exercise: 'bearoff', seedSource: 'pool' });
        expect(get(positionStore).board).toEqual(board().board);
        expect(get(positionStore).id, 'une position engendrée n’a pas d’identifiant').toBe(0);
    });

    test('la saisie est jugée par l’application et l’écart signé part au journal', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue(generated(87.4, 91.2));
        await startTrainingSession({ exercise: 'bearoff', seedSource: 'pool' });
        setTrainingAnswer(0, '90.4');
        setTrainingAnswer(1, '91.2');
        revealQuestion();
        expect(current().faults).toEqual([true, false]);
        await finishTrainingSession();

        const row = db.SaveTrainingSession.mock.calls[0][0];
        expect(row.exercise).toBe('bearoff');
        expect(row.deviations).toBe(2);
        expect(row.items.map((i) => i.deviation)).toEqual([3, 0]);
    });

    // ADR-0041 règle 3 : le refus NOMME le domaine, et rien ne démarre.
    test('une graine hors domaine refuse en le nommant, et rien ne démarre', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue({ generated: false, refusal: 'notBearoff', source: 'board' });
        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'board' })).toBe(false);
        expect(get(trainingSessionStore)).toBeNull();
        expect(get(trainingRefusalStore)).toBe('notBearoff');
    });

    // Le seul refus qui produit quand même une question : un plateau vide
    // retombe sur le vivier, et garde la phrase.
    test('un plateau vide démarre sur le vivier ET dit pourquoi', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue(generated(87.4, 91.2, { refusal: 'emptyBoard' }));
        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'board' })).toBe(true);
        expect(get(trainingSessionStore)).not.toBeNull();
        expect(get(trainingRefusalStore)).toBe('emptyBoard');
    });

    // Le domaine n'est écrit qu'en Go : la liste parcourue est tirée, et c'est
    // le moteur qui dit ce qui convient. Une seconde définition ici finirait
    // par diverger de celle qui refuse.
    test('la source « base » tire jusqu’à trouver un bearoff, et laisse le moteur juger', async () => {
        positionsStore.setIds([7, 8, 9, 10]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateBearoffQuestion.mockResolvedValueOnce({ generated: false, refusal: 'notBearoff' })
            .mockResolvedValueOnce({ generated: false, refusal: 'notBearoff' })
            .mockResolvedValue(generated());
        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'library' })).toBe(true);
        expect(app.GenerateBearoffQuestion.mock.calls[0][0].source).toBe('library');
        expect(app.GenerateBearoffQuestion.mock.calls.length).toBeGreaterThanOrEqual(3);
    });

    test('une base sans course refuse en nommant le domaine, pas « aucune question »', async () => {
        positionsStore.setIds([7, 8, 9]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateBearoffQuestion.mockResolvedValue({ generated: false, refusal: 'notBearoff' });
        expect(await startTrainingSession({ exercise: 'bearoff', seedSource: 'library' })).toBe(false);
        expect(get(trainingRefusalStore)).toBe('notBearoff');
    });
});

describe('la tolérance dite en toutes lettres', () => {
    // La phrase de l'interface porte le nombre en mots — « un demi-pion » —
    // parce qu'un `toFixed(1)` écrivait « 0.5 » au milieu d'une phrase
    // française. Le prix de ce choix est que la prose peut dériver du code :
    // c'est ce que ce test interdit.
    test('la prose des neuf langues et la constante disent la même chose', async () => {
        app.GenerateBearoffQuestion.mockResolvedValue(generated());
        await startTrainingSession({ exercise: 'bearoff', seedSource: 'pool' });
        const [first] = current().question.numbers;
        expect(first.tolerance, 'la phrase dit « un demi-pion »').toBe(0.5);
        expect(fr.training.tolerance).toContain('demi-pion');
    });
});

describe('une session d’Évaluation (#322)', () => {
    /** Une question d'Évaluation telle que le moteur la rend.
     *  @returns {any} */
    function evaluation(extra = {}) {
        return {
            generated: true,
            refusal: '',
            source: 'pool',
            plies: 4,
            position: board(),
            winChance: 71.3,
            cubeVerdict: 'double_take',
            cubeAnswer: 'dt',
            regime: 'evaluated',
            depth: '2-ply',
            ...extra
        };
    }

    test('une seule question porte les deux nombres : les chances saisies, le videau choisi', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation());
        expect(await startTrainingSession({ exercise: 'evaluation', seedSource: 'pool' })).toBe(true);
        expect(app.GenerateEvaluationQuestion.mock.calls[0][0]).toEqual({ source: 'pool' });
        expect(app.GenerateBearoffQuestion, 'Évaluation a son propre générateur').not.toHaveBeenCalled();

        const question = current().question;
        expect(question.numbers).toEqual([
            { type: 'eval.win', value: 71.3, tolerance: 5, precision: 1 },
            { type: 'eval.cube', value: 0, mode: 'chosen', answer: 'dt' }
        ]);
        expect(question.cubeVerdict).toBe('double_take');
    });

    test('la source « plateau » envoie la position telle qu’elle est comme graine', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation());
        await startTrainingSession({ exercise: 'evaluation', seedSource: 'board' });
        const request = app.GenerateEvaluationQuestion.mock.calls[0][0];
        expect(request.source).toBe('board');
        expect(request.seed).toEqual(board());
    });

    test('la position engendrée arrive sur le plateau, sans identifiant, et l’onglet reste ouvert', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation());
        positionStore.set(/** @type {any} */ ({ id: 42 }));
        await startTrainingSession({ exercise: 'evaluation', seedSource: 'pool' });
        expect(get(positionStore).board).toEqual(board().board);
        expect(get(positionStore).id).toBe(0);
        expect(get(activeTabStore)).toBe('training');
        expect(importService.showImportedPosition).not.toHaveBeenCalled();
    });

    test('« Valider » juge les deux, et le journal les distingue par type', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation());
        await startTrainingSession({ exercise: 'evaluation', seedSource: 'pool' });
        setTrainingAnswer(0, '64');
        setTrainingAnswer(1, 'dp');
        revealQuestion();
        expect(current().faults, '7,3 points de trop peu : hors tolérance ; « passe » au lieu de « prend »').toEqual([true, true]);
        await finishTrainingSession();

        const row = db.SaveTrainingSession.mock.calls[0][0];
        const items = /** @type {any[]} */ (row.items);
        expect(row.exercise).toBe('evaluation');
        expect(items.map((i) => i.numberType)).toEqual(['eval.win', 'eval.cube']);
        expect(items[0].hasDeviation).toBe(true);
        expect(items[0].deviation).toBeCloseTo(-7.3, 9);
        expect(items[1].hasDeviation, 'une action de videau est juste ou fausse, sans écart').toBe(false);
        expect(row.deviations).toBe(1);
    });

    // Le verdict vient du moteur : l'interface ne compare aucune équité. À 95 %
    // de chances, un « trop bon » rend « pas de double » juste — et c'est le
    // bouton que le moteur a nommé, pas une règle de l'interface, qui le dit.
    test('le bouton juste est celui que le moteur a nommé, pas un calcul de l’interface', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation({ winChance: 95, cubeVerdict: 'too_good', cubeAnswer: 'nd' }));
        await startTrainingSession({ exercise: 'evaluation', seedSource: 'pool' });
        setTrainingAnswer(0, '95');
        setTrainingAnswer(1, 'nd');
        revealQuestion();
        expect(current().faults).toEqual([false, false]);
    });

    test('rien de choisi est une faute, et le panneau Analyse reste masqué jusqu’à la réponse', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation());
        await startTrainingSession({ exercise: 'evaluation', seedSource: 'pool' });
        expect(get(trainingAnalysisHiddenStore)).toBe(true);
        setTrainingAnswer(0, '71');
        revealQuestion();
        expect(current().faults).toEqual([false, true]);
        expect(get(trainingAnalysisHiddenStore)).toBe(false);
    });

    test('la source « base » tire jusqu’à une décision de videau d’argent, et laisse le moteur juger', async () => {
        positionsStore.setIds([7, 8, 9, 10]);
        db.LoadPosition.mockImplementation((/** @type {any} */ id) => Promise.resolve({ ...board(), id }));
        app.GenerateEvaluationQuestion.mockResolvedValueOnce(/** @type {any} */ ({ generated: false, refusal: 'notMoneyCubeDecision' }))
            .mockResolvedValueOnce(/** @type {any} */ ({ generated: false, refusal: 'notMoneyCubeDecision' }))
            .mockResolvedValue(evaluation({ source: 'library' }));
        const moves = recordIndexMoves();
        expect(await startTrainingSession({ exercise: 'evaluation', seedSource: 'library' })).toBe(true);
        moves.unsubscribe();
        expect(app.GenerateEvaluationQuestion.mock.calls[0][0].source).toBe('library');
        expect(app.GenerateEvaluationQuestion.mock.calls.length).toBeGreaterThanOrEqual(3);
        // Tirée de la base : montrée par son index, jamais par l'import.
        expect(moves.moves).toHaveLength(1);
        expect(positionsStore.idAt(moves.moves[0])).toBe(current().question.positionId);
    });

    test('une base sans décision de videau d’argent refuse en le nommant, et rien ne démarre', async () => {
        positionsStore.setIds([7, 8, 9]);
        db.LoadPosition.mockResolvedValue(board());
        app.GenerateEvaluationQuestion.mockResolvedValue(/** @type {any} */ ({ generated: false, refusal: 'notMoneyCubeDecision' }));
        expect(await startTrainingSession({ exercise: 'evaluation', seedSource: 'library' })).toBe(false);
        expect(get(trainingSessionStore)).toBeNull();
        expect(get(trainingRefusalStore)).toBe('notMoneyCubeDecision');
    });

    test('un plateau vide démarre sur le vivier ET dit pourquoi', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation({ refusal: 'emptyBoard' }));
        expect(await startTrainingSession({ exercise: 'evaluation', seedSource: 'board' })).toBe(true);
        expect(get(trainingRefusalStore)).toBe('emptyBoard');
    });

    test('la tolérance dite en toutes lettres est celle que l’on juge', async () => {
        app.GenerateEvaluationQuestion.mockResolvedValue(evaluation());
        await startTrainingSession({ exercise: 'evaluation', seedSource: 'pool' });
        expect(current().question.numbers[0].tolerance, 'la phrase dit « cinq points »').toBe(5);
        expect(fr.training.toleranceEvaluation).toContain('cinq points');
    });
});

describe('une session de Décision (#323)', () => {
    // Le plateau que le MOTEUR rend avec le coup : c'est lui, et non le damier
    // reconstruit pas à pas par l'interface, qui part au juge.
    const RESULT = { points: [], bearoff: [0, 0], marker: 'rendu par le moteur' };
    const PLAY = {
        notation: '6/4 6/3',
        steps: [
            { from: 6, to: 4, hit: false },
            { from: 6, to: 3, hit: false }
        ],
        result: { board: RESULT }
    };
    const CHECKER = { checkerAnalysis: { moves: [{ move: '6/4 6/3', equityError: 0 }] } };
    const CUBE = { doublingCubeAnalysis: { bestCubeAction: 'No double' } };

    /** @param {Record<number, any>} analyses l'analyse de chaque position de la liste, `null` sans analyse */
    function library(analyses) {
        positionsStore.setIds(Object.keys(analyses).map(Number));
        db.LoadAnalysis.mockImplementation((/** @type {any} */ id) => Promise.resolve(analyses[id] ?? null));
        db.LoadPosition.mockImplementation((/** @type {any} */ id) => Promise.resolve({ ...board(), id }));
        app.LegalMoves.mockResolvedValue(/** @type {any} */ ([PLAY]));
    }

    function playOnBoard() {
        quizPlayStore.update((s) => (s ? playHop(playHop(s, 6, 4), 6, 3) : s));
    }

    /** @param {any} extra */
    function verdict(extra = {}) {
        return { legal: true, matched: true, notation: '6/4 6/3', best: '6/4 6/3', errorMp: 0, ...extra };
    }

    /** Répond à la question courante, quelle que soit sa forme, pour un coût donné. @param {number} errorMp */
    async function answerCurrent(errorMp) {
        if (current().question.prompt === 'cube') {
            db.GradeQuizCube.mockResolvedValueOnce(/** @type {any} */ (verdict({ notation: 'nd', errorMp })));
            await answerDecisionCube('nd');
        } else {
            playOnBoard();
            db.GradeQuizChecker.mockResolvedValueOnce(/** @type {any} */ (verdict({ errorMp })));
            await answerDecisionBoard();
        }
    }

    beforeEach(() => {
        db.LoadAnalysis.mockReset();
        db.LoadAnalysis.mockResolvedValue(/** @type {any} */ (null));
        db.GradeQuizChecker.mockReset();
        db.GradeQuizCheckerMove.mockReset();
        db.GradeQuizCube.mockReset();
        app.LegalMoves.mockReset();
        app.LegalMoves.mockResolvedValue([]);
        quizPlayStore.set(null);
    });

    describe('refuse en le nommant', () => {
        test('sans bibliothèque ouverte, rien ne démarre', async () => {
            databasePathStore.set('');
            library({ 7: CHECKER });
            expect(await startTrainingSession({ exercise: 'decision', seedSource: 'library' })).toBe(false);
            expect(get(trainingRefusalStore)).toBe('noLibrary');
            expect(get(trainingSessionStore)).toBeNull();
            expect(statusBar.setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'training.refusal.noLibrary', i18nParams: null });
        });

        test('sans position analysée dans la liste parcourue, rien ne démarre', async () => {
            library({ 7: null, 8: null });
            expect(await startTrainingSession({ exercise: 'decision', seedSource: 'library' })).toBe(false);
            expect(get(trainingRefusalStore)).toBe('noAnalysis');
            expect(get(trainingSessionStore)).toBeNull();
            expect(statusBar.setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'training.refusal.noAnalysis', i18nParams: null });
        });
    });

    describe('une décision de pions', () => {
        // Le simulacre d'import bascule d'onglet comme le vrai : si la question
        // passait par lui, l'onglet où l'on répond disparaîtrait (#321).
        test('se pose dans l’onglet, sur le plateau armé des coups légaux', async () => {
            library({ 7: CHECKER });
            expect(await startTrainingSession({ exercise: 'decision', seedSource: 'library' })).toBe(true);
            expect(get(activeTabStore)).toBe('training');
            expect(importService.showImportedPosition).not.toHaveBeenCalled();
            expect(get(currentPositionIndexStore)).toBe(0);
            expect(current().question.prompt).toBe('checker');
            expect(current().question.numbers.map((/** @type {any} */ n) => n.type)).toEqual(['decision.checker']);
            expect(get(quizPlayStore), 'le coup se joue sur le plateau').not.toBeNull();
            expect(app.LegalMoves).toHaveBeenCalledWith({ ...board(), id: 7 });
        });

        test('le panneau Analyse, qui porte la réponse, est masqué jusqu’au verdict', async () => {
            library({ 7: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            expect(get(trainingAnalysisHiddenStore)).toBe(true);
            await answerCurrent(0);
            expect(get(trainingAnalysisHiddenStore)).toBe(false);
        });

        test('« Valider » juge le plateau que le moteur a rendu avec ce coup', async () => {
            library({ 7: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            playOnBoard();
            db.GradeQuizChecker.mockResolvedValue(/** @type {any} */ (verdict()));
            await answerDecisionBoard();
            expect(db.GradeQuizChecker).toHaveBeenCalledWith(7, RESULT);
            expect(current().revealed).toBe(true);
            expect(current().faults).toEqual([false]);
            expect(current().verdict.notation).toBe('6/4 6/3');
        });

        test('sans coup complet, « Valider » ne juge rien', async () => {
            library({ 7: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            quizPlayStore.update((s) => (s ? playHop(s, 6, 4) : s));
            await answerDecisionBoard();
            expect(db.GradeQuizChecker).not.toHaveBeenCalled();
            expect(current().revealed).toBe(false);
        });

        test('annuler un pas, puis tout reprendre', async () => {
            library({ 7: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            playOnBoard();
            undoDecisionStep();
            expect(get(quizPlayStore)?.steps).toHaveLength(1);
            resetDecisionPlay();
            expect(get(quizPlayStore)?.steps).toHaveLength(0);
            expect(get(quizPlayStore)?.board.points[6]).toEqual({ checkers: 2, color: 0 });
        });

        test('un juge en échec laisse la question ouverte, et le dit', async () => {
            library({ 7: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            playOnBoard();
            db.GradeQuizChecker.mockRejectedValue(new Error('base fermée'));
            await answerDecisionBoard();
            expect(current().revealed).toBe(false);
            expect(statusBar.setStatusBarMessage).toHaveBeenCalledWith({ i18nKey: 'training.gradeFailed', i18nParams: null });
        });

        test('hors délai, la correction est demandée au juge et s’affiche sans rien coûter', async () => {
            library({ 7: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library', limitSeconds: 15 });
            db.GradeQuizCheckerMove.mockResolvedValue(/** @type {any} */ (verdict({ legal: false, matched: false, notation: '', best: '6/4 6/3' })));
            revealQuestion({ outOfTime: true });
            await flush();
            expect(db.GradeQuizCheckerMove).toHaveBeenCalledWith(7, '');
            expect(current().faults).toEqual([true]);
            expect(current().verdict.best).toBe('6/4 6/3');
        });
    });

    describe('une action de videau', () => {
        test('se choisit, sans plateau à jouer', async () => {
            library({ 7: CUBE });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            expect(current().question.prompt).toBe('cube');
            expect(get(quizPlayStore)).toBeNull();
            db.GradeQuizCube.mockResolvedValue(/** @type {any} */ (verdict({ notation: 'dt', best: 'No double', errorMp: 25 })));
            await answerDecisionCube('dt');
            expect(db.GradeQuizCube).toHaveBeenCalledWith(7, 'dt');
            expect(current().faults).toEqual([true]);
            expect(current().verdict.errorMp).toBe(25);
        });
    });

    describe('la session', () => {
        test('« Terminer » écrit le PR de session, sur l’échelle des statistiques', async () => {
            library({ 7: CHECKER, 8: CUBE });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            await answerCurrent(30);
            await nextTrainingQuestion();
            await answerCurrent(10);
            await finishTrainingSession();

            expect(db.SaveTrainingSession).toHaveBeenCalledTimes(1);
            const row = db.SaveTrainingSession.mock.calls[0][0];
            expect(row.exercise).toBe('decision');
            expect(row.seedSource).toBe('library');
            expect(row.numbersAsked).toBe(2);
            // 500 × (30 + 10) / 1000 / 2
            expect(row.pr).toBeCloseTo(10);
            expect(get(quizPlayStore), 'le plateau redevient celui de l’application').toBeNull();
        });

        test('une position posée ne revient pas, et la liste épuisée le dit', async () => {
            library({ 7: CHECKER, 8: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            const first = current().question.positionId;
            await answerCurrent(0);
            await nextTrainingQuestion();
            expect(current().question.positionId).not.toBe(first);
            await answerCurrent(0);
            await nextTrainingQuestion();
            expect(current().question).toBeNull();
            expect(current().questionError).toBe('decisionsExhausted');
            expect(get(quizPlayStore)).toBeNull();
            expect(current().items, 'les deux décisions répondues restent').toHaveLength(2);
        });

        test('« Quitter » n’écrit rien et désarme le plateau', async () => {
            library({ 7: CHECKER });
            await startTrainingSession({ exercise: 'decision', seedSource: 'library' });
            quitTrainingSession();
            expect(get(quizPlayStore)).toBeNull();
            expect(db.SaveTrainingSession).not.toHaveBeenCalled();
        });

        test('une autre session ne touche pas au coup joué au plateau par une transcription', async () => {
            const transcription = /** @type {any} */ ({ steps: [], board: board().board, plays: [], selected: null });
            quizPlayStore.set(transcription);
            await startTrainingSession({ exercise: 'scores' });
            quitTrainingSession();
            expect(get(quizPlayStore)).toBe(transcription);
            quizPlayStore.set(null);
        });
    });
});
