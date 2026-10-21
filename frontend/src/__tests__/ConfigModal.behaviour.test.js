/**
 * ConfigModal.svelte : chaque réglage modifié appelle le bon binding avec la bonne valeur, une
 * saisie invalide n'atteint pas le binding, une étape refusée ne persiste rien, Échap ferme.
 * Le composant réel est monté ; seuls les bindings Wails (window.go) sont simulés.
 */
import { must } from './helpers/must.js';
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: () => () => {},
    EventsOff: () => {},
    WindowSetTitle: () => {},
    Quit: () => {},
    ClipboardGetText: () => Promise.resolve('')
}));

import ConfigModal from '../components/ConfigModal.svelte';
import { t } from '../i18n';
import { databasePathStore } from '../stores/databaseStore.js';
import { confirmModalStore, resolveConfirm } from '../services/confirmService.js';

/** @type {Record<string, Record<string, import('vitest').Mock>>} */
let go;

/** Every binding resolves to null unless the test names it; a named one is a spy. */
function installGo(/** @type {Record<string, Record<string, (...a: any[]) => any>>} */ overrides) {
    go = {};
    const ns = (/** @type {string} */ path) =>
        new Proxy(
            {},
            {
                get(_t, method) {
                    if (typeof method !== 'string' || method === 'then') return undefined;
                    go[path] ??= {};
                    go[path][method] ??= vi.fn(overrides[path]?.[method] ?? (() => Promise.resolve(null)));
                    return go[path][method];
                }
            }
        );
    const pkgs = new Proxy({}, { get: (_t, pkg) => new Proxy({}, { get: (_u, cls) => ns(`${String(pkg)}.${String(cls)}`) }) });
    // @ts-ignore
    window.go = pkgs;
}

const spy = (/** @type {string} */ path, /** @type {string} */ method) => go[path]?.[method] ?? vi.fn();
const L = (/** @type {string} */ key, /** @type {any} */ params) => get(t)(key, params);

function defaults() {
    return {
        'main.Config': {
            GetLikeLimit: () => Promise.resolve(30),
            GetLikeMaxDistance: () => Promise.resolve(0),
            GetCommentAuthor: () => Promise.resolve('Alice'),
            GetCheckForUpdates: () => Promise.resolve(false),
            GetGammonNetDisplayPly: () => Promise.resolve(2),
            GetGammonNetAnalysisPly: () => Promise.resolve(2),
            GetGammonNetPruneK: () => Promise.resolve(12),
            GetGammonNetCandidates: () => Promise.resolve(10),
            GetGammonNetAutoAnalyze: () => Promise.resolve(false),
            GetWatchFolder: () => Promise.resolve({ on: false, path: '', intervalSeconds: 0 }),
            GetBearoffTSPath: () => Promise.resolve(''),
            GetBearoffRate: () => Promise.resolve(0),
            GetBearoffCores: () => Promise.resolve(0)
        },
        'gui.App': {
            GetIssuerIdentity: () => Promise.resolve({ present: true, name: 'Club', fingerprint: 'ab:cd' }),
            BearoffStatus: () => Promise.resolve(null),
            FolderWatchStatus: () => Promise.resolve({ running: false, folder: '' }),
            StartFolderWatch: (folder) => Promise.resolve({ running: true, folder }),
            StopFolderWatch: () => Promise.resolve()
        },
        'database.Database': {
            GetLibrarySettings: () => Promise.resolve({ errorThresholdMP: 50, blunderThresholdMP: 100 })
        }
    };
}

/** @param {Record<string, Record<string, (...a: any[]) => any>>} [extra] */
async function mount(extra = {}, tab = '') {
    const base = defaults();
    for (const [k, v] of Object.entries(extra)) base[k] = { ...(base[k] ?? {}), ...v };
    installGo(base);
    const onClose = vi.fn();
    const r = render(ConfigModal, { visible: true, onClose });
    await flush();
    const doc = r.container.ownerDocument;
    if (tab) {
        const tabs = [...doc.querySelectorAll('[role="tab"]')];
        const target = tabs.find((b) => b.textContent?.trim() === L(tab));
        await fireEvent.click(must(target));
        await flush();
    }
    return { ...r, doc, onClose };
}

async function flush() {
    for (let i = 0; i < 6; i++) {
        await Promise.resolve();
        await tick();
    }
}

function input(/** @type {Document} */ doc, /** @type {string} */ id) {
    return /** @type {HTMLInputElement} */ (must(doc.getElementById(id)));
}

function button(/** @type {Document} */ doc, /** @type {string} */ label) {
    const b = [...doc.querySelectorAll('button')].find((x) => x.textContent?.trim() === label);
    return /** @type {HTMLButtonElement} */ (must(b));
}

