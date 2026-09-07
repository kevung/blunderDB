<script>
    // Une case de nombre, en mode DÉCLARÉ (ADR-0040 règle 2).
    //
    // Avant « Révéler » elle est vide : la question est justement de savoir ce
    // qui va s'y écrire. Après, elle porte la vérité et devient cliquable —
    // chaque nombre est juste par défaut, on clique celui qu'on a raté. C'est
    // un bouton et non une case à cocher pour que Tab + Espace fasse le même
    // geste que la souris, sans toucher numérotée à retenir.
    //
    // La faute se dit par un glyphe (×) que la couleur redouble ; elle ne se
    // dit jamais par la couleur seule (ADR-0031).
    import { t } from '../i18n';

    let { value = 0, precision = 0, revealed = false, fault = false, locked = false, label = '', onToggle = () => {} } = $props();
</script>

<button
    type="button"
    class="number-cell"
    class:revealed
    class:fault
    disabled={!revealed || locked}
    aria-pressed={fault}
    aria-label={label ? `${label} — ${revealed ? value.toFixed(precision) : ''}` : undefined}
    title={revealed && !locked ? $t('training.faultHint') : undefined}
    onclick={() => onToggle()}
>
    {#if revealed}
        <span class="mark" aria-hidden="true">{fault ? '×' : ''}</span><span class="value">{value.toFixed(precision)}</span>
    {/if}
</button>

<style>
    .number-cell {
        display: inline-flex;
        align-items: baseline;
        justify-content: flex-end;
        gap: 0.15em;
        min-width: 3.4em;
        padding: 0.1em 0.35em;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
        font-variant-numeric: tabular-nums;
        cursor: default;
    }

    .number-cell.revealed:not(:disabled) {
        cursor: pointer;
    }

    .number-cell.revealed:not(:disabled):hover {
        background: var(--color-surface-alt);
    }

    .number-cell.fault {
        color: var(--color-danger);
        border-color: var(--color-danger);
    }

    .mark {
        font-weight: 600;
    }
</style>
