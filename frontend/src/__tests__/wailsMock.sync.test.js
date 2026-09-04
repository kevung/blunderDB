/**
 * wailsMock.sync.test.js
 *
 * `src/__mocks__/wails.js` builds each `create*Mock()` by reflecting on the
 * real generated bindings (`Object.keys()` of the imported module), so it
 * cannot drift from a hand-copied list. That import is the thing to
 * distrust here, not the list: a wrong path, a build/tree-shaking quirk, or
 * a change to how `wailsjs/go/**` is generated could silently make the
 * reflection see fewer names than the file actually exports, and nothing
 * would fail — the mock would just be quietly incomplete again, the exact
 * failure mode #214 opened this file to prevent.
 *
 * So this test does NOT import `REAL_MODULES` and compare it to itself
 * (that would only prove the mock agrees with what it already read) — it
 * re-derives the export lists from the generated files' own source text,
 * the same `export function NAME(` scan `panelDefaults.sync.test.js` and
 * `commandVocabulary.sync.test.js` use for their own independent sources,
 * and checks every one of those names is a key `create*Mock()` produces.
 */

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { createDatabaseMock, createAppMock, createConfigMock, createRuntimeMock } from '../__mocks__/wails.js';

const FRONTEND_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');

/** @param {string} relPath */
function exportedFunctionNames(relPath) {
    const source = fs.readFileSync(path.join(FRONTEND_ROOT, relPath), 'utf8');
    const names = [];
    const re = /^export function (\w+)\(/gm;
    let m;
    while ((m = re.exec(source))) names.push(m[1]);
    return names;
}

const SOURCES = [
    { label: 'wailsjs/go/database/Database.js', file: 'wailsjs/go/database/Database.js', factory: createDatabaseMock },
    { label: 'wailsjs/go/gui/App.js', file: 'wailsjs/go/gui/App.js', factory: createAppMock },
    { label: 'wailsjs/go/main/Config.js', file: 'wailsjs/go/main/Config.js', factory: createConfigMock },
    { label: 'wailsjs/runtime/runtime.js', file: 'wailsjs/runtime/runtime.js', factory: createRuntimeMock }
];

describe('the shared wails mock covers every generated binding', () => {
    test.each(SOURCES)('$label', ({ file, factory }) => {
        const realNames = exportedFunctionNames(file);
        // A generated file with zero parsed exports would make every
        // assertion below vacuously true — guard the scan itself first.
        expect(realNames.length).toBeGreaterThan(0);

        const mock = /** @type {Record<string, unknown>} */ (factory());
        const mockNames = Object.keys(mock);

        const missingFromMock = realNames.filter((n) => !mockNames.includes(n));
        expect(missingFromMock, `create*Mock() is missing: ${missingFromMock.join(', ')}`).toEqual([]);

        const extraInMock = mockNames.filter((n) => !realNames.includes(n));
        expect(extraInMock, `create*Mock() has names ${file} does not export: ${extraInMock.join(', ')}`).toEqual([]);

        for (const name of realNames) {
            expect(typeof mock[name], `${name} should be a mock function`).toBe('function');
        }
    });
});
