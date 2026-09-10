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
 * est classée en 0-ply et le PREMIER est présélectionné.
 *
 * Et **la touche chiffrée commence un jet là où le Cursor est** (ADR-0048
 * décision 1). En bout de document il n'y a rien sous le Cursor : elle VALIDE le
 * candidat sélectionné avant d'ouvrir le jet suivant, si bien que le meilleur
 * coup joué coûte les deux dés et rien de plus, la validation étant portée par
 * la première touche du tour d'après. Sur une Action relue il y a quelque chose :
 * elle en RECOMMENCE le jet, sur place, ce qui est la raison même d'y être
 * revenu. Ce que cette règle exclut a sa sortie : le dernier coup d'une partie,
 * qui n'a pas de tour suivant pour porter sa validation, se valide par Entrée.
 *
 * C'est UN SEUL sens, et c'est ce qui la sépare de l'arbitrage du 2026-09-07
 * qu'elle renverse : celui-ci distinguait « jet corrigeable » de « candidat
 * choisi », donc un HISTORIQUE invisible — « avez-vous touché la liste ? » —,
 * quand le discriminant est ici un objet DESSINÉ, la cellule encadrée du
 * Transcript où l'utilisateur s'est rendu une frappe plus tôt. Un état que l'on
 * voit n'est pas un mode, et un mode est ce que le budget KLM ne savait pas
 * compter (ux.md §1, la réserve sur M).
 *
 * Le discriminant est `replacing`, que le moteur pose sur `annotated.entry` et
 * que le panneau passe en contexte. Pourquoi pas ici : cette machine ne connaît
 * pas le document (fonctionnel.md §1.2), elle en reçoit les deux faits dont elle
 * a besoin.
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
 * Les quatre ont un second déclencheur, à la souris : la rangée `[D] [T] [P]
 * [R]` du panneau et le clic sur le videau dessiné (T2.5). Il ne double pas la
 * règle — [cubeGesture], [beginResign] et [resignWithLevel] sont le corps même
 * de ces touches, appelé par elles.
 *
 * Ce que ces touches ne font PAS : juger. Doubler sans posséder le videau, en
 * partie Crawford ou au-delà du plafond reste transcriptible — c'est une
 * Incohérence du Replay, marquée et jamais refusée (ADR-0044, fonctionnel.md
 * §1.4). Seules `t` et `p` demandent une offre en face, faute de quoi elles ne
 * répondent à rien.
 *
 * # La correction
 *
 * Les gestes de relecture — `h`/`l` pour le Cursor, `i`/`a` pour insérer, `x` et
 * `Suppr` pour supprimer, `s` pour changer de camp — partent de TOUT état, y
 * compris d'un jet à moitié tapé : c'est la ligne « tout » d'ux.md §3, et c'est
 * l'usage qui gouverne, puisque transcrire une vidéo, c'est se reprendre. Chacun
 * coûte une touche, ce qui donne les budgets d'ux.md §4.3 : coup oublié
 * `h`×k `i` jet `l`×k, coup en double `h`×k `x` `l`×k, camp faux `h`×k `s` `l`×k.
 *
 * `Ctrl+Z` et `Ctrl+Maj+Z` sont dans le même tableau mais pas dans [pressKey] :
 * une combinaison Ctrl est toujours globale (`isAlwaysGlobal`), elle est donc
 * liée par le répartiteur, qui pose le geste dans un store que le panneau lit.
 * COMMAND.UNDO et COMMAND.REDO restent nommés ici parce que ce sont des gestes
 * de la Transcription comme les autres, et que le panneau les traduit au même
 * endroit que les autres.
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
    /** Jet saisi, premier candidat présélectionné. */
    ROLL: 'roll',
    /**
     * Jet saisi, liste touchée. Depuis ADR-0048 les deux états répondent la
     * même chose au chiffre — le Cursor décide, pas l'historique — et cette
     * phase ne sert plus qu'à dire que la sélection vient de l'utilisateur, ce
     * dont le panneau se sert pour ne pas la réécrire sous ses doigts.
     */
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
    /**
     * Le coup posé PAR SES PAS et non par un rang dans la liste : celui joué au
     * plateau, dont les dés se déduisent (T2.3), et celui qui n'est dans aucune
     * liste parce qu'il est illégal (T2.4). Aucune touche ne le produit — il
     * naît d'un geste de souris ou d'une notation tapée — mais il est nommé ici
     * avec les autres, pour que le panneau les traduise tous au même endroit.
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
 * Les sortes d'Action dont la saisie passe par deux dés. Exporté parce que le
 * panneau y lit s'il doit offrir sa cible souris (T2.1) : deux listes des mêmes
 * sortes finiraient par diverger.
 */
