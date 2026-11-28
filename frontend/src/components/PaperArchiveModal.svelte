<script>
    // The paper archive of the Match panel (ADR-0119): export a match, every
    // match of the filtered list or the whole library as pages of QR codes,
    // or import pages back — the exported file, or scans and photographs.
    import { t } from '../i18n';
    import Modal from './Modal.svelte';
    import { MAX_SELECTION } from '../utils/multiSelect.js';
    import { exportPaperArchive, importPaperArchive } from '../services/paperArchiveService.js';

    let {
        open = false,
        /** The matches the dialog was opened on: one, or a selection. @type {number[]} */
        matchIds = [],
        /** How many matches the filtered list holds. */
        listedTotal = 0,
        /** Every id of the filtered list, in its order. @type {() => Promise<number[]>} */
        listAll = async () => [],
        /** Called after an import that brought matches in. */
        onimported = () => {},
        /** @type {() => void} */
        onclose = () => {}
    } = $props();

    /** @type {'match' | 'list' | 'library'} */
    let scope = $state('match');
    const tooMany = $derived(scope === 'match' && matchIds.length > MAX_SELECTION);
    // A selection of several matches is the natural scope; a single match otherwise.
    $effect(() => {
        if (open) scope = 'match';
    });
    let busy = $state(false);

    async function doExport() {
        busy = true;
        try {
            const ids = scope === 'library' ? [] : scope === 'list' ? await listAll() : matchIds;
            if (scope !== 'library' && !ids.length) return;
            await exportPaperArchive(ids, { onLaunch: () => onclose() });
        } finally {
            busy = false;
        }
        onclose();
    }

    async function doImport() {
        busy = true;
        try {
            const res = await importPaperArchive(undefined, { onLaunch: () => onclose() });
            if (res?.imported?.length) onimported();
        } finally {
            busy = false;
        }
        onclose();
    }
</script>

<Modal {open} {onclose} size="small" label={$t('paperArchive.title')}>
    <h2 class="archive-title">{$t('paperArchive.title')}</h2>
    <p class="archive-note">{$t('paperArchive.note')}</p>
    <fieldset class="archive-scope" data-testid="paper-archive-scope">
        <legend>{$t('paperArchive.exportWhat')}</legend>
        {#if matchIds.length > 0}
            <label
                ><input type="radio" bind:group={scope} value="match" data-scope="match" />
                {matchIds.length > 1 ? $t('paperArchive.scopeSelection', { n: matchIds.length }) : $t('paperArchive.scopeMatch')}</label
            >
        {/if}
        {#if listedTotal > 1}
            <label><input type="radio" bind:group={scope} value="list" data-scope="list" /> {$t('paperArchive.scopeList', { n: listedTotal })}</label>
        {/if}
        <label><input type="radio" bind:group={scope} value="library" data-scope="library" /> {$t('paperArchive.scopeLibrary')}</label>
    </fieldset>
    {#if tooMany}
        <p class="archive-warning" role="alert" data-testid="paper-archive-too-many">{$t('matchSheet.tooManySelected', { n: matchIds.length, max: MAX_SELECTION })}</p>
    {/if}
    {#snippet footer()}
        <button type="button" onclick={() => onclose()}>{$t('common.cancel')}</button>
        <button type="button" data-testid="paper-archive-import" disabled={busy} onclick={doImport}>{$t('paperArchive.import')}</button>
        <button type="button" class="primary" data-testid="paper-archive-export" disabled={busy || tooMany || (scope === 'match' && matchIds.length === 0)} onclick={doExport}
            >{$t('paperArchive.export')}</button
        >
    {/snippet}
</Modal>

<style>
    .archive-title {
        margin: 0 0 var(--space-2);
        font-size: var(--font-size-dialog-title);
    }
    .archive-note {
        margin: 0 0 var(--space-2);
    }
    .archive-warning {
        margin: var(--space-2) 0 0;
        color: var(--color-danger);
    }
    .archive-scope {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        border: none;
        padding: 0;
        margin: 0;
    }
</style>
