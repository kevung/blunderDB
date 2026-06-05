/**
 * helpers/wailsMock.js
 *
 * Injecte un mock de window.go (bindings Database, Config, App) et de
 * window.runtime (Wails runtime) dans une page Playwright via addInitScript.
 *
 * Usage :
 *   import { installWailsMock } from './helpers/wailsMock.js';
 *   await installWailsMock(page);               // defaults
 *   await installWailsMock(page, { dbExtra: { ComputeEPCFromPosition: ... } });
 *
 * Stratégie : le script est sérialisé et injecté AVANT tout script de la page
 * (Playwright garantit cela). Les Proxy assurent qu'un appel à une méthode
 * non listée retourne Promise.resolve(null) au lieu de planter.
 */

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ dbExtra?: Record<string, unknown> }} [opts]
 */
export async function installWailsMock(page, opts = {}) {
    await page.addInitScript(() => {
        // ── Helpers locaux ───────────────────────────────────────────────────
        const noop = () => {};
        const asyncNull = () => Promise.resolve(null);
        const asyncVoid = () => Promise.resolve(undefined);
        const asyncArr = () => Promise.resolve([]);

        /** Proxy qui retourne asyncNull pour toute méthode non définie. */
        function makeProxy(base) {
            return new Proxy(base, {
                get(target, prop) {
                    return prop in target ? target[prop] : asyncNull;
                },
            });
        }

        // ── window.runtime ───────────────────────────────────────────────────
        window.runtime = new Proxy(
            {
                // Logs — silenced
                LogPrint: noop,
                LogTrace: noop,
                LogDebug: noop,
                LogInfo: noop,
                LogWarning: noop,
                LogError: noop,
                LogFatal: noop,
                // Events
                EventsOnMultiple: () => noop, // retourne une fonction de désabonnement
                EventsOff: noop,
                EventsEmit: noop,
                // File drop
                OnFileDrop: noop,
                OnFileDropOff: noop,
                // Window
                WindowGetSize: () => Promise.resolve({ w: 1280, h: 800 }),
                WindowSetTitle: noop,
                WindowSetSize: noop,
                WindowSetMinSize: noop,
                WindowSetMaxSize: noop,
                WindowCenter: noop,
                WindowMaximise: noop,
                WindowUnmaximise: noop,
                WindowToggleMaximise: noop,
                WindowMinimise: noop,
                WindowUnminimise: noop,
                WindowFullscreen: noop,
                WindowUnfullscreen: noop,
                WindowIsFullscreen: () => Promise.resolve(false),
                WindowIsMaximised: () => Promise.resolve(false),
                WindowIsMinimised: () => Promise.resolve(false),
                WindowIsNormal: () => Promise.resolve(true),
                WindowGetPosition: () => Promise.resolve({ x: 0, y: 0 }),
                WindowSetPosition: noop,
                WindowSetAlwaysOnTop: noop,
                WindowSetBackgroundColour: noop,
                WindowSetSystemDefaultTheme: noop,
                WindowSetLightTheme: noop,
                WindowSetDarkTheme: noop,
                WindowHide: noop,
                WindowShow: noop,
                WindowReload: noop,
                WindowReloadApp: noop,
                // Others
                Quit: noop,
                Hide: noop,
                Show: noop,
                BrowserOpenURL: noop,
                ClipboardGetText: () => Promise.resolve(''),
                ClipboardSetText: asyncVoid,
                CanResolveFilePaths: () => Promise.resolve(false),
                ResolveFilePaths: asyncArr,
                ScreenGetAll: asyncArr,
                Environment: () => Promise.resolve({ buildType: 'dev', platform: 'linux', arch: 'amd64' }),
            },
            { get: (t, p) => (p in t ? t[p] : noop) },
        );

        // ── window.go (Wails bindings, namespaced by Go package) ────────────
        // Database → window.go.database.Database
        // App      → window.go.gui.App
        // Config   → window.go.main.Config
        window.go = {
            database: {
                Database: makeProxy({
                    LoadCommandHistory: asyncArr,
                    SaveCommand: asyncVoid,
                    ClearCommandHistory: asyncVoid,
                    ComputeEPCFromPosition: asyncNull,
                    ComputeStats: asyncNull,
                    LoadAllPositions: asyncArr,
                    LoadAnalysis: asyncNull,
                    SaveSessionState: asyncVoid,
                    LoadSessionState: asyncNull,
                    ClearSessionState: asyncVoid,
                }),
            },
            gui: {
                App: makeProxy({
                    ShowAlert: asyncVoid,
                    ShowQuestionDialog: () => Promise.resolve(false),
                }),
            },
            main: {
                Config: makeProxy({
                    GetLastDatabasePath: () => Promise.resolve(''), // pas d'auto-open
                    SaveLastDatabasePath: asyncVoid,
                    SaveWindowDimensions: asyncVoid,
                    LoadConfig: () => Promise.resolve({}),
                    GetStatsFilter: asyncNull,
                    SaveStatsFilter: asyncVoid,
                    SaveConfig: asyncVoid,
                    // Treat the tour as already seen so the first-run catalog modal
                    // does not auto-open and intercept clicks. Specs that test the
                    // tour open the catalog explicitly.
                    GetTourSeen: () => Promise.resolve(true),
                    SaveTourSeen: asyncVoid,
                }),
            },
        };
    });
}

/**
 * Override dynamiquement une méthode de window.go.database.Database après le chargement
 * de la page. Utile pour modifier les fixtures mid-test.
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} methodName
 * @param {unknown} returnValue  — valeur JSON-sérialisable retournée par la méthode
 */
export async function overrideDbMethod(page, methodName, returnValue) {
    await page.evaluate(
        ({ method, value }) => {
            window.go.database.Database[method] = () => Promise.resolve(value);
        },
        { method: methodName, value: returnValue },
    );
}
