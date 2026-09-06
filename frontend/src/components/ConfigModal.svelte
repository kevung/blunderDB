<script>
    import { get } from 'svelte/store';
    import { configInitialTabStore, statusBarTextStore } from '../stores/uiStore';
    import Modal from './Modal.svelte';
    import { t, tMsg, language, setLanguage, LOCALES, LANGUAGE_LABELS } from '../i18n';
    import { boardColorsStore, setBoardColor, resetBoardColors } from '../stores/boardColorsStore';
    import { uiScaleStore, setUIScale, previewUIScale, MIN_UI_SCALE, MAX_UI_SCALE, UI_SCALE_STEP } from '../stores/uiScaleStore';
    import { panelPositionStore, setPanelPosition, PANEL_BOTTOM, PANEL_SIDE, PANEL_AUTO } from '../stores/panelLayoutStore';
    import { confirmAction } from '../services/confirmService.js';
    import {
        GetIssuerIdentity,
        SetIssuerName,
        ExportIssuerIdentity,
        PickIdentityFile,
        ImportIssuerIdentity,
        RegenerateIssuerIdentity,
        BearoffStatus,
        BearoffPlan,
        GenerateBearoffTable,
        PauseBearoffGeneration,
        CancelBearoffGeneration,
        DiscardBearoffCheckpoint,
        DeleteBearoffTable,
        OpenBearoffFileDialog,
        StartGammonNetBatch,
        StartGammonNetStaleBatch,
        OpenLogsFolder,
        OpenPositionFolderDialog,
        SuggestWatchFolder
    } from '../../wailsjs/go/gui/App.js';
    import { Vacuum, RepairAnalyses, CountPositionsWithoutAnalysis, CountPositionsWithStaleGammonNet } from '../../wailsjs/go/database/Database.js';
    import { GetBearoffTSPath, SaveBearoffTSPath, GetBearoffRate, SaveBearoffRate, GetBearoffCores, SaveBearoffCores } from '../../wailsjs/go/main/Config.js';
    import { bearoffProgressStore, bearoffErrorStore, remainingSeconds } from '../stores/bearoffStore.js';
    import {
        GetGammonNetDisplayPly,
        SaveGammonNetDisplayPly,
        GetGammonNetAnalysisPly,
        SaveGammonNetAnalysisPly,
        GetGammonNetPruneK,
        SaveGammonNetPruneK,
        GetGammonNetCandidates,
        SaveGammonNetCandidates,
        GetGammonNetAutoAnalyze,
        GetWatchFolder,
        SaveGammonNetAutoAnalyze,
        GetCheckForUpdates,
        SaveCheckForUpdates
    } from '../../wailsjs/go/main/Config.js';
    import { watchStatusStore } from '../stores/watchStore.js';
    import { saveWatchSetting, refreshWatchStatus } from '../services/watchService.js';
    import { EventsOn } from '../../wailsjs/runtime/runtime.js';
    import { onDestroy } from 'svelte';
    import { logger } from '../utils/logger.js';

    let { visible = false, onClose } = $props();

    // The issuer identity signs watermarks. It is created on the first watermarked export,
    // so this section reports "not yet" rather than minting a key just because someone
    // opened the settings. See ADR-0007.
    let identity = $state(null);
    let identityPassphrase = $state('');
    let identityMessage = $state('');
    let identityError = $state('');
    // Regenerating is a two-step gesture: the first click only reveals what it does and,
    // more to the point, what it does not do. See RegenerateIssuerIdentity.
    let confirmingRegenerate = $state(false);

    // Compacting the database file (Database.Vacuum) can take a while on a large
    // database and needs headroom on disk, so it is confirmed like a destructive
    // action even though nothing is deleted; the result (or a failure) is reported
    // through the status bar rather than inline, since the modal is usually closed
    // by the time a big VACUUM finishes.
    let vacuumBusy = $state(false);
    let repairBusy = $state(false);

    // Three concerns, and the third is not a preference at all: language, scale and colours
    // are settings one adjusts, whereas the identity is an object one manages, with its own
    // verbs. Keeping them in one column meant eighteen rows and up to seven buttons at
    // different heights; tabs give each concern its own action area and leave a single
    // primary button, always in the same place.
    const TABS = [
        { id: 'interface', labelKey: 'config.interface' },
        { id: 'colors', labelKey: 'config.colors' },
        { id: 'bearoff', labelKey: 'config.bearoffTitle' },
        { id: 'gammonnet', labelKey: 'config.gammonnetTitle' },
        { id: 'watch', labelKey: 'config.watchTitle' },
        { id: 'identity', labelKey: 'config.identityTitle' }
    ];
    let activeTab = $state('interface');

    $effect(() => {
        if (visible) {
            // A caller may request a specific tab (EPC panel → Bearoff).
            const requested = get(configInitialTabStore);
            if (requested) {
                activeTab = requested;
                configInitialTabStore.set(null);
            }
            identityMessage = '';
            identityError = '';
            confirmingRegenerate = false;
            GetIssuerIdentity()
                .then((info) => (identity = info))
                .catch((error) => logger.error('Error loading issuer identity:', error));
        }
    });

    async function renameIdentity(event) {
        const name = event.currentTarget.value.trim();
        if (!name || name === identity?.name) return;
        try {
            identity = await SetIssuerName(name);
            identityMessage = '';
            identityError = '';
        } catch (error) {
            identityError = String(error);
        }
    }

    async function exportIdentity() {
        identityMessage = '';
        identityError = '';
        try {
            const path = await ExportIssuerIdentity(identityPassphrase);
            if (!path) return; // dialog cancelled
            identity = await GetIssuerIdentity();
            identityPassphrase = '';
            identityMessage = $t('config.identityExported', { path });
        } catch (error) {
            identityError = String(error);
        }
    }

    async function regenerateIdentity() {
        identityMessage = '';
        identityError = '';
        try {
            identity = await RegenerateIssuerIdentity(identity?.name ?? '');
            confirmingRegenerate = false;
            identityMessage = $t('config.identityRegenerated', { fingerprint: identity.fingerprint });
        } catch (error) {
            identityError = String(error);
        }
    }

    async function importIdentity() {
        identityMessage = '';
        identityError = '';
        try {
            const pick = await PickIdentityFile();
            if (pick.cancelled) return;
            if (pick.needsPassphrase && !identityPassphrase) {
                identityError = $t('config.identityPassphraseNeeded');
                return;
            }
            identity = await ImportIssuerIdentity(pick.path, identityPassphrase);
            identityPassphrase = '';
            identityMessage = $t('config.identityImported', { name: identity.name });
        } catch (error) {
            identityError = String(error);
        }
    }

    // Board colour settings, in display order. Each maps a store key to a label.
    const COLOR_SETTINGS = [
        { key: 'background', labelKey: 'config.colorBackground' },
        { key: 'border', labelKey: 'config.colorBorder' },
        { key: 'point1', labelKey: 'config.colorPoint1' },
        { key: 'point2', labelKey: 'config.colorPoint2' },
        { key: 'checker1', labelKey: 'config.colorChecker1' },
        { key: 'checker2', labelKey: 'config.colorChecker2' },
        { key: 'dice', labelKey: 'config.colorDice' },
        { key: 'diceDot', labelKey: 'config.colorDiceDot' },
        { key: 'cube', labelKey: 'config.colorCube' }
    ];

    function onLanguageChange(event) {
        setLanguage(event.currentTarget.value);
    }

    function onColorChange(key, event) {
        setBoardColor(key, event.currentTarget.value);
    }

    // Live, lightweight preview while dragging (CSS zoom only)...
    function onUIScaleInput(event) {
        previewUIScale(Number(event.currentTarget.value));
    }

    // ...and the expensive board re-fit + persistence only once, on release.
    function onUIScaleChange(event) {
        setUIScale(Number(event.currentTarget.value));
    }

    const PANEL_POSITION_OPTIONS = [
        { value: PANEL_BOTTOM, labelKey: 'config.panelPositionBottom' },
        { value: PANEL_SIDE, labelKey: 'config.panelPositionSide' },
        { value: PANEL_AUTO, labelKey: 'config.panelPositionAuto' }
    ];

    function onPanelPositionChange(event) {
        setPanelPosition(event.currentTarget.value);
    }

    // The Bearoff tab (ADR-0027, #308). Nothing is downloaded and nothing is
    // embedded: every table is generated here, and this tab is where the user
    // sees what that costs before asking for it.
    /** @type {any} */
    let bearoff = $state(null);
    /** @type {any} */
    let bearoffPlan = $state(null);
    let bearoffExternal = $state('');
    // The domain to generate, named by its label ("TS-06-11", "OS-08"): one
    // picker for both axes, because each row already says which it is and two
    // blocks would only ask the user to know the difference first.
    let bearoffDomain = $state('TS-06-11');
    let bearoffCores = $state(0);
    let bearoffRate = $state(0);
    // A run outlives this modal, so its progress lives in a store.
    const bearoffProgress = bearoffProgressStore;
    const bearoffErrorStoreRef = bearoffErrorStore;

    async function refreshBearoff() {
        try {
            bearoff = await BearoffStatus();
            bearoffExternal = await GetBearoffTSPath();
            bearoffRate = await GetBearoffRate();
            const cores = await GetBearoffCores();
            if (bearoffCores === 0 && cores > 0) bearoffCores = cores;
            bearoffPlan = await BearoffPlan(bearoffRate, bearoffCores);
        } catch (error) {
            logger.error('Error loading bearoff status:', error);
        }
    }

    $effect(() => {
        if (visible) {
            bearoffErrorStore.set('');
            refreshBearoff();
        }
    });

    // The selected domain, which is what the size / memory / time line and the
    // Generate button both speak about.
    let bearoffSelected = $derived(bearoffPlan?.candidates?.find((/** @type {any} */ c) => c.domain === bearoffDomain) ?? null);

    // Anything paused, whatever the domain: the "TS-06-09 interrompue à 43 %"
    // line at launch.
    let bearoffInterrupted = $derived(bearoffPlan?.candidates?.filter((/** @type {any} */ c) => c.interrupted) ?? []);

    // The measured remaining time, recomputed on a ticking clock so it counts
    // down between two progress reports rather than freezing.
    let bearoffNow = $state(Date.now());
    $effect(() => {
        if (!$bearoffProgress) return;
        const id = setInterval(() => (bearoffNow = Date.now()), 1000);
        return () => clearInterval(id);
    });
    let bearoffRemaining = $derived(remainingSeconds($bearoffProgress, bearoffNow));

    /** @param {number} n */
    function bearoffBytes(n) {
        if (!n) return '—';
        if (n >= 1e9) return `${(n / 1e9).toFixed(1)} ${get(t)('config.bearoffGB')}`;
        return `${Math.round(n / 1e6)} ${get(t)('config.bearoffMB')}`;
    }

    /** @param {number|null} seconds */
    function bearoffDuration(seconds) {
        if (!seconds || seconds < 0) return '—';
        if (seconds < 90) return get(t)('config.bearoffSeconds', { n: Math.max(1, Math.round(seconds)) });
        if (seconds < 5400) return get(t)('config.bearoffMinutes', { n: Math.round(seconds / 60) });
        return get(t)('config.bearoffHours', { n: (seconds / 3600).toFixed(1) });
    }

    /** @param {any} candidate */
    async function startBearoffGeneration(candidate = bearoffSelected) {
        if (!candidate) return;
        bearoffErrorStore.set('');
        bearoffProgressStore.set({ domain: '', done: 0, total: 0, startedAt: Date.now(), firstDone: 0 });
        try {
            await SaveBearoffCores(bearoffCores);
            await GenerateBearoffTable(candidate.kind, candidate.points, candidate.checkers, bearoffCores);
        } catch (error) {
            bearoffErrorStore.set(String(error));
            bearoffProgressStore.set(null);
        }
    }

    // Pause keeps the checkpoint, Cancel does not — the two buttons are two
    // different promises and must not share a handler.
    async function pauseBearoffGeneration() {
        try {
            await PauseBearoffGeneration();
        } finally {
            bearoffProgressStore.set(null);
            await refreshBearoff();
        }
    }

    async function cancelBearoffGeneration() {
        try {
            await CancelBearoffGeneration();
        } finally {
            bearoffProgressStore.set(null);
            await refreshBearoff();
        }
    }

    /** @param {number} checkers */
    async function discardBearoffCheckpoint(checkers) {
        try {
            await DiscardBearoffCheckpoint(6, checkers);
        } finally {
            await refreshBearoff();
        }
    }

    /** @param {string} name */
    async function deleteBearoffTable(name) {
        if (!(await confirmAction(get(t)('config.confirmDeleteBearoff', { name }), { confirmLabel: get(t)('common.delete') }))) return;
        bearoffErrorStore.set('');
        try {
            await DeleteBearoffTable(name);
            await refreshBearoff();
        } catch (error) {
            bearoffErrorStore.set(String(error));
        }
    }

    // Opt-in update check (#241): off by default, loaded/saved the same way
    // every other plain-boolean setting on this modal is.
    let checkForUpdates = $state(false);

    async function refreshCheckForUpdates() {
        try {
            checkForUpdates = await GetCheckForUpdates();
        } catch (error) {
            logger.error('Error loading check-for-updates setting:', error);
        }
    }

    $effect(() => {
        if (visible) {
            refreshCheckForUpdates();
        }
    });

    function onCheckForUpdatesChange(event) {
        checkForUpdates = event.currentTarget.checked;
        SaveCheckForUpdates(checkForUpdates).catch((error) => logger.error('Error saving check-for-updates setting:', error));
    }

    const unsubBearoff = [
        EventsOn('bearoff:progress', (p) => {
            bearoffProgressStore.update((cur) => {
                // Time the run from its FIRST progress report: the successor
                // lists are built before any of them, and counting that
                // set-up would inflate every remaining time that follows.
                if (!cur || cur.domain !== p.domain) {
                    return { ...p, startedAt: Date.now(), firstDone: p.done };
                }
                return { ...cur, ...p };
            });
        }),
        EventsOn('bearoff:done', (d) => {
            bearoffProgressStore.set(null);
            bearoffErrorStore.set('');
            // The rate this machine just measured, so the next estimate is
            // about it rather than about the machine the constant came from.
            if (d?.rate > 0) {
                SaveBearoffRate(d.rate).catch((error) => logger.error('Error saving the bearoff rate:', error));
            }
            refreshBearoff();
        }),
        EventsOn('bearoff:error', (e) => {
            bearoffProgressStore.set(null);
            bearoffErrorStore.set(e?.message ?? String(e));
            refreshBearoff();
        })
    ];
    onDestroy(() => unsubBearoff.forEach((off) => off && off()));

    // Compacts the currently open database file. Goes through the same
    // Database.Vacuum() the CLI's `blunderdb vacuum` uses (CLI/GUI parity) —
    // WAL checkpoint, free-space guard, VACUUM, ANALYZE all happen there.
    async function vacuumDatabase() {
        if (!(await confirmAction(get(t)('config.vacuumConfirm'), { confirmLabel: get(t)('config.vacuumConfirmButton') }))) return;
        vacuumBusy = true;
        try {
            const result = await Vacuum();
            const reclaimed = Math.max(0, (result?.SizeBefore ?? 0) - (result?.SizeAfter ?? 0));
            if (reclaimed > 0) {
                statusBarTextStore.set(tMsg('config.vacuumDone', { mb: (reclaimed / 1e6).toFixed(1) }));
            } else {
                statusBarTextStore.set(tMsg('config.vacuumNothing'));
            }
        } catch (error) {
            statusBarTextStore.set(tMsg('config.vacuumError', { error: String(error) }));
        } finally {
            vacuumBusy = false;
        }
    }

    // Recomputes the scalar columns of every analysis from its stored JSON.
    // Same Database.RepairAnalyses the CLI's `blunderdb repair` and the
    // daemon's analyses.repair go through (CLI/GUI parity, G.14 #242): it was
    // reachable over HTTP since the daemon existed and from nowhere else.
    //
    // Confirmed like the vacuum: it rewrites a column of every analysis in the
    // database, which is not something to start by brushing a button.
    async function repairAnalyses() {
        if (!(await confirmAction(get(t)('config.repairConfirm'), { confirmLabel: get(t)('config.repairConfirmButton') }))) return;
        repairBusy = true;
        try {
            const changed = await RepairAnalyses();
            if (changed > 0) {
                statusBarTextStore.set(tMsg('config.repairDone', { n: changed }));
            } else {
                statusBarTextStore.set(tMsg('config.repairNothing'));
            }
        } catch (error) {
            statusBarTextStore.set(tMsg('config.repairError', { error: String(error) }));
        } finally {
            repairBusy = false;
        }
    }

    // Opens the folder holding blunderDB's GUI log file
    // ($XDG_STATE_HOME/blunderDB, see internal/applog) in the platform's
    // file manager — logging.go writes only to stderr otherwise, invisible
    // once the app is launched by a double-click with no attached terminal.
    async function openLogsFolder() {
        try {
            await OpenLogsFolder();
        } catch (error) {
            statusBarTextStore.set(String(error));
        }
    }

    async function pickBearoffExternal() {
        bearoffErrorStore.set('');
        try {
            const path = await OpenBearoffFileDialog();
            if (!path) return;
            await SaveBearoffTSPath(path);
            await refreshBearoff();
        } catch (error) {
            bearoffErrorStore.set(String(error));
        }
    }

    async function clearBearoffExternal() {
        bearoffErrorStore.set('');
        try {
            await SaveBearoffTSPath('');
            await refreshBearoff();
        } catch (error) {
            bearoffErrorStore.set(String(error));
        }
    }

    // gammonNet settings (ADR-0011, ADR-0013). Two depths, named separately:
    // displayPly is interactive comfort (the live evaluation panel, #125);
    // analysisPly is what the batch job (#129) writes into a Position's
    // Analysis row. Loaded once when the tab becomes visible, like bearoff.
    let gnDisplayPly = $state(2);
    let gnAnalysisPly = $state(2);
    let gnPruneK = $state(12);
    let gnCandidates = $state(10);
    let gnAutoAnalyze = $state(false);
    // Catch-up (#130, ADR-0015): the same batch #129 wires after import, run
    // on demand for a library built before gammonNet existed. The count is
    // informational only — StartGammonNetBatch re-derives its own list.
    let gnMissingCount = $state(null);
    let gnCatchUpStarting = $state(false);
    // Re-analysis of stale positions (#191): a position whose gammonNet
    // analysis is entirely its own but was written at an older EngineVersion
    // or a different depth than gnAnalysisPly now asks for. The count is
    // informational only — StartGammonNetStaleBatch re-derives its own list
    // — and depends on gnAnalysisPly, so it is refreshed whenever that
    // changes, not just when the tab opens.
    let gnStaleCount = $state(null);
    let gnStaleStarting = $state(false);

    const GAMMONNET_PLY_OPTIONS = [0, 1, 2, 3, 4];

    // Le dossier surveillé (#258, fiche I.2). Rien n'est deviné : tant que
    // l'utilisateur n'a pas désigné un dossier, il n'y a pas de surveillance,
    // et le bouton « proposer » ne propose un chemin que si ce chemin existe
    // vraiment sur cette machine.
    let watchOn = $state(false);
    let watchFolder = $state('');
    let watchInterval = $state(0);
    let watchError = $state('');

    async function refreshWatch() {
        try {
            const [on, folder, seconds] = await GetWatchFolder();
            watchOn = on;
            watchFolder = folder;
            watchInterval = seconds;
        } catch (error) {
            logger.error('Error loading the watched-folder setting:', error);
        }
        await refreshWatchStatus();
    }

    async function applyWatch(on, folder, seconds) {
        watchError = '';
        try {
            await saveWatchSetting(on, folder, seconds);
            watchOn = on && !!folder;
            watchFolder = folder;
            watchInterval = seconds;
        } catch (error) {
            // Un dossier qui n'existe plus, un partage démonté : la case
            // revient à « non » plutôt que d'afficher une surveillance qui ne
            // tourne pas.
            watchError = String(error).replace(/^Error:\s*/, '');
            watchOn = false;
        }
    }

    async function chooseWatchFolder() {
        try {
            const dir = await OpenPositionFolderDialog();
            if (dir) await applyWatch(true, dir, watchInterval);
        } catch (error) {
            logger.error('Error choosing the watched folder:', error);
        }
    }

    async function suggestWatchFolder() {
        try {
            const dir = await SuggestWatchFolder();
            if (dir) {
                await applyWatch(true, dir, watchInterval);
            } else {
                watchError = $t('config.watchNoSuggestion');
            }
        } catch (error) {
            logger.error('Error suggesting a watched folder:', error);
        }
    }

    async function refreshGammonNet() {
        try {
            [gnDisplayPly, gnAnalysisPly, gnPruneK, gnCandidates, gnAutoAnalyze] = await Promise.all([
                GetGammonNetDisplayPly(),
                GetGammonNetAnalysisPly(),
                GetGammonNetPruneK(),
                GetGammonNetCandidates(),
                GetGammonNetAutoAnalyze()
            ]);
        } catch (error) {
            logger.error('Error loading gammonNet settings:', error);
        }
        try {
            gnMissingCount = await CountPositionsWithoutAnalysis();
        } catch (error) {
            logger.error('Error counting positions without analysis:', error);
            gnMissingCount = null;
        }
        await refreshGammonNetStaleCount();
    }

    async function refreshGammonNetStaleCount() {
        try {
            gnStaleCount = await CountPositionsWithStaleGammonNet(gnAnalysisPly);
        } catch (error) {
            logger.error('Error counting stale gammonNet positions:', error);
            gnStaleCount = null;
        }
    }

    // StatusBar.svelte owns the live progress chip/cancel button for every
    // gammonNet batch run, whichever of the three triggers started it
    // (auto-after-import #129, this button, or a future CLI/serve run has no
    // GUI to show). Starting it here is fire-and-forget: no local progress
    // state to duplicate.
    async function startGammonNetCatchUp() {
        gnCatchUpStarting = true;
        try {
            StartGammonNetBatch(gnAnalysisPly, gnPruneK, gnCandidates);
        } finally {
            gnCatchUpStarting = false;
        }
    }

    // Re-analysis (#191): same fire-and-forget shape as the catch-up above —
    // StartGammonNetStaleBatch shares gnBatchCancel with StartGammonNetBatch
    // on the Go side, so only one of the two ever runs at a time.
    async function startGammonNetStaleRerun() {
        gnStaleStarting = true;
        try {
            StartGammonNetStaleBatch(gnAnalysisPly, gnPruneK, gnCandidates);
        } finally {
            gnStaleStarting = false;
        }
    }

    $effect(() => {
        if (visible) {
            refreshGammonNet();
            refreshWatch();
        }
    });

    function onGnDisplayPlyChange(event) {
        gnDisplayPly = Number(event.currentTarget.value);
        SaveGammonNetDisplayPly(gnDisplayPly).catch((error) => logger.error('Error saving gammonNet display ply:', error));
    }

    function onGnAnalysisPlyChange(event) {
        gnAnalysisPly = Number(event.currentTarget.value);
        refreshGammonNetStaleCount();
        SaveGammonNetAnalysisPly(gnAnalysisPly).catch((error) => logger.error('Error saving gammonNet analysis ply:', error));
    }

    function onGnPruneKChange(event) {
        const k = Number(event.currentTarget.value);
        if (!Number.isFinite(k) || k < 1) return;
        gnPruneK = k;
        SaveGammonNetPruneK(gnPruneK).catch((error) => logger.error('Error saving gammonNet prune k:', error));
    }

    function onGnCandidatesChange(event) {
        const n = Number(event.currentTarget.value);
        if (!Number.isFinite(n) || n < 1) return;
        gnCandidates = n;
        SaveGammonNetCandidates(gnCandidates).catch((error) => logger.error('Error saving gammonNet candidate count:', error));
    }

    function onWatchToggle(event) {
        void applyWatch(event.currentTarget.checked, watchFolder, watchInterval);
    }

    function onWatchIntervalChange(event) {
        const seconds = parseInt(event.currentTarget.value, 10);
        void applyWatch(watchOn, watchFolder, Number.isFinite(seconds) ? seconds : 0);
    }

    function onGnAutoAnalyzeChange(event) {
        gnAutoAnalyze = event.currentTarget.checked;
        SaveGammonNetAutoAnalyze(gnAutoAnalyze).catch((error) => logger.error('Error saving gammonNet auto-analyze:', error));
    }
