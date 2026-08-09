/**
 * i18nKeys.sync.test.js
 *
 * Every literal translation key used in the interface must exist. When one does not, the
 * engine falls back to the key itself and the user reads "export.descMany" in the middle of
 * a sentence — which is exactly what shipped, unnoticed, in the export dialog's summary.
 *
 * This test walks the sources for `$t('...')`, `tr('...')`, `tMsg('...')` and
 * `translate('...')`, and checks each key against every locale. English is the fallback, so
 * a key missing there shows raw text in all nine languages.
 *
 * KNOWN_GAPS holds the keys that were already missing when this guard was written. They are
 * real defects — status messages that display raw keys — but they belong to the match,
 * tournament and status-bar areas rather than to export, and are listed here so the guard
 * can be added without dragging that work along. Removing an entry is always welcome; a new
 * one must not appear.
 */

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { LOCALES } from '../i18n/index.js';

const SRC = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const KNOWN_GAPS = new Set([
    'match.commentUpdated',
    'match.errorDeleting',
    'match.errorSettingTournament',
    'match.errorSwapping',
    'match.errorUpdating',
    'match.errorUpdatingComment',
    'match.matchDeleted',
    'match.matchUpdated',
    'match.noMovesFound',
    'match.swappedPlayers',
    'match.tournamentCleared',
    'match.tournamentSet',
    'tournament.created',
    'tournament.deleted',
    'tournament.errorCreating',
    'tournament.errorOpening',
    'tournament.errorSwapping',
    'tournament.noMovesFound',
    'tournament.swappedPlayers'
]);

function sourceFiles(dir) {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) return entry.name === '__tests__' ? [] : sourceFiles(full);
        return /\.(svelte|js)$/.test(entry.name) ? [full] : [];
    });
}

function usedKeys() {
    // Only literal keys: `$t(setting.labelKey)` and friends are built at runtime and cannot
    // be checked here. A trailing dot marks a prefix being concatenated, likewise dynamic.
    const pattern = /(?:\$t|\btr|\btMsg|\btranslate)\(\s*'([a-zA-Z][\w.]*)'/g;
    const keys = new Set();
    for (const file of sourceFiles(SRC)) {
        if (file.includes(path.join('i18n', 'index.js'))) continue;
        const text = fs.readFileSync(file, 'utf8');
        for (const [, key] of text.matchAll(pattern)) {
            if (!key.endsWith('.')) keys.add(key);
        }
    }
    return [...keys].sort();
}

function resolve(dictionary, key) {
    return key.split('.').reduce((node, part) => (node == null ? undefined : node[part]), dictionary);
}

describe('translation keys', () => {
    const keys = usedKeys();

    test('the interface uses a meaningful number of keys', () => {
        expect(keys.length).toBeGreaterThan(500);
    });

    test.each(LOCALES)('%s defines every key the interface asks for', (locale) => {
        const dictionary = JSON.parse(fs.readFileSync(path.join(SRC, 'i18n', 'locales', `${locale}.json`), 'utf8'));
        const missing = keys.filter((key) => !KNOWN_GAPS.has(key) && typeof resolve(dictionary, key) !== 'string');
        expect(missing).toEqual([]);
    });

    test('the known gaps are still gaps, and no more', () => {
        const english = JSON.parse(fs.readFileSync(path.join(SRC, 'i18n', 'locales', 'en.json'), 'utf8'));
        const fixed = [...KNOWN_GAPS].filter((key) => typeof resolve(english, key) === 'string');
        expect(fixed, 'these keys now exist — remove them from KNOWN_GAPS').toEqual([]);
    });
});
