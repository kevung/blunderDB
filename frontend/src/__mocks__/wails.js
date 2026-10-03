/**
 * src/__mocks__/wails.js — full-surface mocks of the generated Wails bindings (`wailsjs/go/**`,
 * `wailsjs/runtime/runtime.js`). Every name `Object.keys()` finds on the real module becomes a
 * `vi.fn()` (resolving `undefined`) unless `overrides` supplies one, so a mock cannot drift behind
 * a Go rename and a new binding call does not fail with "X is not a function".
 * `wailsMock.sync.test.js` checks the reflection saw every export.
 *
 * Usage, from a `vi.mock(...)` factory (hoisted, hence the dynamic import):
 *
 *   vi.mock('../../wailsjs/go/database/Database.js', async () => {
 *       const { createDatabaseMock } = await import('../__mocks__/wails.js');
 *       return createDatabaseMock({ ListPositionIDs: vi.fn().mockResolvedValue([1, 2, 3]) });
 *   });
 *
 * A safe default; a hand-picked partial mock remains right when a test asserts an exact call list.
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

/**
 * Point the library bindings of a Database mock (CountPositions, ListPositionIDs windows,
 * IndexOfPosition) at `ids`, as the paged library reads them. `db` is the mocked module; `ids`
 * may be replaced later by calling again.
 * @param {any} db
 * @param {number[]} ids
 */
export function useLibrary(db, ids) {
    db.CountPositions.mockImplementation(async () => ids.length);
    db.ListPositionIDs.mockImplementation(async (offset = 0, limit = 0) => (limit > 0 ? ids.slice(offset, offset + limit) : ids.slice(offset)));
    db.IndexOfPosition.mockImplementation(async (id) => ids.indexOf(id));
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
