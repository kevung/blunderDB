/**
 * Les Incohérences d'une Action, nommées par la règle qu'elles enfreignent
 * (`transcript.InconsistencyRule`) : « videau impossible » recouvre six raisons,
 * et la marque dit laquelle. Une Incohérence sans règle (document d'un binaire
 * antérieur) retombe sur le nom de son espèce.
 */

// Clés écrites en toutes lettres : une clé construite à l'exécution échappe à la
// garde des traductions orphelines.
/** @type {Record<string, string>} */
export const RULE_KEY = {
    dance_playable: 'transcript.rule.dance_playable',
    unreachable_board: 'transcript.rule.unreachable_board',
    unknown_kind: 'transcript.rule.unknown_kind',
    double_turn: 'transcript.rule.double_turn',
    double_at_opening: 'transcript.rule.double_at_opening',
    double_pending: 'transcript.rule.double_pending',
    cube_not_held: 'transcript.rule.cube_not_held',
    crawford_cube: 'transcript.rule.crawford_cube',
    cube_capped: 'transcript.rule.cube_capped',
    no_double_to_answer: 'transcript.rule.no_double_to_answer',
    game_over: 'transcript.rule.game_over',
    match_won: 'transcript.rule.match_won',
    opening_double: 'transcript.rule.opening_double',
    opening_winner: 'transcript.rule.opening_winner',
    dice_unused: 'transcript.rule.dice_unused',
    unrecorded: 'transcript.rule.unrecorded',
    money_score: 'transcript.rule.money_score',
    negative_score: 'transcript.rule.negative_score',
    score_differs: 'transcript.rule.score_differs',
    roll_out_of_order: 'transcript.rule.roll_out_of_order',
    action_out_of_order: 'transcript.rule.action_out_of_order'
};

/** @type {Record<string, string>} */
export const KIND_KEY = {
    illegal_move: 'transcript.inconsistency.illegal_move',
    double_turn: 'transcript.inconsistency.double_turn',
    impossible_cube: 'transcript.inconsistency.impossible_cube',
    past_end: 'transcript.inconsistency.past_end',
    inconsistent_dice: 'transcript.inconsistency.inconsistent_dice',
    unrecorded_move: 'transcript.inconsistency.unrecorded_move',
    score_mismatch: 'transcript.inconsistency.score_mismatch',
    timecode_backwards: 'transcript.inconsistency.timecode_backwards'
};

/**
 * La raison courte d'une Incohérence, avec l'élément en cause : le joueur
 * nommé, les dés, la valeur du videau, le score dérivé.
 *
 * @param {{kind: string, rule?: string, args?: Record<string, number>}} flaw
 * @param {(key: string, params?: Record<string, any>) => string} t
 * @param {(side: number) => string} playerName
 * @returns {string}
 */
export function flawText(flaw, t, playerName) {
    const key = flaw.rule ? RULE_KEY[flaw.rule] : undefined;
    if (!key) return KIND_KEY[flaw.kind] ? t(KIND_KEY[flaw.kind]) : flaw.kind;
    /** @type {Record<string, any>} */
    const params = { ...(flaw.args ?? {}) };
    if (typeof params.player === 'number') params.player = playerName(params.player - 1);
    return t(key, params);
}
