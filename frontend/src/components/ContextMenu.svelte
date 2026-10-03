<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { closeOnEscape } from '../services/escapeService.js';
    import { registerKeys } from '../services/keyDispatch.js';

    /**
     * Reusable context-menu popover.
     *
     * @typedef {{ label: string, onClick: () => void, shortcut?: string, disabled?: boolean }} MenuItem
     *
     * Props:
     *   x       {number}     - Client X pixel where the menu appears
     *   y       {number}     - Client Y pixel where the menu appears
     *   items   {MenuItem[]} - Menu items to display
     *   onClose {() => void} - Called when the menu should be dismissed
     */
    let { x = 0, y = 0, items = [], onClose } = $props();

    /** @type {HTMLElement | null} */
    let menuEl = $state(null);

    onMount(() => {
        // Focus the first item immediately for keyboard access; the opener gets the focus back
        // on close so a keyboard user does not restart from the top of the page.
        const opener = /** @type {HTMLElement | null} */ (document.activeElement);
        /** @type {HTMLElement | null | undefined} */ (menuEl?.querySelector('button:not(:disabled)'))?.focus();
        return () => {
            const a = document.activeElement;
            if (opener?.isConnected && (!a || a === document.body || menuEl?.contains(a) || !a.isConnected)) opener.focus();
        };
    });

    // Escape closes the menu before anything else sees it — the panels' tiers,
    // the global dispatcher (escapeService.js).
    $effect(() => closeOnEscape(() => onClose?.()));

    /** @param {KeyboardEvent} event */
    function handleKeyDown(event) {
        if (menuEl && ['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
            const enabled = /** @type {HTMLElement[]} */ ([...menuEl.querySelectorAll('button:not(:disabled)')]);
            if (enabled.length === 0) return;
            const at = enabled.indexOf(/** @type {HTMLElement} */ (document.activeElement));
            let next;
            if (event.key === 'ArrowDown') next = (at + 1) % enabled.length;
            else if (event.key === 'ArrowUp') next = (at - 1 + enabled.length) % enabled.length;
            else next = event.key === 'Home' ? 0 : enabled.length - 1;
            event.preventDefault();
            event.stopImmediatePropagation();
            enabled[next].focus();
            return;
        }
        if (event.key === 'Tab' && menuEl) {
            // Trap focus inside menu
            const focusable = [...menuEl.querySelectorAll('button')];
            if (focusable.length === 0) return;
            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        }
    }

    /* Un seul menu à la fois : un clic droit hors de celui-ci le ferme, en capture, avant que
       l'objet visé n'ouvre le sien. */
    /** @param {MouseEvent} event */
    function handleWindowContextMenu(event) {
        if (menuEl && !menuEl.contains(/** @type {Node} */ (event.target))) onClose?.();
    }

    function handleWindowClick(event) {
        if (menuEl && !menuEl.contains(event.target)) {
            onClose?.();
        }
    }

    function handleItemClick(item) {
        item.onClick();
        onClose?.();
    }

    $effect(() => registerKeys('contextMenu', handleKeyDown));
</script>

<svelte:window onclick={handleWindowClick} oncontextmenucapture={handleWindowContextMenu} />

<div bind:this={menuEl} class="context-menu" style="left:{x}px; top:{y}px" role="menu" aria-label={$t('common.contextMenu')}>
    {#each items as item (item.label)}
        <button class="context-menu-item" role="menuitem" disabled={item.disabled} onclick={() => handleItemClick(item)}>
            {item.label}{#if item.shortcut}<kbd>{item.shortcut}</kbd>{/if}
        </button>
    {/each}
</div>

<style>
    .context-menu {
        position: fixed;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        border-radius: 4px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.14);
        z-index: 1000;
        min-width: 170px;
        padding: 3px 0;
    }

    .context-menu-item {
        display: block;
        width: 100%;
        text-align: left;
        background: none;
        border: none;
        padding: 6px 14px;
        font-size: var(--font-size-base);
        cursor: pointer;
        color: var(--color-text);
        border-radius: 0;
    }

    .context-menu-item:hover,
    .context-menu-item:focus {
        background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface));
        outline: none;
    }

    .context-menu-item:disabled {
        opacity: 0.5;
        cursor: default;
    }

    .context-menu-item kbd {
        float: right;
        margin-left: 18px;
        opacity: 0.6;
    }

    .context-menu-item:focus-visible {
        outline: 2px solid var(--color-primary);
        outline-offset: -2px;
    }
</style>
