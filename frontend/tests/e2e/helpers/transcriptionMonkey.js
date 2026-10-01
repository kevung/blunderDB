/**
 * helpers/transcriptionMonkey.js — un faux moteur de transcription À ÉTAT,
 * lent et désordonné, pour le test « singe » (transcription-monkey.spec.js).
 *
 * Le moteur figé de `transcriptionDraft.js` suffit à compter des gestes ; un
 * singe, lui, doit voir le document bouger pour que « le curseur affiché
 * correspond au document » ait un sens. Ce mock tient donc deux brouillons
 * (42 et 43, aux joueurs distincts) dont chaque geste change l'état : curseur,
 * dés tapés, Actions ajoutées, remplacées ou retirées, pile d'annulation. Il
 * n'imite pas les règles du jeu : ce qui est vérifié est le panneau, pas le
 * moteur (les règles appartiennent aux tests Go du paquet `transcript`).
 *
 * Chaque réponse arrive après un délai tiré d'une graine : deux appels
 * concurrents (un geste en vol et une réouverture, `LegalMoves` d'un jet déjà
 * remplacé) reviennent dans le désordre.
 *
 * Instrumentation lue par la spec, sur `window` :
 *   - `__monkeyInflight` : appels du moteur en cours ;
 *   - `__monkeyState(id)` : l'état annoté courant d'un brouillon ;
 *   - `__monkeyViolations` : gestes reçus pour un brouillon alors qu'un AUTRE
 *     était affiché (`__monkeyDisplayed`, posé par la spec depuis le store) ;
 *   - `__monkeyIntent` : le brouillon que l'utilisateur a demandé en dernier
 *     (ouvert, créé ; null après Terminer ou Abandonner).
 */

import { contactPosition } from './transcriptionDraft.js';

const PLAYS = ['8/5 6/5', '13/10 24/23', '24/21 6/5', '13/10 6/5', '8/5 24/23', '13/10 13/12'];

/** Ce que `LegalMoves` rend pour tout jet : quelques coups, dans l'ordre du générateur. */
const legalPlays = PLAYS.map((notation) => ({
    notation,
    steps: notation.split(' ').map((hop) => {
        const [from, to] = hop.split('/').map(Number);
        return { from, to, hit: false };
    })
}));

const rankedMoves = PLAYS.map((notation, index) => ({
    index,
    move: notation,
    equity: 0.2 - index * 0.03,
    analysisDepth: '0-ply',
    analysisEngine: 'gammonNet'
}));

/** Les deux lignes de `ListTranscriptions`, aux joueurs distincts. */
export const DRAFTS = [
    { id: 42, player1: 'Kévin', player2: 'Alice', actions: 10 },
    { id: 43, player1: 'Bruno', player2: 'Chloé', actions: 3 }
];

/**
 * Installe le faux moteur. À appeler après `installWailsMock`, avant `page.goto`.
 *
 * @param {import('@playwright/test').Page} page
 * @param {{seed: number, maxDelay?: number}} opts
 */
