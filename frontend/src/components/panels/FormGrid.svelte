<script>
    // A launcher's settings (ADR-0085, G6): two columns, label │ control, one row per setting,
    // labels at the start. Rows are `FormRow`s; the button that starts goes in the header strip.

    /** @type {{ children: import('svelte').Snippet }} */
    let { children } = $props();
</script>

<div class="form-grid" data-testid="form-grid">{@render children()}</div>

<style>
    .form-grid {
        display: grid;
        grid-template-columns: max-content 1fr;
        align-items: center;
        gap: var(--space-1) var(--space-2);
        padding: var(--space-2);
        text-align: start;
    }
</style>
