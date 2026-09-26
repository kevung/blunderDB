/**
 * transcriptionKeys.js — la machine à états du clavier de la Transcription
 * (ux.md §3), en fonction PURE : `pressKey(state, event, contexte)` rend
 * l'état suivant et les gestes à envoyer au moteur Go ; le panneau ne porte
 * aucune règle.
 *
 * Pure parce que le budget d'ux.md §4.1 est un compte de touches, mesurable
 * seulement en appelant `pressKey` (`transcriptionKeys.turn.test.js`).
 *
 * Deux règles portent le budget :
 * - au second dé, les coups légaux sont classés en 0-ply et le premier est
 *   présélectionné, sans confirmation ;
 * - la touche chiffrée commence un jet là où le Cursor est (ADR-0048
 *   décision 1) : en bout de document elle VALIDE le candidat puis ouvre le
 *   jet suivant ; sur une Action relue elle RECOMMENCE le jet sur place. Le
 *   dernier coup d'une partie se valide par Entrée.
 * Le discriminant est `replacing` (`annotated.entry`, la cellule encadrée
 * du Transcript) : un état visible, pas un mode (ux.md §1). La machine ne
 * connaît pas le document (fonctionnel.md §1.2).
 *
 * Les candidats viennent du moteur : la machine pose `awaitingCandidates`,
 * le panneau répond par [applyCandidates]. Un jet sans coup légal crée
 * l'Action `dance`.
 *
 * Videau : `d`, `t`, `p`, `r` coûtent une touche car le moteur valide seul le
 * candidat en attente (`transcript.cubeGesture`, ux.md §4.2). `r` ouvre un
 * état qui attend un niveau et rend l'état d'avant sur `Échap`. La souris
 * passe par [cubeGesture], [beginResign], [resignWithLevel], même règle. Rien
 * n'est refusé : une offre illégale est marquée en Incohérence (ADR-0044,
 * fonctionnel.md §1.4). `t`/`p` exigent une offre en face, sinon la touche
 * remonte au répartiteur (`p` = pips).
 *
 * Correction : `h`/`l`, `i`/`a`, `x`/`Suppr`, `s` partent de TOUT état (ux.md
 * §3, ligne « tout » ; §4.3). `Ctrl+Z`/`Ctrl+Maj+Z` sont liés par le
 * répartiteur (`isAlwaysGlobal`) mais nommés ici (COMMAND.UNDO/REDO).
 *
 * Clavier (utils/keys.js) : chiffres par `event.code` (AZERTY), lettres par
 * `event.key`.
 */

import { isBareLetter } from '../utils/keys.js';

/** Les cinq états d'ux.md §3. */
export const PHASE = Object.freeze({
    /** Dés attendus, aucun dé saisi. */
    DICE: 'dice',
    /** Dés attendus, un dé saisi. */
    DIE1: 'die1',
    /** Jet saisi, premier candidat présélectionné. */
    ROLL: 'roll',
    /**
     * Liste touchée par l'utilisateur, pour que le panneau ne réécrive pas la
     * sélection. Ne change pas le sens du chiffre (ADR-0048 décision 1).
     */
    CANDIDATE: 'candidate',
    /**
     * Résignation annoncée, niveau attendu (`1`/`2`/`3`, `Échap` annule). Seul
     * état qui détourne les chiffres des dés, d'où un état et non un drapeau.
     */
    RESIGN: 'resign'
});

/** Les gestes que la machine demande ; le panneau les traduit en appels Go. */
export const COMMAND = Object.freeze({
    DIE: 'die',
    CLEAR: 'clear',
    VALIDATE: 'validate',
    SELECT: 'select',
    /**
     * Coup posé par ses pas plutôt que par un rang : joué au plateau, ou
     * illégal. Aucune touche ne le produit ; nommé ici pour que le panneau
     * traduise tous les gestes au même endroit.
     */
    ENTER_PLAY: 'enter_play',
    DANCE: 'dance',
    DOUBLE: 'double',
    TAKE: 'take',
    PASS: 'pass',
    RESIGN: 'resign',
    CURSOR_BACK: 'cursor_back',
    CURSOR_FORWARD: 'cursor_forward',
    INSERT_BEFORE: 'insert_before',
    INSERT_AFTER: 'insert_after',
    DELETE: 'delete',
    FLIP_SIDE: 'flip_side',
    UNDO: 'undo',
    REDO: 'redo'
});

