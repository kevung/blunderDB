<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { GetMCPHost } from '../../wailsjs/go/main/Config.js';
    import { DEFAULT_MCP_PORT, mcpHostStatusStore, saveMCPHost, refreshMCPHostStatus } from '../services/mcpHostService.js';
    import { logger } from '../utils/logger.js';

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
    .setting-input {
        flex: 1;
        min-width: 0;
        max-width: 220px;
    }
</style>
