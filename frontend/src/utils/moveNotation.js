// The same checker play is written differently by different engines: XG keeps the hit marker
// and chains one checker's two dice ("13/7*"), gammonNet writes every hop ("13/8 8/7"), gnubg's
// SGF candidates write no hit at all ("6/2 2/1" for a play hitting on 2). canonicalMove folds
// the spellings into the play's shape, as engine.CanonicalMove does in Go, for display; the shape
// drops the hits, and "6/5 5/1" and "6/2* 2/1" (4-1) share it while leaving different boards.
// Whether two notations are the same play is therefore never read from their text:
// resolveBoard replays each on the position, as domain.PlayBoards does in Go, and two notations
// are the same play when they leave the same board.

/**
 * A checker play in a form two engines can be compared on: hit markers dropped, "(n)"
 * expanded, chained hops merged, steps sorted.
 *
 * @param {string} move
 * @returns {string}
 */
export function canonicalMove(move) {
    /** @type {Array<{from: string, to: string}>} */
    const hops = [];
    for (let tok of String(move ?? '')
        .trim()
        .split(/\s+/)
        .filter(Boolean)) {
        tok = tok.replaceAll('*', '');
        let count = 1;
        const paren = tok.indexOf('(');
        if (paren >= 0 && tok.endsWith(')')) {
            const n = Number(tok.slice(paren + 1, -1));
            if (Number.isInteger(n) && n > 0) count = n;
            tok = tok.slice(0, paren);
        }
        // "8/5*/4" is one checker hopping twice, written as one token.
        const points = tok.split('/');
        if (points.length < 2) points.push('');
        for (let i = 0; i < count; i++) {
            for (let j = 0; j + 1 < points.length; j++) hops.push({ from: points[j], to: points[j + 1] });
        }
    }
    for (let merged = true; merged;) {
        merged = false;
        outer: for (let i = 0; i < hops.length; i++) {
            if (!hops[i].to) continue;
            for (let j = 0; j < hops.length; j++) {
                if (i === j || hops[j].from !== hops[i].to) continue;
                hops[i] = { from: hops[i].from, to: hops[j].to };
                hops.splice(j, 1);
                merged = true;
                break outer;
            }
        }
    }
    return hops
        .map((h) => (h.to ? `${h.from}/${h.to}` : h.from))
        .sort()
        .join(' ');
}

/**
 * Whether an analysis engine's dialect, as blunderDB's importers write it, carries hit markers;
 * undefined for an engine it does not name. A stored rollout names its plays as they were
 * picked, possibly from an unmarked analysis. engine.EngineMarksHits in Go.
 *
 * @param {string | undefined} engine
 * @returns {boolean | undefined}
 */
export function engineMarksHits(engine) {
    const e = String(engine ?? '');
    if (e.startsWith('blunderDB rollout')) return false;
    if (e === 'XG' || e.startsWith('gammonNet') || e.startsWith('blunderDB')) return true;
    if (e === 'GNUbg' || e === 'BGBlitz' || e.startsWith('HedgeHog')) return false;
    return undefined;
}

/**
 * Per analysis engine of a candidate list, whether its dialect writes its hits: engineMarksHits,
 * else whether any of that engine's own candidates carries a "*". engine.CandidateMarksHits.
 *
 * @param {Array<{ move?: string, analysisEngine?: string }>} moves
 * @returns {Record<string, boolean>}
 */
export function candidateMarksHits(moves) {
    /** @type {Record<string, boolean>} */
    const out = {};
    for (const m of moves ?? []) {
        const e = m?.analysisEngine ?? '';
        const known = engineMarksHits(e);
        out[e] = known ?? (out[e] || marksHits([m?.move]));
    }
    return out;
}

/**
 * Whether any of the notations carries a "*": what can be told of a source whose dialect is
 * unknown, from the plays it wrote.
 *
 * @param {Array<string | undefined>} moves
 */
export function marksHits(moves) {
    return (moves ?? []).some((m) => String(m ?? '').includes('*'));
}

// Board geometry, as domain.Position: index 0 is White's bar, 25 Black's, 1..24 the points;
// Black (0) moves high to low, White (1) low to high.
const BLACK = 0;
const WHITE = 1;
const NONE = -1;
const OFF = -1;

