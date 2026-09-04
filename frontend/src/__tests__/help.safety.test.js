// The four help tabs are injected with {@html} (HelpModal.svelte), so nothing
// in them is escaped at render time. Fiche D.15 found that the eslint exclusion
// covering src/i18n/help/*.js had been hiding a real hole in exactly that spot:
// the About tab interpolated `{appVersion}`/`{dbVersion}` into raw HTML.
//
// Now that the bundles are generated (see docs/adr, "the in-app help is
// generated from the documentation"), the escaping lives in one place —
// cmd/help-gen escapes every string it takes from a .rst or a .po before
// applying its fixed HTML vocabulary — and the lint exclusion is legitimate: the
// files carry no logic. This suite is what replaces the linting: it checks the
// *shipped* corpus, in all nine languages, for the things {@html} would execute.
//
// It deliberately looks at the generated output rather than at the generator, so
// a hand-edit of a bundle (or a hand-written prose fragment copied through
// verbatim) is caught just the same.

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { LOCALES } from '../i18n/index.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const HELP_DIR = path.join(__dirname, '../i18n/help');
const TABS = ['manual', 'shortcuts', 'commands', 'about'];

// HelpModal substitutes exactly these two, HTML-escaping their values first.
const ALLOWED_PLACEHOLDERS = new Set(['{appVersion}', '{dbVersion}']);

const bundles = LOCALES.map((lang) => [lang, path.join(HELP_DIR, `${lang}.js`)]);

describe('the help corpus is safe to inject with {@html}', () => {
    test('every documentation language has a bundle', () => {
        expect(LOCALES.length).toBeGreaterThan(1);
        for (const [lang, file] of bundles) {
            expect(fs.existsSync(file), `no help bundle for ${lang}`).toBe(true);
        }
    });

    test.each(bundles)('%s carries no script, event handler or javascript: URL', async (lang, file) => {
        const mod = await import(/* @vite-ignore */ file);
        for (const tab of TABS) {
            const html = mod.default[tab];
            expect(typeof html, `${lang}.js has no "${tab}" tab`).toBe('string');
            expect(html, `${lang}/${tab}: <script>`).not.toMatch(/<\s*script/i);
            expect(html, `${lang}/${tab}: <iframe>/<object>/<embed>`).not.toMatch(/<\s*(iframe|object|embed|form)/i);
            expect(html, `${lang}/${tab}: inline event handler`).not.toMatch(/\son[a-z]+\s*=/i);
            expect(html, `${lang}/${tab}: javascript:/data: URL`).not.toMatch(/(href|src)\s*=\s*["']?\s*(javascript|data):/i);
        }
    });

    test.each(bundles)('%s interpolates nothing but the two version placeholders', async (lang, file) => {
        const mod = await import(/* @vite-ignore */ file);
        for (const tab of TABS) {
            const placeholders = mod.default[tab].match(/\{[A-Za-z][A-Za-z0-9_]*\}/g) ?? [];
            const unexpected = placeholders.filter((p) => !ALLOWED_PLACEHOLDERS.has(p));
            expect(unexpected, `${lang}/${tab}: unknown placeholder(s) ${unexpected.join(', ')} — HelpModal only escapes appVersion/dbVersion`).toEqual([]);
            if (tab !== 'about') {
                expect(placeholders, `${lang}/${tab}: version placeholders belong to the About tab only`).toEqual([]);
            }
        }
    });

    // Every external link in the corpus opens a browser window from a desktop
    // app; rel="noopener noreferrer" is what keeps the opened page from
    // reaching back into the WebView through window.opener.
    test.each(bundles)('%s opens every external link with rel="noopener noreferrer"', async (lang, file) => {
        const mod = await import(/* @vite-ignore */ file);
        for (const tab of TABS) {
            for (const anchor of mod.default[tab].match(/<a\s[^>]*>/g) ?? []) {
                expect(anchor, `${lang}/${tab}: link without target/rel hardening`).toMatch(/rel="noopener noreferrer"/);
                expect(anchor, `${lang}/${tab}: link must be https`).toMatch(/href="https:\/\//);
            }
        }
    });

    // The bundles are a build artefact of cmd/help-gen; a hand-edit is a change
    // that the next `make help` silently reverts. The banner says so, and this
    // test makes sure it is still there to be read.
    test.each(bundles)('%s is marked as generated', (lang, file) => {
        expect(fs.readFileSync(file, 'utf8').startsWith('// GENERATED FILE')).toBe(true);
    });
});
