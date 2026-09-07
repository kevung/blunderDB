/**
 * transcriptionKeys.js — la machine à états du clavier de la Transcription.
 *
 * C'est le tableau d'ux.md §3, écrit une fois, en fonction PURE : la frappe
 * n'est ni un `await`, ni un composant, ni un store. `pressKey(state, event,
 * contexte)` rend l'état suivant et la liste des gestes à envoyer au moteur Go ;
 * le panneau se charge des allers-retours Wails et n'a plus aucune règle à lui.
 *
 * Pourquoi cette séparation. Le budget d'ux.md §4.1 est un COMPTE DE TOUCHES —
 * meilleur coup joué en 2 K, n-ième coup en (n+1) K — et un compte ne se mesure
 * pas dans un composant : il se mesure en appelant `pressKey` autant de fois que
 * l'utilisateur presse une touche et en regardant ce qui en sort
 * (`transcriptionKeys.turn.test.js`). Une machine enfouie dans le `.svelte`
 * n'aurait laissé qu'un test de bout en bout, trop lent pour tenir une ligne par
 * transition.
 *
 * # Les deux règles qui font le budget
 *
 * Le second dé n'est pas confirmé : dès qu'il tombe, la liste des coups légaux
 * est classée en 0-ply et le PREMIER est présélectionné. Et un chiffre depuis
 * « candidat choisi » VALIDE le coup avant d'ouvrir le jet suivant : le meilleur
 * coup joué coûte donc les deux dés et rien de plus, la validation étant portée
 * par la première touche du tour d'après. Ce que cette règle exclut a sa sortie :
 * le dernier coup d'une partie se valide par Entrée.
 *
 * Tant que la liste n'a pas été touchée, l'état est « jet corrigeable » : un
 * chiffre y RECOMMENCE le jet plutôt que de valider. Dès que `j`/`k` bouge la
 * sélection, on est en « candidat choisi » et le chiffre suivant valide.
 *
 * # Ce que la machine ne décide pas
 *
 * La liste des candidats. Elle est calculée par le moteur (LegalMoves + une
 * évaluation 0-ply) APRÈS le second dé, donc après la frappe : la machine pose
 * `awaitingCandidates` et le panneau lui rend la réponse par [applyCandidates].
 * C'est de là que sort la danse — un jet sans aucun coup légal crée l'Action
 * `dance` aussitôt, zéro touche de plus (ux.md §3, dernière ligne).
 *
 * # Le videau et la résignation
 *
 * `d`, `t`, `p` et `r` ne coûtent chacune qu'une touche parce que le moteur
 * valide de lui-même le candidat resté en attente (`transcript.cubeGesture`) :
 * la machine n'émet donc PAS de `validate` avant elles, et « double + prise »
 * tient dans `d` `t` (ux.md §4.2). `r` est la seule touche qui ouvre un état :
 * elle attend un niveau, met l'état d'avant de côté et le rend tel quel si
 * `Échap` tombe entre les deux — « résignation gammon » coûte `r` `2`.
 *
 * Ce que ces touches ne font PAS : juger. Doubler sans posséder le videau, en
 * partie Crawford ou au-delà du plafond reste transcriptible — c'est une
 * Incohérence du Replay, marquée et jamais refusée (ADR-0044, fonctionnel.md
 * §1.4). Seules `t` et `p` demandent une offre en face, faute de quoi elles ne
 * répondent à rien.
 *
 * Conventions de clavier (voir utils/keys.js) : les CHIFFRES sont positionnels
 * (`event.code`, pour que la rangée du haut d'un AZERTY marche sans Maj), les
 * LETTRES sont lues au caractère produit (`event.key`).
 */

import { isBareLetter } from '../utils/keys.js';

/** Les cinq états d'ux.md §3. */
export const PHASE = Object.freeze({
    /** Dés attendus, aucun dé saisi. */
    DICE: 'dice',
    /** Dés attendus, un dé saisi. */
    DIE1: 'die1',
    /** Jet saisi, premier candidat présélectionné : un chiffre recommence le jet. */
    ROLL: 'roll',
    /** Candidat choisi : un chiffre valide et ouvre le tour suivant. */
    CANDIDATE: 'candidate',
    /**
     * Résignation annoncée, son niveau attendu : `1` simple, `2` gammon, `3`
     * backgammon, `Échap` annule. C'est le seul état qui détourne les chiffres
     * des dés, et c'est tout ce qui le distingue — d'où un état de la machine
     * et non un drapeau du panneau.
     */
    RESIGN: 'resign'
});

