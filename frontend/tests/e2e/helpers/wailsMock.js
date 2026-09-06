/**
 * helpers/wailsMock.js
 *
 * Injecte un mock de window.go (bindings Database, Config, App) et de
 * window.runtime (Wails runtime) dans une page Playwright via addInitScript.
 *
 * Usage :
 *   import { installWailsMock, overrideDbMethod } from './helpers/wailsMock.js';
 *   await installWailsMock(page);                                    // defaults
 *   await installWailsMock(page, { database: { ListPositionIDs: [...] } }); // constantes par méthode
 *   await overrideDbMethod(page, 'ComputeEPCFromPosition', result);   // per-method override
 *
 * Stratégie : le script est sérialisé et injecté AVANT tout script de la page
 * (Playwright garantit cela). Les Proxy assurent qu'un appel à une méthode
 * non listée retourne Promise.resolve(null) au lieu de planter.
 *
 * Chaque appel de binding est journalisé dans window.__wailsCalls
 * ({ ns, method, args }) : une spec peut ainsi vérifier ce que le frontend a
 * demandé au backend (voir getWailsCalls).
 */

/**
 * @typedef {Object} MockOverrides
 * @property {Record<string, unknown>} [database]  valeur constante retournée par window.go.database.Database.<m>
 * @property {Record<string, unknown>} [app]       idem pour window.go.gui.App
 * @property {Record<string, unknown>} [config]    idem pour window.go.main.Config
 * @property {Record<string, unknown>} [runtime]   idem pour window.runtime (ClipboardGetText…)
 */

/**
 * @param {import('@playwright/test').Page} page
 * @param {MockOverrides} [overrides]  valeurs JSON-sérialisables, une par méthode
 */
export async function installWailsMock(page, overrides = {}) {
    await page.addInitScript((overrides) => {
        // ── Helpers locaux ───────────────────────────────────────────────────
        const noop = () => {};
        const asyncNull = () => Promise.resolve(null);
        const asyncVoid = () => Promise.resolve(undefined);
        const asyncArr = () => Promise.resolve([]);
        const constant = (value) => () => Promise.resolve(value);

        window.__wailsCalls = [];
        function record(ns, method, args) {
            const calls = window.__wailsCalls;
            calls.push({ ns, method, args });
            if (calls.length > 500) calls.shift();
        }

        /**
         * LoadPositionsByIDs est la seule méthode que le mock ne peut pas
         * servir par une constante : depuis D.8 (#208) la recherche, le
         * drill-down Stats et le deck Anki ne rapportent QUE des ids, et c'est
         * elle qui charge la fenêtre affichée. Une constante ferait montrer au
         * plateau la bibliothèque entière quels que soient les ids demandés —
         * la spec dirait « 3 / 3 » là où l'app affiche « 1 / 2 ».
         *
         * Sa valeur déclarée est donc lue comme LE CATALOGUE du backend
         * factice : elle répond aux ids demandés, dans l'ordre demandé. Exposée
         * sur window pour que les overrides d'après chargement
         * (overrideDbMethod, overrideDbMethodThen) suivent la même règle.
         */
        window.__mockPositionsByIDs = (catalog) => (ids) =>
            Promise.resolve((Array.isArray(ids) ? ids : []).map((id) => (Array.isArray(catalog) ? catalog.find((p) => p && p.id === id) : undefined)).filter(Boolean));

        /** Applique les constantes d'override sur un objet de base. */
        function withOverrides(base, ns) {
            for (const [method, value] of Object.entries(overrides[ns] || {})) {
                base[method] = method === 'LoadPositionsByIDs' && Array.isArray(value) ? window.__mockPositionsByIDs(value) : constant(value);
            }
            return base;
        }

        /** Proxy qui journalise chaque appel et retourne asyncNull pour toute méthode non définie. */
        function makeProxy(base, ns) {
            return new Proxy(base, {
                get(target, prop) {
                    const fn = prop in target ? target[prop] : asyncNull;
                    if (typeof fn !== 'function') return fn;
                    return (...args) => {
                        record(ns, String(prop), args);
                        return fn(...args);
                    };
                }
            });
        }

        // ── window.runtime ───────────────────────────────────────────────────
        window.runtime = new Proxy(
            withOverrides(
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
                    Environment: () => Promise.resolve({ buildType: 'dev', platform: 'linux', arch: 'amd64' })
                },
                'runtime'
            ),
            {
                get(target, prop) {
                    const fn = prop in target ? target[prop] : noop;
                    if (typeof fn !== 'function') return fn;
                    return (...args) => {
                        record('runtime', String(prop), args);
                        return fn(...args);
                    };
                }
            }
        );

        // ── window.go (Wails bindings, namespaced by Go package) ────────────
        // Database → window.go.database.Database
        // App      → window.go.gui.App
        // Config   → window.go.main.Config
        window.go = {
            database: {
                Database: makeProxy(
                    withOverrides(
                        {
                            LoadCommandHistory: asyncArr,
                            SaveCommand: asyncVoid,
                            ClearCommandHistory: asyncVoid,
                            ComputeEPCFromPosition: asyncNull,
                            ComputeStats: asyncNull,
                            ListPositionIDs: asyncArr,
                            LoadPositionsByIDs: asyncArr,
                            // D.8 (#208) : la recherche, Stats et Anki ne
                            // rendent plus que des ids ; sans stub la méthode
                            // renverrait null et toute recherche paraîtrait vide.
                            LoadPositionIDsByFilters: asyncArr,
                            LoadAnalysis: asyncNull,
                            SaveSessionState: asyncVoid,
                            LoadSessionState: asyncNull,
                            ClearSessionState: asyncVoid
                        },
                        'database'
                    ),
                    'database'
                )
            },
            gui: {
                App: makeProxy(
                    withOverrides(
                        {
                            ShowAlert: asyncVoid,
                            ShowQuestionDialog: () => Promise.resolve(false)
                        },
                        'app'
                    ),
                    'app'
                )
            },
            main: {
                Config: makeProxy(
                    withOverrides(
                        {
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
                            SaveTourSeen: asyncVoid
                        },
                        'config'
                    ),
                    'config'
                )
            }
        };
    }, overrides);
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
            window.go.database.Database[method] = method === 'LoadPositionsByIDs' && Array.isArray(value) ? window.__mockPositionsByIDs(value) : () => Promise.resolve(value);
        },
        { method: methodName, value: returnValue }
    );
}

