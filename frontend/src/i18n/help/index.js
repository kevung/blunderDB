import { logger } from '../../utils/logger.js';
import { language, LOCALES, FALLBACK_LOCALE } from '../index.js';
import { writable, derived } from 'svelte/store';

// Every help bundle (manual/shortcuts/commands/about HTML, ~1.5 MB uncompressed since
// the manual tab became a rendering of manuel.rst), English included, is fetched on
// demand: the interface only ever needs the language on screen, and nobody reads the
// help tabs until the modal is shown. English is fetched alongside any other language
// because it is the per-tab fallback below. Loading is triggered by HelpModal when it
// opens (or the language changes while it's open), not by a language switch elsewhere
// in the app.
//
// A bundle is a gzip-compressed JSON file under public/help/ (written by cmd/help-gen),
// so that the binary, which embeds the built frontend as is, carries ~0.3x of the text.
// It is inflated here with DecompressionStream, available in every engine Wails uses
// (WebKitGTK >= 2.40, WKWebView >= 16.4, WebView2). A response that is already plain
// JSON (a server or a test that inflated it) is read as is.
const maps = {};

// Bumped whenever a lazily-loaded help bundle lands, so the derived `help`
// store below re-evaluates even though `language` itself did not change.
const helpVersion = writable(0);

const failed = new Set();

async function fetchBundle(lang) {
    const response = await fetch(`${import.meta.env.BASE_URL}help/${lang}.bin`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    const bytes = new Uint8Array(await response.arrayBuffer());
    const gzipped = bytes[0] === 0x1f && bytes[1] === 0x8b;
    if (gzipped && typeof DecompressionStream === 'undefined') throw new Error('DecompressionStream is not available');
    const text = gzipped ? await new Response(new Response(bytes).body.pipeThrough(new DecompressionStream('gzip'))).text() : new TextDecoder().decode(bytes);
    return JSON.parse(text);
}

async function loadBundle(lang) {
    if (maps[lang] || !LOCALES.includes(lang)) return;
    try {
        maps[lang] = await fetchBundle(lang);
        failed.delete(lang);
    } catch (error) {
        failed.add(lang);
        logger.error(`could not load the ${lang} help bundle:`, error);
    }
}

/** Fetch and cache the help bundle for `lang` and the English fallback; a no-op once cached. Never rejects. */
export async function loadHelpFor(lang) {
    await Promise.all([...new Set([FALLBACK_LOCALE, lang])].map(loadBundle));
    helpVersion.update((n) => n + 1);
}

// Reactive help content for the active language. Falls back to English per-tab,
// so a partially-translated locale, or one whose bundle failed to load, still
// renders English. `ready` is false until loadHelpFor has settled; `failed` is
// true when not even English could be loaded.
export const help = derived([language, helpVersion], ([$lang]) => {
    const fallback = maps[FALLBACK_LOCALE] || {};
    const m = maps[$lang] || {};
    return {
        ready: !!maps[FALLBACK_LOCALE] && (!!maps[$lang] || failed.has($lang)),
        failed: failed.has(FALLBACK_LOCALE),
        manual: m.manual ?? fallback.manual ?? '',
        shortcuts: m.shortcuts ?? fallback.shortcuts ?? '',
        commands: m.commands ?? fallback.commands ?? '',
        about: m.about ?? fallback.about ?? ''
    };
});
