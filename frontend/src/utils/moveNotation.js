// The same checker play is written differently by different engines: XG keeps the hit marker
// and chains one checker's two dice ("13/7*"), gammonNet writes every hop ("13/8 8/7"). A
// rollout names its plays the way it was asked, or in gammonNet's dialect when it chose them;
// the analysis table names them in the dialect of the engine that analysed the position.
// canonicalMove folds the two, as engine.CanonicalMove does in Go.

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
        const slash = tok.indexOf('/');
        const hop = slash < 0 ? { from: tok, to: '' } : { from: tok.slice(0, slash), to: tok.slice(slash + 1) };
        for (let i = 0; i < count; i++) hops.push({ ...hop });
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
