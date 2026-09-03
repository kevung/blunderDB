/**
 * helpers/showcase.js
 *
 * Jeu de données « vitrine » pour la capture d'écran documentaire
 * (screenshot.spec.js) : une bibliothèque d'une trentaine de positions, un
 * match de quatre parties et, pour la position affichée, une analyse complète
 * au format que positionService verse dans analysisStore (domain.PositionAnalysis
 * côté Go : checkerAnalysis.moves, doublingCubeAnalysis).
 *
 * Tout est factice mais plausible : équités normalisées (ADR-0019), erreurs
 * positives croissantes, probabilités en pourcentage, score en points « away ».
 */

import { openLibraryMock, libraryMockAfter } from './fixtures.js';

/** Construit un tableau de 26 points vide (0 et 25 sont les barres). */
function emptyPoints() {
    return Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
}

/** Plateau à partir d'une liste { point: [pions, couleur] }. */
function board(layout) {
    const pts = emptyPoints();
    for (const [point, [checkers, color]] of Object.entries(layout)) pts[Number(point)] = { checkers, color };
    return { points: pts, bearoff: [0, 0] };
}

function position(id, layout, { dice = [3, 1], score = [7, 7], cube = { owner: -1, value: 0 }, onRoll = 0, decisionType = 0 } = {}) {
    return {
        id,
        board: board(layout),
        cube,
        dice,
        score,
        player_on_roll: onRoll,
        decision_type: decisionType,
        has_jacoby: 0,
        has_beaver: 0
    };
}

// ── La position affichée ─────────────────────────────────────────────────────

/**
 * Milieu de partie, joueur du bas (couleur 0) au trait avec 6-3, dans un match
 * en 7 points à 4-away / 5-away, videau au centre. Deux pions arrière à
 * ramener, l'adversaire tient son 19 et son 20.
 */
const showcaseLayout = {
    24: [1, 0],
    21: [1, 0],
    13: [4, 0],
    8: [3, 0],
    6: [4, 0],
    5: [2, 0],
    1: [2, 1],
    12: [4, 1],
    17: [3, 1],
    19: [4, 1],
    20: [2, 1]
};

export const showcasePositionId = 4033;

export const showcasePosition = position(showcasePositionId, showcaseLayout, { dice: [6, 3], score: [4, 5] });

/** Décision de videau prise trois coups plus tôt, dans la même partie (pas de double). */
const showcaseCubePositionId = 4030;
const showcaseCubePosition = position(showcaseCubePositionId, { ...showcaseLayout, 24: [2, 0], 21: [0, -1], 13: [3, 0] }, { dice: [0, 0], score: [4, 5], decisionType: 1 });

/**
 * Le coup joué dans le match : pas le meilleur, une petite erreur. Écrit dans
 * l'ordre canonique (pion le plus arriéré en premier, `orderMoveTokens`) —
 * l'affichage réordonne de toute façon, mais la comparaison de ce fichier
 * avec le texte rendu ne le fait pas.
 */
export const showcasePlayedMove = '21/18 13/7';

function move(index, text, equity, error, chances, depth) {
    const [pw, pg, pb, ow, og, ob] = chances;
    return {
        index,
        analysisDepth: depth,
        analysisEngine: 'XG',
        move: text,
        equity,
        equityError: error,
        playerWinChance: pw,
        playerGammonChance: pg,
        playerBackgammonChance: pb,
        opponentWinChance: ow,
        opponentGammonChance: og,
        opponentBackgammonChance: ob
    };
}

export const showcaseCubeAnalysis = {
    analysisDepth: '3-ply',
    analysisEngine: 'XG',
    playerWinChances: 52.84,
    playerGammonChances: 13.92,
    playerBackgammonChances: 0.63,
    opponentWinChances: 47.16,
    opponentGammonChances: 12.47,
    opponentBackgammonChances: 0.51,
    cubelessNoDoubleEquity: 0.064,
    cubelessDoubleEquity: 0.128,
    cubefulNoDoubleEquity: 0.102,
    cubefulNoDoubleError: 0,
    cubefulDoubleTakeEquity: 0.037,
    cubefulDoubleTakeError: 0.065,
    cubefulDoublePassEquity: 1.0,
    cubefulDoublePassError: 0.898,
    bestCubeAction: 'No double, take',
    wrongPassPercentage: 0,
    wrongTakePercentage: 0
};

