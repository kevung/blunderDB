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
 * - `EPC` et `EDIT` : jamais. La position est montrée telle quelle pour que les
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
    if (mode === 'EPC' || mode === 'EDIT') return false;
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
