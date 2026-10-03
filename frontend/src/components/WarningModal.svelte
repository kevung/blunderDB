<script>
    import Modal from './Modal.svelte';
    import { t } from '../i18n';

    /**
     * 'info' (default): a message and a close button. 'confirm': a destructive
     * confirmation where Enter confirms and Escape cancels whatever has focus;
     * onConfirm/onClose are required.
     */
    let { message = '', visible = false, onClose = () => {}, mode = 'info', onConfirm = () => {}, confirmLabel = '', cancelLabel = '', choices = [], onChoose = () => {} } = $props();

    function handleKeyDown(event) {
        if (mode === 'confirm' && event.key === 'Enter') {
            // preventDefault so a focused button's own native Enter-activates-click doesn't
            // also fire — this handler is the single source of truth for what Enter does here.
            event.preventDefault();
            if (choices.length) onChoose((choices.find((c) => c.primary) ?? choices[0]).value);
            else onConfirm();
        }
    }
</script>

{#snippet confirmActions()}
    <button onclick={onClose}>{cancelLabel || $t('common.cancel')}</button>
    {#if choices.length}
        {#each choices as c (c.value)}
            <button class={c.primary ? 'primary' : ''} data-testid="choice-{c.value}" onclick={() => onChoose(c.value)}>{c.label}</button>
        {/each}
    {:else}
        <button class="danger" onclick={onConfirm}>{confirmLabel || $t('common.delete')}</button>
    {/if}
{/snippet}

<!-- Confirm mode must be able to layer above any other modal or always-mounted panel it was
     triggered from (e.g. the config modal's bearoff delete): the raised layer. -->
<Modal
    open={visible}
    onclose={onClose}
    size="medium"
    layer={mode === 'confirm' ? 'raised' : 'base'}
    closeOnOverlay
    closeButton={mode === 'info'}
    label={mode === 'confirm' ? $t('warning.confirmTitle') : $t('warning.title')}
    onkeydown={handleKeyDown}
    footer={mode === 'confirm' ? confirmActions : undefined}
>
    <div class="message">
        <p><span class="highlight">{message.split('\n')[0]}</span></p>
        <p>{message.split('\n').slice(1).join('\n')}</p>
    </div>
</Modal>

<style>
    .message p {
        margin: 0 0 8px;
        text-align: justify;
        white-space: pre-wrap; /* Preserve whitespace for new lines */
    }

    .highlight {
        font-weight: bold;
        color: var(--color-danger);
    }
</style>
