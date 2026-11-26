<script>
    // The print dialog of a position list: the browsed list, a collection or
    // the Basket's positions (positionPrintRequest), always printed whole; it
    // refuses beyond MAX_SHEET_POSITIONS rather than print a part of it.
    import { t } from '../i18n';
    import Modal from './Modal.svelte';
    import { DEFAULT_POSITION_SHEET_OPTIONS, MAX_SHEET_POSITIONS, positionPrintRequest, printPositionSheet } from '../services/positionSheetService.js';

    let options = $state({ ...DEFAULT_POSITION_SHEET_OPTIONS });
    let title = $state('');
    let busy = $state(false);
    const request = $derived($positionPrintRequest);
    const tooMany = $derived(!!request && request.count > MAX_SHEET_POSITIONS);

    $effect(() => {
        if (request) title = request.title;
    });

    /** @type {{ key: keyof typeof DEFAULT_POSITION_SHEET_OPTIONS, label: string, tip: string }[]} */
    const BOXES = [
        { key: 'analysis', label: 'positionSheet.analysis', tip: 'positionSheet.analysisTip' },
        { key: 'verso', label: 'positionSheet.verso', tip: 'positionSheet.versoTip' },
        { key: 'comments', label: 'positionSheet.comments', tip: 'positionSheet.commentsTip' },
        { key: 'expanded', label: 'positionSheet.expanded', tip: 'positionSheet.expandedTip' }
    ];

    function close() {
        positionPrintRequest.set(null);
    }

    async function print() {
        if (!request) return;
        busy = true;
        try {
            await printPositionSheet(request.source, { ...$state.snapshot(options), title });
        } finally {
            busy = false;
        }
        close();
    }
</script>

<Modal open={!!request} onclose={close} size="small" label={$t('positionSheet.title')}>
    <h2 class="print-title">{$t('positionSheet.title')}</h2>
    <div class="print-options" data-testid="position-print-options">
        <p class="print-count">{$t('positionSheet.count', { n: request?.count ?? 0 })}</p>
        <label class="print-heading">
            {$t('positionSheet.heading')}
            <input type="text" bind:value={title} data-testid="position-print-title" />
        </label>
        {#each BOXES as box (box.key)}
            <label title={$t(box.tip)}>
                <input type="checkbox" bind:checked={options[box.key]} disabled={box.key === 'verso' && !options.analysis} data-option={box.key} />
                {$t(box.label)}
            </label>
        {/each}
        {#if tooMany}
            <p class="print-warning" role="alert" data-testid="position-print-too-many">{$t('positionSheet.tooMany', { n: request?.count ?? 0, max: MAX_SHEET_POSITIONS })}</p>
        {/if}
    </div>
    {#snippet footer()}
        <button type="button" onclick={close}>{$t('common.cancel')}</button>
        <button type="button" class="primary" data-testid="position-print-go" disabled={busy || tooMany || !request?.count} onclick={print}>{$t('positionSheet.print')}</button>
    {/snippet}
</Modal>

<style>
    .print-title {
        margin: 0 0 var(--space-2);
        font-size: var(--font-size-dialog-title);
    }
    .print-options {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
    }
    .print-options label {
        display: flex;
        gap: var(--space-1);
        align-items: center;
    }
    .print-heading input {
        flex: 1;
    }
    .print-count,
    .print-warning {
        margin: 0;
    }
    .print-warning {
        color: var(--color-danger, #b00020);
    }
</style>
