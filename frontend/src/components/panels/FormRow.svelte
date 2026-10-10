<script>
    // One row of a `FormGrid`: its label in the first column, its controls in the second.

    /** @type {{ label: string, for?: string, children: import('svelte').Snippet }} */
    let { label, for: htmlFor = undefined, children } = $props();
</script>

<label class="form-label" for={htmlFor}>{label}</label>
<div class="form-control">{@render children()}</div>

<style>
    .form-label {
        color: var(--color-text-muted);
        white-space: nowrap;
    }
    .form-control {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: var(--space-1) var(--space-2);
        min-width: 0;
    }
</style>