/** Analyse complète de la position affichée (LoadAnalysis). */
export const showcaseAnalysis = {
    positionId: showcasePositionId,
    xgid: 'XGID=-b----E-C---dD---c-dbB-a-a:0:0:1:63:3:2:0:7:10',
    player1: 'Alice',
    player2: 'Bob',
    analysisType: 'CheckerMove',
    analysisEngineVersion: 'XG 2.19',
    checkerAnalysis: {
        moves: [
            move(0, '24/18 21/18', 0.087, 0, [52.31, 13.42, 0.61, 47.69, 12.08, 0.52], '4-ply'),
            move(1, '21/18 13/7', 0.041, 0.046, [51.2, 14.11, 0.7, 48.8, 12.95, 0.58], '4-ply'),
            move(2, '24/15', 0.019, 0.068, [50.56, 13.05, 0.55, 49.44, 13.3, 0.61], '3-ply'),
            move(3, '13/10 13/7', -0.012, 0.099, [49.87, 14.6, 0.74, 50.13, 14.21, 0.66], '3-ply'),
            move(4, '8/2 5/2', -0.048, 0.135, [48.92, 13.87, 0.66, 51.08, 13.72, 0.63], '3-ply'),
            move(5, '24/18 13/10', -0.071, 0.158, [48.31, 12.96, 0.58, 51.69, 14.05, 0.7], '2-ply'),
            move(6, '21/15 8/5', -0.104, 0.191, [47.54, 12.4, 0.49, 52.46, 14.88, 0.77], '2-ply'),
            move(7, '13/4', -0.166, 0.253, [46.02, 12.11, 0.47, 53.98, 15.42, 0.83], '2-ply'),
            move(8, '8/5 8/2', -0.221, 0.308, [44.71, 11.62, 0.45, 55.29, 16.1, 0.9], '2-ply')
        ]
    },
    doublingCubeAnalysis: null,
    allCubeAnalyses: [],
    playedMove: showcasePlayedMove,
    playedCubeAction: '',
    playedMoves: [showcasePlayedMove],
    playedCubeActions: [],
    creationDate: '2026-03-15T09:12:00Z',
    lastModifiedDate: '2026-03-15T09:12:00Z'
};

/** Analyse de la décision de videau qui précède (chargée par l'onglet Analyse en mode match). */
export const showcaseCubeRecord = {
    ...showcaseAnalysis,
    positionId: showcaseCubePositionId,
    analysisType: 'DoublingCube',
    checkerAnalysis: { moves: [] },
    doublingCubeAnalysis: showcaseCubeAnalysis,
    allCubeAnalyses: [showcaseCubeAnalysis],
    playedMove: '',
    playedMoves: [],
    playedCubeAction: 'No Double',
    playedCubeActions: ['No Double']
};

/** Table id de position → analyse, pour overrideDbMethodByArg(page, 'LoadAnalysis', …). */
export const showcaseAnalyses = {
    [showcasePositionId]: showcaseAnalysis,
    [showcaseCubePositionId]: showcaseCubeRecord
};

// ── Le match ─────────────────────────────────────────────────────────────────

/** Ligne du panneau Match (GetAllMatches / GetMatchByID). */
export const showcaseMatch = {
    id: 11,
    player1_name: 'Alice',
    player2_name: 'Bob',
    match_length: 7,
    match_date: '2026-03-14',
    game_count: 4,
    // Index (0-based) du coup sur lequel la revue s'ouvre : la position vitrine.
    last_visited_position: 32,
    event: 'Spring Open',
    location: 'Paris',
    round: 'Round 3',
    file_path: '/home/alice/matches/spring-open-r3.xg',
    import_date: '2026-03-15',
    pr: 4.12,
    pr2: 6.35,
    mwc_loss: 0.0213
};

/** Parties du match (GetGamesByMatch) : Alice mène 3-2, quatrième partie en cours. */
export const showcaseGames = [
    { game_number: 1, initial_score: [0, 0], winner: 0, points_won: 2 },
    { game_number: 2, initial_score: [2, 0], winner: 1, points_won: 2 },
    { game_number: 3, initial_score: [2, 2], winner: 0, points_won: 1 },
    { game_number: 4, initial_score: [3, 2], winner: -1, points_won: 0 }
];

/**
 * Trente-six coups (GetMatchMovePositions) : 10 + 8 + 6 + 12. Les positions
 * autres que la vitrine ne sont jamais affichées, elles reprennent le même
 * plateau avec d'autres dés. Le coup d'index 32 est la position vitrine (32e coup de pions sur 35, la barre d'état ne compte pas les décisions de videau) ;
 * celui d'index 29 est la décision de videau qui la précède.
 */