/**
 * Les sortes d'Action saisies par deux dés. Exporté pour que le panneau n'en
 * tienne pas une seconde liste.
 */
export const DICE_KINDS = new Set(['opening', 'checker', 'dance']);

/** Le niveau d'une résignation : simple, gammon, backgammon. */
const RESIGN_LEVELS = new Set([1, 2, 3]);

/**
 * @typedef {{phase: string, dice: number[], selected: number, candidateCount: number, awaitingCandidates: boolean, tie: boolean, retyped: boolean, resume: KeyState|null}} KeyState
 */

/**
 * @typedef {{kind: string, value?: number, index?: number}} KeyCommand
 */

/**
 * @typedef {{handled: boolean, state: KeyState, commands: KeyCommand[]}} KeyResult
 */

/**
 * L'état initial. `tie` retient une ouverture à égalité, affichée « relance »
 * (elle reste au document sans Move ni Position, fonctionnel.md §1.2).
 *
 * @returns {KeyState}
 */
export function initialKeyState() {
    return {
        phase: PHASE.DICE,
        dice: [0, 0],
        selected: 0,
        candidateCount: 0,
        awaitingCandidates: false,
        tie: false,
        // Le jet en cours a été TAPÉ dans cette saisie, et non chargé d'une
        // Action relue : c'est ce qui sépare, sur la dernière Action, le
        // chiffre qui corrige son jet de celui qui ouvre le suivant (ADR-0051).
        retyped: false,
        // L'état à rendre si la résignation est abandonnée par Échap. Nul
        // partout ailleurs : seule la phase RESIGN en pose un.
        resume: null
    };
}

/**
 * Le dé qu'une touche désigne (positionnel : `Digit3` = `Numpad3` = 3), ou 0.
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
 * +1 / −1 / 0 : `h`/`l` et gauche/droite. Horizontal parce que les deux
 * colonnes du Transcript sont les deux camps ; `j`/`k` restent aux candidats.
 *
 * @param {KeyboardEvent} event
 * @returns {number}
 */
export function cursorDelta(event) {
    if (isBareLetter(event, 'h') || event.key === 'ArrowLeft') return -1;
    if (isBareLetter(event, 'l') || event.key === 'ArrowRight') return 1;
    return 0;
}

/**
 * Les gestes qui mènent le Cursor de l'arrêt `from` à `to` (rangs de
 * [cursorStop]). Le moteur ne connaît que `cursor_back`/`cursor_forward` : un
 * saut est la répétition du pas.
 *
 * @param {number} from
 * @param {number} to
 * @returns {{kind: string}[]}
 */
export function cursorCommands(from, to) {
    const kind = to < from ? COMMAND.CURSOR_BACK : COMMAND.CURSOR_FORWARD;
    return Array.from({ length: Math.abs(to - from) }, () => ({ kind }));
}

/**
 * L'Action `index` suit-elle un trou (l'Incohérence `double_turn`), case où le
 * Cursor s'arrête (ADR-0054) ?
 *
 * @param {any} annotated
 * @param {number} index
 */
export function holeBefore(annotated, index) {
    const info = (annotated?.actions ?? [])[index];
    return !!info?.inconsistencies?.some((/** @type {{kind: string}} */ f) => f.kind === 'double_turn');
}

/**
 * Le rang d'un arrêt du Cursor dans l'ordre de `h`/`l` : chaque Action,
 * précédée de son trou s'il y en a un, puis le bout du document. Un trou
 * traversé est un pas de plus (ADR-0054).
 *
 * @param {any} annotated
 * @param {number} index - l'Action visée, ou celle devant laquelle est le trou
 * @param {boolean} [hole] - le trou devant l'Action plutôt qu'elle
 */
export function cursorStop(annotated, index, hole = false) {
    let rank = index;
    for (let i = 0; i < index; i++) if (holeBefore(annotated, i)) rank++;
    if (!hole && holeBefore(annotated, index)) rank++;
    return rank;
}

/**
 * Le rang où le Cursor est : sur le trou si une insertion y est ouverte.
 *
 * @param {any} annotated
 */
export function currentStop(annotated) {
    const at = annotated?.cursor ?? 0;
    const e = annotated?.entry;
    const onHole = !!e && e.replacing === false && e.at === at && holeBefore(annotated, at);
    return cursorStop(annotated, at, onHole);
}

