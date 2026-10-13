<script>
    // Ce que font Entrée et les chiffres à l'instant (ADR-0092), sur la barre du panneau et
    // sur celle du théâtre : la frappe change de sens selon la place du Cursor.
    import { t } from '../i18n';

    /** @type {{ state: {mode: string, n: number, move: string, rolled: boolean, phase?: string} | null, dark?: boolean }} */
    let { state: badge = null, dark = false } = $props();

    let label = $derived(!badge ? '' : badge.mode === 'edit' ? $t('transcription.mode.edit', { n: badge.n }) : $t('transcription.mode.entry'));
    let enter = $derived(badge?.move ? $t('transcription.modeEnter', { move: badge.move }) : '');
    // Le sens des chiffres suit la phase : sans dés à la réponse au videau, le second dé
    // d'un jet commencé, puis le mode.
    let digits = $derived(
        !badge
            ? ''
            : badge.phase === 'answer'
              ? $t('transcription.modeDigits.answer')
              : badge.phase === 'die1'
                ? $t('transcription.modeDigits.die1')
                : badge.mode === 'edit'
                  ? $t('transcription.modeDigits.edit', { n: badge.n })
                  : badge.rolled
                    ? $t('transcription.modeDigits.entry')
                    : $t('transcription.modeDigits.roll')
    );
</script>

{#if badge}
    <!-- Rogné à l'étroit, jamais masqué : le mode d'abord, le texte entier au survol. -->
    <span class="mode-badge" class:dark data-testid="transcription-mode" data-mode={badge.mode} title={[label, enter, digits].filter(Boolean).join(' · ')}
        >{label}{#if enter}<span class="mode-key"> · {enter}</span>{/if}<span class="mode-key"> · {digits}</span></span
    >
{/if}

<style>
    /* Toujours visible : la frappe change de sens avec lui. */
    .mode-badge {
        display: inline-block;
        min-width: 0;
        max-width: 100%;
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
        vertical-align: middle;
        padding: 0 var(--space-1);
        border: 1px solid var(--color-border);
        border-radius: var(--radius-sm, 6px);
        font-size: var(--font-size-small);
        color: var(--color-text);
        background: var(--color-surface-alt);
    }

    .mode-badge[data-mode='edit'] {
        border-color: var(--color-primary);
        color: var(--color-primary);
    }

    .mode-key {
        color: var(--color-text-muted);
    }

    .mode-badge.dark {
        border-color: rgb(255 255 255 / 0.5);
        color: white;
        background: transparent;
    }

    .mode-badge.dark .mode-key {
        color: rgb(255 255 255 / 0.7);
    }
</style>
