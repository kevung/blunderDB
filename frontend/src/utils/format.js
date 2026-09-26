/**
 * format.js — date, number and plural formatting in the interface's active language. Reads
 * `language` non-reactively (like `translate()`), so it works from plain `.js`; a component that
 * must re-render on a language change reads it in a `$derived`/`$effect` depending on `$language`.
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
 * The CLDR plural category of `n` in the active language, instead of `n === 1 ? … : …`, which is
 * wrong outside English/French (Russian has three categories).
 */
export function pluralCategory(n, options) {
    return new Intl.PluralRules(activeLocale(), options).select(n);
}
