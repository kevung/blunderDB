/*
 * Géométrie de l'arbre : pure, sans DOM, pour que la mise en page se teste sans monter de
 * composant. Une place est une boîte de deux lignes ; les colonnes sont les tours du moteur,
 * et chaque place se centre sur celles dont elle reçoit le vainqueur.
 */

export const BOX_W = 168;
export const BOX_H = 40;
export const GAP_X = 40;
export const GAP_Y = 12;
const PITCH = BOX_H + GAP_Y;

/**
 * @typedef {{ side: number, section?: string, key: string, loser?: boolean }} Feed
 * @typedef {{ key: string, round: number, feeds?: Feed[] }} LayoutMatch
 * @typedef {{ m: any, x: number, y: number, skeleton?: boolean }} Node
 * @typedef {{ key: string, d: string, loser: boolean }} Edge
 */

/**
 * @param {{ matches: LayoutMatch[] }} section
 * @returns {{ nodes: Node[], edges: Edge[], width: number, height: number }}
 */
export function layoutSection(section) {
    const matches = section.matches || [];
    const rounds = matches.reduce((n, m) => Math.max(n, m.round + 1), 0);
    /** @type {Map<string, Node>} */
    const placed = new Map();
    /** @type {Node[]} */
    const nodes = [];

    for (let r = 0; r < rounds; r++) {
        const col = matches.filter((m) => m.round === r);
        let next = 0;
        const wanted = col.map((m, i) => {
            const ys = (m.feeds || [])
                .filter((f) => !f.section)
                .map((f) => placed.get(f.key))
                .filter((n) => !!n)
                .map((n) => /** @type {Node} */ (n).y);
            return { m, i, y: ys.length ? ys.reduce((a, b) => a + b, 0) / ys.length : null };
        });
        // Une place sans aval placé (premier tour, ou alimentée d'une autre section) prend la
        // première ligne libre ; les autres se centrent sur leurs sources, sans chevaucher.
        wanted.sort((p, q) => (p.y ?? Infinity) - (q.y ?? Infinity) || p.i - q.i);
        for (const w of wanted) {
            let y = w.y ?? next;
            if (y < next) y = next;
            next = y + PITCH;
            const n = { m: w.m, x: r * (BOX_W + GAP_X), y };
            placed.set(w.m.key, n);
            nodes.push(n);
        }
    }

    /** @type {Edge[]} */
    const edges = [];
    for (const n of nodes) {
        for (const f of n.m.feeds || []) {
            const from = f.section ? null : placed.get(f.key);
            if (!from) continue;
            const x1 = from.x + BOX_W;
            const y1 = from.y + BOX_H / 2;
            const x2 = n.x;
            // Le trait arrive à la hauteur du siège (A en haut, B en bas), pas du milieu.
            const y2 = n.y + (f.side === 0 ? BOX_H / 4 : (3 * BOX_H) / 4);
            const mx = x1 + GAP_X / 2;
            edges.push({ key: `${f.key}>${n.m.key}:${f.side}`, d: `M${x1},${y1} H${mx} V${y2} H${x2}`, loser: !!f.loser });
        }
    }

    const width = rounds ? rounds * BOX_W + (rounds - 1) * GAP_X : 0;
    const height = nodes.reduce((h, n) => Math.max(h, n.y + BOX_H), 0);
    return { nodes, edges, width, height };
}

/**
 * Les poules : un tableau de résultats croisés, joueur en ligne contre joueur en colonne.
 *
 * @param {{ matches: any[] }} section
 * @returns {{ players: { id: string, name: string }[], cell: (i: number, j: number) => { m: any, own: number, other: number } | null }}
 */
export function crossTable(section) {
    /** @type {{ id: string, name: string }[]} */
    const players = [];
    const seen = new Set();
    for (const m of section.matches || []) {
        for (const [id, name] of /** @type {[string, string][]} */ ([
            [m.a, m.aName],
            [m.b, m.bName]
        ])) {
            if (id && !seen.has(id)) {
                seen.add(id);
                players.push({ id, name: name || id });
            }
        }
    }
    return {
        players,
        cell(i, j) {
            const a = players[i]?.id;
            const b = players[j]?.id;
            for (const m of section.matches || []) {
                if (m.a === a && m.b === b) return { m, own: m.scoreA || 0, other: m.scoreB || 0 };
                if (m.a === b && m.b === a) return { m, own: m.scoreB || 0, other: m.scoreA || 0 };
            }
            return null;
        }
    };
}