export const showcaseMovePositions = (() => {
    const games = [
        { game: 1, moves: 10, score: [7, 7] },
        { game: 2, moves: 8, score: [5, 7] },
        { game: 3, moves: 6, score: [5, 5] },
        { game: 4, moves: 12, score: [4, 5] }
    ];
    const out = [];
    let id = 4001;
    for (const { game, moves, score } of games) {
        for (let m = 1; m <= moves; m++) {
            const index = out.length;
            const onRoll = (m + 1) % 2;
            if (index === 32) {
                out.push({ game_number: game, move_number: m, move_type: 'checker', checker_move: showcasePlayedMove, cube_action: '', position: showcasePosition });
            } else if (index === 29) {
                out.push({ game_number: game, move_number: m, move_type: 'cube', checker_move: '', cube_action: 'No Double', position: showcaseCubePosition });
            } else {
                const dice = [((m * 5) % 6) + 1, ((m * 3) % 6) + 1];
                out.push({
                    game_number: game,
                    move_number: m,
                    move_type: 'checker',
                    checker_move: `${13 - (m % 5)}/${10 - (m % 5)}`,
                    cube_action: '',
                    position: position(id, showcaseLayout, { dice, score, onRoll })
                });
            }
            id++;
        }
    }
    return out;
})();

// ── La bibliothèque ──────────────────────────────────────────────────────────

/** Quelques plateaux de référence, déclinés avec d'autres dés et scores. */
const libraryLayouts = [
    // Position de départ
    { 24: [2, 0], 13: [5, 0], 8: [3, 0], 6: [5, 0], 1: [2, 1], 12: [5, 1], 17: [3, 1], 19: [5, 1] },
    // Course : tout le monde rentre
    { 6: [4, 0], 5: [4, 0], 4: [3, 0], 3: [2, 0], 2: [2, 0], 19: [3, 1], 20: [4, 1], 21: [3, 1], 22: [3, 1], 23: [2, 1] },
    // Blitz : trois points fermés, un pion adverse à la barre
    { 6: [3, 0], 5: [3, 0], 4: [2, 0], 8: [2, 0], 13: [3, 0], 22: [2, 0], 25: [1, 1], 12: [4, 1], 17: [3, 1], 19: [4, 1], 20: [3, 1] },
    // Holding game : ancre sur le 20 (côté adverse) et pions au milieu
    { 20: [2, 0], 13: [5, 0], 8: [3, 0], 6: [5, 0], 1: [2, 1], 12: [4, 1], 17: [4, 1], 19: [3, 1], 21: [2, 1] },
    // La vitrine elle-même
    showcaseLayout
];

const libraryDice = [
    [3, 1],
    [6, 5],
    [4, 4],
    [5, 2],
    [6, 3],
    [2, 1]
];
const libraryScores = [
    [7, 7],
    [5, 3],
    [2, 4],
    [-1, -1],
    [1, 3]
];

/** Trente positions (ids 3001–3030). */
export const showcaseLibrary = Array.from({ length: 30 }, (_, i) =>
    position(3001 + i, libraryLayouts[i % libraryLayouts.length], {
        dice: libraryDice[i % libraryDice.length],
        score: libraryScores[i % libraryScores.length],
        onRoll: i % 2
    })
);

/**
 * Overrides pour installWailsMock : bibliothèque de 30 positions, un match
 * en base, ses parties et ses coups. Compléter après le chargement par
 * overrideDbMethodByArg(page, 'LoadAnalysis', showcaseAnalyses).
 */
export function showcaseMock() {
    return openLibraryMock({
        // Disposition par défaut (panneau en bas), à la hauteur juste
        // suffisante pour les neuf coups sans défilement. Sans valeur, le
        // panneau prend la hauteur de son contenu et le plateau, ajusté avant
        // que la table n'arrive, déborde dessous.
        config: { GetLanguage: 'en', GetPanelPosition: 'bottom', GetPanelHeight: 280, GetPanelWidth: 520 },
        database: {
            // Pagination (0.35.0): the library loads by id window
            // (ListPositionIDs + LoadPositionsByIDs), LoadAllPositions is
            // dead code on the frontend — overriding it alone left the
            // status bar showing the fixtures' 3-position default instead
            // of the 30-position showcase library.
            ...libraryMockAfter(showcaseLibrary),
            GetAllMatches: [showcaseMatch],
            GetMatchByID: showcaseMatch,
            GetGamesByMatch: showcaseGames,
            GetMatchMovePositions: showcaseMovePositions,
            GetPositionProvenance: [showcaseMatch]
        }
    });
}

