<script>
    import { get } from 'svelte/store';
    import { configInitialTabStore } from '../stores/uiStore';
    import { trapFocus } from '../utils/focusTrap.js';
    import { t, language, setLanguage, LOCALES, LANGUAGE_LABELS } from '../i18n';
    import { boardColorsStore, setBoardColor, resetBoardColors } from '../stores/boardColorsStore';
    import { uiScaleStore, setUIScale, previewUIScale, MIN_UI_SCALE, MAX_UI_SCALE, UI_SCALE_STEP } from '../stores/uiScaleStore';
    import { panelPositionStore, setPanelPosition, PANEL_BOTTOM, PANEL_SIDE, PANEL_AUTO } from '../stores/panelLayoutStore';
    import {
        GetIssuerIdentity,
        SetIssuerName,
        ExportIssuerIdentity,
        PickIdentityFile,
        ImportIssuerIdentity,
        RegenerateIssuerIdentity,
        BearoffStatus,
        DownloadBearoffDB,
        CancelBearoffDownload,
        DeleteBearoffDB,
        OpenBearoffFileDialog
    } from '../../wailsjs/go/gui/App.js';
    import { GetBearoffTsPath, SaveBearoffTsPath } from '../../wailsjs/go/main/Config.js';
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

    // Three concerns, and the third is not a preference at all: language, scale and colours
    // are settings one adjusts, whereas the identity is an object one manages, with its own
    // verbs. Keeping them in one column meant eighteen rows and up to seven buttons at
    // different heights; tabs give each concern its own action area and leave a single
    // primary button, always in the same place.
    const TABS = [
        { id: 'interface', labelKey: 'config.interface' },
        { id: 'colors', labelKey: 'config.colors' },
        { id: 'bearoff', labelKey: 'config.bearoffTitle' },
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

    // Two-sided bearoff data sources (ADR-0009): status of the optional
    // TS-06-11 download plus the user-supplied external .bd path.
    let bearoff = $state(null);
    let bearoffExternal = $state('');
    let bearoffProgress = $state(null); // {received, total} while downloading
    let bearoffError = $state('');

    async function refreshBearoff() {
        try {
            bearoff = await BearoffStatus();
            bearoffExternal = await GetBearoffTsPath();
        } catch (error) {
            logger.error('Error loading bearoff status:', error);
        }
    }

    $effect(() => {
        if (visible) {
            bearoffError = '';
            refreshBearoff();
        }
    });

    const unsubBearoff = [
        EventsOn('bearoff:progress', (p) => (bearoffProgress = p)),
        EventsOn('bearoff:done', () => {
            bearoffProgress = null;
            bearoffError = '';
            refreshBearoff();
        }),
        EventsOn('bearoff:error', (e) => {
            bearoffProgress = null;
            bearoffError = e?.message ?? String(e);
            refreshBearoff();
        })
    ];
    onDestroy(() => unsubBearoff.forEach((off) => off && off()));

    async function startBearoffDownload() {
        bearoffError = '';
        bearoffProgress = { received: bearoff?.partial_bytes ?? 0, total: bearoff?.expected_bytes ?? 0 };
        try {
            await DownloadBearoffDB();
            await refreshBearoff();
        } catch (error) {
            bearoffError = String(error);
            bearoffProgress = null;
        }
    }

    async function cancelBearoffDownload() {
        try {
            await CancelBearoffDownload();
        } finally {
            bearoffProgress = null;
            await refreshBearoff();
        }
    }

    async function deleteBearoffDownload() {
        bearoffError = '';
        try {
            await DeleteBearoffDB();
            await refreshBearoff();
        } catch (error) {
            bearoffError = String(error);
        }
    }

    async function pickBearoffExternal() {
        bearoffError = '';
        try {
            const path = await OpenBearoffFileDialog();
            if (!path) return;
            await SaveBearoffTsPath(path);
            await refreshBearoff();
        } catch (error) {
            bearoffError = String(error);
        }
    }

    async function clearBearoffExternal() {
        bearoffError = '';
        try {
            await SaveBearoffTsPath('');
            await refreshBearoff();
        } catch (error) {
            bearoffError = String(error);
        }
    }

    const gb = (bytes) => (bytes / 1e9).toFixed(2);

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }
</script>

