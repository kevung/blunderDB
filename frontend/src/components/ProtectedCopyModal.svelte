<script>
    // Asks for the password of a protected file. This is the ONE prompt a recipient ever
    // sees: once opened, the result is an ordinary database, nothing further is asked, and
    // nothing about the opening is recorded anywhere. See ADR-0007.
    import { onMount, onDestroy } from 'svelte';
    import { trapFocus } from '../utils/focusTrap.js';
    import { t } from '../i18n';

    let { visible = false, fileName = '', error = '', onSubmit = () => {}, onCancel = () => {} } = $props();

    let password = $state('');
    let passwordVisible = $state(false);
    // Offered, never assumed: once opened, the recipient has the same content twice under
    // two names, which is a nuisance — but the protected file is theirs to keep if they
    // want to pass it on, so the box starts unticked.
    let removeContainer = $state(false);
    let input = $state(null);

    $effect(() => {
        if (visible && input) input.focus();
    });

    function submit() {
        if (password) onSubmit(password, removeContainer);
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
            <div class="password-row">
                <input bind:this={input} bind:value={password} type={passwordVisible ? 'text' : 'password'} aria-label={$t('issuance.passwordLabel')} placeholder={$t('issuance.passwordLabel')} />
                <!-- Same behaviour as the export dialog: revealed only while the button is
                     held, never toggled, so a password cannot be left showing on screen. -->
                <button
                    type="button"
                    class="reveal"
                    aria-label={$t('issuance.revealPassword')}
                    title={$t('issuance.revealPassword')}
                    onpointerdown={() => (passwordVisible = true)}
                    onpointerup={() => (passwordVisible = false)}
                    onpointerleave={() => (passwordVisible = false)}
                    onpointercancel={() => (passwordVisible = false)}
                    onkeydown={(e) => {
                        if (e.key === ' ' || e.key === 'Enter') passwordVisible = true;
                    }}
                    onkeyup={() => (passwordVisible = false)}
                    onblur={() => (passwordVisible = false)}
                >
                    {passwordVisible ? '🙈' : '👁'}
                </button>
            </div>
            <label class="remove-row">
                <input type="checkbox" bind:checked={removeContainer} />
                {$t('issuance.removeContainer')}
            </label>
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

    .password-row {
        display: flex;
        gap: 6px;
        align-items: stretch;
    }

    input {
        flex: 1;
        min-width: 0;
        box-sizing: border-box;
        padding: 6px 8px;
        font-size: 14px;
    }

    .reveal {
        flex: none;
        padding: 0 8px;
        cursor: pointer;
        line-height: 1;
    }

    .remove-row {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-top: 10px;
        font-size: 12px;
        color: #3c4043;
        cursor: pointer;
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
