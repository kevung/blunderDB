/**
 * Le nom d'un type de nombre, en une clé de traduction. Les types (`tp4.last`,
 * `pips.bottom`) portent un point que `$t` lirait comme une clé imbriquée ;
 * cette table est la seule traversée, partagée par la fiche de score et le
 * bilan.
 */
const LABEL_KEYS = Object.freeze({
    'tp2.live': 'training.numbers.tp2Live',
    'tp2.last': 'training.numbers.tp2Last',
    'tp4.live': 'training.numbers.tp4Live',
    'tp4.last': 'training.numbers.tp4Last',
    gv1: 'training.numbers.gv1',
    gv2: 'training.numbers.gv2',
    gv4: 'training.numbers.gv4',
    // Nommés par le JOUEUR, pas par la géométrie (le plateau se retourne) ni
    // la couleur (la palette est modifiable).
    'pips.bottom': 'board.player1',
    'pips.top': 'board.player2',
    // L'EPC des deux camps, nommés comme les comptes de pions : par le
    // JOUEUR. Le même camp ne peut pas s'appeler « bas » ici et « joueur 1 »
    // là — un concept, un terme.
    'epc.bottom': 'board.player1',
    'epc.top': 'board.player2',
    // Décision : un seul nombre par question, la décision que la
    // position porte. Deux types et non un, parce que le journal les compte à
    // part — on peut bien jouer les pions et mal lire le videau.
    'decision.checker': 'training.numbers.decisionChecker',
    'decision.cube': 'training.numbers.decisionCube',
    // Évaluation : chances de gain du joueur au trait, et action de videau —
    // même terme qu'en Décision, compté à part au journal.
    'eval.win': 'training.numbers.evalWin',
    'eval.cube': 'training.numbers.decisionCube'
});

/**
 * @param {string} type
 * @returns {string} la clé de traduction, ou le type brut s'il est inconnu
 * (exercice qu'une version ne sert plus).
 */
export function numberTypeLabelKey(type) {
    return LABEL_KEYS[type] || type;
}

/** Les types que cette version sait nommer. */
export const KNOWN_NUMBER_TYPES = Object.freeze(Object.keys(LABEL_KEYS));