/**
 * +1 / −1 / 0 : `j`/`k` et bas/haut.
 *
 * @param {KeyboardEvent} event
 * @returns {number}
 */
export function selectionDelta(event) {
    if (isBareLetter(event, 'j') || event.key === 'ArrowDown') return 1;
    if (isBareLetter(event, 'k') || event.key === 'ArrowUp') return -1;
    return 0;
}

/**
 * La touche ne me concerne pas : elle remonte au répartiteur.
 *
 * @param {KeyState} state
 * @returns {KeyResult}
 */
const ignored = (state) => ({ handled: false, state, commands: [] });
/**
 * La touche est à moi, sans rien à faire.
 *
 * @param {KeyState} state
 * @returns {KeyResult}
 */
const swallowed = (state) => ({ handled: true, state, commands: [] });

/**
 * @param {number} n
 * @param {number} max
 */
const clamp = (n, max) => Math.min(Math.max(n, 0), max);

/**
 * @param {KeyState} state - l'état rendu par `initialKeyState` ou par un appel précédent
 * @param {KeyboardEvent} event
 * @param {{expects?: string, replacing?: boolean, editing?: boolean, last?: boolean}} context -
 *   `expects` : la sorte d'Action attendue au Cursor (`annotated.entry.kind`,
 *   à défaut `annotated.next.expects`) ; `replacing` : le Cursor est sur une
 *   Action existante (ADR-0048 décision 1) ; `editing` : le Cursor tient une
 *   cellule sans dé tapé ; `last` : l'Action remplacée est la dernière
 *   (ADR-0051).
 * @returns {KeyResult}
 */
