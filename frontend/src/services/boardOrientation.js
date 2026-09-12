/**
 * Dans quel sens le plateau est dessiné.
 *
 * Deux questions vivent ici, et elles n'ont pas la même réponse — c'est tout
 * l'objet du fichier. Elles ont longtemps coïncidé, parce que le camp au trait
 * était toujours en bas ; le jour où il a cessé de l'être (la transcription),
 * les trois endroits qui les avaient confondues se sont mis à répondre chacun
 * une chose différente : les pions dessinés d'un côté, le point cliqué d'un
 * autre, les flèches du coup sélectionné d'un troisième.
 *
 *   • [boardIsMirrored] — faut-il RETOURNER la position du modèle ? C'est une
 *     question de COORDONNÉES : quand la réponse est oui, le point `p` du
 *     modèle se dessine à la place `25 - p` de l'écran, et le clic fait la
 *     conversion inverse.
 *
 *   • [labelsFlipped] — les points sont-ils NUMÉROTÉS depuis le camp d'en face ?
 *     C'est une question d'ÉTIQUETTES : les points se comptent depuis le jan du
 *     camp au trait, et le jan dessiné en bas à droite est celui de la couleur 0
 *     de la position AFFICHÉE. Rien n'est retourné pour autant.
 *
 * Et il faut bien les deux, parce que le code reçoit des points dans DEUX
 * numérotations, jamais dans une seule :
 *
 *   • ABSOLUE — celle du damier du modèle, le jan du joueur 1 en 1-6 et celui du
 *     joueur 2 en 19-24. C'est celle des `CheckerStep` que `domain.LegalMoves`
 *     rend (`singleMoves` parcourt `Board.Points[src]`), donc celle du coup en
 *     cours et du filtre par point. Elle se convertit avec [screenOfModelPoint].
 *
 *   • RELATIVE AU CAMP AU TRAIT — celle de toute notation de backgammon, « 24 »
 *     nommant toujours les pions arriérés de celui qui joue (`domain.pointLabel`
 *     écrit 25-idx pour White). C'est celle des flèches, lues d'une notation.
 *     Elle se convertit avec [screenOfNotationPoint].
 *
 * Les deux numérotations coïncident quand le camp au trait est le joueur 1, et
 * les deux conversions coïncidaient quand le camp au trait était toujours en
 * bas : rien ne signalait qu'on employait l'une pour l'autre. Une position de la
 * bibliothèque étant enregistrée normalisée — camp au trait = joueur 1 —, aucun
 * des deux cas de divergence ne se rencontre hors transcription.
 *
 * Le composant n'a pas de test de rendu — two.js dessine sur un canvas que jsdom
 * n'implémente pas —, et c'est la raison pour laquelle ces deux réponses sont
 * ici plutôt que dans `Board.svelte` : elles sont la règle, elles se vérifient,
 * et une règle qui ne se vérifie nulle part dérive (même leçon que
 * services/boardRedraw.js).
 */

/**
 * Faut-il retourner la position du modèle pour l'afficher ?
 *
 * Rappel du STOCKAGE : toute position enregistrée est normalisée, camp au trait
 * en `player_on_roll = 0` — donc en bas, sans miroir.
 *
 * - `EVAL` et `EDIT` : jamais. La position est montrée telle quelle pour que les
 *   coordonnées de la souris soient celles du modèle ; le miroir de la recherche
 *   est appliqué ailleurs, au moment de chercher.
 * - `TRANSCRIBE` : seulement si l'utilisateur le demande. Le moteur y rend une
 *   position ABSOLUE — couleur 0 = joueur 1 — dont `player_on_roll` change à
 *   chaque demi-coup ; retourner sur le trait y ferait osciller le damier d'un
 *   tour sur l'autre, les pions de celui qu'on vient de regarder passant en
 *   haut. Le joueur 1 reste donc en bas et le trait se lit aux dés, qui changent
 *   de côté.
 * - Mode match : le joueur 1 reste en bas de même. La position est normalisée,
 *   et c'est `MatchMovePosition.player_on_roll` qui dit qui avait le trait :
 *   quand c'était le joueur 2, on retourne pour le renvoyer en haut.
 * - Sinon : le camp au trait en bas, ce qui ne retourne qu'une position éditée
 *   n'ayant pas encore été normalisée par l'enregistrement.
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
 * Les points de la position AFFICHÉE sont-ils numérotés depuis le camp du
 * joueur 2 (le label du point `p` vaut alors `25 - p`) ?
 *
 * Toujours la même règle, quel que soit le mode : le jan du bas appartient à la
 * couleur 0, et la numérotation part du jan du camp au trait.
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
 * La place, à l'écran, d'un point donné dans la numérotation ABSOLUE du modèle —
 * un pas de `domain.LegalMoves`, un point cliqué rendu au modèle.
 *
 * La conversion est celle du miroir, et elle est involutive : le clic s'en sert
 * dans l'autre sens, de l'écran vers le modèle.
 *
 * @param {number} point
 * @param {boolean} mirrored la réponse de [boardIsMirrored]
 */
export function screenOfModelPoint(point, mirrored) {
    return mirrored ? opposite(point) : point;
}

/**
 * La place, à l'écran, d'un point donné dans la numérotation RELATIVE AU CAMP AU
 * TRAIT — celle que porte toute notation, donc les flèches du coup choisi.
 *
 * Ce n'est PAS la conversion ci-dessus : la notation est déjà écrite du point de
 * vue de celui qui joue, et la seule question est de savoir si celui-ci est
 * dessiné en bas (numérotation de l'écran) ou en haut (numérotation inversée) —
 * c'est-à-dire exactement ce que dit [labelsFlipped], et exactement ce que les
 * étiquettes affichent sous chaque flèche.
 *
 * @param {number} point
 * @param {boolean} flipped la réponse de [labelsFlipped]
 */
export function screenOfNotationPoint(point, flipped) {
    return flipped ? opposite(point) : point;
}
