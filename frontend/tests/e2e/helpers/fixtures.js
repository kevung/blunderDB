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
        bearoff: [0, 0]
    },
    cube: { owner: -1, value: 0 },
    dice: [3, 1],
    score: [5, 5],
    player_on_roll: 0,
    decision_type: 0,
    has_jacoby: 0,
    has_beaver: 0
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
        bearoff: [0, 0]
    },
    cube: { owner: -1, value: 0 },
    dice: [4, 2],
    score: [5, 5],
    player_on_roll: 0,
    decision_type: 0,
    has_jacoby: 0,
    has_beaver: 0
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
        source: 'test'
    },
    {
        id: 2,
        date: '2026-01-20',
        player1: 'Charlie',
        player2: 'Alice',
        score: '7pt',
        result: 2,
        source: 'test'
    }
];

// ── Résultat stats factice ────────────────────────────────────────────────────

export const statsResult = {
    prGlobal: 3.14,
    prChecker: 2.5,
    prCube: 0.64,
    totals: {
        numDecisions: 42,
        numCheckerDecisions: 35,
        numCubeDecisions: 7
    },
    byDecisionType: [],
    byDice: []
};

// ── Résultat EPC factice ──────────────────────────────────────────────────────

/**
 * Retour simulé de ComputeEPCFromPosition pour positionA, au contrat typé de
 * engine/race (ADR-0009) : { bottom, top, race? }. L'adversaire est hors de
 * son jan → pas d'EPC côté top, pas de zone course (pas de bearoff pur).
 */
export const epcResultA = {
    bottom: {
        all_in_home: true,
        checker_count: 15,
        epc: {
            epc: 66.47,
            pipCount: 61,
            wastage: 5.47,
            meanRolls: 11.074,
            stdDev: 2.341
        }
    },
    top: { all_in_home: false, checker_count: 15 }
};

/** Retour simulé de ComputeEPCFromPosition pour positionB (différent de A). */
export const epcResultB = {
    bottom: {
        all_in_home: true,
        checker_count: 15,
        epc: {
            epc: 72.34,
            pipCount: 67,
            wastage: 5.34,
            meanRolls: 12.056,
            stdDev: 2.512
        }
    },
    top: { all_in_home: false, checker_count: 15 }
};

// ── Bibliothèque ouverte ─────────────────────────────────────────────────────

/**
 * Position C — troisième position de la bibliothèque (jeu de contact).
 * Sert de dernier élément : l'app s'ouvre dessus après restauration de session.
 */
export const positionC = {
    ...positionA,
    id: 1003,
    board: {
        points: (() => {
            const pts = emptyPoints();
            pts[6] = { checkers: 5, color: 0 };
            pts[8] = { checkers: 3, color: 0 };
            pts[13] = { checkers: 5, color: 0 };
            pts[24] = { checkers: 2, color: 0 };
            pts[1] = { checkers: 2, color: 1 };
            pts[12] = { checkers: 5, color: 1 };
            pts[17] = { checkers: 3, color: 1 };
            pts[19] = { checkers: 5, color: 1 };
            return pts;
        })(),
        bearoff: [0, 0]
    },
    dice: [6, 5]
};

/** Les trois positions de la bibliothèque factice, dans l'ordre du backend. */
export const libraryPositions = [positionA, positionB, positionC];

/** Chemin de la base factice mémorisée dans la config. */
export const libraryDbPath = '/tmp/e2e-library.db';

/**
 * Overrides pour installWailsMock qui font démarrer l'app sur une base
 * ouverte : le chemin mémorisé existe, la version du schéma concorde, aucune
 * session à restaurer → loadAllPositions affiche libraryPositions.
 *
 * @param {{ database?: object, app?: object, config?: object, runtime?: object }} [extra]
 */
export function openLibraryMock(extra = {}) {
    return {
        config: { GetLastDatabasePath: libraryDbPath, ...(extra.config || {}) },
        app: { PathExists: true, ...(extra.app || {}) },
        runtime: { ...(extra.runtime || {}) },
        database: {
            IsProtectedCopyPath: false,
            OpenDatabase: null,
            CheckDatabaseVersion: '2.15.0',
            GetDatabaseVersion: '2.15.0',
            IsReadOnly: false,
            LoadSessionState: null,
            LoadAllPositions: libraryPositions,
            GetAllMatches: [],
            GetAllTournaments: [],
            GetAllCollections: [],
            LoadSearchHistory: [],
            LoadFilters: [],
            ...(extra.database || {})
        }
    };
}

// ── Match factice de deux parties ────────────────────────────────────────────

/** Ligne du panneau Match (forme renvoyée par GetAllMatches). */
export const matchSample = {
    id: 7,
    player1_name: 'Alice',
    player2_name: 'Bob',
    match_length: 7,
    match_date: '2026-01-15',
    game_count: 2,
    last_visited_position: 0,
    event: '',
    location: '',
    round: '',
    file_path: '/tmp/alice-bob.xg',
    import_date: '2026-01-16'
};

/** Parties du match (GetGamesByMatch). */
export const matchGames = [
    { game_number: 1, initial_score: [0, 0], winner: 0, points_won: 1 },
    { game_number: 2, initial_score: [1, 0], winner: -1, points_won: 0 }
];

/**
 * Six coups joués (GetMatchMovePositions) : trois par partie, tous des coups de
 * pions pour que j/k et les flèches les parcourent un à un.
 */
export const matchMovePositions = [1, 2].flatMap((game) =>
    [1, 2, 3].map((move) => ({
        game_number: game,
        move_number: move,
        move_type: 'checker',
        checker_move: `${13 - move}/${10 - move}`,
        cube_action: '',
        position: { ...positionC, id: 2000 + game * 10 + move, dice: [move + 1, move], score: [game - 1, 0] }
    }))
);

// ── Position collée / importée ───────────────────────────────────────────────

/** Texte XGID tel que copié depuis XG (position de départ, 3-1 à jouer). */
export const xgidSample = 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:7:10';

/** Position que ParsePositionText renvoie pour xgidSample (pas encore en base : id 0). */
export const pastedPosition = { ...positionC, id: 0, dice: [3, 1], score: [0, 0] };

/** Retour de ParsePositionText : position seule, sans analyse ni commentaire. */
export const parsedPositionResult = { position: pastedPosition, analysis: null, comment: '' };
