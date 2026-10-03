<script>
    // The one "new …" gesture of a panel: a "+" and the name of what it creates, at the right of
    // the panel's header strip. A panel opens its creation field from it rather than leaving an
    // unlabelled field at the bottom of the list.

    /** @type {{ label: string, title?: string, disabled?: boolean, onclick: () => void }} */
    let { label, title = undefined, disabled = false, onclick } = $props();
</script>

<button type="button" class="new-btn" data-testid="panel-new" {disabled} title={title ?? label} {onclick}>
    <span aria-hidden="true">+</span>
    {label}
</button>

<style>
    .new-btn {
        margin-left: auto;
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        font-size: var(--font-size-small);
        cursor: pointer;
    }
    .new-btn:disabled {
        opacity: 0.5;
        cursor: default;
    }
</style>
