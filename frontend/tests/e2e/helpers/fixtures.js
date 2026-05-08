/**
 * helpers/fixtures.js
 *
 * Données factices pour les specs E2E Playwright.
 * Structures minimales respectant les types attendus par les composants Svelte.
 */

// ── Position factice ─────────────────────────────────────────────────────────

/** Construit un tableau de 26 points vide. */
function emptyPoints() {
    return Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
}

/**
 * Position A — mid-game, checkers répartis sur le plateau.
 * EPC attendu : valeur non-nulle (checkers en home board uniquement).
 */
export const positionA = {
    id: 1001,
    board: {
        points: (() => {
            const pts = emptyPoints();
            // Checkers en home board (points 1–6) pour joueur 0
            pts[1] = { checkers: 2, color: 0 };
            pts[2] = { checkers: 2, color: 0 };
            pts[3] = { checkers: 3, color: 0 };
            pts[4] = { checkers: 3, color: 0 };
            pts[5] = { checkers: 3, color: 0 };
            pts[6] = { checkers: 2, color: 0 };
            // Adversaire (couleur 1) hors home board → EPC N/A côté adversaire
            pts[19] = { checkers: 5, color: 1 };
            pts[24] = { checkers: 2, color: 1 };
            return pts;
        })(),
        bearoff: [0, 0],
    },
    cube: { owner: -1, value: 0 },
    dice: [3, 1],
    score: [5, 5],
    player_on_roll: 0,
    decision_type: 0,
    has_jacoby: 0,
    has_beaver: 0,
};

/**
 * Position B — position différente avec des pips différents.
 * Permet de vérifier que l'EPC change quand on change de position.
 */
export const positionB = {
    id: 1002,
    board: {
        points: (() => {
            const pts = emptyPoints();
            // Distribution différente pour obtenir un EPC différent de positionA
            pts[1] = { checkers: 3, color: 0 };
            pts[2] = { checkers: 3, color: 0 };
            pts[3] = { checkers: 3, color: 0 };
            pts[4] = { checkers: 3, color: 0 };
            pts[5] = { checkers: 3, color: 0 };
            pts[19] = { checkers: 5, color: 1 };
            pts[24] = { checkers: 2, color: 1 };
            return pts;
        })(),
        bearoff: [0, 0],
    },
    cube: { owner: -1, value: 0 },
    dice: [4, 2],
    score: [5, 5],
    player_on_roll: 0,
    decision_type: 0,
    has_jacoby: 0,
    has_beaver: 0,
};

// ── Match factice ─────────────────────────────────────────────────────────────

export const matchesSample = [
    {
        id: 1,
        date: '2026-01-15',
        player1: 'Alice',
        player2: 'Bob',
        score: '5pt',
        result: 1,
        source: 'test',
    },
    {
        id: 2,
        date: '2026-01-20',
        player1: 'Charlie',
        player2: 'Alice',
        score: '7pt',
        result: 2,
        source: 'test',
    },
];

// ── Résultat stats factice ────────────────────────────────────────────────────

export const statsResult = {
    prGlobal: 3.14,
    prChecker: 2.5,
    prCube: 0.64,
    totals: {
        numDecisions: 42,
        numCheckerDecisions: 35,
        numCubeDecisions: 7,
    },
    byDecisionType: [],
    byDice: [],
};

// ── Résultat EPC factice ──────────────────────────────────────────────────────

/** Retour simulé de ComputeEPCFromPosition pour positionA. */
export const epcResultA = {
    bottomEPC: {
        epc: 66.47,
        pipCount: 61,
        wastage: 5.47,
        meanRolls: 11.074,
        stdDev: 2.341,
    },
    topEPC: null,
    error: null,
};

/** Retour simulé de ComputeEPCFromPosition pour positionB (différent de A). */
export const epcResultB = {
    bottomEPC: {
        epc: 72.34,
        pipCount: 67,
        wastage: 5.34,
        meanRolls: 12.056,
        stdDev: 2.512,
    },
    topEPC: null,
    error: null,
};
