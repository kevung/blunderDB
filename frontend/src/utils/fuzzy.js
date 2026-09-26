// fuzzy.js — approximate matching for the command palette: the query's characters appear IN
// ORDER ("trsc" finds "Transcription"), case and accents folded. The score rewards consecutive
// runs, word starts, a match at the start and a short text; only the order between two scores
// for the same query means anything.

/**
 * Lower-case and strip diacritics (NFD, drop combining marks). One code unit in, one out for
 * Latin, Greek and Cyrillic, so indices into the folded text are indices into the original.
 *
 * @param {string} text
 * @returns {string}
 */
export function fold(text) {
    let out = '';
    for (const ch of String(text ?? '')) {
        const base = ch.normalize('NFD').replace(/[̀-ͯ]/g, '');
        const lower = base.toLowerCase();
        // Keep the one-to-one mapping when folding changed the length (a
        // ligature, an unusual case mapping): fall back to the lower-cased
        // original character.
        out += lower.length === ch.length ? lower : ch.toLowerCase().length === ch.length ? ch.toLowerCase() : ch;
    }
    return out;
}

const WORD_SEPARATORS = /[\s\-_/·.,:;'’()«»"!?#]/;

/**
 * @param {string} text
 * @param {number} i
 */
function isWordStart(text, i) {
    return i === 0 || WORD_SEPARATORS.test(text[i - 1]);
}

/**
 * Whether q[qi..] is a subsequence of t[ti..].
 * @param {string} q @param {number} qi @param {string} t @param {number} ti
 */
function fitsAfter(q, qi, t, ti) {
    for (; qi < q.length; qi++) {
        const at = t.indexOf(q[qi], ti);
        if (at === -1) return false;
        ti = at + 1;
    }
    return true;
}

/**
 * Match `query` against `text`: greedy, but a later occurrence at a word start beats an earlier
 * one mid-word ("ta" in "Table d'équité" takes the T of "Table").
 *
 * @param {string} query
 * @param {string} text
 * @returns {{ score: number, indices: number[] } | null} null when the query's
 *   characters do not all appear in order; an empty query matches with score 0.
 */
export function fuzzyMatch(query, text) {
    const q = fold(query).replace(/\s+/g, '');
    const t = fold(text);
    if (q.length === 0) return { score: 0, indices: [] };
    if (q.length > t.length) return null;

    // Exact substring: the strongest signal, scored above any scattered match.
    const at = t.indexOf(q);
    if (at !== -1) {
        const indices = Array.from({ length: q.length }, (_, k) => at + k);
        let score = 100 + q.length * 10;
        if (at === 0) score += 50;
        else if (isWordStart(t, at)) score += 30;
        score -= Math.min(t.length - q.length, 40) * 0.5;
        return { score, indices };
    }

    const indices = [];
    let ti = 0;
    for (let qi = 0; qi < q.length; qi++) {
        const ch = q[qi];
        let found = -1;
        let firstAny = -1;
        for (let j = ti; j < t.length; j++) {
            if (t[j] !== ch) continue;
            if (firstAny === -1) firstAny = j;
            // Consecutive to the previous match, or a word start: take it.
            if ((indices.length > 0 && j === indices[indices.length - 1] + 1) || isWordStart(t, j)) {
                found = j;
                break;
            }
        }
        // A later word start is only worth it if the rest of the query still
        // fits after it ("μτρκβ" in "Μήτρα του κύβου" must not jump to the τ
        // of "του" and lose the ρ).
        if (found !== -1 && found !== firstAny && !fitsAfter(q, qi + 1, t, found + 1)) found = firstAny;
        if (found === -1) found = firstAny;
        if (found === -1) return null;
        indices.push(found);
        ti = found + 1;
    }

    let score = 0;
    for (let k = 0; k < indices.length; k++) {
        const i = indices[k];
        score += 1;
        if (isWordStart(t, i)) score += 8;
        if (k > 0 && i === indices[k - 1] + 1) score += 5;
        if (k > 0) score -= Math.min(i - indices[k - 1] - 1, 10) * 0.5;
    }
    if (indices[0] === 0) score += 10;
    score -= Math.min(t.length - q.length, 40) * 0.25;
    return { score, indices };
}

/**
 * Split `text` into runs for highlighting the matched characters.
 *
 * @param {string} text
 * @param {number[]} indices sorted positions of the matched characters
 * @returns {Array<{ text: string, hit: boolean }>}
 */
export function highlightSegments(text, indices) {
    const hits = new Set(indices);
    /** @type {Array<{ text: string, hit: boolean }>} */
    const segments = [];
    for (let i = 0; i < text.length; i++) {
        const hit = hits.has(i);
        const last = segments[segments.length - 1];
        if (last && last.hit === hit) last.text += text[i];
        else segments.push({ text: text[i], hit });
    }
    return segments;
}
