/**
 * format.js — date, number and plural formatting tied to the interface's
 * active language (#213), replacing the scattered hand-rolled formatters and
 * hardcoded locale tags that used to disagree with whatever language the user
 * actually selected (`'fr-FR'` in CommentPanel, the browser's own locale in
 * SearchPanel, the `'sv-SE'` ISO-order trick in MatchInfoBar/metadataStatus.js,
 * home-made digit padding in CollectionPanel/matchTable.js).
 *
 * Every function reads `language` non-reactively via `get()` — the same
 * pattern `translate()` uses in `i18n/index.js` — so it can be called from
 * plain `.js` modules as well as component script blocks. A component that
 * needs the formatted value to re-render on a language change should read it
 * inside a `$derived`/`$effect` that also depends on `$language` (see
 * `i18n/index.js`'s header comment on the same trade-off for `translate()`).
 */

import { get } from 'svelte/store';
import { language } from '../i18n/index.js';

function activeLocale() {
    return get(language);
}

function toDate(value) {
    if (value == null || value === '') return null;
    const d = value instanceof Date ? value : new Date(value);
    return isNaN(d.getTime()) ? null : d;
}

/**
 * Format a date (day/month/year) in the active language. `value` is a Date,
 * a parseable string, or a timestamp. Returns '' for a missing/invalid input.
 */
export function formatDate(value, options) {
    const d = toDate(value);
    if (!d) return '';
    return new Intl.DateTimeFormat(activeLocale(), options ?? { year: 'numeric', month: '2-digit', day: '2-digit' }).format(d);
}

/** Same as formatDate(), with hour:minute appended per the active language's conventions. */
export function formatDateTime(value, options) {
    const d = toDate(value);
    if (!d) return '';
    return new Intl.DateTimeFormat(activeLocale(), options ?? { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(d);
}

/** Format a plain number (grouping/decimal separators) in the active language. */
export function formatNumber(value, options) {
    if (value == null || typeof value !== 'number' || Number.isNaN(value)) return '';
    return new Intl.NumberFormat(activeLocale(), options).format(value);
}

/** Format a ratio (0..1) as a percentage in the active language. */
export function formatPercent(value, options) {
    if (value == null || typeof value !== 'number' || Number.isNaN(value)) return '';
    return new Intl.NumberFormat(activeLocale(), { style: 'percent', ...options }).format(value);
}

/**
 * Minimal Intl.PluralRules wrapper: the CLDR plural category ('one', 'other',
 * and for some languages 'few'/'many'/'two'/'zero') for `n` in the active
 * language. Use it to pick a message form instead of a hand-rolled
 * `n === 1 ? singular : plural`, which is wrong outside English/French (e.g.
 * Russian has three categories, and this project ships to `ru`).
 */
export function pluralCategory(n, options) {
    return new Intl.PluralRules(activeLocale(), options).select(n);
}
