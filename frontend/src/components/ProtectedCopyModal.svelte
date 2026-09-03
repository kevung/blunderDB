<script>
    // Asks for the password of a protected file. This is the ONE prompt a recipient ever
    // sees: once opened, the result is an ordinary database, nothing further is asked, and
    // nothing about the opening is recorded anywhere. See ADR-0007.
    import Modal from './Modal.svelte';
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
        if (event.key === 'Enter') {
            event.stopImmediatePropagation();
            submit();
        }
    }
</script>

<Modal open={visible} onclose={onCancel} size="medium" compactTitle closeButton={false} onkeydown={handleKeyDown}>
    {#snippet title()}{$t('issuance.protectedTitle')}{/snippet}
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
    {#snippet footer()}
        <button type="button" onclick={onCancel}>{$t('common.cancel')}</button>
        <button type="button" class="primary" onclick={submit} disabled={!password}>
            {$t('issuance.openCopy')}
        </button>
    {/snippet}
</Modal>

<style>
    .file {
        margin: 0 0 8px;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-base);
        color: #555;
        word-break: break-all;
    }

    .hint {
        margin: 0 0 10px;
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
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
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
        color: #3c4043;
        cursor: pointer;
    }

    .error {
        color: #b3261e;
        font-size: var(--font-size-base);
        margin: 6px 0 0;
    }
</style>