/** Les gestes que la machine demande ; le panneau les traduit en appels Go. */
export const COMMAND = Object.freeze({
    DIE: 'die',
    CLEAR: 'clear',
    VALIDATE: 'validate',
    SELECT: 'select',
    DANCE: 'dance',
    DOUBLE: 'double',
    TAKE: 'take',
    PASS: 'pass',
    RESIGN: 'resign'
});

/** Les sortes d'Action dont la saisie passe par deux dés. */
const DICE_KINDS = new Set(['opening', 'checker', 'dance']);

/** Le niveau d'une résignation : simple, gammon, backgammon. */
const RESIGN_LEVELS = new Set([1, 2, 3]);

/**
 * L'état initial : dés attendus, rien de saisi.
 *
 * `tie` retient qu'une ouverture est tombée à égalité, ce que le panneau affiche
 * « relance » (fonctionnel.md §1.2 : l'égalité reste dans le document et ne
 * produit ni Move ni Position).
 */
export function initialKeyState() {
    return {
        phase: PHASE.DICE,
        dice: [0, 0],
        selected: 0,
        candidateCount: 0,
        awaitingCandidates: false,
        tie: false,
        // L'état à rendre si la résignation est abandonnée par Échap. Nul
        // partout ailleurs : seule la phase RESIGN en pose un.
        resume: null
    };
}

/**
 * Le dé qu'une touche désigne, ou 0. Positionnel : `Digit3` et `Numpad3` valent
 * tous deux 3, quelle que soit la disposition du clavier.
 *
 * @param {KeyboardEvent} event
 * @returns {number} 1 à 6, ou 0
 */
export function dieOf(event) {
    if (event.ctrlKey || event.metaKey || event.altKey) return 0;
    const m = /^(?:Digit|Numpad)([1-6])$/.exec(event.code ?? '');
    return m ? Number(m[1]) : 0;
}

/** +1 (candidat suivant), −1 (précédent), 0 sinon : `j`/`k` et bas/haut. */
export function selectionDelta(event) {
    if (isBareLetter(event, 'j') || event.key === 'ArrowDown') return 1;
    if (isBareLetter(event, 'k') || event.key === 'ArrowUp') return -1;
    return 0;
}

/** Un résultat « la touche ne me concerne pas » : elle remonte au répartiteur. */
const ignored = (state) => ({ handled: false, state, commands: [] });
/** Un résultat « la touche est à moi, mais il n'y a rien à faire ». */
const swallowed = (state) => ({ handled: true, state, commands: [] });

const clamp = (n, max) => Math.min(Math.max(n, 0), max);

/**
 * Applique une touche.
 *
 * @param {object} state - l'état rendu par `initialKeyState` ou par un appel précédent
 * @param {KeyboardEvent} event
 * @param {{expects?: string}} context - `expects` est `annotated.next.expects`,
 *   la sorte d'Action que le document attend.
 * @returns {{handled: boolean, state: object, commands: {kind: string, value?: number, index?: number}[]}}
 */
