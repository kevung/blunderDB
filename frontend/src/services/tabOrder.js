/**
 * L'ordre des onglets, réconcilié avec celui que l'utilisateur a enregistré.
 *
 * L'ordre est persisté (`GetTabOrder`), et une version qui livre un onglet
 * neuf trouve donc une liste qui l'ignore. L'ajouter à la FIN était le premier
 * réflexe — il n'était pas perdu — mais un onglet conçu pour être entre Eval et
 * Anki qui apparaît après Métadonnées n'est pas à sa place : sa place fait
 * partie de ce qu'on a décidé (ADR-0040 règle 1). Il s'insère donc auprès de
 * son voisin de gauche par défaut, sans toucher à l'ordre choisi.
 */

/**
 * @template {{id: string}} T
 * @param {readonly T[]} defaults la liste canonique, dans son ordre de conception
 * @param {unknown} order les identifiants enregistrés, dans l'ordre choisi
 * @returns {T[]}
 */
export function applyTabOrder(defaults, order) {
    if (!Array.isArray(order) || order.length === 0) return [...defaults];
    const byId = new Map(defaults.map((tab) => [tab.id, tab]));
    const ordered = order.map((id) => byId.get(id)).filter(Boolean);
    defaults.forEach((tab, index) => {
        if (ordered.some((t) => t.id === tab.id)) return;
        // Après le plus proche voisin de gauche encore présent ; à la fin s'il
        // n'en reste aucun (l'utilisateur a tout réarrangé devant lui).
        let at = ordered.length;
        for (let i = index - 1; i >= 0; i--) {
            const position = ordered.findIndex((t) => t.id === defaults[i].id);
            if (position !== -1) {
                at = position + 1;
                break;
            }
        }
        ordered.splice(at, 0, tab);
    });
    return ordered;
}