export function pressKey(state, event, { expects = 'checker', replacing = false, editing = false, last = false } = {}) {
    // PHASE.RESIGN capte tout jusqu'à son niveau : les chiffres y sont des
    // niveaux, pas des dés.
    if (state.phase === PHASE.RESIGN) return resignLevel(state, event);

    // Le Cursor se déplace depuis tout état (après RESIGN, modal) et réarme
    // la machine ; le panneau la recharge sur l'Action visée.
    const step = cursorDelta(event);
    if (step !== 0) {
        return {
            handled: true,
            state: initialKeyState(),
            commands: [{ kind: step < 0 ? COMMAND.CURSOR_BACK : COMMAND.CURSOR_FORWARD }]
        };
    }

    // Les quatre gestes de correction, aussi depuis « tout état » (ux.md §3,
    // budgets §4.3). Lus avant les dés : ce ne sont pas des chiffres, et une
    // saisie de jet à moitié tapée n'empêche pas la relecture.
    const edit = editCommand(event);
    if (edit) {
        return { handled: true, state: initialKeyState(), commands: [{ kind: edit }] };
    }

    // `r` ouvre l'attente du niveau depuis tout état ; `Échap` rend l'état
    // d'avant intact (ux.md §3).
    if (isBareLetter(event, 'r')) {
        return { handled: true, ...beginResign(state) };
    }

    // `d` double depuis tout état (une touche : le moteur valide d'abord le
    // candidat en attente). Jamais avalée pour videau impossible — Incohérence
    // marquée, jamais refusée (ADR-0044, fonctionnel.md §1.4).
    if (isBareLetter(event, 'd')) {
        return { handled: true, ...cubeGesture(state, COMMAND.DOUBLE) };
    }

    // `t`/`p` répondent à une offre et CORRIGENT la cellule au Cursor (le
    // moteur écrit au rang de l'Entry, sans supprimer ni insérer). Hors de ces
    // cas la touche reste disponible au répartiteur global (`p` = pips).
    if (expects === 'take' || editing) {
        if (isBareLetter(event, 't')) return { handled: true, ...cubeGesture(state, COMMAND.TAKE) };
        if (isBareLetter(event, 'p')) return { handled: true, ...cubeGesture(state, COMMAND.PASS) };
    }

    // Une réponse à un double n'est pas une saisie de dés : ses touches sont
    // `t`/`p`, traitées juste au-dessus.
    if (!DICE_KINDS.has(expects)) return ignored(state);

    const die = dieOf(event);
    if (die > 0) return enterDie(state, die, expects, replacing, last);

    const delta = selectionDelta(event);
    if (delta !== 0) return moveSelection(state, delta);

    if (event.key === 'Enter') {
        // Entrée valide seule : le dernier coup d'une partie n'a pas de tour
        // suivant pour porter sa validation (ux.md §3).
        if (state.phase !== PHASE.ROLL && state.phase !== PHASE.CANDIDATE) return ignored(state);
        return { handled: true, state: initialKeyState(), commands: [{ kind: COMMAND.VALIDATE }] };
    }

    // Retour arrière efface les dés et reste prise même à vide (ailleurs elle
    // réinitialise le plateau). Échap n'est prise que s'il y a une saisie à
    // abandonner, sinon elle ferme le panneau.
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
 * Le geste de correction d'une touche, ou null : `i` insère devant, `a`
 * derrière, `x`/`Suppr` suppriment, `s` change le camp. Rien n'est refusé ni
 * réparé (ADR-0044) ; la suppression est annulable par `Ctrl+Z`, sans
 * confirmation.
 *
 * @param {KeyboardEvent} event
 * @returns {string|null}
 */
function editCommand(event) {
    if (isBareLetter(event, 'i')) return COMMAND.INSERT_BEFORE;
    if (isBareLetter(event, 'a')) return COMMAND.INSERT_AFTER;
    if (isBareLetter(event, 'x')) return COMMAND.DELETE;
    if (isBareLetter(event, 's')) return COMMAND.FLIP_SIDE;
    if (event.key === 'Delete' && !event.ctrlKey && !event.metaKey && !event.altKey) return COMMAND.DELETE;
    return null;
}

/**
 * Le niveau d'une résignation : `1`/`2`/`3` la crée, `Échap` rend l'état
 * d'avant, tout le reste est avalé. Le camp est celui au trait, que seul le
 * moteur connaît.
 *
 * @param {KeyState} state
 * @param {KeyboardEvent} event
 * @returns {KeyResult}
 */
function resignLevel(state, event) {
    if (event.key === 'Escape') return { handled: true, ...cancelResign(state) };
    const level = dieOf(event);
    if (!RESIGN_LEVELS.has(level)) return swallowed(state);
    return { handled: true, ...resignWithLevel(state, level) };
}

/**
 * Un geste de videau au clic, même corps que `d`/`t`/`p` : pas de `validate`,
 * le moteur valide seul, rien n'est jugé (ADR-0044). C'est le panneau qui
 * éteint un bouton sans objet.
 *
 * @param {KeyState} state
 * @param {string} kind - COMMAND.DOUBLE, COMMAND.TAKE ou COMMAND.PASS
 * @returns {{state: KeyState, commands: KeyCommand[]}}
 */
export function cubeGesture(state, kind) {
    return { state: initialKeyState(), commands: [{ kind }] };
}

/**
 * La résignation annoncée (`r` ou `[R]`) : l'état d'avant est mis de côté
 * pour `Échap`.
 *
 * @param {KeyState} state
 * @returns {{state: KeyState, commands: KeyCommand[]}}
 */
export function beginResign(state) {
    return { state: { ...initialKeyState(), phase: PHASE.RESIGN, resume: state }, commands: [] };
}

/**
 * @param {KeyState} state
 * @param {number} level
 * @returns {{state: KeyState, commands: KeyCommand[]}}
 */
export function resignWithLevel(state, level) {
    if (!RESIGN_LEVELS.has(level)) return { state, commands: [] };
    return { state: initialKeyState(), commands: [{ kind: COMMAND.RESIGN, value: level }] };
}

/**
 * @param {KeyState} state
 * @returns {{state: KeyState, commands: KeyCommand[]}}
 */
export function cancelResign(state) {
    return { state: state.resume ?? initialKeyState(), commands: [] };
}

/**
 * Une entrée du menu contextuel : mener le Cursor à la cellule cliquée, puis
 * la correction (qui agit sur l'Action au Cursor).
 *
 * @param {number} from - le Cursor actuel
 * @param {number} to - la cellule cliquée
 * @param {string} kind - COMMAND.INSERT_BEFORE, INSERT_AFTER, DELETE ou FLIP_SIDE
 */
export function menuCommands(from, to, kind) {
    return [...cursorCommands(from, to), { kind }];
}

/**
 * Un chiffre : premier dé, puis second ; sur une ouverture le second valide.
 *
 * Sur un jet déjà saisi (ADR-0048 décision 1) : en bout de document il valide
 * puis ouvre le jet suivant ; sur une Action relue il recommence sur place —
 * sauf sur la dernière Action une fois son jet retapé (ADR-0051), où il valide.
 *
 * Pas « valider partout » : `validate` n'est pas gardé par `entryDiffers`
 * comme `commitCorrection` ; il réécrirait l'Action et renverrait le Cursor à
 * `doc.Return`, mettant fin à la relecture.
 *
 * @param {KeyState} state
 * @param {number} die
 * @param {string} expects
 * @param {boolean} [replacing]
 * @param {boolean} [last] - l'Action remplacée est la dernière du document
 * @returns {KeyResult}
 */
function enterDie(state, die, expects, replacing = false, last = false) {
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
                // Une ouverture RESSAISIE ne relance pas la partie : elle décide
                // à nouveau qui commence et rend le Cursor là où la relecture
                // l'avait pris, sans enchaîner sur les candidats du premier coup.
                if (replacing && !last) {
                    return { handled: true, state: initialKeyState(), commands };
                }
                // Le gagnant joue les deux dés de l'ouverture sans les
                // ressaisir ; le moteur les repose sur l'Entry suivante.
                const roll = dice[0] >= dice[1] ? [dice[0], dice[1]] : [dice[1], dice[0]];
                return {
                    handled: true,
                    state: { ...initialKeyState(), phase: PHASE.ROLL, dice: roll, awaitingCandidates: true, retyped: true },
                    commands
                };
            }
            return {
                handled: true,
                state: { ...initialKeyState(), phase: PHASE.ROLL, dice, awaitingCandidates: true, retyped: true },
                commands: [{ kind: COMMAND.DIE, value: die }]
            };
        }

        case PHASE.ROLL:
        case PHASE.CANDIDATE: {
            // Relue : recommencer sur place (le moteur repart de zéro, rien
            // n'est validé). Bout de document : valider d'abord.
            const inPlace = replacing && !(last && state.retyped);
            const commands = inPlace ? [{ kind: COMMAND.DIE, value: die }] : [{ kind: COMMAND.VALIDATE }, { kind: COMMAND.DIE, value: die }];
            return {
                handled: true,
                state: { ...initialKeyState(), phase: PHASE.DIE1, dice: [die, 0] },
                commands
            };
        }

        default:
            return ignored(state);
    }
}

