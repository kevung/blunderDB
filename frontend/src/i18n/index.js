// Lightweight i18n engine. `t` is a derived store yielding a translation FUNCTION, used as
// `$t('key', { param })`, so a locale switch re-renders every call. `translate()` / `get(t)` are
// non-reactive helpers for plain `.js` files: their output does not follow a language change.
// Fallback chain: locale -> English -> the key itself. `"Merge {n} names"` + { n: 3 } -> "Merge 3 names".

import { writable, derived, get } from 'svelte/store';
import { SaveLanguage } from '../../wailsjs/go/main/Config.js';

import en from './locales/en.json';

// Supported locale codes (order = order shown in the language selector).
export const LOCALES = ['en', 'fr', 'de', 'it', 'es', 'fi', 'ja', 'el', 'ru'];
export const FALLBACK_LOCALE = 'en';

// Native names (endonyms) for the language selector.
export const LANGUAGE_LABELS = {
    en: 'English',
    fi: 'Suomi',
    el: 'Ελληνικά',
    ja: '日本語',
    de: 'Deutsch',
    es: 'Español',
    fr: 'Français',
    it: 'Italiano',
    ru: 'Русский'
};

// Non-English locales are fetched on demand (bundling all nine put 511 kB of JSON in the main
// chunk). English stays static: it is the fallback and the seed of `language`, needed before
// any async load resolves.
const localeLoaders = import.meta.glob('./locales/*.json');
/** @type {Record<string, any>} */
const messages = { en };

/** @param {string} lang */
async function ensureLocaleLoaded(lang) {
    if (messages[lang]) return;
    const loader = localeLoaders[`./locales/${lang}.json`];
    if (!loader) return;
    const mod = /** @type {{ default?: any }} */ (await loader());
    messages[lang] = mod.default ?? mod;
}

// Resolve a dotted key path against a nested dictionary.
/**
 * @param {any} dict
 * @param {string} key
 */
function lookup(dict, key) {
    if (!dict) return undefined;
    return key.split('.').reduce((/** @type {any} */ o, /** @type {string} */ k) => (o == null ? undefined : o[k]), dict);
}

// Replace {placeholders} with values from params. Unknown placeholders are left intact.
/**
 * @param {unknown} str
 * @param {Record<string, unknown> | null | undefined} params
 */
function interpolate(str, params) {
    if (typeof str !== 'string' || !params) return str;
    return str.replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m));
}

/**
 * A translation function (the value of `$t`, the shape of `translate`); a key missing everywhere
 * resolves to itself.
 *
 * @typedef {(key: string, params?: Record<string, unknown> | null) => string} Translate
 */

/**
 * @param {string} lang
 * @param {string} key
 * @param {Record<string, unknown> | null | undefined} params
 * @returns {string}
 */
function translateFor(lang, key, params) {
    let raw = lookup(messages[lang], key);
    if (raw === undefined && lang !== FALLBACK_LOCALE) {
        raw = lookup(messages[FALLBACK_LOCALE], key);
    }
    if (raw === undefined) raw = key; // final fallback: surface the key for visibility
    return /** @type {string} */ (interpolate(raw, params));
}

// Current locale. Initialized to the fallback; overwritten at startup by
// initLanguage() once the persisted config is read.
export const language = writable(FALLBACK_LOCALE);

// Reactive translation function. Components: `{$t('toolbar.newDatabase')}`.
/** @type {import('svelte/store').Readable<Translate>} */
export const t = derived(language, ($lang) => (/** @type {string} */ key, /** @type {Record<string, unknown> | null | undefined} */ params) => translateFor($lang, key, params));

/**
 * Un bloc entier de messages, dans la langue courante, replié sur l'anglais clé par clé : la page
 * de salle, écrite en Go, rend ainsi les codes du moteur avec les mêmes mots que le front.
 *
 * @param {string} section
 */
export function messageBlock(section) {
    const base = lookup(messages[FALLBACK_LOCALE], section);
    const own = lookup(messages[get(language)], section);
    return mergeBlocks(base, own);
}

/**
 * Repli profond : l'anglais dessous, la langue de l'utilisateur dessus.
 *
 * @param {any} base
 * @param {any} own
 * @returns {any}
 */
function mergeBlocks(base, own) {
    if (own === undefined) return base === undefined ? {} : base;
    if (base === undefined || typeof base !== 'object' || typeof own !== 'object') return own;
    const out = { ...base };
    for (const [k, v] of Object.entries(own)) out[k] = mergeBlocks(base[k], v);
    return out;
}

// Non-reactive translation for use outside Svelte components (.js modules).
/**
 * @param {string} key
 * @param {Record<string, unknown> | null} [params]
 * @returns {string}
 */
export function translate(key, params) {
    return translateFor(get(language), key, params);
}

// A *deferred* status-bar message: StatusBar resolves it through `$t`, so it re-translates on a
// language change, unlike a translate() string. Plain strings may still be stored as-is.
/**
 * @typedef {{i18nKey: string, i18nParams: Record<string, any>|null}} StatusMessage
 */

/**
 * @param {string} key
 * @param {Record<string, any>} [params]
 * @returns {StatusMessage}
 */
export function tMsg(key, params) {
    return { i18nKey: key, i18nParams: params ?? null };
}

// Resolve a plain string or a tMsg() descriptor with `tfn` (e.g. the value of `$t`).
/**
 * @param {string | StatusMessage | null | undefined} value
 * @param {Translate} tfn
 * @returns {string | null | undefined}
 */
export function resolveStatusMessage(value, tfn) {
    if (value && typeof value === 'object' && value.i18nKey) {
        return tfn(value.i18nKey, value.i18nParams);
    }
    // tMsg() always names a key, so what is left is a plain string or nothing.
    return /** @type {string | null | undefined} */ (value);
}

// Change and persist the language. The dictionary is awaited before the store flips, so
// `$t(...)` never renders raw keys.
/** @param {string} lang */
export async function setLanguage(lang) {
    const next = LOCALES.includes(lang) ? lang : FALLBACK_LOCALE;
    await ensureLocaleLoaded(next);
    language.set(next);
    try {
        await SaveLanguage(next);
    } catch (e) {
        // Persistence is best-effort; the in-memory switch still applies.
        // eslint-disable-next-line no-console
        console.error('Failed to persist language preference:', e);
    }
}

// Apply a persisted language at startup without re-persisting it.
/** @param {string} lang */
export async function initLanguage(lang) {
    const next = LOCALES.includes(lang) ? lang : FALLBACK_LOCALE;
    await ensureLocaleLoaded(next);
    language.set(next);
}
