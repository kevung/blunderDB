<!--
  The one overlay + dialog box every modal of the application is built on. It owns
  what the thirteen modals used to each carry a copy of: the backdrop, the box and
  its sizes, the close cross, the focus trap, and the Escape key. A modal only
  brings its content (`title` / `children` / `footer` snippets) and its own keys
  through `onkeydown`.

  Keyboard handling sits on the dialog element, not on window. The focus trap keeps
  the focus inside the box (the overlay itself carries tabindex="-1", so a click on
  plain text lands on it rather than on <body>), which means every key pressed while
  the modal is open bubbles through here. Escape is stopped with
  stopImmediatePropagation, as HelpModal did: the global dispatcher is another
  window-level listener, and closing a modal on a key it also handles would let it
  re-process the same event and re-open the dialog (a load-order race).

  Layout: the overlay scrolls, the box never does — WebKitGTK failed to repaint a
  box that was its own scroll container when its height changed sharply between
  branches (the export dialog's form vs its completion screen). `.modal-scroll`
  centres the box; auto margins do not centre inside a scrolling container.
-->
<script>
    import { trapFocus } from '../utils/focusTrap.js';
    import { t } from '../i18n';

    let {
        open = false,
        onclose = () => {},
        // small | medium | large | wide | panel | auto
        size = 'medium',
        // base (1000) | raised (1100, above another modal) | top (2000, above everything)
        layer = 'base',
        closeOnOverlay = false,
        closeOnEscape = true,
        closeButton = true,
        // Accessible name when there is no `title` snippet (tables, plain messages).
        label = '',
        // ADR-0008: two compact utility dialogs keep the panel-size title.
        compactTitle = false,
        // left | center — the box's text alignment, inherited by title and footer.
        align = 'left',
        // Extra keys of the modal, called for every key Escape did not consume.
        onkeydown,
        title,
        children,
        footer
    } = $props();

    const titleId = $props.id();
    let box = $state();

    function handleKeydown(event) {
        if (event.key === 'Escape') {
            if (!closeOnEscape) return;
            event.preventDefault();
            event.stopImmediatePropagation();
            onclose();
            return;
        }
        onkeydown?.(event);
    }

    function handleClick(event) {
        if (closeOnOverlay && box && !box.contains(event.target)) onclose();
    }
</script>

{#if open}
    <div
        class="modal-overlay layer-{layer}"
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        aria-label={title ? undefined : label}
        tabindex="-1"
        onkeydown={handleKeydown}
        onclick={handleClick}
        use:trapFocus
    >
        <div class="modal-scroll">
            <div class="modal-box size-{size} align-{align}" bind:this={box}>
                {#if title}
                    <h2 class="modal-title" class:compact={compactTitle} id={titleId}>{@render title()}</h2>
                {/if}
                {@render children?.()}
                {#if footer}
                    <div class="modal-footer">{@render footer()}</div>
                {/if}
                {#if closeButton}
                    <!-- Last in the DOM so the first control of the dialog takes the focus on
                         open, and the cross closes the Tab cycle rather than opening it. -->
                    <button type="button" class="modal-close" aria-label={$t('common.close')} title={$t('common.close')} onclick={onclose}>×</button>
                {/if}
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        inset: 0;
        background-color: rgba(0, 0, 0, 0.5);
        overflow-y: auto;
        z-index: 1000;
        outline: none;
    }

    .layer-raised {
        z-index: 1100;
    }

    .layer-top {
        z-index: 2000;
    }

    /* min-height rather than height: it fills the window when the box is short, so the
       box is centred, and grows past it when the box is tall, so the top stays reachable
       instead of being clipped. */
    .modal-scroll {
        min-height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 20px;
        box-sizing: border-box;
    }

    .modal-box {
        position: relative;
        display: flex;
        flex-direction: column;
        gap: 12px;
        flex: none;
        box-sizing: border-box;
        padding: 20px;
        background-color: white;
        border-radius: 6px;
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
        font-size: var(--font-size-base);
        text-align: left;
    }

    .align-center {
        text-align: center;
    }

    .size-small {
        width: min(320px, 100%);
    }

    .size-medium {
        width: min(460px, 100%);
    }

    .size-large {
        width: min(520px, 100%);
    }

    .size-wide {
        width: min(900px, 100%);
    }

    /* A fixed frame the content fills (tabbed help): no padding, the content lays itself out. */
    .size-panel {
        width: 80vw;
        height: 70vh;
        padding: 0;
        gap: 0;
    }

    .size-auto {
        width: auto;
        max-width: 100%;
    }

    /* The dialog's vocabulary — title, busy spinner, status line, note box — is styled
       here once; the progress and export dialogs write their own headings and notes in
       their templates (a title that changes with the step, a spinner beside it), hence
       :global under the box. */
    .modal-box :global(.modal-title) {
        margin: 0;
        font-size: var(--font-size-dialog-title);
        color: #333;
    }

    .modal-box :global(.modal-title.compact) {
        font-size: var(--font-size-title);
    }

    .modal-box :global(.spinner) {
        display: inline-block;
        width: 16px;
        height: 16px;
        border: 3px solid #e0e0e0;
        border-top: 3px solid #666;
        border-radius: 50%;
        animation: modal-spin 1s linear infinite;
        margin-left: 10px;
        vertical-align: middle;
    }

    @keyframes -global-modal-spin {
        to {
            transform: rotate(360deg);
        }
    }

    .modal-box :global(.status-text) {
        color: #666;
        margin: 0;
    }

    .modal-box :global(.summary) {
        background-color: #f9f9f9;
        padding: 15px;
        border-radius: 4px;
        border-left: 4px solid #666;
    }

    .modal-box :global(.summary.warning) {
        background-color: #f5f5f5;
        border-left-color: #999;
    }

    .modal-box :global(.summary p) {
        margin: 5px 0;
        color: #555;
    }

    .modal-box :global(.summary strong) {
        color: #333;
    }

    .modal-footer {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        justify-content: flex-end;
    }

    .align-center .modal-footer {
        justify-content: center;
    }

    /* One palette for the footer buttons of every dialog — they used to come in five
       (grey, dark, blue, red, light). Neutral by default, `.primary` for the action the
       dialog exists for, `.danger` for the one destructive action. The buttons belong
       to the modals' own templates, hence :global. */
    .modal-footer :global(button) {
        padding: 8px 16px;
        border: 1px solid #ccc;
        border-radius: 4px;
        background-color: white;
        color: #333;
        font-weight: 500;
        cursor: pointer;
        transition:
            background-color 0.2s ease,
            border-color 0.2s ease;
    }

    .modal-footer :global(button:hover:not(:disabled)) {
        background-color: #f5f5f5;
        border-color: #999;
    }

    .modal-footer :global(button:disabled) {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .modal-footer :global(button.primary) {
        background-color: #333;
        border-color: #333;
        color: white;
    }

    .modal-footer :global(button.primary:hover:not(:disabled)) {
        background-color: #555;
        border-color: #555;
    }

    .modal-footer :global(button.danger) {
        background-color: #b3261e;
        border-color: #b3261e;
        color: white;
    }

    .modal-footer :global(button.danger:hover:not(:disabled)) {
        background-color: #8f1e18;
        border-color: #8f1e18;
    }

    .modal-close {
        position: absolute;
        top: 2px;
        right: 8px;
        padding: 0 4px;
        border: none;
        background: none;
        font-size: var(--font-size-dialog-close);
        font-weight: bold;
        line-height: 1;
        color: #666;
        cursor: pointer;
        z-index: 10;
    }

    .modal-close:hover {
        color: #333;
    }
</style>