// ── Panneaux annexes (galerie de captures, screenshot-panels.spec.js) ────────
//
// Un jeu minimal par panneau, dans le même esprit que la vitrine ci-dessus :
// factice mais plausible, avec les noms Alice/Bob/le tournoi Spring Open déjà
// posés par le match. Chaque panneau récupère ses propres données au montage
// (TabbedPanel démonte/remonte son enfant à chaque bascule d'onglet), donc une
// constante par méthode suffit — aucune de ces captures ne modifie l'état.

/** Panneau Tournois (GetAllTournaments) : le tournoi du match vitrine. */
export const showcaseTournaments = [
    { id: 21, name: 'Spring Open', matchCount: 1, date: '2026-03-14', location: 'Paris', pr: 4.12, mwc_loss: 0.0213, ref_player: 'Alice' },
    { id: 22, name: 'Winter Cup', matchCount: 3, date: '2026-01-18', location: 'Lyon', pr: 5.4, mwc_loss: 0.0388, ref_player: 'Alice' }
];

// Toutes ces chaînes libres (noms, descriptions, commentaires) sont en
// anglais comme le reste de l'interface capturée (config.GetLanguage: 'en',
// même choix que la vitrine ci-dessus) — pas de mélange de langues visible
// dans une capture qui illustre par ailleurs une doc en français.

/** Panneau Collections (GetAllCollections). */
export const showcaseCollections = [
    { id: 1, name: 'Blitzes to review', positionCount: 12, description: 'Three-point-or-better attacks, to replay', updatedAt: '2026-03-10T09:00:00Z' },
    { id: 2, name: 'Marginal takes', positionCount: 7, description: '', updatedAt: '2026-02-20T18:30:00Z' },
    { id: 3, name: 'Backgames', positionCount: 5, description: 'Reference positions from The Theory of Backgammon', updatedAt: '2026-01-05T11:15:00Z' }
];

/** Panneau Anki (GetAllAnkiDecks). */
export const showcaseAnkiDecks = [
    { id: 1, name: 'Blitz', description: 'Attacking plays deck', cardCount: 42, newCount: 5, dueCount: 12 },
    { id: 2, name: "This month's errors", description: '', cardCount: 18, newCount: 0, dueCount: 3 },
    { id: 3, name: 'Cube decisions', description: 'Cash / too good', cardCount: 26, newCount: 2, dueCount: 0 }
];

/** Panneau Commentaires (GetCommentsByPosition) : deux échanges sur la position vitrine. */
export const showcaseComments = [
    { id: 1, positionId: showcasePositionId, text: 'The double looks premature: contact is still too loose.', createdAt: '2026-03-15T09:30:00Z', modifiedAt: '2026-03-15T09:30:00Z' },
    { id: 2, positionId: showcasePositionId, text: 'Agreed, but 21/18 leaves a blot exposed to the next 6-3.', createdAt: '2026-03-15T10:05:00Z', modifiedAt: '2026-03-15T10:05:00Z' }
];

/**
 * Panneau Stats, onglets Dashboard/Erreurs (ComputeStats) : même forme que le
 * contrat Go (PascalCase), reprise du jeu de test statsStore (Alice/Bob).
 */
