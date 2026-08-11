<script>
    import { trapFocus } from '../utils/focusTrap.js';
    import { t } from '../i18n';

    /**
     * Two modes:
     *  - 'info' (default): a single message with a close button — the original warning toast
     *    (e.g. database version mismatch).
     *  - 'confirm': a destructive-action confirmation. Mirrors native window.confirm() keyboard
     *    semantics so it feels familiar and stays fluid to drive without touching the mouse —
     *    Enter always confirms and Escape always cancels, regardless of which button (if any)
     *    has focus. onConfirm/onClose are both required in this mode.
     */
    let { message = '', visible = false, onClose = () => {}, mode = 'info', onConfirm = () => {}, confirmLabel = '', cancelLabel = '' } = $props();

    function handleClose() {
        onClose();
    }

    function handleConfirm() {
        onConfirm();
    }

    function handleKeyDown(event) {
        if (!visible) return;
        if (event.key === 'Escape') {
            event.preventDefault();
            handleClose();
        } else if (mode === 'confirm' && event.key === 'Enter') {
            // preventDefault so a focused button's own native Enter-activates-click doesn't
            // also fire — this handler is the single source of truth for what Enter does here.
            event.preventDefault();
            handleConfirm();
        }
    }

    function handleClickOutside(event) {
        if (!visible) return;
        const modalContent = document.getElementById('modalContent');
        if (modalContent && !modalContent.contains(event.target)) {
            handleClose();
        }
    }

    import { onMount, onDestroy } from 'svelte';

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('click', handleClickOutside);
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
        window.removeEventListener('click', handleClickOutside);
    });
</script>

{#if visible}
    <div
        class="modal-overlay"
        class:confirm-overlay={mode === 'confirm'}
        role="dialog"
        aria-modal="true"
        aria-label={mode === 'confirm' ? $t('warning.confirmTitle') : $t('warning.title')}
        use:trapFocus
    >
        <div class="modal-content" class:confirm-content={mode === 'confirm'} id="modalContent">
            {#if mode === 'info'}
                <div class="close-button" onclick={handleClose}>×</div>
            {/if}
            <div class="tab-content" class:confirm-tab-content={mode === 'confirm'}>
                <p><span class="highlight">{message.split('\n')[0]}</span></p>
                <p>{message.split('\n').slice(1).join('\n')}</p>
                <!-- Ensure the message is displayed correctly -->
            </div>
            {#if mode === 'confirm'}
                <div class="confirm-actions">
                    <button class="btn-cancel" onclick={handleClose}>{cancelLabel || $t('common.cancel')}</button>
                    <button class="btn-confirm" onclick={handleConfirm}>{confirmLabel || $t('common.delete')}</button>
                </div>
            {/if}
        </div>
    </div>
{/if}

<style>
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

    .modal-content {
        background-color: white;
        padding: 0; /* Remove padding */
        border-radius: 4px;
        width: 60%; /* Reduce width */
        height: 50%; /* Reduce height */
        box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
        position: relative;
        display: flex;
        flex-direction: column;
    }

    .close-button {
        position: absolute;
        top: -8px;
        right: 4px;
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        z-index: 10;
        transition:
            background-color 0.3s ease,
            opacity 0.3s ease;
    }

    .tab-content {
        flex-grow: 1;
        overflow-y: auto;
        border-top: 1px solid #ddd;
        padding: 0; /* Remove padding */
        box-sizing: border-box;
        height: calc(100% - 50px); /* Adjust height to ensure uniform tab size */
    }

    .tab-content p {
        margin: 20px; /* Add margin for spacing */
        text-align: justify;
        white-space: pre-wrap; /* Preserve whitespace for new lines */
    }

    .highlight {
        font-weight: bold;
        color: red;
    }

    /* Confirm mode: a small dialog that must be able to layer above any other modal or
       always-mounted panel it was triggered from (e.g. the config modal's bearoff delete),
       so it gets its own, higher stacking context regardless of DOM order. */
    .confirm-overlay {
        z-index: 1100;
    }

    .confirm-content {
        width: min(90%, 420px);
        height: auto;
    }

    .confirm-tab-content {
        border-top: none;
        height: auto;
    }

    .confirm-tab-content p {
        margin: 20px 20px 0 20px;
    }

    .confirm-actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        padding: 16px 20px 20px 20px;
    }

    .btn-cancel,
    .btn-confirm {
        padding: 4px 10px;
        border-radius: 4px;
        cursor: pointer;
    }

    .btn-cancel {
        border: 1px solid #ccc;
        background: white;
        color: #333;
    }

    .btn-confirm {
        border: 1px solid #b3261e;
        background: #b3261e;
        color: white;
    }
</style>
