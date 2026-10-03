import { language, LOCALES, FALLBACK_LOCALE } from '../index.js';
import { writable, derived } from 'svelte/store';

// Every help bundle (manual/shortcuts/commands/about HTML, ~1.5 MB combined
// since the manual tab became a rendering of manuel.rst), English included, is
// fetched on demand: the interface only ever needs the language on screen, and
// nobody reads the help tabs until the modal is shown. English is fetched
// alongside any other language because it is the per-tab fallback below.
// Loading is triggered by HelpModal when it opens (or the language changes
// while it's open), not by a language switch elsewhere in the app.
const helpLoaders = import.meta.glob('./*.js');
const maps = {};

// Bumped whenever a lazily-loaded help bundle lands, so the derived `help`
// store below re-evaluates even though `language` itself did not change.
const helpVersion = writable(0);

async function loadBundle(lang) {
    if (maps[lang] || !LOCALES.includes(lang)) return;
    const loader = helpLoaders[`./${lang}.js`];
    if (!loader) return;
    const mod = await loader();
    maps[lang] = mod.default ?? mod;
}

/** Fetch and cache the help bundle for `lang` and the English fallback; a no-op once cached. */
export async function loadHelpFor(lang) {
    await Promise.all([loadBundle(FALLBACK_LOCALE), loadBundle(lang)]);
    helpVersion.update((n) => n + 1);
}

// Reactive help content for the active language. Falls back to English per-tab,
// so a partially-translated locale still renders English for any missing tab.
// `ready` is false until the bundles requested by loadHelpFor have landed
// (HelpModal shows a spinner meanwhile); helpVersion ticks the store again.
export const help = derived([language, helpVersion], ([$lang]) => {
    const fallback = maps[FALLBACK_LOCALE] || {};
    const m = maps[$lang] || {};
    return {
        ready: !!(maps[FALLBACK_LOCALE] && maps[$lang]),
        manual: m.manual ?? fallback.manual ?? '',
        shortcuts: m.shortcuts ?? fallback.shortcuts ?? '',
        commands: m.commands ?? fallback.commands ?? '',
        about: m.about ?? fallback.about ?? ''
    };
});
