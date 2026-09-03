import { language, LOCALES, FALLBACK_LOCALE } from '../index.js';
import { writable, derived } from 'svelte/store';

import en from './en.js';

// Every non-English help bundle (manual/shortcuts/commands/about HTML, ~628 kB
// combined) is fetched on demand instead of bundled statically (#207): the
// interface only ever needs the language actually on screen. English stays a
// static import — it is the per-tab fallback below. Loading is triggered by
// HelpModal when it opens (or the language changes while it's open), not by
// a language switch elsewhere in the app, since nobody is reading the help
// tabs until the modal is shown.
const helpLoaders = import.meta.glob('./*.js');
const maps = { en };

// Bumped whenever a lazily-loaded help bundle lands, so the derived `help`
// store below re-evaluates even though `language` itself did not change.
const helpVersion = writable(0);

const FALLBACK = en;

/** Fetch and cache the help bundle for `lang`; a no-op once cached. */
export async function loadHelpFor(lang) {
    if (maps[lang] || lang === FALLBACK_LOCALE || !LOCALES.includes(lang)) return;
    const loader = helpLoaders[`./${lang}.js`];
    if (!loader) return;
    const mod = await loader();
    maps[lang] = mod.default ?? mod;
    helpVersion.update((n) => n + 1);
}

// Reactive help content for the active language. Falls back to English per-tab,
// so a partially-translated locale still renders English for any missing tab,
// and so does a locale whose bundle has not been fetched yet (loadHelpFor
// resolves shortly after and helpVersion ticks the store again).
export const help = derived([language, helpVersion], ([$lang]) => {
    const m = maps[$lang] || {};
    return {
        manual: m.manual ?? FALLBACK.manual,
        shortcuts: m.shortcuts ?? FALLBACK.shortcuts,
        commands: m.commands ?? FALLBACK.commands,
        about: m.about ?? FALLBACK.about
    };
});
