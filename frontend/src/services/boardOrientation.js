/**
 * Dans quel sens le plateau est dessiné : deux questions distinctes, qui ne
 * coïncident que tant que le camp au trait est en bas (faux en transcription).
 *
 *   • [boardIsMirrored] — COORDONNÉES : faut-il retourner le modèle ? Si oui,
 *     le point `p` se dessine en `25 - p`, et le clic convertit en sens inverse.
 *   • [labelsFlipped] — ÉTIQUETTES : les points sont-ils numérotés depuis le
 *     camp d'en face ? Le jan du bas est celui de la couleur 0 de la position
 *     affichée ; rien n'est retourné.
 *
 * Le code reçoit des points dans deux numérotations :
 *   • ABSOLUE — le damier du modèle (jan du joueur 1 en 1-6), celle des
 *     `CheckerStep` de `domain.LegalMoves` : [screenOfModelPoint] ;
 *   • RELATIVE AU CAMP AU TRAIT — toute notation (`domain.pointLabel`), donc
 *     les flèches : [screenOfNotationPoint].
 * Elles coïncident pour une position de la bibliothèque (normalisée, camp au
 * trait = joueur 1) ; seule la transcription les sépare.
 *
 * Ici plutôt que dans `Board.svelte`, sans test de rendu (two.js dessine sur
 * un canvas absent de jsdom), pour que la règle soit vérifiée.
 */

/**
 * Faut-il retourner la position du modèle pour l'afficher ? (Stockage :
 * toute position est normalisée, camp au trait = 0, en bas.)
 *
 * - `EVAL`, `EDIT` : jamais, pour que la souris parle en coordonnées du
 *   modèle ; le miroir de la recherche s'applique au moment de chercher.
 * - `TRANSCRIBE` : sur demande seulement. La position y est ABSOLUE et le
 *   trait change à chaque demi-coup : retourner sur le trait ferait osciller
 *   le damier. Le joueur 1 reste en bas ; le trait se lit aux dés.
 * - Match : le joueur 1 reste en bas ; on retourne quand
 *   `MatchMovePosition.player_on_roll` dit que le joueur 2 avait le trait.
 * - Sinon : camp au trait en bas (ne retourne qu'une position éditée non
 *   encore normalisée).
 *
 * @param {{mode: string, position: any, matchContext?: any, transcriptionSwap?: boolean}} params
 * @returns {boolean}
 */
export function boardIsMirrored({ mode, position, matchContext, transcriptionSwap }) {
    if (mode === 'EVAL' || mode === 'EDIT') return false;
    if (mode === 'TRANSCRIBE') return transcriptionSwap === true;
    if (matchContext && matchContext.isMatchMode && matchContext.movePositions?.length > 0) {
        const current = matchContext.movePositions[matchContext.currentIndex];
        return !!current && current.player_on_roll === 1;
    }
    return position?.player_on_roll === 1;
}

/**
 * Les points de la position AFFICHÉE sont-ils numérotés depuis le joueur 2
 * (label de `p` = `25 - p`) ? Même règle dans tous les modes.
 *
 * @param {any} displayPosition la position telle qu'elle est dessinée, miroir compris
 * @returns {boolean}
 */
export function labelsFlipped(displayPosition) {
    return displayPosition?.player_on_roll === 1;
}

/** Le symétrique d'un point ; la barre (0/25) et la sortie (-1) comprises. */
function opposite(point) {
    return point >= 0 && point <= 25 ? 25 - point : point;
}

/**
 * La place à l'écran d'un point en numérotation ABSOLUE (pas de
 * `domain.LegalMoves`). Involutive : le clic s'en sert dans l'autre sens.
 *
 * @param {number} point
 * @param {boolean} mirrored la réponse de [boardIsMirrored]
 */
export function screenOfModelPoint(point, mirrored) {
    return mirrored ? opposite(point) : point;
}

/**
 * La place à l'écran d'un point en numérotation RELATIVE AU CAMP AU TRAIT
 * (notation, flèches). Pas la conversion ci-dessus : seule compte la place du
 * joueur au trait, c'est-à-dire [labelsFlipped].
 *
 * @param {number} point
 * @param {boolean} flipped la réponse de [labelsFlipped]
 */
export function screenOfNotationPoint(point, flipped) {
    return flipped ? opposite(point) : point;
}
