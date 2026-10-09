// Search inside the rendered help: the index only caches the folded text of the nodes on screen, so the text the reader sees is the text searched.
// Accents and case are ignored ("raccourci" finds "Raccourcis"); folding keeps the string
// length (one code point in, one out), so offsets in the folded text are offsets in the node.

/** @param {string} s */
function fold(s) {
    let out = '';
    for (const ch of s) {
        const base = ch.normalize('NFD').replace(/[̀-ͯ]/g, '');
        out += base.length === 1 ? base.toLowerCase() : ch.toLowerCase();
    }
    return out;
}

/**
 * The folded text of every text node under `root`, computed once: folding dominates the cost
 * of a keystroke, and the text does not change while the same help is on screen. Call
 * `search` for each query; build a new index when the content under `root` is replaced.
 *
 * @param {Element|null|undefined} root
 */
export function createHelpIndex(root) {
    /** @type {{node: Node, hay: string}[]} */
    const entries = [];
    if (root) {
        const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
        for (let node = walker.nextNode(); node; node = walker.nextNode()) {
            const text = node.nodeValue || '';
            const hay = fold(text);
            if (hay.length === text.length) entries.push({ node, hay });
        }
    }
    return {
        /**
         * Occurrences of `query` in reading order, counted without a Range each: `count` is the
         * total, `ranges` the first `limit` ones, `at(i)` builds any one on demand.
         *
         * @param {string} query
         * @param {number} [limit]
         */
        search(query, limit = Infinity) {
            const needle = fold(query.trim());
            /** @type {[number, number][]} entry index, offset */
            const hits = [];
            if (needle) {
                for (let e = 0; e < entries.length; e++) {
                    const hay = entries[e].hay;
                    for (let from = hay.indexOf(needle); from !== -1; from = hay.indexOf(needle, from + needle.length)) {
                        hits.push([e, from]);
                    }
                }
            }
            /** @param {number} i */
            const at = (i) => {
                const [e, from] = hits[i];
                const range = document.createRange();
                range.setStart(entries[e].node, from);
                range.setEnd(entries[e].node, from + needle.length);
                return range;
            };
            const ranges = [];
            for (let i = 0; i < hits.length && i < limit; i++) ranges.push(at(i));
            return { count: hits.length, ranges, at };
        }
    };
}

/**
 * Every occurrence of `query` in the text nodes under `root`, in reading order.
 *
 * @param {Element|null|undefined} root
 * @param {string} query
 * @returns {Range[]}
 */
export function findInHelp(root, query) {
    return createHelpIndex(root).search(query).ranges;
}