export function pressKey(state, event, { expects = 'checker' } = {}) {
    // La résignation capte tout tant que son niveau n'est pas donné : ses
    // chiffres SONT des niveaux et non des dés, et rien d'autre ne doit passer
    // entre `r` et la touche qui la termine.
    if (state.phase === PHASE.RESIGN) return resignLevel(state, event);

    // `r` ouvre l'attente du niveau depuis n'importe quel état, et l'état
    // d'avant est mis de côté : `Échap` le rend intact, la résignation
    // « annule sans effet » (ux.md §3, fiche T1.5).
    if (isBareLetter(event, 'r')) {
        return { handled: true, state: { ...initialKeyState(), phase: PHASE.RESIGN, resume: state }, commands: [] };
    }

    // `d` double ou redouble depuis n'importe quel état de saisie. Une seule
    // touche : le moteur valide d'abord le candidat en attente s'il y en a un
    // (transcript.cubeGesture), donc « double + prise » coûte `d` `t` et rien
    // de plus (ux.md §4.2).
    //
    // Elle n'est JAMAIS avalée au motif que le camp ne possède pas le videau,
    // qu'on est en partie Crawford ou que le videau est au plafond : ce sont
    // les trois « videau impossible » de fonctionnel.md §1.4, et une
    // Incohérence est MARQUÉE, jamais refusée (ADR-0044). Le moteur la pose,
    // le panneau l'affiche.
    if (isBareLetter(event, 'd')) {
        return { handled: true, state: initialKeyState(), commands: [{ kind: COMMAND.DOUBLE }] };
    }

    // `t`/`p` ne répondent qu'à une offre. Ce n'est pas un refus : sans double
    // qui précède il n'y a pas de réponse à transcrire, et la touche reste
    // disponible pour le répartiteur global (ux.md §3 : « réponse attendue |
    // t / p »).
    if (expects === 'take') {
        if (isBareLetter(event, 't')) return { handled: true, state: initialKeyState(), commands: [{ kind: COMMAND.TAKE }] };
        if (isBareLetter(event, 'p')) return { handled: true, state: initialKeyState(), commands: [{ kind: COMMAND.PASS }] };
    }

    // Une réponse à un double n'est pas une saisie de dés : ses touches sont
    // `t`/`p`, traitées juste au-dessus.
    if (!DICE_KINDS.has(expects)) return ignored(state);

    const die = dieOf(event);
    if (die > 0) return enterDie(state, die, expects);

    const delta = selectionDelta(event);
    if (delta !== 0) return moveSelection(state, delta);

    if (event.key === 'Enter') {
        // Entrée valide seule. C'est la sortie que la règle « un chiffre
        // valide » laisse ouverte : le dernier coup d'une partie n'a pas de
        // tour suivant pour porter sa validation (ux.md §3).
        if (state.phase !== PHASE.ROLL && state.phase !== PHASE.CANDIDATE) return ignored(state);
        return { handled: true, state: initialKeyState(), commands: [{ kind: COMMAND.VALIDATE }] };
    }

    // Retour arrière efface les deux dés — l'Action n'existe pas encore, il n'y
    // a rien à défaire (ux.md §3). La touche est PRISE même quand rien n'est
    // saisi : ailleurs elle réinitialise le plateau, et le plateau appartient ici
    // au brouillon. Échap, au contraire, n'est prise que s'il y a une saisie à
    // abandonner : sans cela elle ferme le panneau, comme partout.
    if (event.key === 'Backspace' || event.key === 'Escape') {
        if (state.phase === PHASE.DICE) {
            return event.key === 'Backspace' ? swallowed(state) : ignored(state);
        }
        return {
            handled: true,
            state: { ...initialKeyState(), tie: state.tie },
            commands: [{ kind: COMMAND.CLEAR }]
        };
    }

    return ignored(state);
}

/**
 * Le niveau d'une résignation. `1`/`2`/`3` la crée — deux touches en tout avec
 * le `r` qui l'a ouverte, le budget d'ux.md §4.2 — et `Échap` rend la main à
 * l'état d'avant sans avoir créé la moindre Action.
 *
 * Tout le reste est AVALÉ : entre `r` et son chiffre, une touche qui n'est ni
 * un niveau ni l'annulation ne doit pas retomber sur les dés, sinon `r` puis
 * `5` enregistrerait un dé au lieu de ne rien faire.
 *
 * Le camp n'est pas dit ici. Il est celui au trait, et c'est le moteur qui le
 * sait (`transcript.cubeGesture`) : la machine à touches ne connaît pas le
 * document (fonctionnel.md §1.2).
 */
function resignLevel(state, event) {
    if (event.key === 'Escape') {
        return { handled: true, state: state.resume ?? initialKeyState(), commands: [] };
    }
    const level = dieOf(event);
    if (!RESIGN_LEVELS.has(level)) return swallowed(state);
    return { handled: true, state: initialKeyState(), commands: [{ kind: COMMAND.RESIGN, value: level }] };
}

/**
 * Un chiffre. Premier dé, puis second ; sur une ouverture le second dé valide
 * aussitôt. Depuis « jet corrigeable » il RECOMMENCE le jet, depuis « candidat
 * choisi » il VALIDE le candidat et ouvre le tour suivant — c'est la règle qui
 * ramène le meilleur coup joué à deux touches.
 */