</script>

<Modal open={visible} onclose={onClose} size="medium" align="center" compactTitle closeOnOverlay>
    {#snippet title()}{$t('config.title')}{/snippet}

    <div class="tabs" role="tablist">
        {#each TABS as tab (tab.id)}
            <button type="button" class="tab" class:active={activeTab === tab.id} role="tab" aria-selected={activeTab === tab.id} onclick={() => (activeTab = tab.id)}>
                {$t(tab.labelKey)}
            </button>
        {/each}
    </div>

    <div class="tab-body">
        {#if activeTab === 'interface'}
            <div class="setting-row">
                <label for="config-language">{$t('config.language')}</label>
                <select id="config-language" class="setting-select" value={$language} onchange={onLanguageChange}>
                    {#each LOCALES as code (code)}
                        <option value={code}>{LANGUAGE_LABELS[code]}</option>
                    {/each}
                </select>
            </div>
            <div class="setting-row">
                <label for="config-ui-scale">{$t('config.uiScale')}</label>
                <div class="scale-control">
                    <input
                        id="config-ui-scale"
                        type="range"
                        class="setting-range"
                        min={MIN_UI_SCALE}
                        max={MAX_UI_SCALE}
                        step={UI_SCALE_STEP}
                        value={$uiScaleStore}
                        oninput={onUIScaleInput}
                        onchange={onUIScaleChange}
                    />
                    <span class="scale-value">{$uiScaleStore}%</span>
                </div>
            </div>
            <div class="setting-row">
                <label for="config-panel-position">{$t('config.panelPosition')}</label>
                <select id="config-panel-position" class="setting-select" value={$panelPositionStore} onchange={onPanelPositionChange}>
                    {#each PANEL_POSITION_OPTIONS as opt (opt.value)}
                        <option value={opt.value}>{$t(opt.labelKey)}</option>
                    {/each}
                </select>
            </div>
            <p class="setting-note">{$t('config.vacuumIntro')}</p>
            <div class="tab-actions">
                <button class="secondary-button" onclick={vacuumDatabase} disabled={vacuumBusy}>
                    {vacuumBusy ? $t('config.vacuumRunning') : $t('config.vacuumButton')}
                </button>
            </div>
            <p class="setting-note">{$t('config.repairIntro')}</p>
            <div class="tab-actions">
                <button class="secondary-button" onclick={repairAnalyses} disabled={repairBusy}>
                    {repairBusy ? $t('config.repairRunning') : $t('config.repairButton')}
                </button>
            </div>
            <p class="setting-note">{$t('config.logsIntro')}</p>
            <div class="tab-actions">
                <button class="secondary-button" onclick={openLogsFolder}>{$t('config.logsButton')}</button>
            </div>
            <div class="setting-row">
                <label for="config-check-for-updates">{$t('config.checkForUpdates')}</label>
                <input id="config-check-for-updates" type="checkbox" checked={checkForUpdates} onchange={onCheckForUpdatesChange} />
            </div>
            <p class="setting-note">{$t('config.checkForUpdatesNote')}</p>
        {:else if activeTab === 'colors'}
            {#each COLOR_SETTINGS as setting (setting.key)}
                <div class="setting-row">
                    <label for={`config-color-${setting.key}`}>{$t(setting.labelKey)}</label>
                    <input id={`config-color-${setting.key}`} type="color" class="setting-color" value={$boardColorsStore[setting.key]} oninput={(e) => onColorChange(setting.key, e)} />
                </div>
            {/each}
            <div class="tab-actions">
                <button class="secondary-button" onclick={resetBoardColors}>{$t('config.resetColors')}</button>
            </div>
        {:else if activeTab === 'bearoff'}
            <p class="setting-note">{$t('config.bearoffIntro')}</p>
            {#if bearoff}
                <div class="setting-row">
                    <span class="setting-label">{$t('config.bearoffActive')}</span>
                    <code class="identity-fingerprint">
                        {bearoff.active_domain > 0 ? `TS-06-${String(bearoff.active_domain).padStart(2, '0')} — ${bearoff.active_origin}` : $t('config.bearoffNone')}
                    </code>
                </div>
                <div class="setting-row">
                    <span class="setting-label">{$t('config.bearoffEpc')}</span>
                    <span class="setting-value">{bearoff.one_sided_ready ? $t('config.bearoffEpcReady') : $t('config.bearoffEpcMissing')}</span>
                </div>
                {#if bearoffPlan}
                    <div class="setting-row">
                        <span class="setting-label">{$t('config.bearoffDataDir')}</span>
                        <code class="identity-fingerprint">{bearoffPlan.data_dir}</code>
                    </div>
                {/if}

                <!-- What is on disk, with a Delete on each: the only place a
                     user can see the space these take and get it back. -->
                {#if bearoffPlan?.files?.length}
                    <p class="setting-note">{$t('config.bearoffFilesTitle')}</p>
                    <ul class="bearoff-files">
                        {#each bearoffPlan.files as file (file.name)}
                            <li>
                                <code class="identity-fingerprint">{file.name}</code>
                                <span class="setting-value">{bearoffBytes(file.size)} · {$t(`config.bearoffVerdict_${file.verdict}`)}</span>
                                <button class="danger-button" onclick={() => deleteBearoffTable(file.name)}>{$t('common.delete')}</button>
                            </li>
                        {/each}
                    </ul>
                {/if}

                <!-- A run that was paused, offered by name and percentage. It
                     is never restarted on its own: the user asked for it to
                     stop. -->
                {#each bearoffInterrupted as paused (paused.domain)}
                    <div class="setting-row">
                        <span class="setting-label">{$t('config.bearoffInterrupted', { domain: paused.domain, percent: paused.percent.toFixed(0) })}</span>
                        <span class="tab-actions">
                            <button class="secondary-button" onclick={() => startBearoffGeneration(paused)}>{$t('config.bearoffResume')}</button>
                            <button class="danger-button" onclick={() => discardBearoffCheckpoint(paused.checkers)}>{$t('common.delete')}</button>
                        </span>
                    </div>
                {/each}

                {#if $bearoffProgress || bearoff.generating}
                    <div class="setting-row">
                        <span class="setting-label">{$t('config.bearoffGenerating', { domain: $bearoffProgress?.domain || bearoff.generating })}</span>
                        <progress class="bearoff-progress" max={$bearoffProgress?.total ?? 0} value={$bearoffProgress?.done ?? 0}></progress>
                    </div>
                    <p class="setting-note">
                        {#if bearoffRemaining !== null}
                            {$t('config.bearoffRemaining', { time: bearoffDuration(bearoffRemaining) })}
                        {:else}
                            {$t('config.bearoffStarting')}
                        {/if}
                    </p>
                    <div class="tab-actions">
                        <button class="secondary-button" onclick={pauseBearoffGeneration}>{$t('config.bearoffPause')}</button>
                        <button class="secondary-button" onclick={cancelBearoffGeneration}>{$t('common.cancel')}</button>
                    </div>
                {:else if bearoffPlan}
                    <p class="setting-note">{$t('config.bearoffWiderNote')}</p>
                    <div class="setting-row">
                        <label for="config-bearoff-checkers">{$t('config.bearoffCheckers')}</label>
                        <select id="config-bearoff-checkers" class="setting-select" bind:value={bearoffDomain}>
                            <optgroup label={$t('config.bearoffTwoSided')}>
                                {#each bearoffPlan.candidates.filter((/** @type {any} */ c) => c.kind === 'two-sided') as candidate (candidate.domain)}
                                    <option value={candidate.domain} disabled={!candidate.fits}>{candidate.domain}</option>
                                {/each}
                            </optgroup>
                            <optgroup label={$t('config.bearoffOneSided')}>
                                {#each bearoffPlan.candidates.filter((/** @type {any} */ c) => c.kind === 'one-sided') as candidate (candidate.domain)}
                                    <option value={candidate.domain} disabled={!candidate.fits}>{candidate.domain}</option>
                                {/each}
                            </optgroup>
                        </select>
                    </div>
                    <div class="setting-row">
                        <label for="config-bearoff-cores">{$t('config.bearoffCores')}</label>
                        <!-- The one-sided sweep reads only positions below the one it
                             is on, so cores buy it nothing and the picker says so. -->
                        <select id="config-bearoff-cores" class="setting-select" disabled={bearoffSelected?.kind === 'one-sided'} bind:value={bearoffCores} onchange={refreshBearoff}>
                            <option value={0}>{$t('config.bearoffCoresDefault', { n: bearoffPlan.default_cores })}</option>
                            {#each Array.from({ length: bearoffPlan.default_cores + 1 }, (/** @type {unknown} */ _, /** @type {number} */ i) => i + 1) as n (n)}
                                <option value={n}>{n}</option>
                            {/each}
                        </select>
                    </div>
                    {#if bearoffSelected}
                        <p class="setting-note">
                            {#if bearoffSelected.fits}
                                {$t(bearoffSelected.kind === 'one-sided' ? 'config.bearoffCostSequential' : 'config.bearoffCost', {
                                    size: bearoffBytes(bearoffSelected.size),
                                    ram: bearoffBytes(bearoffSelected.ram_needed),
                                    time: bearoffDuration(bearoffSelected.seconds),
                                    cores: bearoffPlan.cores
                                })}
                                <span class="setting-value"> {$t(bearoffSelected.kind === 'one-sided' ? 'config.bearoffPurposeOneSided' : 'config.bearoffPurposeTwoSided')}</span>
                                {#if !bearoffPlan.rate_measured}<span class="setting-value"> {$t('config.bearoffEstimateUnmeasured')}</span>{/if}
                            {:else}
                                {$t('config.bearoffTooBig', {
                                    ram: bearoffBytes(bearoffSelected.ram_needed),
                                    available: bearoffBytes(bearoffPlan.ram_available)
                                })}
                            {/if}
                        </p>
                    {/if}
                    <div class="tab-actions">
                        <button class="secondary-button" disabled={!bearoffSelected?.fits || bearoffSelected?.present} onclick={() => startBearoffGeneration()}>
                            {bearoffSelected?.present ? $t('config.bearoffAlreadyThere') : $t('config.bearoffGenerate')}
                        </button>
                    </div>
                {/if}

                <div class="setting-row">
                    <span class="setting-label">{$t('config.bearoffExternal')}</span>
                    <code class="identity-fingerprint">{bearoffExternal || $t('config.bearoffExternalNone')}</code>
                </div>
                <div class="tab-actions">
                    <button class="secondary-button" onclick={pickBearoffExternal}>{$t('config.bearoffExternalPick')}</button>
                    {#if bearoffExternal}
                        <button class="secondary-button" onclick={clearBearoffExternal}>{$t('config.bearoffExternalClear')}</button>
                    {/if}
                </div>

                {#if $bearoffErrorStoreRef}<p class="setting-note warn">{$bearoffErrorStoreRef}</p>{/if}
            {/if}
        {:else if activeTab === 'gammonnet'}
            <p class="setting-note">{$t('config.gammonnetIntro')}</p>
            <div class="setting-row">
                <label for="config-gn-display-ply">{$t('config.gammonnetDisplayPly')}</label>
                <select id="config-gn-display-ply" class="setting-select" value={gnDisplayPly} onchange={onGnDisplayPlyChange}>
                    {#each GAMMONNET_PLY_OPTIONS as ply (ply)}
                        <option value={ply}>{$t('config.gammonnetPly', { n: ply })}</option>
                    {/each}
                </select>
            </div>
            <p class="setting-note">{$t('config.gammonnetDisplayPlyNote')}</p>
            <div class="setting-row">
                <label for="config-gn-analysis-ply">{$t('config.gammonnetAnalysisPly')}</label>
                <select id="config-gn-analysis-ply" class="setting-select" value={gnAnalysisPly} onchange={onGnAnalysisPlyChange}>
                    {#each GAMMONNET_PLY_OPTIONS as ply (ply)}
                        <option value={ply}>{$t('config.gammonnetPly', { n: ply })}</option>
                    {/each}
                </select>
            </div>
            <p class="setting-note">{$t('config.gammonnetAnalysisPlyNote')}</p>
            <div class="setting-row">
                <label for="config-gn-prune-k">{$t('config.gammonnetPruneK')}</label>
                <input id="config-gn-prune-k" type="number" class="setting-input" min="1" max="64" value={gnPruneK} onchange={onGnPruneKChange} />
            </div>
            <div class="setting-row">
                <label for="config-gn-candidates">{$t('config.gammonnetCandidates')}</label>
                <input id="config-gn-candidates" type="number" class="setting-input" min="1" max="50" value={gnCandidates} onchange={onGnCandidatesChange} />
            </div>
            <div class="setting-row">
                <label for="config-gn-auto-analyze">{$t('config.gammonnetAutoAnalyze')}</label>
                <input id="config-gn-auto-analyze" type="checkbox" checked={gnAutoAnalyze} onchange={onGnAutoAnalyzeChange} />
            </div>
            <p class="setting-note">{$t('config.gammonnetAutoAnalyzeNote')}</p>
            <div class="setting-row">
                <span class="setting-label">
                    {gnMissingCount === null ? $t('config.gammonnetCatchUpUnknown') : $t('config.gammonnetCatchUpCount', { n: gnMissingCount })}
                </span>
                <button class="secondary-button" disabled={gnCatchUpStarting || gnMissingCount === 0} onclick={startGammonNetCatchUp}>
                    {$t('config.gammonnetCatchUpStart')}
                </button>
            </div>
            <p class="setting-note">{$t('config.gammonnetCatchUpNote')}</p>
            <div class="setting-row">
                <span class="setting-label">
                    {gnStaleCount === null ? $t('config.gammonnetStaleUnknown') : $t('config.gammonnetStaleCount', { n: gnStaleCount })}
                </span>
                <button class="secondary-button" disabled={gnStaleStarting || gnStaleCount === 0} onclick={startGammonNetStaleRerun}>
                    {$t('config.gammonnetStaleStart')}
                </button>
            </div>
            <p class="setting-note">{$t('config.gammonnetStaleNote')}</p>
        {:else if activeTab === 'watch'}
            <p class="setting-note">{$t('config.watchIntro')}</p>
            <div class="setting-row">
                <span class="setting-label">{watchFolder || $t('config.watchNoFolder')}</span>
                <button class="secondary-button" onclick={chooseWatchFolder}>{$t('config.watchChoose')}</button>
            </div>
            <div class="setting-row">
                <span class="setting-label">{$t('config.watchSuggestLabel')}</span>
                <button class="secondary-button" onclick={suggestWatchFolder}>{$t('config.watchSuggest')}</button>
            </div>
            <div class="setting-row">
                <label for="config-watch-on">{$t('config.watchEnabled')}</label>
                <input id="config-watch-on" type="checkbox" checked={watchOn} disabled={!watchFolder} onchange={onWatchToggle} />
            </div>
            <div class="setting-row">
                <label for="config-watch-interval">{$t('config.watchInterval')}</label>
                <input id="config-watch-interval" type="number" class="setting-input" min="0" max="3600" value={watchInterval} onchange={onWatchIntervalChange} />
            </div>
            <p class="setting-note">{$t('config.watchIntervalNote')}</p>
            <p class="setting-note">{$t('config.watchOnlyNewNote')}</p>
            {#if watchError}
                <p class="setting-note warn">{watchError}</p>
            {/if}
            <div class="setting-row">
                <span class="setting-label">
                    {$watchStatusStore.running ? $t('config.watchRunning', { folder: $watchStatusStore.folder }) : $t('config.watchStopped')}
                </span>
            </div>
        {:else if activeTab === 'identity'}
            <p class="setting-note">{$t('config.identityIntro')}</p>
            {#if identity?.present}
                <div class="setting-row">
                    <label for="config-identity-name">{$t('config.identityName')}</label>
                    <input id="config-identity-name" type="text" class="setting-input" value={identity.name} onblur={renameIdentity} />
                </div>
                <div class="setting-row">
                    <span class="setting-label">{$t('config.identityFingerprint')}</span>
                    <code class="identity-fingerprint">{identity.fingerprint}</code>
                </div>
            {:else}
                <p class="setting-note">{$t('config.identityNone')}</p>
            {/if}
            <div class="setting-row">
                <label for="config-identity-passphrase">{$t('config.identityPassphrase')}</label>
                <input id="config-identity-passphrase" type="password" class="setting-input" bind:value={identityPassphrase} />
            </div>
            <p class="setting-note warn">{$t('config.identityWarning')}</p>

            {#if confirmingRegenerate}
                <div class="regenerate-confirm">
                    <p class="setting-note warn">{$t('config.identityRegenerateWarning')}</p>
                    <p class="setting-note">{$t('config.identityRegenerateKeep', { fingerprint: identity?.fingerprint ?? '' })}</p>
                    <div class="tab-actions">
                        <button class="secondary-button" onclick={exportIdentity}>{$t('config.identitySaveFirst')}</button>
                        <button class="secondary-button" onclick={() => (confirmingRegenerate = false)}>{$t('common.cancel')}</button>
                        <button class="danger-button" onclick={regenerateIdentity}>{$t('config.identityRegenerateConfirm')}</button>
                    </div>
                </div>
            {:else}
                <div class="tab-actions">
                    <button class="secondary-button" onclick={exportIdentity}>{$t('config.identityExport')}</button>
                    <button class="secondary-button" onclick={importIdentity}>{$t('config.identityImport')}</button>
                    {#if identity?.present}
                        <button class="secondary-button" onclick={() => (confirmingRegenerate = true)}>
                            {$t('config.identityRegenerate')}
                        </button>
                    {/if}
                </div>
            {/if}

            {#if identityMessage}<p class="setting-note ok">{identityMessage}</p>{/if}
            {#if identityError}<p class="setting-note warn">{identityError}</p>{/if}
        {/if}
    </div>

    {#snippet footer()}
        <button class="primary" onclick={onClose}>{$t('common.close')}</button>
    {/snippet}
</Modal>

<style>
    .bearoff-files {
        list-style: none;
        margin: 0 0 0.5rem;
        padding: 0;
    }

    .bearoff-files li {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.15rem 0;
    }

    .bearoff-files li .setting-value {
        margin-left: auto;
    }

    .bearoff-progress {
        flex: 1;
        min-width: 120px;
    }

    .setting-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        margin: 8px 0;
        text-align: left;
    }

    .setting-row label {
        font-weight: 500;
    }

    .setting-select {
        flex: 0 0 auto;
        min-width: 160px;
        padding: 8px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        box-sizing: border-box;
        background-color: white;
    }

    .setting-select:focus {
        outline: none;
        border-color: #6c757d;
        box-shadow: 0 0 5px rgba(108, 117, 125, 0.5);
    }

    .scale-control {
        display: flex;
        align-items: center;
        gap: 10px;
        flex: 0 0 auto;
    }

    .setting-range {
        width: 130px;
        cursor: pointer;
        accent-color: #6c757d;
    }

    .scale-value {
        min-width: 42px;
        text-align: right;
        font-variant-numeric: tabular-nums;
        font-weight: 500;
    }

    .regenerate-confirm {
        border-left: 3px solid #b3261e;
        padding-left: 8px;
        margin: 4px 0;
    }

    /* The only destructive action in the dialog, and the only red button. */
    .danger-button {
        padding: 4px 10px;
        border: 1px solid #b3261e;
        border-radius: 4px;
        background: #b3261e;
        color: white;
        cursor: pointer;
    }

    .setting-note {
        color: var(--color-text-muted);
        margin: 2px 0 6px;
        line-height: 1.35;
    }

    .setting-note.warn {
        color: #b3261e;
    }

    .setting-note.ok {
        color: #1a7f37;
    }

    .setting-input {
        flex: 1;
        min-width: 0;
        max-width: 220px;
    }

    .setting-label {
    }

    /* Monospace looks a size larger than a proportional face at the same nominal size, so
       it takes the small token to sit level with the base-size text beside it (ADR-0008
       rule 4). */
    .identity-fingerprint {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
    }

    .tabs {
        display: flex;
        gap: 2px;
        border-bottom: 1px solid #e0e0e0;
        margin-bottom: 8px;
    }

    .tab {
        flex: 1;
        padding: 6px 8px;
        border: none;
        border-bottom: 2px solid transparent;
        background: none;
        color: #5f6368;
        cursor: pointer;
    }

    .tab:hover {
        color: #202124;
    }

    .tab.active {
        color: var(--color-primary);
        border-bottom-color: var(--color-primary);
        font-weight: 600;
    }

    /* A fixed height, not a floor: a minimum still let the dialog grow with the tallest tab
       — Interface has three rows, Colours has nine, and Identity changes again when the
       regeneration warning unfolds. The box then resized and, since the overlay centres it,
       moved under the pointer at every tab change. The content scrolls inside instead.
       `min()` keeps it from overflowing a short window. */
    .tab-body {
        height: min(300px, 46vh);
        overflow-y: auto;
        text-align: left;
    }

    /* One action area per tab, aligned the same way, so no button ever appears in the
       middle of a list. */
    .tab-actions {
        display: flex;
        flex-wrap: wrap;
        justify-content: flex-end;
        gap: 6px;
        margin-top: 10px;
    }

    .tab-actions button {
        padding: 4px 10px;
    }

    .setting-color {
        flex: 0 0 auto;
        width: 48px;
        height: 28px;
        padding: 0;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        background: none;
        cursor: pointer;
    }

    .secondary-button {
        padding: 4px 10px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        background-color: #f5f5f5;
        cursor: pointer;
    }

    .secondary-button:hover {
        background-color: #e9e9e9;
    }
</style>