export async function installMonkeyEngine(page, opts) {
    await page.addInitScript(
        ({ drafts, plays, moves, seed, maxDelay, position, board }) => {
            // mulberry32 : les délais se rejouent à l'identique d'une graine.
            let s = seed >>> 0;
            const rand = () => {
                s = (s + 0x6d2b79f5) >>> 0;
                let t = s;
                t = Math.imul(t ^ (t >>> 15), t | 1);
                t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
                return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
            };

            window.__monkeyInflight = 0;
            window.__monkeyViolations = [];
            window.__monkeyIntent = null;
            window.__monkeyDisplayed = null;
            window.__monkeyGestures = [];

            const clone = (v) => JSON.parse(JSON.stringify(v));
            const slow = (fn) => {
                window.__monkeyInflight += 1;
                const wait = Math.floor(rand() * maxDelay);
                return new Promise((resolve, reject) =>
                    setTimeout(() => {
                        window.__monkeyInflight -= 1;
                        try {
                            resolve(fn());
                        } catch (err) {
                            reject(err);
                        }
                    }, wait)
                );
            };

            const action = (side, notation) => ({ kind: 'checker', side, notation });
            const docs = new Map();
            let nextId = 50;
            const fresh = (id, player1, player2, count) => {
                const actions = [];
                for (let i = 0; i < count; i += 1) actions.push(action(i % 2, plays[i % plays.length].notation));
                docs.set(id, {
                    id,
                    header: { match_length: 7, player1, player2, jacoby: false, beaver: false },
                    actions,
                    cursor: actions.length,
                    dice: [0, 0],
                    waiting: 'checker',
                    undo: [],
                    redo: []
                });
            };
            for (const d of drafts) fresh(d.id, d.player1, d.player2, d.actions);

            const snapshot = (doc) => clone({ header: doc.header, actions: doc.actions, cursor: doc.cursor, waiting: doc.waiting });
            const restore = (doc, snap) => Object.assign(doc, clone(snap), { dice: [0, 0] });

            const annotate = (doc) => {
                const len = doc.actions.length;
                const actions = doc.actions.map((a, index) => ({
                    index,
                    side: a.side,
                    kind: a.kind,
                    before: { ...clone(position), dice: [3, 1], player_on_roll: a.side },
                    has_position: true,
                    after: clone(board),
                    notation: a.notation,
                    game_index: 0,
                    game_number: 1,
                    score: [0, 0],
                    move_number: index,
                    opens_game: index === 0
                }));
                const nextSide = len ? 1 - doc.actions[len - 1].side : 0;
                const at = doc.cursor;
                const entry =
                    at < len
                        ? { at, kind: 'checker', game_start: at === 0, side: doc.actions[at].side, replacing: true, review: false, selected: true, dice: [...doc.dice] }
                        : { at, kind: doc.waiting, game_start: at === 0, side: nextSide, replacing: false, review: false, selected: false, dice: [...doc.dice] };
                return {
                    id: doc.id,
                    annotated: {
                        document: { format_version: '1', header: clone(doc.header), actions: [], cursor: at },
                        actions,
                        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: len - 1 }],
                        next: {
                            expects: doc.waiting,
                            game_start: len === 0,
                            side: nextSide,
                            position: { ...clone(position), player_on_roll: nextSide },
                            game_number: 1,
                            crawford: false,
                            match_over: false
                        },
                        entry,
                        finished: false,
                        winner: -1,
                        score: [0, 0],
                        cursor: at
                    },
                    can_undo: doc.undo.length > 0,
                    can_redo: doc.redo.length > 0
                };
            };
            window.__monkeyState = (id) => (docs.has(id) ? annotate(docs.get(id)) : null);

            const row = (doc) => ({
                id: doc.id,
                created_at: '2026-09-07T10:00:00Z',
                updated_at: '2026-09-07T10:30:00Z',
                format_version: '1',
                match_id: 0,
                label: `${doc.header.player1} – ${doc.header.player2}`,
                player1: doc.header.player1,
                player2: doc.header.player2,
                match_length: doc.header.match_length,
                action_count: doc.actions.length
            });

            /** Une Action posée au Cursor : remplace, ou ajoute en bout. */
            const record = (doc, a) => {
                if (doc.cursor < doc.actions.length) doc.actions[doc.cursor] = a;
                else doc.actions.push(a);
                doc.cursor = Math.min(doc.cursor + 1, doc.actions.length);
                doc.dice = [0, 0];
            };

            const apply = (doc, g) => {
                const kind = g ? g.Kind : '';
                const len = doc.actions.length;
                const side = doc.cursor < len ? doc.actions[doc.cursor].side : len ? 1 - doc.actions[len - 1].side : 0;
                switch (kind) {
                    case 'enter_die':
                        if (!doc.dice[0]) doc.dice[0] = g.Die;
                        else doc.dice[1] = g.Die;
                        return;
                    case 'clear_dice':
                        doc.dice = [0, 0];
                        return;
                    case 'cursor_back':
                        doc.cursor = Math.max(0, doc.cursor - 1);
                        doc.dice = [0, 0];
                        return;
                    case 'cursor_forward':
                        doc.cursor = Math.min(len, doc.cursor + 1);
                        doc.dice = [0, 0];
                        return;
                    case 'undo':
                        if (doc.undo.length) {
                            doc.redo.push(snapshot(doc));
                            restore(doc, doc.undo.pop());
                        }
                        return;
                    case 'redo':
                        if (doc.redo.length) {
                            doc.undo.push(snapshot(doc));
                            restore(doc, doc.redo.pop());
                        }
                        return;
                }
                doc.undo.push(snapshot(doc));
                doc.redo = [];
                switch (kind) {
                    case 'validate':
                    case 'select_candidate':
                    case 'enter_play':
                    case 'dance':
                        record(doc, action(side, g.Candidate != null ? plays[g.Candidate % plays.length].notation : plays[0].notation));
                        doc.waiting = 'checker';
                        return;
                    case 'double':
                        record(doc, { kind: 'double', side });
                        doc.waiting = 'take';
                        return;
                    case 'take':
                    case 'pass':
                        record(doc, { kind, side });
                        doc.waiting = 'checker';
                        return;
                    case 'resign':
                        record(doc, { kind: 'resign', side });
                        return;
                    case 'delete':
                        if (doc.cursor < len) doc.actions.splice(doc.cursor, 1);
                        else if (len) doc.actions.pop();
                        doc.cursor = Math.min(doc.cursor, doc.actions.length);
                        doc.dice = [0, 0];
                        return;
                    case 'insert_before':
                    case 'insert_after': {
                        const at = kind === 'insert_after' ? Math.min(doc.cursor + 1, len) : doc.cursor;
                        doc.actions.splice(at, 0, action(side, plays[1].notation));
                        doc.cursor = at;
                        doc.dice = [0, 0];
                        return;
                    }
                    case 'flip_side':
                        if (doc.cursor < len) doc.actions[doc.cursor].side = 1 - doc.actions[doc.cursor].side;
                        return;
                    case 'set_header':
                        Object.assign(doc.header, g.Header || {});
                        return;
                    case 'set_length':
                        if (g.HasLength) doc.header.match_length = g.MatchLength;
                        return;
                    case 'swap_players': {
                        const { player1, player2 } = doc.header;
                        doc.header.player1 = player2;
                        doc.header.player2 = player1;
                        return;
                    }
                    default:
                        return;
                }
            };

            const db = window.go.database.Database;
            const app = window.go.gui.App;

            db.ListTranscriptions = () => slow(() => [...docs.values()].map(row).reverse());
            db.OpenTranscription = (id) => {
                window.__monkeyIntent = id;
                return slow(() => annotate(docs.get(id)));
            };
            db.CreateTranscription = (opts) => {
                const id = nextId++;
                window.__monkeyIntent = id;
                fresh(id, 'Nina', 'Omar', 0);
                docs.get(id).header.match_length = opts?.match_length ?? 7;
                return slow(() => annotate(docs.get(id)));
            };
            /** Un brouillon quitte le moteur (Terminer, Abandonner) ; la liste ne reste jamais vide. */
            const release = (id) => {
                docs.delete(id);
                if (window.__monkeyIntent === id) window.__monkeyIntent = null;
                if (!docs.size) fresh(nextId++, 'Kévin', 'Alice', 4);
            };
            db.FinishTranscription = (id) =>
                slow(() => {
                    if (!docs.has(id)) throw new Error(`transcription ${id} is not open`);
                    release(id);
                    return { match_id: 7, replaced: false, to_analyze: 0 };
                });
            db.AbandonTranscription = (id) =>
                slow(() => {
                    release(id);
                    return null;
                });
            db.MatchTranscriptionLosses = () => slow(() => ({ draft_id: 0, imported: false, analyses: 0, comments: 0 }));
            db.ApplyTranscriptionGesture = (id, gesture) => {
                const shown = window.__monkeyDisplayed;
                window.__monkeyGestures.push({ id, kind: gesture?.Kind, shown });
                if (shown != null && shown !== id) window.__monkeyViolations.push({ id, shown, gesture: gesture?.Kind });
                return slow(() => {
                    const doc = docs.get(id);
                    if (!doc) throw new Error(`transcription ${id} is not open`);
                    apply(doc, gesture);
                    return annotate(doc);
                });
            };
            db.TranscriptionMAT = () => slow(() => '');
            db.PendingTranscriptionAnalysis = () => slow(() => null);
            db.SuggestTranscriptionMatFilename = () => slow(() => 'match.mat');
            db.ExportTranscriptionMAT = () => slow(() => null);
            app.OpenExportMatDialog = () => slow(() => '');
            app.LegalMoves = () => slow(() => clone(plays));
            app.EvaluatePositionImmediate = () => slow(() => ({ moves: clone(moves), refused: false }));
        },
        {
            drafts: DRAFTS,
            plays: legalPlays,
            moves: rankedMoves,
            seed: opts.seed,
            maxDelay: opts.maxDelay ?? 60,
            position: contactPosition(),
            board: contactPosition().board
        }
    );
}