/**
 * Fait répondre une méthode Database selon son premier argument : `table`
 * associe à `String(arg)` la valeur JSON-sérialisable à renvoyer, `fallback`
 * couvre les autres appels. C'est ce qui donne à chaque position sa propre
 * analyse (LoadAnalysis(id)) là où une constante les ferait toutes se
 * ressembler.
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} methodName
 * @param {Record<string, unknown>} table
 * @param {unknown} [fallback]
 */
export async function overrideDbMethodByArg(page, methodName, table, fallback = null) {
    await page.evaluate(
        ({ method, table, fallback }) => {
            window.go.database.Database[method] = (arg) => Promise.resolve(Object.hasOwn(table, String(arg)) ? table[String(arg)] : fallback);
        },
        { method: methodName, table, fallback }
    );
}

/**
 * Fait répondre `methodName` par `returnValue`, puis, dès son premier appel,
 * remplace d'autres méthodes Database par des constantes — le mock simule ainsi
 * un backend dont l'état change après une écriture (une position enregistrée
 * apparaît ensuite dans ListPositionIDs, par exemple).
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} methodName
 * @param {unknown} returnValue
 * @param {Record<string, unknown>} afterCall  — { méthode: valeur constante }
 */
export async function overrideDbMethodThen(page, methodName, returnValue, afterCall) {
    await page.evaluate(
        ({ method, value, after }) => {
            window.go.database.Database[method] = () => {
                for (const [m, v] of Object.entries(after)) {
                    window.go.database.Database[m] = m === 'LoadPositionsByIDs' && Array.isArray(v) ? window.__mockPositionsByIDs(v) : () => Promise.resolve(v);
                }
                return Promise.resolve(value);
            };
        },
        { method: methodName, value: returnValue, after: afterCall }
    );
}

/**
 * Appels de bindings journalisés depuis le chargement de la page, filtrés par
 * nom de méthode (tous espaces de noms confondus si `methodName` est omis).
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} [methodName]
 * @returns {Promise<Array<{ ns: string, method: string, args: unknown[] }>>}
 */
export async function getWailsCalls(page, methodName) {
    return page.evaluate((method) => (window.__wailsCalls || []).filter((c) => !method || c.method === method), methodName ?? null);
}

/**
 * Écarte l'écran d'accueil s'il est là.
 *
 * L'accueil (#284) couvre l'application tant qu'aucune base n'est ouverte, et
 * intercepte donc les clics sur les onglets. Une spec qui travaille sur le
 * plateau brouillon — Eval, EPC — est exactement le cas pour lequel le bouton
 * « écarter » existe : elle fait ici le geste que l'utilisateur ferait, plutôt
 * que d'ouvrir une base dont elle n'a pas besoin.
 *
 * Sans effet quand une base est montée (openLibraryMock) : l'accueil n'est
 * alors pas rendu.
 *
 * @param {import('@playwright/test').Page} page
 */
export async function dismissHomeScreen(page) {
    const dismiss = page.locator('[data-testid="home-dismiss"]');
    if (await dismiss.count()) await dismiss.click();
}
