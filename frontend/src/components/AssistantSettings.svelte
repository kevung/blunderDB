<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { GetMCPHost } from '../../wailsjs/go/main/Config.js';
    import { DEFAULT_MCP_PORT, mcpHostStatusStore, saveMCPHost, refreshMCPHostStatus } from '../services/mcpHostService.js';
    import { logger } from '../utils/logger.js';
    import { AssistantSetKey, AssistantHasKey } from '../../wailsjs/go/gui/App.js';
    import { assistantSettingsStore, assistantPresetsStore, loadAssistantSettings, saveAssistantSettings, presetOf, effectiveURL, isRemoteURL } from '../services/assistantService.js';

    // The MCP server the window serves on localhost (ADR-0059): off until the user turns it on.
    let mcp = $state({ on: false, port: DEFAULT_MCP_PORT, write: false });

    onMount(async () => {
        try {
            const saved = await GetMCPHost();
            if (saved) mcp = { on: !!saved.on, port: saved.port || DEFAULT_MCP_PORT, write: !!saved.write };
            await refreshMCPHostStatus();
        } catch (error) {
            logger.error('Error loading the MCP server setting:', error);
        }
    });

    /** @param {Partial<typeof mcp>} change */
    async function updateMCP(change) {
        mcp = { ...mcp, ...change };
        try {
            await saveMCPHost(mcp);
        } catch (error) {
            logger.error('Error saving the MCP server setting:', error);
        }
    }

    // The in-app assistant (ADR-0064): off by default; a remote provider asks for the user's
    // consent first, and its key goes to the system keyring, never to the settings file.
    let keyInput = $state('');
    let hasKey = $state(false);
    const preset = $derived(presetOf($assistantSettingsStore, $assistantPresetsStore));
    // Consent is tied to the address actually used: changing it asks again.
    const endpoint = $derived(effectiveURL($assistantSettingsStore, $assistantPresetsStore));
    const remote = $derived(isRemoteURL(endpoint));

    onMount(async () => {
        await loadAssistantSettings();
        await refreshKey();
    });

    async function refreshKey() {
        try {
            hasKey = !!(await AssistantHasKey($assistantSettingsStore.preset));
        } catch {
            hasKey = false;
        }
    }

    /** @param {Partial<import('../services/assistantService.js').AssistantSettings>} change */
    async function updateAssistant(change) {
        try {
            await saveAssistantSettings({ ...$assistantSettingsStore, ...change });
        } catch (error) {
            logger.error('Error saving the assistant setting:', error);
        }
    }

    /** @param {Event & { currentTarget: HTMLSelectElement }} e */
    async function onPreset(e) {
        // Another provider: its own address and model, and its own consent.
        await updateAssistant({ preset: e.currentTarget.value, baseURL: '', model: '', remoteAck: '' });
        await refreshKey();
    }

    async function saveKey() {
        try {
            await AssistantSetKey($assistantSettingsStore.preset, keyInput);
            keyInput = '';
        } catch (error) {
            logger.error('Error storing the API key:', error);
        }
        await refreshKey();
    }

    /** @param {Event & { currentTarget: HTMLInputElement }} e */
    function onPort(e) {
        const port = Math.round(Number(e.currentTarget.value));
        updateMCP({ port: port > 0 && port <= 65535 ? port : DEFAULT_MCP_PORT });
    }
</script>

