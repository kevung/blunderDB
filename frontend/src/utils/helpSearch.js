// Search inside the rendered help: no index, the text the reader sees is the text searched.
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
 * Every occurrence of `query` in the text nodes under `root`, in reading order.
 *
 * @param {Element|null|undefined} root
 * @param {string} query
 * @returns {Range[]}
 */
export function findInHelp(root, query) {
    const needle = fold(query.trim());
    if (!root || !needle) return [];
    const ranges = [];
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
        const text = node.nodeValue || '';
        const hay = fold(text);
        if (hay.length !== text.length) continue;
        for (let from = hay.indexOf(needle); from !== -1; from = hay.indexOf(needle, from + needle.length)) {
            const range = document.createRange();
            range.setStart(node, from);
            range.setEnd(node, from + needle.length);
            ranges.push(range);
        }
    }
    return ranges;
}
