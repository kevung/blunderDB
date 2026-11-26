<script>
    // The print dialog of the Match panel: what the match sheet holds, then the
    // sheet itself (matchSheetService). The batch prints every match of the
    // filtered list, not only the pages read, one sheet each, in one document,
    // and refuses beyond MAX_SHEET_MATCHES rather than print a part of it.
    import { t } from '../i18n';
    import Modal from './Modal.svelte';
    import { DEFAULT_SHEET_OPTIONS, MAX_SHEET_MATCHES, printMatchSheet } from '../services/matchSheetService.js';

    let {
        open = false,
        /** The match the dialog was opened on. */
        matchId = null,
        /** How many matches the filtered list holds (CountMatches), for the batch. */
        listedTotal = 0,
        /** Every id of the filtered list, in its order, for the batch. @type {() => Promise<number[]>} */
        listAll = async () => [],
        /** @type {() => void} */
        onclose = () => {}
    } = $props();

    let options = $state({ ...DEFAULT_SHEET_OPTIONS });
    let batch = $state(false);
    let busy = $state(false);
    const tooMany = $derived(batch && listedTotal > MAX_SHEET_MATCHES);

    /** @type {{ key: keyof typeof DEFAULT_SHEET_OPTIONS, label: string, tip: string }[]} */
    const BOXES = [
        { key: 'opponentBlunders', label: 'matchSheet.opponentBlunders', tip: 'matchSheet.opponentBlundersTip' },
        { key: 'analysis', label: 'matchSheet.analysis', tip: 'matchSheet.analysisTip' },
        { key: 'comments', label: 'matchSheet.comments', tip: 'matchSheet.commentsTip' },
        { key: 'expanded', label: 'matchSheet.expanded', tip: 'matchSheet.expandedTip' },
        { key: 'reimportCode', label: 'matchSheet.reimportCode', tip: 'matchSheet.reimportCodeTip' }
    ];

    async function print() {
        busy = true;
        try {
            const ids = batch ? await listAll() : matchId == null ? [] : [matchId];
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
        {#if listedTotal > 1}
            <label title={$t('matchSheet.batchTip')}>
                <input type="checkbox" bind:checked={batch} data-option="batch" />
                {$t('matchSheet.batch', { n: listedTotal })}
            </label>
        {/if}
        {#if tooMany}
            <p class="print-warning" role="alert" data-testid="match-print-too-many">{$t('matchSheet.tooMany', { n: listedTotal, max: MAX_SHEET_MATCHES })}</p>
        {/if}
    </div>
    {#snippet footer()}
        <button type="button" onclick={() => onclose()}>{$t('common.cancel')}</button>
        <button type="button" class="primary" data-testid="match-print-go" disabled={busy || tooMany || (!batch && matchId == null)} onclick={print}>{$t('matchSheet.print')}</button>
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
    .print-warning {
        margin: 0;
        color: var(--color-danger, #b00020);
    }
</style>