/**
 * `j`/`k`, bas/haut et la molette déplacent la sélection, bornée, sans
 * boucler. La phase passe à CANDIDATE.
 *
 * @param {KeyState} state
 * @param {number} delta
 * @returns {KeyResult}
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
 * Choisir un candidat au clic. Le simple clic sélectionne ; le double-clic
 * valide (ADR-0048 décision 11), seul chemin souris pour le dernier coup.
 *
 * @param {KeyState} state
 * @param {number} index
 * @returns {{state: KeyState, commands: KeyCommand[]}}
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
 * La réponse du moteur : combien de coups légaux. Aucun → danse créée sans
 * touche de plus ; sinon le premier est présélectionné.
 *
 * @param {KeyState} state
 * @param {number} count
 * @returns {{state: KeyState, commands: KeyCommand[]}}
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

/**
 * Le jet d'un seul clic sur une case du triangle : les deux dés passent par
 * `enterDie`, exactement comme au clavier. Le clic reste plus lent qu'une
 * frappe (ux.md §4.1) : ADR-0048 R3 demande que tout soit atteignable à la
 * souris, pas au même prix.
 *
 * @param {KeyState} state
 * @param {number} d1 - le dé fort, celui que porte l'étiquette de la case
 * @param {number} d2
 * @param {{expects?: string, replacing?: boolean, last?: boolean}} context
 * @returns {{state: KeyState, commands: KeyCommand[]}}
 */
export function enterDicePair(state, d1, d2, { expects = 'checker', replacing = false, last = false } = {}) {
    const first = enterDie(state, d1, expects, replacing, last);
    const second = enterDie(first.state, d2, expects, replacing, last);
    return { state: second.state, commands: [...first.commands, ...second.commands] };
}

/**
 * Un seul dé au clic (rangée de l'ouverture, où chaque camp donne le sien).
 *
 * @param {KeyState} state
 * @param {number} die
 * @param {{expects?: string, replacing?: boolean, last?: boolean}} context
 * @returns {{state: KeyState, commands: KeyCommand[]}}
 */
export function enterSingleDie(state, die, { expects = 'checker', replacing = false, last = false } = {}) {
    const result = enterDie(state, die, expects, replacing, last);
    return { state: result.state, commands: result.commands };
}
