<script>
    // The print dialog of the Match panel: what the match sheet holds, then the
    // sheet itself (matchSheetService). The batch prints the matches the list
    // shows, one sheet each, in one document.
    import { t } from '../i18n';
    import Modal from './Modal.svelte';
    import { DEFAULT_SHEET_OPTIONS, printMatchSheet } from '../services/matchSheetService.js';

    let {
        open = false,
        /** The match the dialog was opened on. */
        matchId = null,
        /** The ids of the matches the list shows, for the batch. @type {number[]} */
        listedIds = [],
        onclose = () => {}
    } = $props();

    let options = $state({ ...DEFAULT_SHEET_OPTIONS });
    let batch = $state(false);
    let busy = $state(false);

    const BOXES = [
        { key: 'opponentBlunders', label: 'matchSheet.opponentBlunders', tip: 'matchSheet.opponentBlundersTip' },
        { key: 'analysis', label: 'matchSheet.analysis', tip: 'matchSheet.analysisTip' },
        { key: 'comments', label: 'matchSheet.comments', tip: 'matchSheet.commentsTip' },
        { key: 'expanded', label: 'matchSheet.expanded', tip: 'matchSheet.expandedTip' },
        { key: 'reimportCode', label: 'matchSheet.reimportCode', tip: 'matchSheet.reimportCodeTip' }
    ];

    async function print() {
        const ids = batch ? listedIds : matchId == null ? [] : [matchId];
        busy = true;
        try {
            await printMatchSheet(ids, $state.snapshot(options));
        } finally {
            busy = false;
        }
        onclose();
    }
</script>

<Modal {open} {onclose} size="small" label={$t('matchSheet.title')}>
    <h2 class="print-title">{$t('matchSheet.title')}</h2>
    <div class="print-options" data-testid="match-print-options">
        {#each BOXES as box (box.key)}
            <label title={$t(box.tip)}>
                <input type="checkbox" bind:checked={options[box.key]} data-option={box.key} />
                {$t(box.label)}
            </label>
        {/each}
        {#if listedIds.length > 1}
            <label title={$t('matchSheet.batchTip')}>
                <input type="checkbox" bind:checked={batch} data-option="batch" />
                {$t('matchSheet.batch', { n: listedIds.length })}
            </label>
        {/if}
    </div>
    {#snippet footer()}
        <button type="button" onclick={onclose}>{$t('common.cancel')}</button>
        <button type="button" class="primary" data-testid="match-print-go" disabled={busy || (!batch && matchId == null)} onclick={print}>{$t('matchSheet.print')}</button>
    {/snippet}
</Modal>

<style>
    .print-title {
        margin: 0 0 var(--space-2);
        font-size: 1.1em;
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
</style>
