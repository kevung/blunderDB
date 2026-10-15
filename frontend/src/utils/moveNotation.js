// The same checker play is written differently by different engines: XG keeps the hit marker
// and chains one checker's two dice ("13/7*"), gammonNet writes every hop ("13/8 8/7"). A
// rollout names its plays the way it was asked, or in gammonNet's dialect when it chose them;
// the analysis table names them in the dialect of the engine that analysed the position.
// canonicalMove folds the two into the play's shape, as engine.CanonicalMove does in Go; the
// shape drops the hits, and "6/5* 5/1*" and "6/2 2/1*" (4-1) share it while leaving different
// boards. playKey and samePlay add where the play hits, as engine.PlayKey and engine.SamePlay do.

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
 * The points a play hits on, as its "*" markers name them, sorted and without repeats.
 *
 * @param {string} move
 * @returns {string[]}
 */
export function hitPoints(move) {
    const hits = new Set();
    for (let tok of String(move ?? '')
        .trim()
        .split(/\s+/)
        .filter(Boolean)) {
        const paren = tok.indexOf('(');
        if (paren >= 0 && tok.endsWith(')')) tok = tok.slice(0, paren);
        for (const p of tok.split('/').slice(1)) {
            if (p.endsWith('*')) hits.add(p.replace(/\*+$/, ''));
        }
    }
    return [...hits].sort();
}

/**
 * canonicalMove with the play's hit points: equal keys name the same play in any dialect that
 * writes its hits.
 *
 * @param {string} move
 * @returns {string}
 */
export function playKey(move) {
    const canon = canonicalMove(move);
    const hits = hitPoints(move);
    return hits.length ? `${canon} *${hits.join(',')}` : canon;
}

/**
 * Whether a source writes its hits: whether any play of its list carries a "*". A source that
 * never does (gnubg's SGF candidates) says nothing about hits.
 *
 * @param {Array<string | undefined>} moves
 */
export function marksHits(moves) {
    return (moves ?? []).some((m) => String(m ?? '').includes('*'));
}

/**
 * Whether two notations name the same play, given whether each one's source writes its hits:
 * when both do, a missing "*" means no hit; when one does not, the shape decides and the caller
 * must make sure it names a single play.
 *
 * @param {string} a
 * @param {boolean} aMarks
 * @param {string} b
 * @param {boolean} bMarks
 */
export function samePlay(a, aMarks, b, bMarks) {
    if (canonicalMove(a) !== canonicalMove(b)) return false;
    if (!aMarks || !bMarks) return true;
    return hitPoints(a).join(',') === hitPoints(b).join(',');
}
