<script>
    // Asks for the password of a protected copy. This is the ONE prompt the recipient of a
    // watermarked copy ever sees: once opened, the result is an ordinary database and
    // nothing in the product interrupts them again. See ADR-0007.
    import { onMount, onDestroy } from 'svelte';
    import { trapFocus } from '../utils/focusTrap.js';
    import { t } from '../i18n';

    let { visible = false, fileName = '', error = '', onSubmit = () => {}, onCancel = () => {} } = $props();

    let password = $state('');
    let input = $state(null);

    $effect(() => {
        if (visible && input) input.focus();
    });

    function submit() {
        if (password) onSubmit(password);
    }

    function handleKeyDown(event) {
        if (!visible) return;
        if (event.key === 'Escape') {
            event.stopImmediatePropagation();
            onCancel();
        } else if (event.key === 'Enter') {
            event.stopImmediatePropagation();
            submit();
        }
    }

    onMount(() => window.addEventListener('keydown', handleKeyDown));
    onDestroy(() => window.removeEventListener('keydown', handleKeyDown));
</script>

{#if visible}
    <div class="modal-overlay" role="dialog" aria-modal="true" aria-label={$t('issuance.protectedTitle')} use:trapFocus>
        <div class="modal-content">
            <h2>{$t('issuance.protectedTitle')}</h2>
            <p class="file">{fileName}</p>
            <p class="hint">{$t('issuance.protectedHint')}</p>
            <input bind:this={input} bind:value={password} type="password" aria-label={$t('issuance.passwordLabel')} placeholder={$t('issuance.passwordLabel')} />
            {#if error}<p class="error">{error}</p>{/if}
            <div class="buttons">
                <button type="button" onclick={onCancel}>{$t('common.cancel')}</button>
                <button type="button" class="primary" onclick={submit} disabled={!password}>
                    {$t('issuance.openCopy')}
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        inset: 0;
        background-color: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }

    .modal-content {
        background: white;
        padding: 20px;
        border-radius: 6px;
        min-width: 340px;
        max-width: 460px;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
    }

    h2 {
        margin: 0 0 6px;
        font-size: 16px;
    }

    .file {
        margin: 0 0 8px;
        font-family: monospace;
        font-size: 12px;
        color: #555;
        word-break: break-all;
    }

    .hint {
        margin: 0 0 10px;
        font-size: 12px;
        color: #666;
        line-height: 1.35;
    }

    input {
        width: 100%;
        box-sizing: border-box;
        padding: 6px 8px;
        font-size: 14px;
    }

    .error {
        color: #b3261e;
        font-size: 12px;
        margin: 6px 0 0;
    }

    .buttons {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        margin-top: 14px;
    }

    button {
        padding: 6px 14px;
        cursor: pointer;
    }

    .primary {
        background: #1a73e8;
        color: white;
        border: 1px solid #1a73e8;
    }

    .primary:disabled {
        opacity: 0.5;
        cursor: default;
    }
</style>
