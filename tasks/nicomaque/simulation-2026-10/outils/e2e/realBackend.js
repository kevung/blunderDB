/**
 * realBackend.js — branche le vrai front Svelte sur le vrai `Database` Go servi par le shim.
 *
 * Écrit à neuf, sans `installWailsMock` : la consigne de la simulation est qu'aucune réponse du
 * backend ne soit fabriquée. Seules les liaisons natives sans équivalent navigateur (dialogues,
 * fenêtre, presse-papiers) sont figées ici, et toutes sont journalisées dans `window.__calls`.
 *
 *   window.go.database.Database  → Proxy : chaque appel = POST shim /call/<Méthode>
 *   window.go.main.Config        → préférences en mémoire (langue fr, thème clair, visite vue)
 *   window.go.gui.App            → dialogues figés : la base au démarrage, le dossier de la page
 *                                  murale, SaveCSV écrit dans le dossier de sortie par le shim
 *   window.runtime               → événements locaux, presse-papiers capturé (window.__clipboard)
 */

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ shimUrl: string, dbPath: string, outDir: string, language?: string, questionAnswer?: string }} opts
 */
export async function installRealBackend(page, opts) {
    await page.addInitScript((o) => {
        const calls = [];
        window.__calls = calls;
        window.__clipboard = [];
        window.__opened = [];
        window.__dialogs = [];
        window.__questionAnswer = o.questionAnswer ?? '';
        const rec = (ns, name, args) => calls.push({ ns, name, args, t: Date.now() });
        const resolved = (v) => Promise.resolve(v);
        const noop = () => {};

        // ── Database : le vrai backend ──────────────────────────────────
        const callShim = (name, args) =>
            fetch(`${o.shimUrl}/call/${name}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(args)
            })
                .then((r) => r.json())
                .then((r) => {
                    if (r.error) throw r.error;
                    return r.result ?? null;
                });
        const database = new Proxy(
            {},
            {
                get(_t, prop) {
                    if (typeof prop !== 'string' || prop === 'then') return undefined;
                    return (...args) => {
                        rec('database', prop, args);
                        return callShim(prop, args);
                    };
                }
            }
        );

        // ── Config : préférences en mémoire ─────────────────────────────
        const prefs = {
            Language: o.language ?? 'fr',
            Theme: 'light',
            TourSeen: true,
            LastDatabasePath: o.dbPath,
            PageStep: 10,
            UIScale: 100,
            MCPHost: { on: false, port: 0, write: false },
            WatchFolder: { on: false, path: '', intervalSeconds: 0 }
        };
        const config = new Proxy(
            {},
            {
                get(_t, prop) {
                    if (typeof prop !== 'string' || prop === 'then') return undefined;
                    return (...args) => {
                        rec('config', prop, args);
                        if (prop === 'LoadConfig') return resolved({});
                        if (prop.startsWith('Get')) return resolved(prefs[prop.slice(3)] ?? null);
                        if (prop.startsWith('Save')) {
                            prefs[prop.slice(4)] = args[0];
                            return resolved(null);
                        }
                        return resolved(null);
                    };
                }
            }
        );

        // ── App : dialogues natifs figés ────────────────────────────────
        const appImpl = {
            StartupFilePath: () => resolved(o.dbPath),
            PathExists: () => resolved(true),
            IsDirectory: () => resolved(false),
            CheckForUpdate: () => resolved(null),
            OpenDirectionOutputDialog: () => resolved(o.outDir),
            SaveCSV: (name, body) =>
                fetch(`${o.shimUrl}/savecsv`, { method: 'POST', body: JSON.stringify({ Name: name, Body: body }) })
                    .then((r) => r.json())
                    .then((r) => {
                        if (r.error) throw r.error;
                        return r.result;
                    }),
            ShowAlert: (msg) => {
                window.__dialogs.push({ kind: 'alert', msg });
                return resolved(null);
            },
            ShowQuestionDialog: (title, message, buttons) => {
                window.__dialogs.push({ kind: 'question', title, message, buttons });
                return resolved(window.__questionAnswer || (buttons && buttons[0]) || '');
            },
            FolderWatchStatus: () => resolved(null),
            GetMCPHostStatus: () => resolved({}),
            ConfigureMCPHost: () => resolved({}),
            RolloutPresets: () => resolved([]),
            AssistantPresets: () => resolved([]),
            AssistantHasKey: () => resolved(false),
            BearoffStatus: () => resolved(null)
        };
        const app = new Proxy(appImpl, {
            get(t, prop) {
                if (typeof prop !== 'string' || prop === 'then') return undefined;
                const fn = t[prop] ?? (() => resolved(null));
                return (...args) => {
                    rec('app', prop, args);
                    return fn(...args);
                };
            }
        });

        window.go = { database: { Database: database }, gui: { App: app }, main: { Config: config } };

        // ── runtime : événements locaux, presse-papiers capturé ─────────
        const listeners = new Map();
        const runtimeImpl = {
            EventsOnMultiple: (name, cb, max) => {
                const l = { cb, left: max > 0 ? max : Infinity };
                if (!listeners.has(name)) listeners.set(name, new Set());
                listeners.get(name).add(l);
                return () => listeners.get(name)?.delete(l);
            },
            EventsOff: (name) => listeners.delete(name),
            EventsOffAll: () => listeners.clear(),
            EventsEmit: (name, ...data) => {
                for (const l of listeners.get(name) ?? []) {
                    l.cb(...data);
                    if (--l.left <= 0) listeners.get(name).delete(l);
                }
            },
            ClipboardSetText: (s) => {
                window.__clipboard.push(s);
                return resolved(true);
            },
            ClipboardGetText: () => resolved(window.__clipboard.at(-1) ?? ''),
            BrowserOpenURL: (u) => {
                window.__opened.push(u);
            },
            WindowGetSize: () => resolved({ w: window.innerWidth, h: window.innerHeight }),
            WindowGetPosition: () => resolved({ x: 0, y: 0 }),
            WindowIsFullscreen: () => resolved(false),
            WindowIsMaximised: () => resolved(false),
            WindowIsMinimised: () => resolved(false),
            WindowIsNormal: () => resolved(true),
            CanResolveFilePaths: () => resolved(false),
            ResolveFilePaths: () => resolved([]),
            ScreenGetAll: () => resolved([]),
            Environment: () => resolved({ buildType: 'dev', platform: 'linux', arch: 'amd64' })
        };
        window.runtime = new Proxy(runtimeImpl, {
            get(t, prop) {
                if (typeof prop !== 'string' || prop === 'then') return undefined;
                const fn = t[prop] ?? noop;
                return (...args) => {
                    if (!prop.startsWith('Log')) rec('runtime', prop, args);
                    return fn(...args);
                };
            }
        });
    }, opts);
}

/** Appelle le vrai backend depuis la spec, hors interface (vérifications d'oracle). */
export async function shimCall(shimUrl, method, ...args) {
    const r = await fetch(`${shimUrl}/call/${method}`, { method: 'POST', body: JSON.stringify(args) });
    const j = await r.json();
    if (j.error) throw new Error(`${method}: ${j.error}`);
    return j.result;
}

/** Les avertissements de page murale que le GUI aurait remontés en barre d'état. */
export async function shimPageWarnings(shimUrl) {
    return (await fetch(`${shimUrl}/warnings`)).json();
}
