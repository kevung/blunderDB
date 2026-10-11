<script>
    // A count of positions is a way to those positions (ADR-0085, G7): a button drawn as the
    // status bar's dotted link, never inert text.

    /** @type {{ label: string | number, onclick: (event: MouseEvent) => void, title?: string, testid?: string }} */
    let { label, onclick, title = undefined, testid = 'count-link' } = $props();
</script>

<button type="button" class="count-link" data-testid={testid} {title} {onclick}>{label}</button>

<style>
    /* No `font: inherit`: style.css already sets it (ADR-0008). */
    .count-link {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        cursor: pointer;
        text-decoration: underline dotted;
        font-variant-numeric: tabular-nums;
        flex-shrink: 0;
    }
    .count-link:hover,
    .count-link:focus-visible {
        color: var(--color-primary);
    }
</style>