export const DICE_KINDS = new Set(['opening', 'checker', 'dance']);

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
 * @param {{expects?: string, replacing?: boolean}} context - `expects` est
 *   `annotated.next.expects`, la sorte d'Action que le document attend ;
 *   `replacing` est `annotated.entry.replacing`, vrai quand le Cursor est sur
 *   une Action existante que la saisie remplacerait (ADR-0048 décision 1).
 * @returns {{handled: boolean, state: object, commands: {kind: string, value?: number, index?: number}[]}}
 */
export function pressKey(state, event, { expects = 'checker', replacing = false } = {}) {
    // La résignation capte tout tant que son niveau n'est pas donné : ses
    // chiffres SONT des niveaux et non des dés, et rien d'autre ne doit passer
    // entre `r` et la touche qui la termine.
    if (state.phase === PHASE.RESIGN) return resignLevel(state, event);

    // Le Cursor se déplace depuis tout état de saisie ordinaire (ux.md §3,
    // ligne « tout ») : c'est le geste de la relecture, et il doit marcher
    // pendant une saisie de dés comme devant une réponse au videau. Il rend la
    // machine à son état initial — le panneau la réarme sur l'Action visée.
    //
    // Il est lu APRÈS la phase de résignation : celle-ci est un état modal
    // bref, entre `r` et le chiffre du niveau, où rien d'autre ne doit passer.
    const step = cursorDelta(event);
    if (step !== 0) {
        return {
            handled: true,
            state: initialKeyState(),
            commands: [{ kind: step < 0 ? COMMAND.CURSOR_BACK : COMMAND.CURSOR_FORWARD }]
        };
    }

    // Les quatre gestes de correction, eux aussi depuis « tout état » (ux.md §3).
    // Ils portent les budgets d'ux.md §4.3 : le coup oublié coûte `i` et rien de
    // plus avant le jet, le coup en double `x`, le camp faux `s` — une touche
    // chacun, quelle que soit la distance déjà parcourue par le Cursor.
    //
    // Ils sont lus AVANT les dés : `i`, `a`, `x` et `s` ne sont pas des chiffres,
    // et une saisie de jet à moitié tapée n'est pas une raison de refuser la
    // relecture, pas plus qu'elle ne l'est pour `h`/`l`. La machine repart à zéro
    // et le panneau la réarme sur ce que le moteur a rendu.
    const edit = editCommand(event);
    if (edit) {
        return { handled: true, state: initialKeyState(), commands: [{ kind: edit }] };
    }

    // `r` ouvre l'attente du niveau depuis n'importe quel état, et l'état
    // d'avant est mis de côté : `Échap` le rend intact, la résignation
    // « annule sans effet » (ux.md §3, fiche T1.5).
    if (isBareLetter(event, 'r')) {
        return { handled: true, ...beginResign(state) };
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
        return { handled: true, ...cubeGesture(state, COMMAND.DOUBLE) };
    }

    // `t`/`p` ne répondent qu'à une offre. Ce n'est pas un refus : sans double
    // qui précède il n'y a pas de réponse à transcrire, et la touche reste
    // disponible pour le répartiteur global (ux.md §3 : « réponse attendue |
    // t / p »).
    if (expects === 'take') {
        if (isBareLetter(event, 't')) return { handled: true, ...cubeGesture(state, COMMAND.TAKE) };
        if (isBareLetter(event, 'p')) return { handled: true, ...cubeGesture(state, COMMAND.PASS) };
    }

    // Une réponse à un double n'est pas une saisie de dés : ses touches sont
    // `t`/`p`, traitées juste au-dessus.
    if (!DICE_KINDS.has(expects)) return ignored(state);

    const die = dieOf(event);
    if (die > 0) return enterDie(state, die, expects, replacing);

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
 * Le geste de correction qu'une touche désigne, ou null : `i` insère devant,
 * `a` derrière, `x` et `Suppr` suppriment, `s` change le camp de l'Action au
 * Cursor (ux.md §3, lignes « tout »).
 *
 * Aucun de ces gestes ne juge quoi que ce soit. Insérer une Action du même camp
 * que sa voisine crée un double trait, supprimer en crée un autre, changer un
 * camp peut rendre illégaux les coups qui suivent : ce sont des Incohérences que
 * le Replay MARQUE, et rien ici ne les refuse ni ne les répare (ADR-0044,
 * fonctionnel.md §1.4). C'est aussi pourquoi la suppression ne demande pas de
 * confirmation : elle est annulable par `Ctrl+Z`, et une boîte de dialogue par
 * pion effacé rendrait la relecture impraticable.
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
    if (event.key === 'Escape') return { handled: true, ...cancelResign(state) };
    const level = dieOf(event);
    if (!RESIGN_LEVELS.has(level)) return swallowed(state);
    return { handled: true, ...resignWithLevel(state, level) };
}

