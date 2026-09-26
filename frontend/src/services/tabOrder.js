/**
 * L'ordre des onglets, réconcilié avec l'ordre enregistré (`GetTabOrder`). Un
 * onglet nouveau s'insère auprès de son voisin de gauche par défaut, pas à la
 * fin : sa place fait partie de sa conception (ADR-0040 règle 1).
 */

/**
 * Anciens identifiants d'onglet encore présents dans un état enregistré, et
 * leur nom actuel (`epc` → `eval` ; la mesure EPC garde son nom).
 */
const LEGACY_TAB_IDS = Object.freeze({ epc: 'eval' });

/**
 * L'identifiant d'onglet actuel d'un identifiant relu depuis un état
 * enregistré ; tout autre valeur revient telle quelle.
 * @param {unknown} id
 * @returns {unknown}
 */
export function normalizeTabId(id) {
    return typeof id === 'string' && Object.hasOwn(LEGACY_TAB_IDS, id) ? LEGACY_TAB_IDS[id] : id;
}

/**
 * @template {{id: string}} T
 * @param {readonly T[]} defaults la liste canonique, dans son ordre de conception
 * @param {unknown} order les identifiants enregistrés, dans l'ordre choisi
 * @returns {T[]}
 */
export function applyTabOrder(defaults, order) {
    if (!Array.isArray(order) || order.length === 0) return [...defaults];
    const byId = new Map(defaults.map((tab) => [tab.id, tab]));
    const ordered = order.map((id) => byId.get(normalizeTabId(id))).filter(Boolean);
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
