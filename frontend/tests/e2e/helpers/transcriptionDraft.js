/**
 * helpers/transcriptionDraft.js — un brouillon de transcription ouvert, pour
 * les specs de budget (T3.5, ux.md §4).
 *
 * ## Ce que ce faux moteur rend, et ce qu'il ne rend pas
 *
 * Le panneau est un CLIENT du moteur Go (ADR-0045 règle 9) : chaque geste part
 * dans `ApplyTranscriptionGesture` et revient en document annoté entier. Un
 * navigateur n'a pas ce moteur, et le réécrire en JavaScript en ferait une
 * seconde version qui dériverait — c'est exactement ce que l'en-tête de
 * `TranscriptView.svelte` refuse.
 *
 * Ce mock ne le réécrit donc pas. Il **fige** un document annoté — un brouillon
 * de sept points, dix Actions, une position de contact — et le rend tel quel à
 * chaque geste, après avoir noté le geste. Deux faits seulement y sont dérivés,
 * et chacun parce qu'un tableau de budgets serait injouable sans lui : la
 * position du Cursor (`cursor_back` recule d'un, `cursor_forward` avance d'un),
 * parce que les corrections d'ux.md §4.3 se comptent en pas de Cursor et que le
 * panneau réarme sa liste sur l'Action visée ; et l'Action attendue après un
 * double, parce que `t` et `p` ne sont des touches de réponse que devant une
 * offre. Tout le reste — score, Crawford, Incohérences, camp au trait — reste ce
 * qu'il était, et appartient aux tests Go du paquet `transcript`
 * (`gestures_test.go`, `replay_test.go`, `correction_test.go`).
 *
 * Ce que les specs de budget mesurent est donc exactement ceci : **combien de
 * gestes l'application réelle demande pour que la suite de gestes qui consigne
 * le coup parte au moteur**. Que cette suite produise le bon document est déjà
 * tenu, en Go, une Action par Kind.
 *
 * ## Le jet fixé, et pourquoi ce jet-là
 *
 * Les dix-sept coups légaux du 3-1 sont ceux de
 * `src/__tests__/transcriptionFilter.test.js`, dans le même ordre : le douzième,
 * `8/5 8/7`, est celui d'ux.md §4.1 « coup loin dans la liste ». Les deux
 * mesures parlent ainsi du même jet et du même rang.
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
const legalPlays = PLAYS.map(([notation, froms]) => ({
    notation,
    steps: froms.map((from) => ({ from, to: from - 1, hit: false }))
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
 * Installe le faux moteur. À appeler APRÈS `installWailsMock`, avant
 * `page.goto` : le script s'exécute après celui du mock, donc `window.go`
 * existe déjà et n'est que complété.
 *
 * `expects` permet de poser l'Action attendue — `'take'` pour la réponse à un
 * double d'ux.md §4.2, `'opening'` pour l'ouverture — et `cursor` où le Cursor
 * commence, la fin du document par défaut.
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
                return { id: row.id, annotated, can_undo: true, can_redo: true };
            };

            db.ListTranscriptions = () => Promise.resolve([row]);
            db.OpenTranscription = () => Promise.resolve(state());
            db.CreateTranscription = () => Promise.resolve(state());
            db.TranscriptionMAT = () => Promise.resolve('');
            db.ApplyTranscriptionGesture = (_id, gesture) => {
                // Les DEUX seules choses que ce faux moteur dérive.
                //
                // 1. Le Cursor : les budgets de correction d'ux.md §4.3 se
                //    comptent en pas de Cursor (`h`×k … `l`×k), et le panneau
                //    réarme sa liste sur l'Action visée.
                // 2. L'Action attendue après un double : `t` et `p` ne sont
                //    des touches de réponse que lorsque le moteur attend une
                //    réponse (`expects === 'take'`, transcriptionKeys.js), donc
                //    sans cela le tableau §4.2 ne serait pas jouable du tout.
                //
                // Tout le reste du document reste figé, et appartient aux tests
                // Go du paquet transcript.
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
