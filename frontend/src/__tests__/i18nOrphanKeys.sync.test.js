/**
 * i18nOrphanKeys.sync.test.js
 *
 * The mirror image of i18nKeys.sync.test.js: instead of guarding against a key the interface
 * asks for and no locale defines, this guards against a key every locale carries that the
 * interface no longer asks for. Those accumulate silently — a panel gets folded into another
 * (filterLibrary.* and most of searchHistory.* survived a full release cycle after their
 * panels merged into SearchPanel) or a field gets renamed — and nobody notices because an
 * unused translation never breaks anything at runtime. They just add 9x translator effort to
 * features nobody sees.
 *
 * "Used" here is deliberately loose: a key counts as referenced if it appears anywhere in the
 * sources as a quoted string literal, not only inside a `$t('...')`-shaped call. Several
 * components look up keys indirectly — a config array holds `{ labelKey: 'tabbedPanel.search' }`
 * and a different line calls `$t(tab.labelKey)` — so a call-site-only scan (like
 * i18nKeys.sync.test.js uses, safe there because it only needs to find keys, not rule every
 * other key out) would flag most of the app's tab/table/tour definitions as orphaned. A loose
 * "does this exact string occur anywhere" check accepts a little false-negative risk (a key
 * that happens to appear in a comment) in exchange for not drowning in false positives.
 *
 * DYNAMIC_PREFIXES covers the remaining case: keys assembled by concatenation at runtime
 * (`$t('epc.race.verdicts.' + verdict)`), where even the leaf key itself never appears as a
 * literal anywhere — only the prefix does. Discover new ones by grepping the sources for
 * `$t('` (or tr/tMsg/translate/get(t)) followed by a dotted prefix and a `' +`.
 */

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import en from '../i18n/locales/en.json';

const SRC = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

// Prefixes of keys built at runtime by string concatenation, so no literal scan can ever see
// the full leaf key used. Every subkey under one of these is assumed live.
const DYNAMIC_PREFIXES = ['search.filters.', 'search.filterGroups.', 'epc.race.cubeStates.', 'epc.race.verdicts.'];

function sourceFiles(dir) {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) return entry.name === '__tests__' ? [] : sourceFiles(full);
        return /\.(svelte|js)$/.test(entry.name) ? [full] : [];
    });
}

// One blob of every source file's text (i18n/index.js excluded — it's the engine, not a
// consumer, and its generic string handling would make every key look "used").
const allSource = sourceFiles(SRC)
    .filter((f) => !f.includes(path.join('i18n', 'index.js')))
    .map((f) => fs.readFileSync(f, 'utf8'))
    .join('\n');

function isReferenced(key) {
    return allSource.includes(`'${key}'`) || allSource.includes(`"${key}"`);
}

// Flatten a nested dictionary into a flat list of dotted leaf keys.
function leafKeys(dict, prefix = '', out = []) {
    for (const [k, v] of Object.entries(dict)) {
        const nk = prefix ? `${prefix}.${k}` : k;
        if (v && typeof v === 'object') leafKeys(v, nk, out);
        else out.push(nk);
    }
    return out;
}

describe('translation keys are not orphaned', () => {
    const defined = leafKeys(en);

    test('every defined key is referenced somewhere, or covered by a dynamic prefix', () => {
        const orphans = defined.filter((key) => !isReferenced(key) && !DYNAMIC_PREFIXES.some((prefix) => key.startsWith(prefix)));
        expect(orphans).toEqual([]);
    });

    test('every dynamic prefix still matches at least one defined key (catches typos/renames)', () => {
        const unmatched = DYNAMIC_PREFIXES.filter((prefix) => !defined.some((key) => key.startsWith(prefix)));
        expect(unmatched).toEqual([]);
    });
});