/** @param {number} mover */
const barOf = (mover) => (mover === BLACK ? 25 : 0);
/** @param {number} mover @param {number} idx — how far idx is from bearing off. */
const progress = (mover, idx) => (idx === OFF ? 0 : idx === barOf(mover) ? 25 : mover === WHITE ? 25 - idx : idx);

/**
 * @typedef {{ points: Array<{ checkers: number, color: number }>, bearoff: number[] }} Board
 * @typedef {{ from: number, to: number, hit: boolean }} Step
 */

/** @param {Board} b @param {number} mover @param {number} dest */
function blocked(b, mover, dest) {
    const pt = b.points[dest];
    return pt.color === 1 - mover && pt.checkers >= 2;
}

/** @param {Board} b @param {number} mover */
function allHome(b, mover) {
    if (b.points[barOf(mover)].checkers > 0) return false;
    for (let i = 1; i <= 24; i++) {
        const pt = b.points[i];
        if (pt.color !== mover || pt.checkers === 0) continue;
        if (mover === BLACK ? i > 6 : i < 19) return false;
    }
    return true;
}

/** @param {Board} b @param {number} mover @param {number} src @param {number} die */
function canBearOff(b, mover, src, die) {
    const need = mover === BLACK ? src : 25 - src;
    if (die === need) return true;
    if (die < need) return false;
    if (mover === BLACK) {
        for (let i = src + 1; i <= 6; i++) if (b.points[i].color === BLACK && b.points[i].checkers > 0) return false;
    } else {
        for (let i = 19; i < src; i++) if (b.points[i].color === WHITE && b.points[i].checkers > 0) return false;
    }
    return true;
}

/**
 * The legal single-die steps from one point, domain's singleMoves narrowed to `from`.
 * @param {Board} b @param {number} mover @param {number} die @param {number} from
 * @returns {Step | null}
 */
function stepFrom(b, mover, die, from) {
    const bar = barOf(mover);
    if (b.points[bar].checkers > 0) {
        if (from !== bar) return null;
        const entry = mover === BLACK ? 25 - die : die;
        if (blocked(b, mover, entry)) return null;
        return { from, to: entry, hit: b.points[entry].color === 1 - mover && b.points[entry].checkers === 1 };
    }
    if (from < 1 || from > 24) return null;
    const pt = b.points[from];
    if (pt.checkers <= 0 || pt.color !== mover) return null;
    const dest = mover === BLACK ? from - die : from + die;
    if (dest >= 1 && dest <= 24) {
        if (blocked(b, mover, dest)) return null;
        return { from, to: dest, hit: b.points[dest].color === 1 - mover && b.points[dest].checkers === 1 };
    }
    return allHome(b, mover) && canBearOff(b, mover, from, die) ? { from, to: OFF, hit: false } : null;
}

/** @param {Board} b @param {number} mover @param {Step} s @returns {Board} */
function applyStep(b, mover, s) {
    const points = b.points.map((p) => ({ checkers: p.checkers, color: p.color }));
    const bearoff = [...b.bearoff];
    points[s.from].checkers--;
    if (points[s.from].checkers === 0) points[s.from].color = NONE;
    if (s.to === OFF) {
        bearoff[mover]++;
        return { points, bearoff };
    }
    if (s.hit) {
        const ob = barOf(1 - mover);
        points[ob].checkers++;
        points[ob].color = 1 - mover;
        points[s.to] = { checkers: 0, color: NONE };
    }
    points[s.to].checkers++;
    points[s.to].color = mover;
    return { points, bearoff };
}

/** @param {Board} b */
const boardKey = (b) => b.points.map((p) => (p.checkers ? `${p.color}:${p.checkers}` : '')).join(',') + '|' + b.bearoff.join(',');

/**
 * A notation's hops as board indices and the points it stars, null when unreadable.
 * @param {string} notation @param {number} mover
 * @returns {{ segs: Array<[number, number]>, starred: Set<number> } | null}
 */
