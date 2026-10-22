<script>
    import { toCssPx } from '../utils/uiZoom.js';
    import { onMount, tick } from 'svelte';
    import { t } from '../i18n';
    import { closeOnEscape } from '../services/escapeService.js';
    import { registerKeys } from '../services/keyDispatch.js';

    /**
     * Reusable context-menu popover.
     *
     * @typedef {{ label: string, onClick?: () => void, shortcut?: string, disabled?: boolean, keepOpen?: boolean, separatorBefore?: boolean, submenu?: MenuItem[] }} MenuItem
     *
     * Props:
     *   x           {number}      - Client X pixel where the menu appears
     *   y           {number}      - Client Y pixel where the menu appears
     *   items       {MenuItem[]}  - Menu items to display; `keepOpen` leaves the menu open on click;
     *                               `submenu` makes the item open a nested menu (hover, click, →)
     *   onClose     {() => void}  - Called when the menu should be dismissed
     *   returnFocus {HTMLElement} - Where the focus goes on close, instead of the opener
     *   testid      {string}      - data-testid of the menu
     *   children    {Snippet}     - Extra content under the items (a field an item opens)
     */
    let { x = 0, y = 0, items = [], onClose, returnFocus = null, testid = undefined, children = undefined } = $props();

    /** @type {HTMLElement | null} */
    let menuEl = $state(null);

    // Anchored to a button near an edge, the menu would spill out of the window: it is moved
    // back in once its size is known, and again when a field it opens makes it grow.
    let left = $state(0);
    let top = $state(0);
    /** @param {number} ax @param {number} ay */
    function fit(ax, ay) {
        // The anchor and the box are in window pixels; left/top are CSS pixels of the zoomed UI.
        const r = menuEl?.getBoundingClientRect();
        left = toCssPx(r && ax + r.width > window.innerWidth ? Math.max(0, window.innerWidth - r.width - 4) : ax);
        top = toCssPx(r && ay + r.height > window.innerHeight ? Math.max(0, window.innerHeight - r.height - 4) : ay);
    }
    $effect(() => {
        const ax = x;
        const ay = y;
        fit(ax, ay);
        if (!menuEl || typeof ResizeObserver === 'undefined') return;
        const observer = new ResizeObserver(() => fit(ax, ay));
        observer.observe(menuEl);
        return () => observer.disconnect();
    });

    onMount(() => {
        // Focus the first item immediately for keyboard access; the opener gets the focus back
        // on close so a keyboard user does not restart from the top of the page.
        const opener = /** @type {HTMLElement | null} */ (document.activeElement);
        /** @type {HTMLElement | null | undefined} */ (menuEl?.querySelector('button:not(:disabled)'))?.focus();
        return () => {
            const a = document.activeElement;
            const back = returnFocus ?? opener;
            if (back?.isConnected && (!a || a === document.body || menuEl?.contains(a) || !a.isConnected)) back.focus({ preventScroll: true });
        };
    });

    // Escape closes the menu before anything else sees it — the panels' tiers,
    // the global dispatcher (escapeService.js).
    $effect(() => closeOnEscape(() => onClose?.()));

    // Which submenu is open at each depth: openPath[d] is the index, in the menu of depth d, of
    // the item whose submenu shows. Opening one at depth d closes everything deeper.
    /** @type {number[]} */
    let openPath = $state([]);

    /** @param {number} depth @param {number} index @param {boolean} focusFirst */
    function openSubmenu(depth, index, focusFirst) {
        if (openPath[depth] !== index || openPath.length !== depth + 1) openPath = [...openPath.slice(0, depth), index];
        if (!focusFirst) return;
        tick().then(() => {
            const sub = menuEl?.querySelector(`.context-submenu[data-depth="${depth + 1}"]`);
            /** @type {HTMLElement | null | undefined} */ (sub && levelButtons(sub)[0])?.focus();
        });
    }

    /** Closes the submenus deeper than `depth` and gives the focus back to the item that opened the first of them. @param {number} depth */
    function closeSubmenus(depth, refocus = false) {
        if (openPath.length <= depth) return;
        const index = openPath[depth];
        openPath = openPath.slice(0, depth);
        if (refocus) /** @type {HTMLElement | null | undefined} */ (menuEl?.querySelector(`[data-depth="${depth}"][data-index="${index}"]`))?.focus();
    }

    /** The enabled items of one menu level, not those of its submenus. @param {Element} menu */
    function levelButtons(menu) {
        return /** @type {HTMLElement[]} */ ([...menu.querySelectorAll('button:not(:disabled)')].filter((b) => b.closest('[role="menu"]') === menu));
    }

    // Escape closes the deepest open submenu first, then the menu (the stack in escapeService.js
    // takes the last entry first).
    $effect(() => {
        const depth = openPath.length - 1;
        if (depth < 0) return;
        return closeOnEscape(() => closeSubmenus(depth, true));
    });

    /** @param {KeyboardEvent} event */
    function handleKeyDown(event) {
        // A field in the menu keeps its own caret keys.
        const inField = event.target instanceof Element && event.target.matches('input, textarea');
        const active = /** @type {HTMLElement | null} */ (document.activeElement);
        if (menuEl && !inField && (event.key === 'ArrowRight' || event.key === 'ArrowLeft')) {
            const depth = Number(active?.dataset?.depth ?? -1);
            if (event.key === 'ArrowRight' && active?.getAttribute('aria-haspopup') === 'menu') openSubmenu(depth, Number(active.dataset.index), true);
            else if (event.key === 'ArrowLeft' && depth > 0) closeSubmenus(depth - 1, true);
            else return;
            event.preventDefault();
            event.stopImmediatePropagation();
            return;
        }
        if (menuEl && !inField && ['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
            const level = active?.closest('[role="menu"]');
            const enabled = levelButtons(level && menuEl.contains(level) ? level : menuEl);
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
            const focusable = /** @type {HTMLElement[]} */ ([...menuEl.querySelectorAll('button:not(:disabled), input')]);
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

    // A menu opened by a click mounts while that click is still bubbling: it would reach this
    // window listener and close the menu it just opened.
    const openingEvent = typeof window !== 'undefined' ? window.event : undefined;

    /** @param {MouseEvent} event */
    function handleWindowClick(event) {
        if (event === openingEvent) return;
        if (menuEl && !menuEl.contains(/** @type {Node} */ (event.target))) {
            onClose?.();
        }
    }

    /** @param {MenuItem} item */
    function handleItemClick(item) {
        item.onClick?.();
        if (!item.keepOpen) onClose?.();
    }

    // A submenu that would leave the window opens on the other side of its item, or moves up.
    /** @param {HTMLElement} node */
    function fitSubmenu(node) {
        const r = node.getBoundingClientRect();
        if (r.right > window.innerWidth) {
            node.style.left = 'auto';
            node.style.right = '100%';
        }
        if (r.bottom > window.innerHeight) node.style.top = `${-toCssPx(Math.min(r.top, r.bottom - window.innerHeight + 4))}px`;
    }

    $effect(() => registerKeys('contextMenu', handleKeyDown));
</script>

<svelte:window onclick={handleWindowClick} oncontextmenucapture={handleWindowContextMenu} />

{#snippet level(/** @type {MenuItem[]} */ list, /** @type {number} */ depth)}
    {#each list as item, index (item.label)}
        {#if item.separatorBefore}<hr class="context-menu-sep" />{/if}
        {#if item.submenu}
            <div class="context-menu-branch">
                <button
                    class="context-menu-item"
                    role="menuitem"
                    aria-haspopup="menu"
                    aria-expanded={openPath[depth] === index}
                    data-depth={depth}
                    data-index={index}
                    disabled={item.disabled}
                    onmouseenter={() => openSubmenu(depth, index, false)}
                    onclick={() => openSubmenu(depth, index, true)}
                >
                    {item.label}<span class="context-menu-arrow" aria-hidden="true">▸</span>
                </button>
                {#if openPath[depth] === index}
                    <div class="context-menu context-submenu" role="menu" aria-label={item.label} data-depth={depth + 1} use:fitSubmenu>
                        {@render level(item.submenu, depth + 1)}
                    </div>
                {/if}
            </div>
        {:else}
            <button
                class="context-menu-item"
                role="menuitem"
                data-depth={depth}
                data-index={index}
                disabled={item.disabled}
                onmouseenter={() => closeSubmenus(depth)}
                onclick={() => handleItemClick(item)}
            >
                {item.label}{#if item.shortcut}<kbd>{item.shortcut}</kbd>{/if}
            </button>
        {/if}
    {/each}
{/snippet}

<div bind:this={menuEl} class="context-menu" style="left:{left}px; top:{top}px" role="menu" aria-label={$t('common.contextMenu')} data-testid={testid}>
    {@render level(items, 0)}
    {@render children?.()}
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

    .context-menu-branch {
        position: relative;
    }

    /* Beside its item, in the CSS pixels of the zoomed interface like the menu itself. */
    .context-submenu {
        position: absolute;
        left: 100%;
        top: -3px;
    }

    .context-menu-arrow {
        float: right;
        margin-left: 18px;
        opacity: 0.6;
    }

    .context-menu-sep {
        margin: 4px 0;
        border: 0;
        border-top: 1px solid var(--color-border, rgb(128 128 128 / 0.4));
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
    .context-menu-item:focus,
    .context-menu-item[aria-expanded='true'] {
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