{#if visible}
    <div class="modal-overlay" onclick={onClose} onkeydown={handleKeyDown} role="dialog" aria-modal="true" aria-label={$t('config.title')} use:trapFocus>
        <div class="modal-content" onclick={(e) => e.stopPropagation()}>
            <div class="close-button" onclick={onClose}>×</div>
            <h2>{$t('config.title')}</h2>

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
                            <code class="identity-fingerprint">TS-06-{String(bearoff.active_domain).padStart(2, '0')} — {bearoff.active_origin}</code>
                        </div>

                        {#if bearoffProgress || bearoff.downloading}
                            <div class="setting-row">
                                <span class="setting-label">{$t('config.bearoffDownloading')}</span>
                                <progress class="bearoff-progress" max={bearoffProgress?.total ?? bearoff.expected_bytes} value={bearoffProgress?.received ?? 0}></progress>
                                <span class="setting-label">{gb(bearoffProgress?.received ?? 0)} / {gb(bearoffProgress?.total ?? bearoff.expected_bytes)} {$t('config.bearoffGB')}</span>
                            </div>
                            <div class="tab-actions">
                                <button class="secondary-button" onclick={cancelBearoffDownload}>{$t('common.cancel')}</button>
                            </div>
                        {:else if bearoff.downloaded}
                            <p class="setting-note ok">{$t('config.bearoffDownloaded', { gb: gb(bearoff.size_bytes) })}</p>
                            <div class="tab-actions">
                                <button class="danger-button" onclick={deleteBearoffDownload}>{$t('config.bearoffDelete')}</button>
                            </div>
                        {:else}
                            <p class="setting-note">{$t('config.bearoffDownloadNote', { gb: gb(bearoff.expected_bytes) })}</p>
                            {#if bearoff.partial_bytes > 0}
                                <p class="setting-note">{$t('config.bearoffPartial', { gb: gb(bearoff.partial_bytes) })}</p>
                            {/if}
                            <div class="tab-actions">
                                <button class="secondary-button" onclick={startBearoffDownload}>
                                    {bearoff.partial_bytes > 0 ? $t('config.bearoffResume') : $t('config.bearoffDownload')}
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

                        {#if bearoffError}<p class="setting-note warn">{bearoffError}</p>{/if}
                    {/if}
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

            <div class="modal-buttons">
                <button class="primary-button" onclick={onClose}>{$t('common.close')}</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .bearoff-progress {
        flex: 1;
        min-width: 120px;
    }

    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }

    /* One type scale for the whole dialog. It used to carry four at once in a single tab —
       11 px notes, 12 px labels and buttons, 15 px selects, and form controls with no size
       at all, which fall back to the browser's own control font in another family. What
       separates a label from a value here is weight and colour, not size. */
    .modal-content {
        font-size: var(--font-size-base);
        background-color: white;
        padding: 1rem;
        border-radius: 4px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        z-index: 1000;
        width: 90%;
        max-width: 360px;
        max-height: 80vh;
        position: relative;
        display: flex;
        flex-direction: column;
        text-align: center;
    }

    input,
    select,
    button {
        font: inherit;
    }

    h2 {
        margin: 0 0 1rem;
        font-size: var(--font-size-title);
    }

    .close-button {
        position: absolute;
        top: 8px;
        right: 8px;
        font-size: 1.5rem;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        z-index: 10;
        transition:
            background-color 0.3s ease,
            opacity 0.3s ease;
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
        border: 1px solid #ccc;
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
        color: #666;
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
       it is nudged down to sit level with the text beside it. */
    .identity-fingerprint {
        font-family: monospace;
        font-size: 0.92em;
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
        font: inherit;
        color: #5f6368;
        cursor: pointer;
    }

    .tab:hover {
        color: #202124;
    }

    .tab.active {
        color: #1a73e8;
        border-bottom-color: #1a73e8;
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
        border: 1px solid #ccc;
        border-radius: 4px;
        background: none;
        cursor: pointer;
    }

    .secondary-button {
        padding: 4px 10px;
        border: 1px solid #ccc;
        border-radius: 4px;
        background-color: #f5f5f5;
        cursor: pointer;
    }

    .secondary-button:hover {
        background-color: #e9e9e9;
    }

    .modal-buttons {
        margin-top: 10px;
        display: flex;
        justify-content: center;
        gap: 10px;
    }

    .modal-buttons button {
        padding: 6px 14px;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }

    .primary-button {
        background-color: #6c757d;
        color: white;
    }

    .primary-button:hover {
        background-color: #5a6268;
    }
</style>
