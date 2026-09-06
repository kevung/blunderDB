/**
 * Le nom d'un type de nombre, en une clé de traduction.
 *
 * Les types (`tp4.last`, `gv2`, `pips.bottom`) portent un point : ce sont des
 * identifiants du journal, pas des chemins de traduction, et les faire passer
 * tels quels à `$t` les ferait lire comme des clés imbriquées. Cette table est
 * la seule traversée entre les deux, et elle sert aux deux lecteurs — la fiche
 * de score et le détail du bilan — donc un type ne peut pas s'appeler
 * autrement d'un endroit à l'autre.
 */
const LABEL_KEYS = Object.freeze({
    'tp2.live': 'training.numbers.tp2Live',
    'tp2.last': 'training.numbers.tp2Last',
    'tp4.live': 'training.numbers.tp4Live',
    'tp4.last': 'training.numbers.tp4Last',
    gv1: 'training.numbers.gv1',
    gv2: 'training.numbers.gv2',
    gv4: 'training.numbers.gv4',
    'pips.bottom': 'training.numbers.pipsBottom',
    'pips.top': 'training.numbers.pipsTop'
});

/**
 * @param {string} type
 * @returns {string} la clé de traduction, ou le type lui-même s'il est
 * inconnu — un journal peut porter le type d'un exercice que cette version ne
 * sert plus, et l'afficher brut vaut mieux que l'effacer.
 */
export function numberTypeLabelKey(type) {
    return LABEL_KEYS[type] || type;
}

/** Les types que cette version sait nommer. */
export const KNOWN_NUMBER_TYPES = Object.freeze(Object.keys(LABEL_KEYS));
