<script>
    import { onMount } from 'svelte';
    import { t, language } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { syncSettingsStore, syncStateStore, loadSync, saveSyncSettings, setSyncToken, syncNow, syncErrorWords, syncUnapplied, syncSkipped, syncSkipReasonKey } from '../services/syncService.js';

    let serverInput = $state('');
    let tokenInput = $state('');
    let saveError = $state('');

    onMount(async () => {
        await loadSync();
        serverInput = $syncSettingsStore.server;
    });

    const st = $derived($syncStateStore);
    const lastDate = $derived(st.lastSuccess ? new Date(st.lastSuccess).toLocaleString($language) : '');

    /** @param {Partial<import('../services/syncService.js').SyncSettings>} change */
    async function update(change) {
        saveError = '';
        try {
            await saveSyncSettings({ ...$syncSettingsStore, ...change });
        } catch (error) {
            logger.error('Error saving the sync setting:', error);
            saveError = String(error);
        }
    }

    async function saveToken() {
        saveError = '';
        try {
            await setSyncToken($syncSettingsStore.server, tokenInput);
            tokenInput = '';
        } catch (error) {
            logger.error('Error storing the sync token:', error);
            saveError = String(error);
        }
    }
</script>

<div class="sync-settings">
    <h4>{$t('sync.title')}</h4>
    <p class="setting-note">{$t('sync.intro')}</p>
    <div class="setting-row">
        <label for="config-sync-server">{$t('sync.server')}</label>
        <input id="config-sync-server" type="url" class="setting-input" placeholder="https://" autocomplete="off" bind:value={serverInput} onchange={() => update({ server: serverInput.trim() })} />
    </div>
    <div class="setting-row">
        <label for="config-sync-token">{$t('sync.token')}</label>
        <input id="config-sync-token" type="password" class="setting-input" autocomplete="off" bind:value={tokenInput} />
        <button class="secondary-button" onclick={saveToken} disabled={!$syncSettingsStore.server}>
            {tokenInput.trim() ? $t('sync.tokenSave') : $t('sync.tokenRemove')}
        </button>
    </div>
    <p class="setting-note" data-testid="sync-token-state">{st.hasToken ? $t('sync.tokenStored') : $t('sync.tokenNone')}</p>
    <div class="setting-row">
        <label for="config-sync-auto">{$t('sync.auto')}</label>
        <input id="config-sync-auto" type="checkbox" checked={$syncSettingsStore.auto} onchange={(e) => update({ auto: e.currentTarget.checked })} />
    </div>
    <div class="setting-row">
        <span class="setting-label" data-testid="sync-last">
            {#if st.running}
                {$t('sync.running')}
            {:else if lastDate}
                {$t('sync.last', { date: lastDate })}
            {:else}
                {$t('sync.never')}
            {/if}
        </span>
        <button class="secondary-button" onclick={syncNow} disabled={st.running || !$syncSettingsStore.server || !st.hasToken}>{$t('sync.now')}</button>
    </div>
    {#if st.error}
        {@const w = syncErrorWords(st.error)}
        <p class="setting-note warn" role="alert" data-testid="sync-error">
            {w.key ? $t(w.key, { ...w.params, axis: w.params.axis ? $t(String(w.params.axis)) : '' }) : w.verbatim}
            {#if w.detail}<span class="detail">{w.detail}</span>{/if}
        </p>
    {:else if !st.running && st.result}
        <p class="setting-note ok" data-testid="sync-result">
            {$t('sync.result', { sent: st.result.sent?.records ?? 0, received: st.result.received?.records ?? 0 })}
        </p>
        {#if syncUnapplied(st.result) > 0}
            <p class="setting-note warn" role="alert" data-testid="sync-unapplied">
                {$t('sync.unapplied', {
                    conflicts: (st.result.sent?.conflicts ?? 0) + (st.result.received?.conflicts ?? 0),
                    missing: (st.result.sent?.missing ?? 0) + (st.result.received?.missing ?? 0),
                    unsupported: (st.result.sent?.unsupported ?? 0) + (st.result.received?.unsupported ?? 0)
                })}
            </p>
        {/if}
    {/if}
    {#if syncSkipped(st).length > 0}
        <div class="setting-note warn" role="alert" data-testid="sync-skipped">
            {$t('sync.skipped.title', { count: syncSkipped(st).length })}
            <ul>
                {#each syncSkipped(st) as k (k.direction + k.family + k.uuid)}
                    <li>
                        {$t('sync.skipped.item', {
                            family: k.family,
                            id: k.uuid,
                            direction: $t(k.direction === 'received' ? 'sync.skipped.received' : 'sync.skipped.sent'),
                            reason: $t(syncSkipReasonKey(k.reason))
                        })}
                    </li>
                {/each}
            </ul>
        </div>
    {/if}
    {#if saveError}
        <p class="setting-note warn" role="alert">{saveError}</p>
    {/if}
</div>

<style>
    h4 {
        margin: 12px 0 4px;
        text-align: left;
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
    .setting-note {
        color: var(--color-text-muted);
        margin: 2px 0 6px;
        line-height: 1.35;
        text-align: left;
    }
    .setting-note.warn {
        color: var(--color-danger);
    }
    .setting-note.ok {
        color: var(--color-primary);
    }
    .detail {
        display: block;
        color: var(--color-text-muted);
    }
    .secondary-button {
        padding: 4px 10px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        background-color: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
    }
    .secondary-button:disabled {
        opacity: 0.5;
        cursor: default;
    }
    .setting-input {
        flex: 1;
        min-width: 0;
        max-width: 280px;
    }
</style>