export const showcaseStatsResult = {
    Totals: { NumPositions: 486, NumMatches: 9, NumTournaments: 2, NumDecisions: 486 },
    PRGlobal: 4.71,
    PRChecker: 4.28,
    PRCube: 6.02,
    PRRolling: { 5: 3.9, 10: 4.3, 50: 4.6, 100: 4.7, 250: 4.71 },
    MWCGlobal: 0.048,
    MWCChecker: 0.034,
    MWCCube: 0.041,
    MWCRolling: { 5: 0.03, 10: 0.036, 50: 0.045, 100: 0.047, 250: 0.048 },
    MWCAvailable: true,
    PerTournament: [
        { ID: 21, Name: 'Spring Open', PR: 4.12, MWC: 0.0213, NumDecisions: 210 },
        { ID: 22, Name: 'Winter Cup', PR: 5.4, MWC: 0.0388, NumDecisions: 276 }
    ],
    PerMatch: [
        { ID: 11, Date: '2026-03-14T00:00:00Z', PlayerName: 'Alice', PR: 4.12, MWC: 0.0213, NumDecisions: 70 },
        { ID: 12, Date: '2026-01-18T00:00:00Z', PlayerName: 'Alice', PR: 5.4, MWC: 0.0388, NumDecisions: 92 }
    ],
    CubeActionBreakdown: [
        { Action: 'NoDouble', PR: 3.9, MWC: 0.019, NumDecisions: 140, BlunderCount: 6 },
        { Action: 'DoubleTake', PR: 6.8, MWC: 0.052, NumDecisions: 96, BlunderCount: 15 },
        { Action: 'DoublePass', PR: 2.6, MWC: 0.014, NumDecisions: 48, BlunderCount: 3 }
    ],
    ErrorHistogram: [
        { MinMP: 0, MaxMP: 5, Count: 240 },
        { MinMP: 5, MaxMP: 10, Count: 128 },
        { MinMP: 10, MaxMP: 25, Count: 64 },
        { MinMP: 25, MaxMP: 50, Count: 32 },
        { MinMP: 50, MaxMP: 100, Count: 15 },
        { MinMP: 100, MaxMP: -1, Count: 7 }
    ],
    TopBlunders: [
        {
            PositionID: showcasePositionId,
            MatchID: 11,
            TournamentID: 21,
            ErrorMP: 460,
            MWCLoss: 0.065,
            Description: '',
            DecisionType: 0,
            MatchDate: '2026-03-14T00:00:00Z',
            PlayerNames: 'Alice vs Bob'
        }
    ]
};

/** Onglet Joueurs (GetPlayerTable). */
export const showcasePlayerTable = [
    { name: 'Alice', matches: 9, wins: 6, losses: 3, decisions: 486, pr: 4.71, luck_known: true, luck_rate_mp: 8, luck_rolls: 512 },
    { name: 'Bob', matches: 5, wins: 2, losses: 3, decisions: 260, pr: 6.15, luck_known: true, luck_rate_mp: -4, luck_rolls: 288 },
    { name: 'Charlie', matches: 3, wins: 1, losses: 2, decisions: 140, pr: 5.02, luck_known: false, luck_rate_mp: 0, luck_rolls: 0 }
];

/**
 * Panneau Eval (App.EvaluatePositionImmediate) sur la position vitrine :
 * mêmes six coups classés que showcaseAnalysis, forme gammonNet ({moves,
 * cube}) plutôt que XG (champ analysisEngine ignoré, showProvenance=false).
 */
export const showcaseEvalResult = {
    moves: [
        move(0, '24/18 21/18', 0.082, 0, [52.1, 13.6, 0.6, 47.9, 12.2, 0.53], '2-ply'),
        move(1, '21/18 13/7', 0.038, 0.044, [51.05, 14.2, 0.71, 48.95, 13.1, 0.59], '2-ply'),
        move(2, '24/15', 0.021, 0.061, [50.6, 13.1, 0.56, 49.4, 13.4, 0.62], '2-ply'),
        move(3, '13/10 13/7', -0.01, 0.092, [49.9, 14.7, 0.75, 50.1, 14.3, 0.67], '2-ply'),
        move(4, '8/2 5/2', -0.045, 0.127, [48.98, 13.9, 0.67, 51.02, 13.8, 0.64], '2-ply')
    ],
    cube: null
};

/**
 * Overrides supplémentaires pour la galerie de captures : la vitrine
 * (showcaseMock) plus les panneaux Tournois, Collections, Anki, Commentaires
 * et Stats. Le panneau Eval (App.EvaluatePositionImmediate) est du namespace
 * `app` (gui.App), pas `database` — installWailsMock accepte les deux.
 */
export function showcaseGalleryMock() {
    const base = showcaseMock();
    return {
        ...base,
        app: {
            ...base.app,
            EvaluatePositionImmediate: showcaseEvalResult
        },
        database: {
            ...base.database,
            GetAllTournaments: showcaseTournaments,
            GetTournamentMatches: [showcaseMatch],
            GetAllCollections: showcaseCollections,
            GetAllAnkiDecks: showcaseAnkiDecks,
            GetCommentsByPosition: showcaseComments,
            ComputeStats: showcaseStatsResult,
            GetPlayerTable: showcasePlayerTable,
            // StatsFilterBar treats an empty GetAllPlayerNames as "database
            // empty" and replaces the whole filter row with an import hint.
            GetAllPlayerNames: ['Alice', 'Bob', 'Charlie'],
            GetStatsDateRange: { DateFrom: '2026-01-18', DateTo: '2026-03-14' }
        }
    };
}
