<script>
    import { onMount } from 'svelte';

    /**
     * Reusable context-menu popover.
     *
     * @typedef {{ label: string, onClick: () => void }} MenuItem
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
        // Focus the first item immediately for keyboard access
        menuEl?.querySelector('button')?.focus();
    });

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            event.stopPropagation();
            onClose?.();
        }
        if (event.key === 'Tab' && menuEl) {
            // Trap focus inside menu
            const focusable = [...menuEl.querySelectorAll('button')];
            if (focusable.length === 0) return;
            const first = focusable[0];
            const last  = focusable[focusable.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        }
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
</script>

<svelte:window onkeydown={handleKeyDown} onclick={handleWindowClick} />

<div
    bind:this={menuEl}
    class="context-menu"
    style="left:{x}px; top:{y}px"
    role="menu"
    aria-label="Actions"
>
    {#each items as item (item.label)}
        <button
            class="context-menu-item"
            role="menuitem"
            onclick={() => handleItemClick(item)}
        >{item.label}</button>
    {/each}
</div>

<style>
    .context-menu {
        position: fixed;
        background: #fff;
        border: 1px solid #ddd;
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
        font-size: 12px;
        cursor: pointer;
        color: #333;
        border-radius: 0;
    }

    .context-menu-item:hover,
    .context-menu-item:focus {
        background: #f0f4ff;
        outline: none;
    }

    .context-menu-item:focus-visible {
        outline: 2px solid #1976d2;
        outline-offset: -2px;
    }
</style>
