<script>
    // L'assistant d'import de document : lit un PDF, des images ou un Relevé,
    // fait confirmer une fois les défauts du document et le sens de jeu,
    // montre les entrées à vérifier avec leur note, puis importe — au même
    // niveau que `document import --accept-complete --fallback`.
    import { t } from '../i18n';
    import Modal from './Modal.svelte';
    import { documentImportStore } from '../stores/documentImportStore.js';
    import { refusedRanks, pendingConfirmations, confirmDocument, setChoice, runDocumentImport, closeDocumentImport } from '../services/documentImportService.js';

    let { visible = false } = $props();

    const s = $derived($documentImportStore);
    const summary = $derived(s.summary);
    const pending = $derived(pendingConfirmations(summary));
    const refused = $derived(refusedRanks(summary, s));
    const defaults = $derived(summary?.defaults ?? null);
    const busy = $derived(s.phase === 'reading' || s.phase === 'importing');
    const canImport = $derived(s.phase === 'review' && refused.length === 0 && (summary?.entries ?? 0) > 0);
    const fallbackRanks = $derived((s.acceptComplete ? summary?.fallbackComplete : summary?.fallback) ?? []);
</script>

<Modal open={visible} onclose={closeDocumentImport} size="wide" label={$t('documentImport.title')}>
    <h2 class="doc-title">{$t('documentImport.title')}</h2>
    {#if s.phase === 'reading'}
        <p data-testid="document-import-reading">{$t('documentImport.reading')}</p>
    {/if}
    {#if s.error}
        <p class="doc-error" role="alert" data-testid="document-import-error">{s.error}</p>
    {/if}
    {#if summary && (s.phase === 'review' || s.phase === 'importing')}
        <p class="doc-counts" data-testid="document-import-summary">
            {$t('documentImport.summary', { title: summary.title, entries: summary.entries, sure: summary.sure, toVerify: summary.toVerify, uncertain: summary.uncertain })}
        </p>
        {#if defaults}
            <section class="doc-block" data-testid="document-import-defaults">
                <h3>{$t('documentImport.defaults')}</h3>
                <p>{defaults.description} ({defaults.context})</p>
                {#if defaults.confirmed}
                    <p>{$t('documentImport.defaultsConfirmed')}</p>
                {:else}
                    <p>{$t('documentImport.defaultsAsk')}</p>
                {/if}
                {#if defaults.direction}
                    <p data-testid="document-import-direction">
                        {$t('documentImport.direction', { description: defaults.direction.description })}
                        {defaults.direction.confirmed ? $t('documentImport.directionConfirmed') : $t('documentImport.directionAsk')}
                    </p>
                {/if}
                {#if pending.defaults || pending.direction}
                    <button type="button" data-testid="document-import-confirm" disabled={busy} onclick={() => confirmDocument(pending)}>{$t('documentImport.confirm')}</button>
                {/if}
            </section>
        {/if}
        <fieldset class="doc-choices">
            <label>
                <input type="checkbox" data-testid="document-import-complete" checked={s.acceptComplete} onchange={(e) => setChoice('acceptComplete', e.currentTarget.checked)} />
                {$t('documentImport.acceptComplete', { n: summary.complete })}
            </label>
            <label>
                <input type="checkbox" data-testid="document-import-fallback" checked={s.fallback} disabled={!defaults?.confirmed} onchange={(e) => setChoice('fallback', e.currentTarget.checked)} />
                {$t('documentImport.fallback', { n: fallbackRanks.length })}
            </label>
            <label class="doc-collection">
                {$t('documentImport.collection')}
                <input type="text" value={s.collection} placeholder={summary.title} oninput={(e) => setChoice('collection', e.currentTarget.value)} />
            </label>
        </fieldset>
        <section class="doc-block">
            <h3>{$t('documentImport.review', { n: s.review.length })}</h3>
            {#if s.review.length === 0}
                <p>{$t('documentImport.reviewNone')}</p>
            {:else}
                <ul class="doc-review" data-testid="document-import-review">
                    {#each s.review as item (item.rank)}
                        <li data-rank={item.rank} class:held={refused.includes(item.rank)}>
                            <strong>{item.reference}</strong>
                            <span
                                >{item.state === 'sure'
                                    ? $t('documentImport.state.sure')
                                    : item.state === 'uncertain'
                                      ? $t('documentImport.state.uncertain')
                                      : $t('documentImport.state.to-verify')}</span
                            >
                            {#if item.note}<em>{item.note}</em>{/if}
                        </li>
                    {/each}
                </ul>
            {/if}
            {#if refused.length > 0}
                <p class="doc-refused" data-testid="document-import-refused">{$t('documentImport.refused', { ranks: refused.join(', ') })}</p>
            {/if}
        </section>
    {/if}
    {#if s.phase === 'done' && s.result}
        <p data-testid="document-import-done">
            {$t('documentImport.imported', { collection: s.result.collection, added: s.result.added, repeated: s.result.repeated })}
        </p>
        <p>
            {$t('documentImport.importedDetail', { entries: s.result.entries, positions: s.result.positions, fresh: s.result.newPositions, complete: s.result.complete, fallback: s.result.fallback })}
        </p>
    {/if}
    {#snippet footer()}
        {#if s.phase === 'done'}
            <button type="button" class="primary" data-testid="document-import-close" onclick={closeDocumentImport}>{$t('common.close')}</button>
        {:else}
            <button type="button" onclick={closeDocumentImport}>{$t('common.cancel')}</button>
            <button type="button" class="primary" data-testid="document-import-run" disabled={!canImport} onclick={runDocumentImport}>{$t('documentImport.run')}</button>
        {/if}
    {/snippet}
</Modal>

<style>
    .doc-title {
        margin: 0 0 var(--space-2);
        font-size: var(--font-size-dialog-title);
    }
    .doc-block h3 {
        margin: var(--space-2) 0 var(--space-1);
        font-size: var(--font-size-base);
    }
    .doc-choices {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        border: none;
        padding: 0;
        margin: var(--space-2) 0;
    }
    .doc-review {
        max-height: 14em;
        overflow-y: auto;
        margin: 0;
        padding-left: var(--space-3);
    }
    .doc-review li.held {
        font-weight: 600;
    }
    .doc-review em {
        display: block;
        font-style: normal;
        opacity: 0.8;
    }
    .doc-error,
    .doc-refused {
        color: var(--color-danger, inherit);
    }
</style>