beforeEach(() => {
    Element.prototype.animate ??= () => ({ cancel() {}, finished: Promise.resolve(), onfinish: null });
    databasePathStore.set('');
});
afterEach(() => {
    cleanup();
    resolveConfirm(false);
    databasePathStore.set('');
});

describe('ConfigModal — Interface tab', () => {
    test('loads the saved settings into the fields on opening', async () => {
        const { doc } = await mount({ 'main.Config': { GetLikeLimit: () => Promise.resolve(77), GetCommentAuthor: () => Promise.resolve('Bob') } });
        expect(input(doc, 'config-like-limit').value).toBe('77');
        expect(input(doc, 'config-comment-author').value).toBe('Bob');
    });

    test('the neighbour cap is saved clamped to its maximum', async () => {
        const { doc } = await mount();
        await fireEvent.change(input(doc, 'config-like-limit'), { target: { value: '9999' } });
        await flush();
        expect(spy('main.Config', 'SaveLikeLimit')).toHaveBeenCalledWith(500);
        expect(input(doc, 'config-like-limit').value).toBe('500');
    });

    test('a neighbour cap below one is raised to one', async () => {
        const { doc } = await mount();
        await fireEvent.change(input(doc, 'config-like-limit'), { target: { value: '-4' } });
        await flush();
        expect(spy('main.Config', 'SaveLikeLimit')).toHaveBeenCalledWith(1);
    });

    test('a non-numeric neighbour cap is refused: nothing is saved', async () => {
        const { doc } = await mount();
        await fireEvent.change(input(doc, 'config-like-limit'), { target: { value: '' } });
        await flush();
        expect(spy('main.Config', 'SaveLikeLimit')).not.toHaveBeenCalled();
        expect(input(doc, 'config-like-limit').value).toBe('30');
    });

    test('a cap already at its maximum shows the maximum again after an over-limit entry', async () => {
        const { doc } = await mount({ 'main.Config': { GetLikeLimit: () => Promise.resolve(500) } });
        await fireEvent.change(input(doc, 'config-like-limit'), { target: { value: '800' } });
        await flush();
        expect(input(doc, 'config-like-limit').value).toBe('500');
    });

    test('a negative maximum distance is saved as zero', async () => {
        const { doc } = await mount();
        await fireEvent.change(input(doc, 'config-like-max-distance'), { target: { value: '-3' } });
        await flush();
        expect(spy('main.Config', 'SaveLikeMaxDistance')).toHaveBeenCalledWith(0);
    });

    test('the comment author is saved, then re-read from the configuration', async () => {
        const { doc } = await mount({ 'main.Config': { SaveCommentAuthor: () => Promise.resolve() } });
        go['main.Config'].GetCommentAuthor.mockResolvedValue('Carol (trimmed)');
        await fireEvent.change(input(doc, 'config-comment-author'), { target: { value: 'Carol ' } });
        await flush();
        expect(spy('main.Config', 'SaveCommentAuthor')).toHaveBeenCalledWith('Carol ');
        expect(input(doc, 'config-comment-author').value).toBe('Carol (trimmed)');
    });

    test('ticking the update check saves true', async () => {
        const { doc } = await mount();
        await fireEvent.click(input(doc, 'config-check-for-updates'));
        await flush();
        expect(spy('main.Config', 'SaveCheckForUpdates')).toHaveBeenCalledWith(true);
    });

    test('Escape closes the modal', async () => {
        const { doc, onClose } = await mount();
        await fireEvent.keyDown(input(doc, 'config-like-limit'), { key: 'Escape' });
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('the Close button closes without saving anything', async () => {
        const { doc, onClose } = await mount();
        await fireEvent.click(button(doc, L('common.close')));
        expect(onClose).toHaveBeenCalledTimes(1);
        const saves = Object.entries(go['main.Config'] ?? {}).filter(([m, f]) => m.startsWith('Save') && f.mock.calls.length);
        expect(saves).toEqual([]);
    });
});

describe('ConfigModal — gammonNet tab', () => {
    test('the display and analysis depths are saved as numbers', async () => {
        const { doc } = await mount({}, 'config.gammonnetTitle');
        await fireEvent.change(input(doc, 'config-gn-display-ply'), { target: { value: '3' } });
        await fireEvent.change(input(doc, 'config-gn-analysis-ply'), { target: { value: '1' } });
        await flush();
        expect(spy('main.Config', 'SaveGammonNetDisplayPly')).toHaveBeenCalledWith(3);
        expect(spy('main.Config', 'SaveGammonNetAnalysisPly')).toHaveBeenCalledWith(1);
    });

    test('a pruning width below one is refused; a valid one is saved', async () => {
        const { doc } = await mount({}, 'config.gammonnetTitle');
        await fireEvent.change(input(doc, 'config-gn-prune-k'), { target: { value: '0' } });
        await flush();
        expect(spy('main.Config', 'SaveGammonNetPruneK')).not.toHaveBeenCalled();
        expect(input(doc, 'config-gn-prune-k').value).toBe('12');
        await fireEvent.change(input(doc, 'config-gn-prune-k'), { target: { value: '20' } });
        await flush();
        expect(spy('main.Config', 'SaveGammonNetPruneK')).toHaveBeenCalledWith(20);
    });

    test('a candidate count below one is refused', async () => {
        const { doc } = await mount({}, 'config.gammonnetTitle');
        await fireEvent.change(input(doc, 'config-gn-candidates'), { target: { value: '-1' } });
        await flush();
        expect(spy('main.Config', 'SaveGammonNetCandidates')).not.toHaveBeenCalled();
    });

    test('auto-analysis is saved when ticked', async () => {
        const { doc } = await mount({}, 'config.gammonnetTitle');
        await fireEvent.click(input(doc, 'config-gn-auto-analyze'));
        await flush();
        expect(spy('main.Config', 'SaveGammonNetAutoAnalyze')).toHaveBeenCalledWith(true);
    });
});

describe('ConfigModal — Library tab', () => {
    test('a threshold typed in equity is saved in millipoints', async () => {
        databasePathStore.set('/db/a.db');
        const { doc } = await mount({}, 'config.libraryTitle');
        expect(input(doc, 'config-error-threshold').value).toBe('0.050');
        await fireEvent.change(input(doc, 'config-error-threshold'), { target: { value: '0.04' } });
        await flush();
        expect(spy('database.Database', 'SaveLibrarySettings')).toHaveBeenCalledWith({ errorThresholdMP: 40, blunderThresholdMP: 100 });
    });

    test('a pair refused by the storage shows its message and keeps the previous value', async () => {
        databasePathStore.set('/db/a.db');
        const { doc } = await mount({ 'database.Database': { SaveLibrarySettings: () => Promise.reject('error threshold above blunder threshold') } }, 'config.libraryTitle');
        await fireEvent.change(input(doc, 'config-error-threshold'), { target: { value: '0.2' } });
        await flush();
        expect(doc.body.textContent).toContain('error threshold above blunder threshold');
        expect(input(doc, 'config-error-threshold').value).toBe('0.050');
    });

    test('a preset saves its own pair', async () => {
        databasePathStore.set('/db/a.db');
        const { doc } = await mount({}, 'config.libraryTitle');
        await fireEvent.click(button(doc, L('config.thresholdPreset_xg')));
        await flush();
        expect(spy('database.Database', 'SaveLibrarySettings')).toHaveBeenCalledWith({ errorThresholdMP: 20, blunderThresholdMP: 80 });
    });

    test('without an open database there is no threshold field to save', async () => {
        const { doc } = await mount({}, 'config.libraryTitle');
        expect(doc.getElementById('config-error-threshold')).toBeNull();
        expect(spy('database.Database', 'GetLibrarySettings')).not.toHaveBeenCalled();
    });
});

describe('ConfigModal — Watched folder tab', () => {
    test('a suggested folder is saved and switched on', async () => {
        const { doc } = await mount({ 'gui.App': { SuggestWatchFolder: () => Promise.resolve('/home/u/xg') } }, 'config.watchTitle');
        await fireEvent.click(button(doc, L('config.watchSuggest')));
        await flush();
        expect(spy('main.Config', 'SaveWatchFolder')).toHaveBeenCalledWith(true, '/home/u/xg', 0);
        expect(spy('gui.App', 'StartFolderWatch')).toHaveBeenCalledWith('/home/u/xg', 0);
        expect(input(doc, 'config-watch-on').checked).toBe(true);
    });

    test('no suggestion found: a message, nothing saved', async () => {
        const { doc } = await mount({ 'gui.App': { SuggestWatchFolder: () => Promise.resolve('') } }, 'config.watchTitle');
        await fireEvent.click(button(doc, L('config.watchSuggest')));
        await flush();
        expect(doc.body.textContent).toContain(L('config.watchNoSuggestion'));
        expect(spy('main.Config', 'SaveWatchFolder')).not.toHaveBeenCalled();
    });

    test('a refused folder unticks the box and shows the reason', async () => {
        const { doc } = await mount(
            {
                'main.Config': {
                    GetWatchFolder: () => Promise.resolve({ on: false, path: '/gone', intervalSeconds: 0 }),
                    SaveWatchFolder: () => Promise.reject(new Error('folder does not exist'))
                }
            },
            'config.watchTitle'
        );
        const box = input(doc, 'config-watch-on');
        expect(box.disabled).toBe(false);
        await fireEvent.click(box);
        await flush();
        expect(doc.body.textContent).toContain('folder does not exist');
        expect(input(doc, 'config-watch-on').checked).toBe(false);
    });
});

describe('ConfigModal — Identity tab', () => {
    test('renaming on blur calls SetIssuerName with the trimmed name', async () => {
        const { doc } = await mount({ 'gui.App': { SetIssuerName: (n) => Promise.resolve({ present: true, name: n, fingerprint: 'ab:cd' }) } }, 'config.identityTitle');
        const field = input(doc, 'config-identity-name');
        field.value = '  Club B ';
        await fireEvent.blur(field);
        await flush();
        expect(spy('gui.App', 'SetIssuerName')).toHaveBeenCalledWith('Club B');
    });

    test('an unchanged or empty name is not sent', async () => {
        const { doc } = await mount({}, 'config.identityTitle');
        const field = input(doc, 'config-identity-name');
        await fireEvent.blur(field);
        field.value = '   ';
        await fireEvent.blur(field);
        await flush();
        expect(spy('gui.App', 'SetIssuerName')).not.toHaveBeenCalled();
    });

    test('regenerating takes two steps; cancelling the second regenerates nothing', async () => {
        const { doc } = await mount({}, 'config.identityTitle');
        await fireEvent.click(button(doc, L('config.identityRegenerate')));
        await flush();
        await fireEvent.click(button(doc, L('common.cancel')));
        await flush();
        expect(spy('gui.App', 'RegenerateIssuerIdentity')).not.toHaveBeenCalled();
        await fireEvent.click(button(doc, L('config.identityRegenerate')));
        await flush();
        await fireEvent.click(button(doc, L('config.identityRegenerateConfirm')));
        await flush();
        expect(spy('gui.App', 'RegenerateIssuerIdentity')).toHaveBeenCalledWith('Club');
    });

    test('importing a protected key without a passphrase is refused before the call', async () => {
        const { doc } = await mount({ 'gui.App': { PickIdentityFile: () => Promise.resolve({ cancelled: false, path: '/k.key', needsPassphrase: true }) } }, 'config.identityTitle');
        await fireEvent.click(button(doc, L('config.identityImport')));
        await flush();
        expect(doc.body.textContent).toContain(L('config.identityPassphraseNeeded'));
        expect(spy('gui.App', 'ImportIssuerIdentity')).not.toHaveBeenCalled();
    });

    test('the export sends the typed passphrase and reports the path', async () => {
        const { doc } = await mount({ 'gui.App': { ExportIssuerIdentity: () => Promise.resolve('/out/id.key') } }, 'config.identityTitle');
        await fireEvent.input(input(doc, 'config-identity-passphrase'), { target: { value: 's3cret' } });
        await fireEvent.click(button(doc, L('config.identityExport')));
        await flush();
        expect(spy('gui.App', 'ExportIssuerIdentity')).toHaveBeenCalledWith('s3cret');
        expect(doc.body.textContent).toContain(L('config.identityExported', { path: '/out/id.key' }));
        expect(input(doc, 'config-identity-passphrase').value).toBe('');
    });
});

describe('ConfigModal — Bearoff tab', () => {
    const plan = { data_dir: '/d', files: [{ name: 'ts-06-11.bin', size: 2e6, verdict: 'ok' }], candidates: [], default_cores: 2, cores: 2 };

    test('deleting a table asks first; a refusal deletes nothing', async () => {
        const { doc } = await mount({ 'gui.App': { BearoffStatus: () => Promise.resolve({ active_domain: 0 }), BearoffPlan: () => Promise.resolve(plan) } }, 'config.bearoffTitle');
        await fireEvent.click(button(doc, L('common.delete')));
        await flush();
        expect(get(confirmModalStore)).not.toBeNull();
        resolveConfirm(false);
        await flush();
        expect(spy('gui.App', 'DeleteBearoffTable')).not.toHaveBeenCalled();
    });

    test('a confirmed deletion names the file', async () => {
        const { doc } = await mount({ 'gui.App': { BearoffStatus: () => Promise.resolve({ active_domain: 0 }), BearoffPlan: () => Promise.resolve(plan) } }, 'config.bearoffTitle');
        await fireEvent.click(button(doc, L('common.delete')));
        await flush();
        resolveConfirm(true);
        await waitFor(() => expect(spy('gui.App', 'DeleteBearoffTable')).toHaveBeenCalledWith('ts-06-11.bin'));
    });
});
