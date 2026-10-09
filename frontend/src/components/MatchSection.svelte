<script>
    // A folded section of the match sheet: what a study of the match needs less
    // often than the summary, the charts and the transcript. Folded by default;
    // the open state is remembered per section, so a user who keeps one open
    // finds it open on the next match.
    import { untrack } from 'svelte';
    import { isSectionOpen, rememberSectionOpen } from '../utils/matchSheet.js';

    /** @type {{ id: string, title: string, hint?: string, onopen?: () => void, children: import('svelte').Snippet }} */
    let { id, title, hint = '', onopen = () => {}, children } = $props();

    // A section's id never changes: it is read once, at mount.
    let open = $state(untrack(() => isSectionOpen(id)));

    $effect(() => {
        if (open) onopen();
    });

    /** @param {Event} e */
    function toggle(e) {
        const now = /** @type {HTMLDetailsElement} */ (e.currentTarget).open;
        if (now === open) return;
        open = now;
        rememberSectionOpen(id, now);
    }
</script>

<details class="match-section" data-testid="match-section-{id}" {open} ontoggle={toggle}>
    <summary title={hint || undefined}>{title}</summary>
    <div class="body">{@render children()}</div>
</details>

<style>
    .match-section {
        border-top: 1px solid var(--color-border);
        text-align: left;
    }
    summary {
        padding: 6px 12px;
        cursor: pointer;
        font-weight: 600;
        color: var(--color-text);
        user-select: none;
    }
    summary:hover {
        background: var(--color-surface-alt);
    }
    .body {
        padding: 0 12px 8px;
    }
</style>
