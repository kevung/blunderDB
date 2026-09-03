<script>
    // A text field with a dropdown of matching entities. MatchPanel's inline
    // tournament picker and TournamentPanel's "add a match" field were the same
    // widget written twice: a filtered list under (or above) the input, picked
    // with the mouse, dismissed on blur, closed on Escape. Neither could be
    // driven from the keyboard; this one can (arrows, Enter).
    //
    // The dropdown is position:fixed and measured from the input's rect, so it
    // escapes the overflow:auto table containers the panels scroll in; it flips
    // above the input when the space below is too short (`placement="auto"`) or
    // always (`placement="above"`, for a field at the bottom of a pane).

    import { tick } from 'svelte';

    let {
        /** The typed text (bindable). */
        value = $bindable(''),
        /** Entities to choose from. */
        items = [],
        /** Text shown for an item; also what `value` becomes on selection. */
        label = (item) => item?.name ?? String(item),
        /** Keeps the list keyed; defaults to the label. */
        key = (item) => label(item),
        /** Which items match the typed text. Empty text matches everything. */
        filter = (item, query) => label(item).toLowerCase().includes(query.toLowerCase()),
        /** An item was picked (mouse or Enter on a highlighted row). */
        onSelect = undefined,
        /** Enter with nothing highlighted: the typed text itself is the answer. */
        onSubmit = undefined,
        /** Escape. The input is blurred first. */
        onCancel = undefined,
        /** Focus left the field (after `blurDelay`, so a click on an option lands first). */
        onDismiss = undefined,
        /** Focus entered the field — a chance to (re)load `items`. */
        onFocus = undefined,
        /** Picking an item writes its label into the field and closes the list
         *  (an inline cell editor). `false` keeps the field and the list as they
         *  are, for adding several entities in a row. */
        fillOnSelect = true,
        placement = 'auto',
        maxHeight = 120,
        blurDelay = 150,
        autofocus = false,
        placeholder = '',
        /** 'inline' = the blue-bordered look of an edited cell; 'field' = a plain form field. */
        variant = 'inline',
        /** Custom rendering of one option; receives the item. */
        item = undefined
    } = $props();

    let inputEl = $state(null);
    let open = $state(false);
    let active = $state(-1);
    let dropdownStyle = $state('');

    const filtered = $derived(items.filter((entry) => filter(entry, value)));
    const showList = $derived(open && filtered.length > 0);

    $effect(() => {
        // Keep the highlight inside the list as the filter narrows it.
        if (active >= filtered.length) active = filtered.length - 1;
    });

    $effect(() => {
        if (autofocus && inputEl) inputEl.focus();
    });

    function computePosition() {
        if (!inputEl) return;
        const rect = inputEl.getBoundingClientRect();
        const spaceBelow = window.innerHeight - rect.bottom;
        const spaceAbove = rect.top;
        const above = placement === 'above' || (spaceBelow < maxHeight && spaceAbove > spaceBelow);
        const common = `position:fixed; left:${rect.left}px; width:${rect.width}px;`;
        dropdownStyle = above
            ? `${common} bottom:${window.innerHeight - rect.top}px; max-height:${Math.min(maxHeight, spaceAbove)}px;`
            : `${common} top:${rect.bottom}px; max-height:${Math.min(maxHeight, spaceBelow)}px;`;
    }

    function show() {
        computePosition();
        open = true;
    }

    function pick(entry) {
        onSelect?.(entry);
        if (fillOnSelect) {
            value = label(entry);
            open = false;
            active = -1;
        }
    }

    function handleFocus() {
        onFocus?.();
        show();
    }

    function handleInput() {
        active = -1;
        show();
    }

    function handleBlur() {
        setTimeout(() => {
            open = false;
            active = -1;
            onDismiss?.();
        }, blurDelay);
    }

    async function moveActive(delta) {
        if (!open) show();
        await tick();
        if (filtered.length === 0) return;
        active = (active + delta + filtered.length) % filtered.length;
    }

    function handleKeyDown(event) {
        switch (event.key) {
            case 'ArrowDown':
            case 'ArrowUp':
                event.stopPropagation();
                event.preventDefault();
                moveActive(event.key === 'ArrowDown' ? 1 : -1);
                break;
            case 'Enter':
                event.stopPropagation();
                event.preventDefault();
                if (showList && active >= 0) pick(filtered[active]);
                else onSubmit?.(value);
                break;
            case 'Escape':
                event.stopPropagation();
                event.preventDefault();
                open = false;
                active = -1;
                inputEl?.blur();
                onCancel?.();
                break;
            default:
                break;
        }
    }
</script>

<div class="entity-autocomplete">
    <input
        bind:this={inputEl}
        type="text"
        class="input {variant}"
        bind:value
        {placeholder}
        role="combobox"
        aria-expanded={showList}
        aria-autocomplete="list"
        onfocus={handleFocus}
        oninput={handleInput}
        onblur={handleBlur}
        onkeydown={handleKeyDown}
    />
    {#if showList}
        <div class="dropdown" style={dropdownStyle} role="listbox">
            {#each filtered as entry, i (key(entry))}
                <!-- mousedown, not click: keeps the focus (and so the list) on the input -->
                <div
                    class="option"
                    class:active={i === active}
                    role="option"
                    aria-selected={i === active}
                    tabindex="-1"
                    onmousedown={(e) => {
                        e.preventDefault();
                        pick(entry);
                    }}
                >
                    {#if item}{@render item(entry)}{:else}{label(entry)}{/if}
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
    .entity-autocomplete {
        position: relative;
        width: 100%;
    }

    .input {
        width: 100%;
        font-size: var(--font-size-small);
        box-sizing: border-box;
        outline: none;
    }

    .input.inline {
        padding: 2px 4px;
        border: 1px solid var(--color-primary);
        border-radius: 2px;
    }

    .input.field {
        padding: 2px 6px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
    }

    .input.field:focus {
        border-color: #99c;
    }

    .dropdown {
        overflow-y: auto;
        background: white;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.18);
        z-index: 9999;
    }

    .option {
        padding: 3px 8px;
        font-size: var(--font-size-small);
        cursor: pointer;
    }

    .option:hover,
    .option.active {
        background: #e3f2fd;
    }
</style>
