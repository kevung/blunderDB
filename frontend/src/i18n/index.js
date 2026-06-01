// Lightweight i18n engine for blunderDB.
//
// Design (see tasks/i18n plan):
//  - `language` is a writable store holding the current locale code.
//  - `t` is a derived store that yields a translation FUNCTION, so components
//    use it reactively as `$t('some.key', { param })`. Because it depends on
//    `language`, switching the locale re-renders every `$t(...)` call.
//  - `translate()` / `get(t)` are non-reactive helpers for use in plain `.js`
//    files (commandProcessor, services, stores) where `$` auto-subscription is
//    unavailable. Messages emitted from those files resolve at call time and do
//    NOT live-update on language change — acceptable for transient log/status text.
//
// Fallback chain: selected locale -> English -> the key itself.
// Interpolation: `"Merge {n} names"` + { n: 3 } -> "Merge 3 names".

import { writable, derived, get } from 'svelte/store';
import { SaveLanguage } from '../../wailsjs/go/main/Config.js';

import en from './locales/en.json';
import fi from './locales/fi.json';
import el from './locales/el.json';
import ja from './locales/ja.json';
import de from './locales/de.json';
import es from './locales/es.json';
import fr from './locales/fr.json';
import it from './locales/it.json';
import ru from './locales/ru.json';

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

const messages = { en, fi, el, ja, de, es, fr, it, ru };

// Resolve a dotted key path against a nested dictionary.
function lookup(dict, key) {
    if (!dict) return undefined;
    return key.split('.').reduce((o, k) => (o == null ? undefined : o[k]), dict);
}

// Replace {placeholders} with values from params. Unknown placeholders are left intact.
function interpolate(str, params) {
    if (typeof str !== 'string' || !params) return str;
    return str.replace(/\{(\w+)\}/g, (m, k) => (k in params ? String(params[k]) : m));
}

function translateFor(lang, key, params) {
    let raw = lookup(messages[lang], key);
    if (raw === undefined && lang !== FALLBACK_LOCALE) {
        raw = lookup(messages[FALLBACK_LOCALE], key);
    }
    if (raw === undefined) raw = key; // final fallback: surface the key for visibility
    return interpolate(raw, params);
}

// Current locale. Initialized to the fallback; overwritten at startup by
// initLanguage() once the persisted config is read.
export const language = writable(FALLBACK_LOCALE);

// Reactive translation function. Components: `{$t('toolbar.newDatabase')}`.
export const t = derived(language, ($lang) => (key, params) => translateFor($lang, key, params));

// Non-reactive translation for use outside Svelte components (.js modules).
export function translate(key, params) {
    return translateFor(get(language), key, params);
}

// Build a *deferred* status-bar message descriptor instead of a resolved string.
// The status bar (StatusBar.svelte) holds the descriptor and resolves it through
// the reactive `$t` store, so the message re-translates live when the language
// changes — unlike a string produced by translate(), which is frozen at call time.
// Plain strings (player names, technical tokens) may still be stored as-is.
export function tMsg(key, params) {
    return { i18nKey: key, i18nParams: params ?? null };
}

// Resolve a value that may be either a plain string or a tMsg() descriptor.
// `tfn` is a translation function (e.g. the value of the `$t` store).
export function resolveStatusMessage(value, tfn) {
    if (value && typeof value === 'object' && value.i18nKey) {
        return tfn(value.i18nKey, value.i18nParams);
    }
    return value;
}

// Change the active language and persist it to the Go config.
export async function setLanguage(lang) {
    const next = LOCALES.includes(lang) ? lang : FALLBACK_LOCALE;
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
export function initLanguage(lang) {
    language.set(LOCALES.includes(lang) ? lang : FALLBACK_LOCALE);
}