function parsePlay(notation, mover) {
    const tokens = String(notation ?? '')
        .trim()
        .split(/\s+/)
        .filter(Boolean);
    if (!tokens.length) return null;
    /** @type {Array<[number, number]>} */
    const segs = [];
    const starred = new Set();
    /** @param {string} label @param {boolean} from */
    const index = (label, from) => {
        const l = label.toLowerCase();
        if (l === 'bar') return from ? barOf(mover) : null;
        if (l === 'off') return from ? null : OFF;
        if (!/^\d+$/.test(l)) return null;
        const n = Number(l);
        if (n < 1 || n > 25) return null;
        if (n === 25) return from ? barOf(mover) : null;
        return mover === WHITE ? 25 - n : n;
    };
    for (let tok of tokens) {
        let count = 1;
        const paren = tok.indexOf('(');
        if (paren >= 0 && tok.endsWith(')')) {
            count = Number(tok.slice(paren + 1, -1));
            if (!Number.isInteger(count) || count < 1 || count > 4) return null;
            tok = tok.slice(0, paren);
        }
        const labels = tok.split('/');
        if (labels.length < 2) return null;
        const pts = [];
        for (let j = 0; j < labels.length; j++) {
            const star = labels[j].endsWith('*');
            const idx = index(labels[j].replace(/\*+$/, ''), j === 0);
            if (idx === null || (star && (j === 0 || idx < 1 || idx > 24))) return null;
            if (star) starred.add(idx);
            pts.push(idx);
        }
        for (let c = 0; c < count; c++) for (let j = 0; j + 1 < pts.length; j++) segs.push([pts[j], pts[j + 1]]);
    }
    return { segs, starred };
}

/**
 * The boards a notation may leave from position, following the points it writes and the dice:
 * domain.PlayBoards in Go. marks says whether the notation's dialect writes its hits: then the
 * points it stars are exactly those it hits on; otherwise a starred point must be hit and an
 * unstarred one may be. Keys of the distinct boards, empty when it names no play.
 *
 * @param {any} position a domain.Position as the bindings send it
 * @param {string} notation
 * @param {boolean} marks
 * @returns {string[]}
 */
export function playBoards(position, notation, marks) {
    const [d1, d2] = position?.dice ?? [];
    const points = position?.board?.points;
    if (!(d1 >= 1 && d1 <= 6 && d2 >= 1 && d2 <= 6) || !Array.isArray(points) || points.length < 26) return [];
    const mover = position.player_on_roll === WHITE ? WHITE : BLACK;
    const parsed = parsePlay(notation, mover);
    if (!parsed) return [];
    const { segs, starred } = parsed;
    /** @type {Board} */
    const start = {
        points: points.map((/** @type {any} */ p) => ({ checkers: p?.checkers ?? 0, color: p?.checkers ? p.color : NONE })),
        bearoff: [...(position.board.bearoff ?? [0, 0])]
    };
    const out = new Set();

    /** @param {Board} b @param {number[]} dice @param {Array<[number, number]>} rest @param {Set<number>} hits */
    const solve = (b, dice, rest, hits) => {
        if (!rest.length) {
            for (const s of starred) if (!hits.has(s)) return;
            if (marks && hits.size !== starred.size) return;
            out.add(boardKey(b));
            return;
        }
        // Segments are tried in every order: "2/1 6/2" is "6/2 2/1" when the checker on 2 is the
        // one that just arrived.
        rest.forEach(([from, to], i) => {
            if (i > 0 && rest[i - 1][0] === from && rest[i - 1][1] === to) return;
            advance(b, dice, from, to, [...rest.slice(0, i), ...rest.slice(i + 1)], hits);
        });
    };
    /** @param {Board} b @param {number[]} dice @param {number} at @param {number} target @param {Array<[number, number]>} rest @param {Set<number>} hits */
    const advance = (b, dice, at, target, rest, hits) => {
        const tried = new Set();
        dice.forEach((die, i) => {
            if (tried.has(die)) return;
            tried.add(die);
            const step = stepFrom(b, mover, die, at);
            if (!step) return;
            const nb = applyStep(b, mover, step);
            const h = step.hit ? new Set([...hits, step.to]) : hits;
            const left = [...dice.slice(0, i), ...dice.slice(i + 1)];
            if (step.to === target) solve(nb, left, rest, h);
            else if (step.to !== OFF && progress(mover, step.to) > progress(mover, target)) advance(nb, left, step.to, target, rest, h);
        });
    };
    solve(start, d1 === d2 ? [d1, d1, d1, d1] : [d1, d2], segs, new Set());
    return [...out];
}

/**
 * The one board a notation leaves from position (playBoards), null when it names no play or
 * may leave several: refused, never guessed. engine.ResolveBoard in Go.
 *
 * @param {any} position
 * @param {string} notation
 * @param {boolean} marks
 * @returns {string | null}
 */
export function resolveBoard(position, notation, marks) {
    const boards = playBoards(position, notation, marks);
    return boards.length === 1 ? boards[0] : null;
}
