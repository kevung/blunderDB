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
 * Conventions de clavier (voir utils/keys.js) : les CHIFFRES sont positionnels
 * (`event.code`, pour que la rangée du haut d'un AZERTY marche sans Maj), les
 * LETTRES sont lues au caractère produit (`event.key`).
 */

import { isBareLetter } from '../utils/keys.js';

/** Les quatre états d'ux.md §3. */
export const PHASE = Object.freeze({
    /** Dés attendus, aucun dé saisi. */
    DICE: 'dice',
    /** Dés attendus, un dé saisi. */
    DIE1: 'die1',
    /** Jet saisi, premier candidat présélectionné : un chiffre recommence le jet. */
    ROLL: 'roll',
    /** Candidat choisi : un chiffre valide et ouvre le tour suivant. */
    CANDIDATE: 'candidate'
});

/** Les gestes que la machine demande ; le panneau les traduit en appels Go. */
export const COMMAND = Object.freeze({
    DIE: 'die',
    CLEAR: 'clear',
    VALIDATE: 'validate',
    SELECT: 'select',
    DANCE: 'dance',
    CURSOR_BACK: 'cursor_back',
    CURSOR_FORWARD: 'cursor_forward'
});

/** Les sortes d'Action dont la saisie passe par deux dés. */
const DICE_KINDS = new Set(['opening', 'checker', 'dance']);

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
        tie: false
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

/**
 * +1 (Action suivante), −1 (précédente), 0 sinon : `h`/`l` et gauche/droite.
 *
 * Le Cursor est une CELLULE du Transcript et les deux colonnes sont les deux
 * camps : le déplacer d'une Action, c'est passer d'une cellule à l'autre, donc
 * d'un camp à l'autre. D'où l'axe horizontal, quand `j`/`k` gardent le vertical
 * pour la liste des candidats (ux.md §3).
 */
export function cursorDelta(event) {
    if (isBareLetter(event, 'h') || event.key === 'ArrowLeft') return -1;
    if (isBareLetter(event, 'l') || event.key === 'ArrowRight') return 1;
    return 0;
}

/**
 * Les gestes qui mènent le Cursor de `from` à `to` — ce qu'un clic sur une
 * cellule du Transcript demande. Le moteur ne connaît que « recule » et
 * « avance » (`cursor_back`/`cursor_forward` d'apply.go), qui rechargent
 * l'Action visée : un saut est donc la répétition du pas, et non un geste de
 * plus à écrire côté Go pour la souris seule.
 *
 * @param {number} from
 * @param {number} to
 * @returns {{kind: string}[]}
 */
export function cursorCommands(from, to) {
    const kind = to < from ? COMMAND.CURSOR_BACK : COMMAND.CURSOR_FORWARD;
    return Array.from({ length: Math.abs(to - from) }, () => ({ kind }));
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
    // Le Cursor se déplace depuis TOUT état (ux.md §3, ligne « tout ») : c'est
    // le geste de la relecture, et il doit marcher pendant une saisie de dés
    // comme devant une réponse au videau. Il est donc lu avant tout le reste,
    // et il rend la machine à son état initial — le panneau la réarme sur
    // l'Action visée, dont le moteur recharge dés et coup.
    const step = cursorDelta(event);
    if (step !== 0) {
        return {
            handled: true,
            state: initialKeyState(),
            commands: [{ kind: step < 0 ? COMMAND.CURSOR_BACK : COMMAND.CURSOR_FORWARD }]
        };
    }

    // Une réponse à un double n'est pas une saisie de dés : ses touches sont
    // `t`/`p` et elles sont à T1.4. Rien n'est avalé ici en attendant.
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
