<script>
    import { logger } from '../utils/logger.js';
    import { onDestroy } from 'svelte';
    import { LoadMetadata, SaveMetadata, GetIssuanceInfo } from '../../wailsjs/go/database/Database.js';
    import { activeTabStore } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { t } from '../i18n';
    import PanelHeader from './panels/PanelHeader.svelte';
    import FormGrid from './panels/FormGrid.svelte';
    import FormRow from './panels/FormRow.svelte';

    let user = $state('');
    let description = $state('');
    let dateOfCreation = $state('');
    let databaseVersion = $state('');
    let loaded = $state(false);

    // The origin block appears only when this database was watermarked by its producer. An
    // ordinary database shows the panel exactly as it always was. See ADR-0007.
    let issuance = $state(null);

    async function loadMetadata() {
        try {
            const metadata = await LoadMetadata();
            user = metadata.user || '';
            description = metadata.description || '';
            dateOfCreation = metadata.dateOfCreation || '';
            databaseVersion = metadata.database_version || '';
            loaded = true;
        } catch (error) {
            logger.error('Error loading metadata:', error);
        }
        try {
            issuance = await GetIssuanceInfo();
        } catch (error) {
            logger.error('Error loading issuance info:', error);
            issuance = null;
        }
    }

    function verdictOf(watermark) {
        if (!watermark.signatureValid) return $t('issuance.signatureInvalid');
        return watermark.issuedByYou ? $t('issuance.signatureYours') : $t('issuance.signatureValid');
    }

    function shortDate(stamp) {
        return stamp && stamp.length >= 10 ? stamp.slice(0, 10) : (stamp ?? '');
    }

    async function saveMetadata() {
        if (!loaded) return;
        try {
            await SaveMetadata({ user, description, dateOfCreation });
        } catch (error) {
            logger.error('Error saving metadata:', error);
        }
    }

    // Load when tab becomes active, save when leaving
    let wasActive = false;
    $effect(() => {
        const value = $activeTabStore;
        if (value === 'metadata' && $databaseLoadedStore) {
            loadMetadata().then(() => {
                wasActive = true;
            });
        } else if (wasActive) {
            saveMetadata();
            wasActive = false;
        }
    });

    onDestroy(() => {
        if (wasActive) saveMetadata();
    });
</script>

<div class="metadata-panel">
    <PanelHeader title={$t('metadata.title')} />
    <!-- One grid for the whole panel (ADR-0085, G6): label at the start, control after it, so
         every row lines up. A rule closes the watermark block; it is read-only and written by
         whoever produced the file, everything under the rule belongs to whoever holds it
         (ADR-0007). Each field saves when it loses focus: no button, the mechanism is enough. -->
    <div class="fields">
        <FormGrid>
            {#if issuance?.watermark}
                <FormRow label={$t('issuance.originLabel')}><span class="value strong">{issuance.watermark.origin}</span></FormRow>

                {#if issuance.watermark.note}
                    <FormRow label={$t('issuance.note')}><span class="value note">{issuance.watermark.note}</span></FormRow>
                {/if}

                <FormRow label={$t('issuance.signature')}>
                    <span class="value">
                        {issuance.watermark.issuerName}
                        <span class="sep">·</span>
                        <code>{issuance.watermark.issuerFingerprint}</code>
                        <span class="mark" class:invalid={!issuance.watermark.signatureValid}>
                            {verdictOf(issuance.watermark)}
                        </span>
                    </span>
                </FormRow>

                <FormRow label={$t('issuance.markedOn')}><span class="value">{shortDate(issuance.watermark.issuedAt)}</span></FormRow>

                <hr />
            {/if}

            <FormRow label={$t('metadata.user')} for="meta-user">
                <input id="meta-user" class="short" type="text" bind:value={user} onblur={saveMetadata} />
            </FormRow>

            <FormRow label={$t('metadata.created')} for="meta-date">
                <input id="meta-date" class="short" type="date" bind:value={dateOfCreation} onchange={saveMetadata} />
            </FormRow>

            <FormRow label={$t('metadata.version')} for="meta-version">
                <input id="meta-version" class="short" type="text" bind:value={databaseVersion} readonly />
            </FormRow>

            <FormRow label={$t('metadata.description')} for="meta-description">
                <textarea id="meta-description" bind:value={description} onblur={saveMetadata} rows="2"></textarea>
            </FormRow>
        </FormGrid>
    </div>
</div>

<style>
    /* One type scale for the whole panel. Form controls do not inherit the page font on
       their own — left alone they render in the browser's own control font, larger and in a
       different family than everything around them. */
    .metadata-panel {
        font-size: var(--font-size-base);
        height: 100%;
        display: flex;
        flex-direction: column;
        background: var(--color-surface);
        box-sizing: border-box;
        text-align: start;
    }

    .fields {
        overflow-y: auto;
        min-height: 0;
    }

    input,
    textarea {
        min-width: 0;
    }

    /* A name, a date or a version reads in two dozen characters; stretched across the dock it
       sends the eye far from its label. The description is prose and keeps the full width. */
    input.short {
        width: 24ch;
        max-width: 100%;
    }

    textarea {
        flex: 1;
        resize: vertical;
    }

    input:read-only {
        background: var(--color-surface-alt);
        color: var(--color-text-muted);
    }

    input:focus,
    textarea:focus {
        outline: none;
        border-color: var(--color-primary);
    }

    hr {
        grid-column: 1 / -1;
        width: 100%;
        margin: 3px 0;
        border: none;
        border-top: 1px solid var(--color-border);
    }

    .value {
        color: var(--color-text);
        overflow-wrap: anywhere;
    }

    .value.strong {
        font-weight: 600;
        color: var(--color-text);
    }

    .value.note {
        font-style: italic;
    }

    /* Monospace looks a size larger than a proportional face at the same nominal size, so
       it takes the small token to sit level with the base-size text beside it (ADR-0008
       rule 4). */
    code {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .sep {
        color: var(--color-text-muted);
    }

    /* The only colour in the panel: a watermark that does not verify must catch the eye. */
    .mark {
        font-weight: 600;
        color: color-mix(in srgb, #1a7f37 70%, var(--color-text));
    }

    .mark.invalid {
        color: var(--color-danger);
    }
</style>