<div class="assistant-settings">
    <h4>{$t('assistant.mcpTitle')}</h4>
    <p class="setting-note">{$t('assistant.mcpIntro')}</p>
    <div class="setting-row">
        <label for="config-mcp-on">{$t('assistant.mcpEnabled')}</label>
        <input id="config-mcp-on" type="checkbox" checked={mcp.on} onchange={(e) => updateMCP({ on: e.currentTarget.checked })} />
    </div>
    <div class="setting-row">
        <label for="config-mcp-port">{$t('assistant.mcpPort')}</label>
        <input id="config-mcp-port" type="number" class="setting-input" min="1" max="65535" value={mcp.port} onchange={onPort} />
    </div>
    <div class="setting-row">
        <label for="config-mcp-write">{$t('assistant.mcpWrite')}</label>
        <input id="config-mcp-write" type="checkbox" checked={mcp.write} onchange={(e) => updateMCP({ write: e.currentTarget.checked })} />
    </div>
    <p class="setting-note">{$t('assistant.mcpWriteNote')}</p>
    {#if $mcpHostStatusStore.error}
        <p class="setting-note warn">{$t('assistant.mcpError', { error: $mcpHostStatusStore.error })}</p>
    {:else if $mcpHostStatusStore.running}
        <p class="setting-note ok" data-testid="mcp-url">{$t('assistant.mcpRunning', { url: $mcpHostStatusStore.url })}</p>
    {:else}
        <p class="setting-note">{$t('assistant.mcpStopped')}</p>
    {/if}

    <h4>{$t('assistant.title')}</h4>
    <p class="setting-note">{$t('assistant.intro')}</p>
    <div class="setting-row">
        <label for="config-assistant-on">{$t('assistant.enabled')}</label>
        <input id="config-assistant-on" type="checkbox" checked={$assistantSettingsStore.on} onchange={(e) => updateAssistant({ on: e.currentTarget.checked })} />
    </div>
    <div class="setting-row">
        <label for="config-assistant-preset">{$t('assistant.preset')}</label>
        <select id="config-assistant-preset" value={$assistantSettingsStore.preset} onchange={onPreset}>
            {#each $assistantPresetsStore as p (p.id)}
                <option value={p.id}>{p.name}</option>
            {/each}
        </select>
    </div>
    <div class="setting-row">
        <label for="config-assistant-url">{$t('assistant.baseURL')}</label>
        <input
            id="config-assistant-url"
            type="text"
            class="setting-input"
            placeholder={preset?.baseURL || ''}
            value={$assistantSettingsStore.baseURL}
            onchange={(e) => updateAssistant({ baseURL: e.currentTarget.value.trim() })}
        />
    </div>
    <div class="setting-row">
        <label for="config-assistant-model">{$t('assistant.model')}</label>
        <input
            id="config-assistant-model"
            type="text"
            class="setting-input"
            placeholder={preset?.model || ''}
            value={$assistantSettingsStore.model}
            onchange={(e) => updateAssistant({ model: e.currentTarget.value.trim() })}
        />
    </div>
    <div class="setting-row">
        <label for="config-assistant-key">{$t('assistant.key')}</label>
        <input id="config-assistant-key" type="password" class="setting-input" autocomplete="off" bind:value={keyInput} />
        <button class="secondary-button" onclick={saveKey}>{$t('assistant.keySave')}</button>
    </div>
    <p class="setting-note" data-testid="assistant-key-state">{hasKey ? $t('assistant.keyStored') : $t('assistant.keyNone')}</p>
    <div class="setting-row">
        <label for="config-assistant-write">{$t('assistant.write')}</label>
        <input id="config-assistant-write" type="checkbox" checked={$assistantSettingsStore.write} onchange={(e) => updateAssistant({ write: e.currentTarget.checked })} />
    </div>
    {#if remote}
        <p class="setting-note warn" data-testid="assistant-privacy">{$t('assistant.privacyWarning', { provider: endpoint || preset?.name || '' })}</p>
        <div class="setting-row">
            <label for="config-assistant-ack">{$t('assistant.privacyAccept')}</label>
            <input
                id="config-assistant-ack"
                type="checkbox"
                checked={!!endpoint && $assistantSettingsStore.remoteAck === endpoint}
                disabled={!endpoint}
                onchange={(e) => updateAssistant({ remoteAck: e.currentTarget.checked ? endpoint : '' })}
            />
        </div>
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
    .secondary-button {
        padding: 4px 10px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        background-color: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
    }
    .setting-input {
        flex: 1;
        min-width: 0;
        max-width: 220px;
    }
</style>
