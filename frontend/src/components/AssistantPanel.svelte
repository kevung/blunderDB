<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import {
        assistantSettingsStore,
        assistantEntriesStore,
        assistantPendingStore,
        assistantBusyStore,
        assistantErrorStore,
        loadAssistantSettings,
        askAssistant,
        confirmAssistant,
        cancelAssistant,
        resetAssistant
    } from '../services/assistantService.js';

    // A sentence becomes a view: the assistant searches with the application's grammar and the
    // window opens the result in a new view tab named after the sentence. What the tools return
    // is the database's; what the model writes is free text, and is marked so.
    let sentence = $state('');

    onMount(loadAssistantSettings);

    async function send() {
        if (await askAssistant(sentence)) sentence = '';
    }

    /** @param {KeyboardEvent} e */
    function onKey(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            send();
        }
    }
</script>

<div class="assistant-panel" data-testid="assistant-panel">
    {#if !$assistantSettingsStore.on}
        <p class="note">{$t('assistant.disabled')}</p>
    {:else}
        <div class="ask-row">
            <input
                type="text"
                class="ask-input"
                aria-label={$t('assistant.placeholder')}
                placeholder={$t('assistant.placeholder')}
                bind:value={sentence}
                onkeydown={onKey}
                disabled={$assistantBusyStore || !!$assistantPendingStore}
            />
            {#if $assistantBusyStore}
                <button class="secondary-button" onclick={cancelAssistant}>{$t('assistant.cancel')}</button>
            {:else}
                <button class="secondary-button" onclick={send} disabled={!sentence.trim() || !!$assistantPendingStore}>{$t('assistant.send')}</button>
            {/if}
            <button class="secondary-button" onclick={resetAssistant} disabled={$assistantBusyStore}>{$t('assistant.reset')}</button>
        </div>
        {#if $assistantErrorStore}
            <p class="note warn">{$assistantErrorStore === 'privacy' ? $t('assistant.needsAck') : $t('assistant.error', { error: $assistantErrorStore })}</p>
        {/if}
        <ol class="transcript">
            {#each $assistantEntriesStore as entry, i (i)}
                <li class="entry {entry.kind}">
                    {#if entry.kind === 'user'}
                        <span class="who">›</span> {entry.text}
                    {:else if entry.kind === 'model'}
                        <span class="badge">{$t('assistant.freeText')}</span>
                        <span class="free">{entry.text}</span>
                    {:else if entry.kind === 'tool'}
                        <code>{entry.tool}</code> <span class="args">{entry.args}</span>
                    {:else}
                        <code>{entry.tool || ''}</code>
                        <span class="warn">{entry.text === 'declined' ? $t('assistant.declined') : entry.text}</span>
                    {/if}
                </li>
            {/each}
        </ol>
        {#if $assistantPendingStore}
            <div class="pending" role="alertdialog" aria-label={$t('assistant.pendingTitle')}>
                <strong>{$t('assistant.pendingTitle')}</strong> — {$assistantPendingStore.title}
                <pre>{$assistantPendingStore.args}</pre>
                <button class="secondary-button" onclick={() => confirmAssistant(true)} disabled={$assistantBusyStore}>{$t('assistant.confirm')}</button>
                <button class="secondary-button" onclick={() => confirmAssistant(false)} disabled={$assistantBusyStore}>{$t('assistant.decline')}</button>
            </div>
        {/if}
    {/if}
</div>

<style>
    .assistant-panel {
        /* Interface chrome is not text to copy; fields below opt back in. */
        user-select: none;
        -webkit-user-select: none;
        display: flex;
        flex-direction: column;
        gap: 6px;
        text-align: left;
        min-width: 0;
    }
    .ask-row {
        display: flex;
        gap: 6px;
    }
    .ask-input {
        flex: 1;
        min-width: 0;
    }
    .note {
        color: var(--color-text-muted);
        margin: 4px 0;
    }
    .warn {
        color: var(--color-danger);
    }
    .transcript {
        list-style: none;
        margin: 0;
        padding: 0;
        overflow-y: auto;
    }
    .entry {
        margin: 2px 0;
        overflow-wrap: anywhere;
    }
    .entry.tool,
    .entry.error {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
    .badge {
        border: 1px solid var(--color-border);
        border-radius: 3px;
        padding: 0 4px;
        margin-right: 4px;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
    .free {
        font-style: italic;
    }
    .pending {
        border: 1px solid var(--color-primary);
        border-radius: 4px;
        padding: 6px;
    }
    .pending pre {
        white-space: pre-wrap;
        margin: 4px 0;
    }
    .secondary-button {
        padding: 4px 10px;
        border: 1px solid var(--color-border);
        border-radius: 4px;
        background-color: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
    }

    .assistant-panel input {
        user-select: text;
        -webkit-user-select: text;
    }

    .assistant-panel .transcript {
        user-select: text;
        -webkit-user-select: text;
    }
</style>
