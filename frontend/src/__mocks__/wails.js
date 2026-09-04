/**
 * src/__mocks__/wails.js — one shared shape for mocking the generated Wails
 * bindings (`wailsjs/go/**`, `wailsjs/runtime/runtime.js`).
 *
 * 52/94 test files used to hand-write their own partial `vi.mock(...)` object
 * for these modules (D.13, #214) — a subset picked by what the test happened
 * to exercise. That is brittle in one direction only: a test can always name
 * fewer functions than the module exports, so nothing here flags a hand-mock
 * that quietly drifted behind a Go method rename, and a code path that starts
 * calling one more binding fails at the call site with "X is not a function"
 * rather than at the mock's own boundary.
 *
 * These factories mock the FULL surface of each module instead: every name
 * `Object.keys()` finds on the real generated file becomes a `vi.fn()`
 * (resolving `undefined` — the common case for a Go-bound call nothing reads
 * the result of) unless `overrides` supplies its own `vi.fn()`. Because the
 * key list is read from the real module rather than copied by hand, it is
 * automatically complete — adding a Go method regenerates `wailsjs/go/**` and
 * this file's mocks follow without an edit. `wailsMock.sync.test.js` is the
 * belt-and-braces check that the reflection actually saw every export (the
 * same belt commandVocabulary.sync.test.js and friends wear for their own
 * derived-from-source invariants) rather than trusting import machinery
 * silently.
 *
 * Usage — from inside a `vi.mock(...)` factory (which vitest hoists, so the
 * shared module must be reached with a dynamic import, not a top-level one):
 *
 *   vi.mock('../../wailsjs/go/database/Database.js', async () => {
 *       const { createDatabaseMock } = await import('../__mocks__/wails.js');
 *       return createDatabaseMock({
 *           ListPositionIDs: vi.fn().mockResolvedValue([1, 2, 3])
 *       });
 *   });
 *
 * This does not replace a hand-picked partial mock where a test wants to
 * assert that a specific, short list of calls happened and no others — it is
 * a safer *default* for tests that need the module to not blow up on calls
 * they do not care about.
 */

import { vi } from 'vitest';
import * as DatabaseModule from '../../wailsjs/go/database/Database.js';
import * as AppModule from '../../wailsjs/go/gui/App.js';
import * as ConfigModule from '../../wailsjs/go/main/Config.js';
import * as RuntimeModule from '../../wailsjs/runtime/runtime.js';

function mockAll(realModule, overrides) {
    const mock = {};
    for (const name of Object.keys(realModule)) {
        mock[name] = overrides[name] ?? vi.fn().mockResolvedValue(undefined);
    }
    return mock;
}

export function createDatabaseMock(overrides = {}) {
    return mockAll(DatabaseModule, overrides);
}

export function createAppMock(overrides = {}) {
    return mockAll(AppModule, overrides);
}

export function createConfigMock(overrides = {}) {
    return mockAll(ConfigModule, overrides);
}

export function createRuntimeMock(overrides = {}) {
    return mockAll(RuntimeModule, overrides);
}

// Exposed only so wailsMock.sync.test.js can compare the real modules'
// export names against what each factory above produces, without every
// caller needing its own import of all four.
export const REAL_MODULES = {
    database: DatabaseModule,
    gui: AppModule,
    config: ConfigModule,
    runtime: RuntimeModule
};
