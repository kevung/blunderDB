/**
 * transcriptionFilter.js — la liste des candidats réduite par les points d'où
 * part le coup (T2.2, ux.md §4.1).
 *
 * Le geste du coup lointain. Descendre au douzième candidat coûte treize
 * touches (2 pour les dés, onze `j`) soit 3,6 s ; cliquer le point de départ du
 * premier pas connu, puis finir au clavier, coûte 2 K + H + P + 2 B + ≤ 2 K,
 * mesuré à 2,6 s. Le gain n'existe QUE parce que le clic ne rend pas la main au
 * classement complet : `j` continue de se déplacer dans la liste réduite.
 *
 * Ce fichier est PUR, et c'est ce qui compte le plus ici : le filtre est un
 * état d'AFFICHAGE. Il ne crée aucune Action, ne touche pas au document, ne
 * rappelle pas le moteur — la liste des candidats est déjà là
 * (`EvaluatePositionImmediate`, 0-ply), et filtrer, c'est en cacher une partie.
 * Un filtre qui aurait fabriqué un geste aurait fait entrer dans le match un
 * choix d'affichage.
 *
 * Un candidat retenu part de CHAQUE point cliqué : deux clics réduisent, ils
 * ne s'additionnent pas en « ou ». C'est ce que veut dire « un second clic sur
 * un autre point la réduit encore » — sans quoi le second clic élargirait.
 */

/** Les points d'où au moins un candidat fait partir un pas. */
export function sourcesOf(candidates) {
    const points = new Set();
    for (const candidate of candidates ?? []) {
        for (const step of candidate?.steps ?? []) {
            if (Number.isInteger(step?.from)) points.add(step.from);
        }
    }
    return points;
}

/**
 * Les candidats dont un pas part de chacun des `points`. Sans point, la liste
 * entière — le filtre vide est l'absence de filtre, et non une liste vide.
 *
 * @param {{steps?: {from: number}[]}[]} candidates
 * @param {number[]} points
 */
export function filterByPoints(candidates, points) {
    const list = candidates ?? [];
    if (!points?.length) return list;
    return list.filter((candidate) => points.every((point) => (candidate?.steps ?? []).some((step) => step?.from === point)));
}

/**
 * Le filtre après un clic sur `point`, et rien d'autre : quel point est
 * cliquable est une question de la liste, pas du plateau.
 *
 * Trois cas, et le troisième est le plus important. Un point déjà dans le
 * filtre en sort — c'est l'annulation à la souris, sur la cible même qu'on
 * vient de viser. Un point d'où part au moins un des candidats encore visibles
 * s'ajoute. Un point d'où ne part AUCUN d'entre eux ne fait rien : le geste
 * reste disponible pour le déplacement libre de pions (T2.4), qui a son propre
 * état, et un filtre vide n'aurait de toute façon rien à montrer.
 *
 * @param {number[]} points - le filtre courant
 * @param {number} point
 * @param {{steps?: {from: number}[]}[]} candidates - la liste COMPLÈTE du jet
 * @returns {number[] | null} le filtre suivant, ou null si le clic ne fait rien
 */
export function nextFilter(points, point, candidates) {
    const current = points ?? [];
    if (current.includes(point)) return current.filter((p) => p !== point);
    const next = [...current, point];
    return filterByPoints(candidates, next).length ? next : null;
}
