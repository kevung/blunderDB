/**
 * Helper to install/uninstall a Wails window.go mock for component tests.
 *
 * Prefer `vi.mock('../../wailsjs/go/database/Database.js', ...)` directly in each
 * test file (Vitest hoists it automatically). Use this helper only when you
 * need fine-grained per-test overrides at runtime.
 */

const DEFAULT_METHODS = {
    LoadCommandHistory: () => Promise.resolve([]),
    SaveCommand: () => Promise.resolve(undefined),
    ComputeStats: () => Promise.resolve(null)
};

/**
 * Install a mock `window.go.main.Database` on the global object.
 * @param {Record<string, Function>} overrides – per-test method overrides.
 */
export function installWailsMock(overrides = {}) {
    globalThis.window = globalThis.window ?? {};
    globalThis.window.go = {
        main: {
            Database: { ...DEFAULT_METHODS, ...overrides }
        }
    };
}

/** Remove the mock from the global object. */
export function uninstallWailsMock() {
    if (globalThis.window) {
        delete globalThis.window.go;
    }
}
