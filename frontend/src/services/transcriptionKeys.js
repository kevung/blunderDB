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
 * (`transcriptionKeys.budget.test.js`). Une machine enfouie dans le `.svelte`
 * n'aurait laissé qu'un test de bout en bout, trop lent pour tenir une ligne par
 * transition.
 *
 * Ce que la machine ne décide pas : la liste des candidats. Elle est calculée
 * par le moteur (LegalMoves + une évaluation 0-ply) après le second dé, donc
 * après la frappe — la machine pose `awaitingCandidates` et laisse le panneau
 * lui rendre la réponse. Le tour de pions lui-même (jet corrigeable, candidats,
 * `j`/`k`, Entrée, danse) est T1.3 ; ce fichier n'en porte pour l'instant que
 * l'ouverture, qui est la même saisie de deux dés.
 *
 * Conventions de clavier (voir utils/keys.js) : les CHIFFRES sont positionnels
 * (`event.code`, pour que la rangée du haut d'un AZERTY marche sans Maj), les
 * LETTRES sont lues au caractère produit (`event.key`).
 */

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
    VALIDATE: 'validate'
});

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

/** Un résultat « la touche ne me concerne pas » : elle remonte au répartiteur. */
const ignored = (state) => ({ handled: false, state, commands: [] });

/**
 * Applique une touche.
 *
 * @param {object} state - l'état rendu par `initialKeyState` ou par un appel précédent
 * @param {KeyboardEvent} event
 * @param {{expects?: string}} context - `expects` est `annotated.next.expects`,
 *   la sorte d'Action que le document attend : `'opening'` ou autre chose.
 * @returns {{handled: boolean, state: object, commands: {kind: string, value?: number, index?: number}[]}}
 */
export function pressKey(state, event, { expects = 'checker' } = {}) {
    // Lot T1.2 : seule l'ouverture est saisissable. Le tour de pions — jet
    // corrigeable, candidats, j/k, Entrée — est T1.3, et jusque-là ses touches
    // remontent au répartiteur global plutôt que d'être avalées en silence.
    if (expects !== 'opening') return ignored(state);

    const die = dieOf(event);
    if (die > 0) return enterDie(state, die, expects);

    // Retour arrière efface les deux dés — l'Action n'existe pas encore, il n'y
    // a rien à défaire (ux.md §3). La touche est PRISE même quand rien n'est
    // saisi : ailleurs elle réinitialise le plateau, et le plateau appartient ici
    // au brouillon. Échap, au contraire, n'est prise que s'il y a une saisie à
    // abandonner : sans cela elle ferme le panneau, comme partout.
    if (event.key === 'Backspace' || event.key === 'Escape') {
        if (state.phase === PHASE.DICE) {
            return event.key === 'Backspace' ? { handled: true, state, commands: [] } : ignored(state);
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
 * Un chiffre. Premier dé, puis second ; sur une ouverture le second dé VALIDE
 * aussitôt — les deux dés sont l'Action entière, il n'y a rien à choisir
 * (ux.md §3, dernière ligne : l'ouverture coûte deux touches).
 */
function enterDie(state, die, expects) {
    if (state.phase === PHASE.DICE || state.phase === PHASE.DIE1) {
        if (state.phase === PHASE.DICE) {
            return {
                handled: true,
                state: { ...state, phase: PHASE.DIE1, dice: [die, 0], tie: false },
                commands: [{ kind: COMMAND.DIE, value: die }]
            };
        }
        const dice = [state.dice[0], die];
        const commands = [{ kind: COMMAND.DIE, value: die }, { kind: COMMAND.VALIDATE }];
        if (expects === 'opening' && dice[0] === dice[1]) {
            // Égalité : l'Action `opening` est enregistrée telle quelle et une
            // autre ouverture est attendue. Rien n'est effacé, rien n'est refusé.
            return { handled: true, state: { ...initialKeyState(), tie: true }, commands };
        }
        // Le gagnant du jet joue les DEUX dés comme premier coup de pions : il
        // ne les ressaisit pas (fonctionnel.md §1.2). Le moteur les repose sur
        // l'Entry suivante ; la machine les garde pour les afficher et pour
        // demander les candidats.
        const roll = dice[0] >= dice[1] ? [dice[0], dice[1]] : [dice[1], dice[0]];
        return {
            handled: true,
            state: { ...initialKeyState(), phase: PHASE.ROLL, dice: roll, awaitingCandidates: true },
            commands
        };
    }
    return ignored(state);
}