function enterDie(state, die, expects) {
    switch (state.phase) {
        case PHASE.DICE:
            return {
                handled: true,
                state: { ...state, phase: PHASE.DIE1, dice: [die, 0], tie: false },
                commands: [{ kind: COMMAND.DIE, value: die }]
            };

        case PHASE.DIE1: {
            const dice = [state.dice[0], die];
            if (expects === 'opening') {
                const commands = [{ kind: COMMAND.DIE, value: die }, { kind: COMMAND.VALIDATE }];
                if (dice[0] === dice[1]) {
                    // Égalité : l'Action `opening` est enregistrée telle quelle et
                    // une autre ouverture est attendue. Rien n'est refusé.
                    return { handled: true, state: { ...initialKeyState(), tie: true }, commands };
                }
                // Le gagnant du jet joue les DEUX dés comme premier coup de pions :
                // il ne les ressaisit pas (fonctionnel.md §1.2). Le moteur les
                // repose sur l'Entry suivante ; la machine les garde pour les
                // afficher et pour demander les candidats.
                const roll = dice[0] >= dice[1] ? [dice[0], dice[1]] : [dice[1], dice[0]];
                return {
                    handled: true,
                    state: { ...initialKeyState(), phase: PHASE.ROLL, dice: roll, awaitingCandidates: true },
                    commands
                };
            }
            return {
                handled: true,
                state: { ...initialKeyState(), phase: PHASE.ROLL, dice, awaitingCandidates: true },
                commands: [{ kind: COMMAND.DIE, value: die }]
            };
        }

        case PHASE.ROLL:
            // Jet corrigeable : le chiffre recommence le jet. Le moteur fait de
            // même — `enter_die` sur une Entry aux deux dés pleins repart de zéro.
            return {
                handled: true,
                state: { ...initialKeyState(), phase: PHASE.DIE1, dice: [die, 0] },
                commands: [{ kind: COMMAND.DIE, value: die }]
            };

        case PHASE.CANDIDATE:
            return {
                handled: true,
                state: { ...initialKeyState(), phase: PHASE.DIE1, dice: [die, 0] },
                commands: [{ kind: COMMAND.VALIDATE }, { kind: COMMAND.DIE, value: die }]
            };

        default:
            return ignored(state);
    }
}

/**
 * `j`/`k` et bas/haut déplacent la sélection et FONT SORTIR du jet corrigeable :
 * dès que la liste a été touchée, le chiffre suivant valide au lieu de
 * recommencer le jet. Le déplacement est borné, il ne boucle pas.
 */
function moveSelection(state, delta) {
    if (state.phase !== PHASE.ROLL && state.phase !== PHASE.CANDIDATE) return ignored(state);
    // La liste n'est pas encore revenue du moteur : la touche est à nous, mais
    // il n'y a rien à déplacer.
    if (state.awaitingCandidates || state.candidateCount <= 0) return swallowed(state);

    const selected = clamp(state.selected + delta, state.candidateCount - 1);
    const commands = selected === state.selected ? [] : [{ kind: COMMAND.SELECT, index: selected }];
    return { handled: true, state: { ...state, phase: PHASE.CANDIDATE, selected }, commands };
}

/**
 * Choisir un candidat au clic, dans la liste classée. Même effet qu'un `j`/`k`
 * qui tomberait juste : la sélection bouge et l'on sort du jet corrigeable.
 */
export function selectCandidate(state, index) {
    if (state.candidateCount <= 0) return { state, commands: [] };
    const selected = clamp(index, state.candidateCount - 1);
    return {
        state: { ...state, phase: PHASE.CANDIDATE, selected },
        commands: [{ kind: COMMAND.SELECT, index: selected }]
    };
}

/**
 * La réponse du moteur au jet que la machine attendait : combien de coups légaux.
 *
 * Aucun : c'est une danse, l'Action est créée sans une touche de plus et le trait
 * passe à l'autre camp. Au moins un : le premier est présélectionné, ses flèches
 * partent sur le plateau, et le jet reste corrigeable tant que l'utilisateur n'a
 * pas touché à la liste.
 *
 * @param {object} state
 * @param {number} count
 */
export function applyCandidates(state, count) {
    if (!state.awaitingCandidates) {
        return { state: { ...state, candidateCount: count }, commands: [] };
    }
    if (count === 0) {
        return { state: initialKeyState(), commands: [{ kind: COMMAND.DANCE }] };
    }
    return {
        state: { ...state, awaitingCandidates: false, candidateCount: count, selected: 0, phase: PHASE.ROLL },
        commands: [{ kind: COMMAND.SELECT, index: 0 }]
    };
}