/**
 * Un geste de videau donné au CLIC : le bouton `[D]`, `[T]` ou `[P]` de la
 * rangée, et le clic sur le videau dessiné sur le plateau (T2.5).
 *
 * C'est le corps même des touches `d`, `t` et `p` — elles l'appellent — et non
 * un second chemin qui leur ressemblerait. Ce que la touche ne fait pas, le
 * bouton ne le fait donc pas non plus : aucun `validate` n'est émis devant le
 * geste, parce que le moteur valide de lui-même le candidat resté en attente
 * (`transcript.cubeGesture`), et rien n'est jugé — doubler sans posséder le
 * videau reste transcriptible, l'Incohérence est marquée (ADR-0044).
 *
 * Ce qui distingue le bouton de la touche est AILLEURS, dans le panneau : un
 * bouton s'éteint là où son geste ne répond à rien, quand la touche, elle, ne
 * refuse jamais.
 *
 * @param {object} state
 * @param {string} kind - COMMAND.DOUBLE, COMMAND.TAKE ou COMMAND.PASS
 * @returns {{state: object, commands: {kind: string}[]}}
 */
export function cubeGesture(state, kind) {
    return { state: initialKeyState(), commands: [{ kind }] };
}

/**
 * La résignation annoncée, son niveau attendu : ce que fait `r`, et ce que fait
 * le bouton `[R]` de la rangée.
 *
 * Les deux mènent au même état modal, celui qui met l'état d'avant de côté pour
 * qu'`Échap` — ou le bouton « Annuler » qui le double — le rende intact. Le
 * niveau se donne ensuite d'un chiffre ou d'un clic, en un geste : la
 * résignation coûte deux gestes à la souris comme au clavier (ux.md §4.2).
 */
export function beginResign(state) {
    return { state: { ...initialKeyState(), phase: PHASE.RESIGN, resume: state }, commands: [] };
}

/** Le niveau donné : `1`/`2`/`3` au clavier, un des trois boutons à la souris. */
export function resignWithLevel(state, level) {
    if (!RESIGN_LEVELS.has(level)) return { state, commands: [] };
    return { state: initialKeyState(), commands: [{ kind: COMMAND.RESIGN, value: level }] };
}

/** La résignation abandonnée : `Échap`, ou le bouton qui le double. */
export function cancelResign(state) {
    return { state: state.resume ?? initialKeyState(), commands: [] };
}

/**
 * Les gestes d'une entrée du menu contextuel du Transcript (T2.5) : le Cursor
 * mené jusqu'à la cellule cliquée, puis la correction elle-même.
 *
 * C'est mot pour mot ce que coûte la relecture au clavier — `h`×k puis `x`, la
 * ligne « coup en double » d'ux.md §4.3 — et c'est voulu : `i`, `a`, `x` et `s`
 * agissent sur l'Action AU CURSOR, si bien qu'un clic droit sur une cellule
 * lointaine doit d'abord y amener le Cursor. Le moteur ne connaît que le pas
 * (`cursor_back`/`cursor_forward`), et un saut est la répétition du pas : rien
 * n'est ajouté côté Go pour la souris.
 *
 * @param {number} from - le Cursor actuel
 * @param {number} to - la cellule cliquée
 * @param {string} kind - COMMAND.INSERT_BEFORE, INSERT_AFTER, DELETE ou FLIP_SIDE
 */
export function menuCommands(from, to, kind) {
    return [...cursorCommands(from, to), { kind }];
}

