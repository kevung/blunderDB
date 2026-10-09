<script>
    // Une plage opaque à la place de la réponse : masquer les lignes sur place trahirait encore le
    // meilleur coup par son rang (ADR-0025 règle 3). Avec `onReveal`, un clic la dévoile ; sans,
    // elle est inerte (une question d'entraînement se dévoile par son verdict).
    /** @type {{ onReveal?: (() => void) | null, title?: string, class?: string }} */
    let { onReveal = null, title = '', class: extra = '' } = $props();
</script>

{#if onReveal}
    <button type="button" class="answer-mask {extra}" onclick={onReveal} {title} aria-label={title}>···</button>
{:else}
    <div class="answer-mask inert {extra}" {title}>···</div>
{/if}

<style>
    .answer-mask {
        display: flex;
        align-items: center;
        justify-content: center;
        box-sizing: border-box;
        width: 100%;
        min-height: 6em;
        padding: 14px 0;
        border: 1px dashed var(--color-border);
        border-radius: 3px;
        background: var(--color-surface-alt);
        color: var(--color-text-muted);
        letter-spacing: 0.4em;
        cursor: pointer;
    }
    .answer-mask:hover:not(.inert) {
        background: color-mix(in srgb, var(--color-text) 6%, var(--color-surface-alt));
    }
    .answer-mask.inert {
        cursor: default;
    }
</style>
