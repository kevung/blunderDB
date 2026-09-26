/**
 * helpers/transcriptionDraft.js — un brouillon de transcription ouvert, pour
 * les specs de budget (ux.md §4).
 *
 * Le panneau est un client du moteur Go (ADR-0045 règle 9) ; le réécrire en JS
 * en ferait une seconde version qui dériverait. Ce mock **fige** donc un
 * document annoté (sept points, dix Actions, une position de contact) et le
 * rend tel quel à chaque geste, après l'avoir noté. Trois faits seulement sont
 * dérivés, faute de quoi les budgets seraient injouables : la position du
 * Cursor (§4.3 se compte en pas de Cursor), l'Action attendue après un double
 * (`t`/`p` n'y répondent que devant une offre), et `entry.replacing` (lu du
 * Cursor, il donne son sens à la touche chiffrée, ADR-0048). Le reste
 * appartient aux tests Go du paquet `transcript`.
 *
 * Les specs mesurent donc le nombre de gestes pour que la suite qui consigne
 * le coup parte au moteur, pas la justesse du document.
 *
 * Le jet est 3-1 : son douzième coup, `8/5 8/7`, est le « coup loin dans la
 * liste » d'ux.md §4.1 (ADR-0052). La liste garde les doublons d'ordre que le
 * vrai générateur déduplique ; le coup joué au plateau s'en accommode.
 */

/** Un tableau de 26 points vides. */
function emptyPoints() {
    return Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
}