/**
 * Un chiffre. Premier dé, puis second ; sur une ouverture le second dé valide
 * aussitôt.
 *
 * Sur un jet déjà saisi, il commence un jet LÀ OÙ LE CURSOR EST (ADR-0048
 * décision 1) : en bout de document il valide le candidat puis ouvre le jet
 * suivant — la règle qui ramène le meilleur coup joué à deux touches —, sur une
 * Action relue (`replacing`) il recommence le jet sur place, ce qui est la
 * raison même d'y être revenu.
 *
 * Pourquoi pas « valider partout » : `GestureValidate` passe par `validate`,
 * qui — contrairement à `commitCorrection` — n'est pas gardé par `entryDiffers`,
 * réécrit l'Action à l'identique et rend le Cursor à `doc.Return`. Un chiffre
 * égaré en relecture aurait donc mis fin à la relecture et renvoyé le Cursor en
 * bout de document, quand la règle retenue le garde local.
 */
function enterDie(state, die, expects, replacing = false) {
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
        case PHASE.CANDIDATE: {
            // Le Cursor décide, pas l'historique de la liste : les deux phases
            // répondent la même chose.
            //
            // Sur une Action relue, le chiffre recommence le jet sur place. Le
            // moteur fait de même — `enter_die` sur une Entry aux deux dés
            // pleins repart de zéro — et rien n'est validé, donc le Cursor ne
            // bouge pas.
            //
            // En bout de document, il valide d'abord : la validation du tour
            // est portée par la première touche du tour d'après.
            const commands = replacing ? [{ kind: COMMAND.DIE, value: die }] : [{ kind: COMMAND.VALIDATE }, { kind: COMMAND.DIE, value: die }];
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
 * `j`/`k`, bas/haut et la molette déplacent la sélection. Le déplacement est
 * borné, il ne boucle pas.
 *
 * La phase passe à CANDIDATE, ce qui ne change plus rien au sens du chiffre
 * depuis ADR-0048 — c'est le Cursor qui le décide — et sert seulement à dire
 * que la sélection vient de l'utilisateur.
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
 * qui tomberait juste.
 *
 * Le simple clic SÉLECTIONNE — les flèches du plateau suivent, et c'est là que
 * l'on reconnaît le coup vu sur la vidéo. C'est le double-clic qui valide
 * (ADR-0048 décision 11), seul chemin souris pour le dernier coup d'une partie,
 * qui n'a pas de jet suivant pour porter sa validation.
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
 * passe à l'autre camp. Au moins un : le premier est présélectionné et ses
 * flèches partent sur le plateau.
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

/**
 * Le jet donné d'un seul geste — le clic sur une case du triangle (T2.1).
 *
 * C'est LA MÊME chose que les deux frappes, et pas une seconde règle : les deux
 * dés passent par `enterDie`, dans l'ordre, exactement comme deux touches. Ce
 * qu'un chiffre fait depuis « jet corrigeable » (recommencer) ou depuis
 * « candidat choisi » (valider, puis ouvrir le jet suivant) est donc fait aussi
 * par le clic, sans que rien de tout cela soit réécrit ici.
 *
 * Un clic n'est PAS moins cher qu'une frappe : ux.md §4.1 le mesure à 1,18 s
 * contre 0,56 s pour les deux touches. Le triangle est une entrée pour la
 * souris, jamais un remplacement du clavier — et R3 (ADR-0048) demande que
 * chaque geste soit ATTEIGNABLE à la souris, jamais qu'il y coûte le même temps.
 *
 * @param {object} state
 * @param {number} d1 - le dé fort, celui que porte l'étiquette de la case
 * @param {number} d2
 * @param {{expects?: string, replacing?: boolean}} context
 * @returns {{state: object, commands: {kind: string, value?: number, index?: number}[]}}
 */
export function enterDicePair(state, d1, d2, { expects = 'checker', replacing = false } = {}) {
    const first = enterDie(state, d1, expects, replacing);
    const second = enterDie(first.state, d2, expects, replacing);
    return { state: second.state, commands: [...first.commands, ...second.commands] };
}

/**
 * Un seul dé, au clic : la rangée des six dés de l'ouverture, où chaque camp
 * donne le sien (fonctionnel.md §1.2). Même chemin qu'une touche chiffrée.
 *
 * @param {object} state
 * @param {number} die
 * @param {{expects?: string, replacing?: boolean}} context
 */
export function enterSingleDie(state, die, { expects = 'checker', replacing = false } = {}) {
    const result = enterDie(state, die, expects, replacing);
    return { state: result.state, commands: result.commands };
}
