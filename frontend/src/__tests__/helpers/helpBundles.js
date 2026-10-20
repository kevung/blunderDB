// @ts-nocheck — doublures de test : les signatures typées (Wails, stores) exigent des objets complets.
// The help bundles are gzip-compressed JSON under public/help/ (written by cmd/help-gen).
// Tests read them straight from disk.
import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';
import { fileURLToPath } from 'node:url';

const HELP_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), '../../../public/help');

export const helpBundlePath = (lang) => path.join(HELP_DIR, `${lang}.bin`);

/** The {manual, shortcuts, commands, about} object of one language. */
export function readHelpBundle(lang) {
    return JSON.parse(zlib.gunzipSync(fs.readFileSync(helpBundlePath(lang))).toString('utf8'));
}

/** Languages that have a bundle on disk. */
export const helpBundleLanguages = () =>
    fs
        .readdirSync(HELP_DIR)
        .filter((f) => f.endsWith('.bin'))
        .map((f) => f.replace(/\.bin$/, ''));

/**
 * Replaces fetch with one serving the bundles from disk, gzip-compressed as shipped. `fail`
 * lists the languages whose request answers an error.
 */
export function stubHelpFetch(vi, { fail = [] } = {}) {
    vi.stubGlobal(
        'fetch',
        vi.fn(async (url) => {
            const lang = /help\/([a-z]+)\.bin$/.exec(String(url))?.[1];
            if (!lang || fail.includes(lang)) return new Response('', { status: 500 });
            return new Response(fs.readFileSync(helpBundlePath(lang)), { status: 200 });
        })
    );
}
