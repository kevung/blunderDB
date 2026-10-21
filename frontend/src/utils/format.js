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

// Building an Intl formatter costs far more than formatting with one, and a list formats every
// cell: formatters are kept per (kind, options) for the active language, and the whole cache is
// dropped when the language changes so it never holds more than one locale.
/** @type {Map<string, any>} */
const formatters = new Map();
let formattersLocale = '';

/**
 * @template T
 * @param {string} kind
 * @param {object | undefined} options
 * @param {(locale: string) => T} build
 * @returns {T}
 */
function cachedFormatter(kind, options, build) {
    const locale = activeLocale();
    if (locale !== formattersLocale) {
        formatters.clear();
        formattersLocale = locale;
    }
    const key = options === undefined ? kind : `${kind}|${JSON.stringify(options)}`;
    let f = formatters.get(key);
    if (f === undefined) {
        f = build(locale);
        formatters.set(key, f);
    }
    return f;
}

const DATE_OPTIONS = /** @type {Intl.DateTimeFormatOptions} */ ({ year: 'numeric', month: '2-digit', day: '2-digit' });
const DATE_TIME_OPTIONS = /** @type {Intl.DateTimeFormatOptions} */ ({ year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });

/** @param {Intl.DateTimeFormatOptions} options */
function dateTimeFormat(options) {
    return cachedFormatter('date', options, (locale) => new Intl.DateTimeFormat(locale, options));
}

/** @param {Intl.NumberFormatOptions | undefined} options */
function numberFormat(options) {
    return cachedFormatter('number', options, (locale) => new Intl.NumberFormat(locale, options));
}

function toDate(/** @type {Date | string | number | null | undefined} */ value) {
    if (value == null || value === '') return null;
    const d = value instanceof Date ? value : new Date(value);
    return isNaN(d.getTime()) ? null : d;
}

/**
 * Format a date (day/month/year) in the active language. `value` is a Date,
 * a parseable string, or a timestamp. Returns '' for a missing/invalid input.
 */
export function formatDate(/** @type {Date | string | number | null | undefined} */ value, /** @type {Intl.DateTimeFormatOptions | undefined} */ options = undefined) {
    const d = toDate(value);
    if (!d) return '';
    return dateTimeFormat(options ?? DATE_OPTIONS).format(d);
}

/**
 * The UTC-midnight timestamp of the day a backend date text starts with ("YYYY-MM-DD", RFC 3339,
 * or Go's time.String "YYYY-MM-DD hh:mm:ss +0000 UTC"), or null when it does not start with one. Read the day,
 * never the whole text: WebKitGTK's Date refuses Go's time.String format where V8 accepts it.
 */
export function parseIsoDayUtc(/** @type {string | null | undefined} */ value) {
    if (!value) return null;
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(String(value));
    if (!m) return null;
    return Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
}

/**
 * Format the calendar day of an ISO string ("YYYY-MM-DD", "YYYY-MM-DDT00:00:00Z") in the active
 * language. The day is read from the first ten characters and built as a local date, so a UTC
 * midnight does not slide to the day before west of Greenwich. Anything else comes back as is.
 */
export function formatIsoDay(/** @type {Date | string | number | null | undefined} */ value) {
    if (!value) return '';
    const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(String(value));
    if (!m) return String(value);
    return formatDate(new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])));
}

/**
 * Format the UTC calendar day of a timestamp: the counterpart of reading "YYYY-MM-DD" with
 * `new Date(...)`, which is UTC midnight. Formatting that instant in local time would show the
 * day before west of Greenwich.
 */
export function formatUtcDay(/** @type {number} */ ms) {
    if (!ms) return '';
    const d = new Date(ms);
    return formatDate(new Date(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate()));
}

/** Same as formatDate(), with hour:minute appended per the active language's conventions. */
export function formatDateTime(/** @type {Date | string | number | null | undefined} */ value, /** @type {Intl.DateTimeFormatOptions | undefined} */ options = undefined) {
    const d = toDate(value);
    if (!d) return '';
    return dateTimeFormat(options ?? DATE_TIME_OPTIONS).format(d);
}

/** Format a plain number (grouping/decimal separators) in the active language. */
export function formatNumber(/** @type {number | null | undefined} */ value, /** @type {Intl.NumberFormatOptions | undefined} */ options = undefined) {
    if (value == null || typeof value !== 'number' || Number.isNaN(value)) return '';
    return numberFormat(options).format(value);
}

/** Format a ratio (0..1) as a percentage in the active language. */
export function formatPercent(/** @type {number | null | undefined} */ value, /** @type {Intl.NumberFormatOptions | undefined} */ options = undefined) {
    if (value == null || typeof value !== 'number' || Number.isNaN(value)) return '';
    return numberFormat({ style: 'percent', ...options }).format(value);
}

/**
 * The CLDR plural category of `n` in the active language, instead of `n === 1 ? … : …`, which is
 * wrong outside English/French (Russian has three categories).
 */
export function pluralCategory(/** @type {number} */ n, /** @type {Intl.PluralRulesOptions | undefined} */ options = undefined) {
    return cachedFormatter('plural', options, (locale) => new Intl.PluralRules(locale, options)).select(n);
}