/** Une position de contact : les deux camps se sont dépassés, rien n'est forcé. */
export function contactPosition(dice = [0, 0]) {
    const pts = emptyPoints();
    const put = (idx, n, color) => (pts[idx] = { checkers: n, color });
    put(24, 2, 0);
    put(13, 4, 0);
    put(11, 2, 0);
    put(8, 3, 0);
    put(6, 3, 0);
    put(4, 1, 0);
    put(1, 2, 1);
    put(12, 4, 1);
    put(14, 2, 1);
    put(17, 3, 1);
    put(19, 3, 1);
    put(21, 1, 1);
    return {
        id: 0,
        board: { points: pts, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice,
        score: [0, 0],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

/**
 * Les dix-sept coups du 3-1, ordre du classement 0-ply. `PLAYS[11]` est le
 * rang douze d'ux.md §4.1.
 */
const PLAYS = [
    ['8/5 6/5', [8, 6]],
    ['13/10 24/23', [13, 24]],
    ['24/21 6/5', [24, 6]],
    ['13/10 6/5', [13, 6]],
    ['8/5 24/23', [8, 24]],
    ['13/10 13/12', [13, 13]],
    ['24/21 24/23', [24, 24]],
    ['8/7 8/5', [8, 8]],
    ['13/12 13/10', [13, 13]],
    ['6/3 6/5', [6, 6]],
    ['24/23 13/10', [24, 13]],
    ['8/5 8/7', [8, 8]],
    ['6/3 8/7', [6, 8]],
    ['13/10 8/7', [13, 8]],
    ['24/21 13/12', [24, 13]],
    ['6/5 13/10', [6, 13]],
    ['8/7 13/10', [8, 13]]
];

/** Ce que `LegalMoves` rend : les pas et la notation, dans l'ordre du générateur. */
const legalPlays = PLAYS.map(([notation]) => ({
    notation,
    steps: notation.split(' ').map((hop) => {
        const [from, to] = hop.split('/').map(Number);
        return { from, to, hit: false };
    })
}));

/** Ce que `EvaluatePositionImmediate` rend : le même ordre, avec une équité. */
const rankedMoves = PLAYS.map(([notation], index) => ({
    index,
    move: notation,
    equity: 0.2 - index * 0.03,
    analysisDepth: '0-ply',
    analysisEngine: 'gammonNet'
}));

/** Le nombre d'Actions du document figé — la distance maximale du Cursor. */
export const ACTION_COUNT = 10;

/** Le document annoté figé, Cursor compris. */
function annotatedAt(cursor) {
    const actions = [];
    for (let i = 0; i < ACTION_COUNT; i += 1) {
        actions.push({
            index: i,
            side: i % 2,
            kind: 'checker',
            before: contactPosition([3, 1]),
            has_position: true,
            after: contactPosition().board,
            notation: PLAYS[i][0],
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: i
        });
    }
    return {
        document: {
            format_version: '1',
            header: { match_length: 7, player1: 'Kévin', player2: 'Alice', jacoby: false, beaver: false },
            actions: [],
            cursor
        },
        actions,
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: ACTION_COUNT - 1 }],
        next: { expects: 'checker', side: 0, position: contactPosition(), game_number: 1, crawford: false, match_over: false },
        finished: false,
        winner: -1,
        score: [0, 0],
        cursor
    };
}

/** La ligne que `ListTranscriptions` rend pour ce brouillon. */
export const draftRow = {
    id: 42,
    created_at: '2026-09-07T10:00:00Z',
    updated_at: '2026-09-07T10:30:00Z',
    format_version: '1',
    match_id: 0,
    label: 'Kévin – Alice',
    player1: 'Kévin',
    player2: 'Alice',
    match_length: 7,
    action_count: ACTION_COUNT
};

/**
 * Installe le faux moteur. À appeler après `installWailsMock` (il complète
 * `window.go`), avant `page.goto`. `expects` pose l'Action attendue (`'take'`,
 * `'opening'`) ; `cursor` où le Cursor commence (fin du document par défaut).
 *
 * @param {import('@playwright/test').Page} page
 * @param {{expects?: string, cursor?: number}} [opts]
 */
export async function installTranscriptionEngine(page, opts = {}) {
    await page.addInitScript(
        ({ doc, row, plays, moves, expects, cursor }) => {
            const db = window.go.database.Database;
            const app = window.go.gui.App;

            let at = cursor;
            let waiting = expects;
            const state = () => {
                const annotated = JSON.parse(JSON.stringify(doc));
                annotated.cursor = at;
                annotated.document.cursor = at;
                annotated.next.expects = waiting;
                // Sur une Action existante la saisie remplace, au bout elle
                // ajoute (ADR-0048 décision 1).
                const info = annotated.actions[at];
                annotated.entry = info
                    ? { at, side: info.side, replacing: true, review: false, selected: true, dice: [3, 1] }
                    : { at, side: annotated.next.side, replacing: false, review: false, selected: false, dice: [0, 0] };
                return { id: row.id, annotated, can_undo: true, can_redo: true };
            };

            db.ListTranscriptions = () => Promise.resolve([row]);
            db.OpenTranscription = () => Promise.resolve(state());
            db.CreateTranscription = () => Promise.resolve(state());
            db.TranscriptionMAT = () => Promise.resolve('');
            db.ApplyTranscriptionGesture = (_id, gesture) => {
                // Dérivés : le Cursor (budgets §4.3) et l'Action attendue
                // après un double (`t`/`p`, §4.2). Le reste reste figé.
                const kind = gesture ? gesture.Kind : '';
                if (kind === 'cursor_back') at = Math.max(0, at - 1);
                if (kind === 'cursor_forward') at = Math.min(doc.actions.length, at + 1);
                if (kind === 'double') waiting = 'take';
                if (kind === 'take' || kind === 'pass') waiting = 'checker';
                return Promise.resolve(state());
            };

            app.LegalMoves = () => Promise.resolve(plays);
            app.EvaluatePositionImmediate = () => Promise.resolve({ moves, refused: false });
        },
        {
            doc: annotatedAt(opts.cursor ?? ACTION_COUNT),
            row: draftRow,
            plays: legalPlays,
            moves: rankedMoves,
            expects: opts.expects ?? 'checker',
            cursor: opts.cursor ?? ACTION_COUNT
        }
    );
}

/**
 * Les gestes reçus par le moteur depuis le chargement de la page, dans l'ordre.
 *
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<Array<{Kind: string, [k: string]: any}>>}
 */
export async function sentGestures(page) {
    return page.evaluate(() => (window.__wailsCalls || []).filter((c) => c.method === 'ApplyTranscriptionGesture').map((c) => c.args[1]));
}

/** Les seuls `Kind` des gestes reçus — la forme qu'une spec lit le plus souvent. */
export async function sentKinds(page) {
    return (await sentGestures(page)).map((g) => g.Kind);
}

/** Vide le journal des appels : le compte d'une spec commence à zéro. */
export async function resetGestures(page) {
    await page.evaluate(() => (window.__wailsCalls = []));
}
