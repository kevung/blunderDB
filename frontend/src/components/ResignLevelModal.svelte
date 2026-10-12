<script>
    // The level of a resignation, asked over the theatre, where the panel's own buttons are
    // covered. Its keys are the panel's (transcriptionKeys.js, PHASE.RESIGN): the focus stays on
    // the panel, the buttons only repeat them for the mouse. Escape goes through the overlay
    // stack, above the theatre's own Escape, so it cancels the resignation and nothing else.
    import { t } from '../i18n';
    import { closeOnEscape } from '../services/escapeService.js';
    import { portal } from '../utils/portal.js';

    let {
        open = false,
        /** Who resigns, named in the dialog's label. */
        player = '',
        /** @type {(level: number) => void} */
        onLevel = () => {},
        /** @type {() => void} */
        onCancel = () => {}
    } = $props();

    const LEVELS = [
        { level: 1, key: 'transcription.resignSingle' },
        { level: 2, key: 'transcription.resignGammon' },
        { level: 3, key: 'transcription.resignBackgammon' }
    ];

    $effect(() => {
        if (open) return closeOnEscape(() => onCancel());
    });

    /** @param {MouseEvent} event */
    const keepFocus = (event) => event.preventDefault();
</script>

{#if open}
    <div class="resign-overlay" use:portal={document.body} data-testid="resign-modal">
        <div class="resign-box" role="dialog" aria-modal="true" aria-label={$t('transcription.resignPrompt', { player })}>
            <div class="resign-title">{$t('transcription.resignPrompt', { player })}</div>
            <div class="resign-levels">
                {#each LEVELS as entry (entry.level)}
                    <button
                        type="button"
                        class="resign-level"
                        data-level={entry.level}
                        onmousedown={keepFocus}
                        onclick={() => onLevel(entry.level)}
                        title={$t('transcription.resignLevelTooltip', { n: entry.level })}
                    >
                        <kbd>{entry.level}</kbd>{$t(entry.key)}
                    </button>
                {/each}
            </div>
            <button type="button" class="resign-cancel" onmousedown={keepFocus} onclick={() => onCancel()} title={$t('transcription.resignCancelTooltip')}>
                {$t('transcription.resignCancel')}
            </button>
        </div>
    </div>
{/if}

<style>
    /* Above the theatre (900) and its video, under the application's dialogs (1000). */
    .resign-overlay {
        position: fixed;
        inset: 0;
        z-index: 950;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgb(0 0 0 / 0.35);
    }
    .resign-box {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-3);
        border-radius: 8px;
        background: var(--color-surface);
        color: var(--color-text);
        box-shadow: 0 4px 16px rgb(0 0 0 / 0.4);
    }
    .resign-title {
        font-weight: 600;
    }
    .resign-levels {
        display: flex;
        gap: var(--space-2);
    }
    .resign-level,
    .resign-cancel {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        cursor: pointer;
    }
    kbd {
        font-family: inherit;
        font-size: var(--font-size-small);
        opacity: 0.7;
    }
</style>
