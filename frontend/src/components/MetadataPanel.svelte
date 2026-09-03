<script>
    import { logger } from '../utils/logger.js';
    import { onDestroy } from 'svelte';
    import { LoadMetadata, SaveMetadata, GetIssuanceInfo } from '../../wailsjs/go/database/Database.js';
    import { activeTabStore } from '../stores/uiStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { t } from '../i18n';

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
    <!-- One grid for the whole panel: label on the left, content on the right, in a single
         pair of columns so every row lines up. The watermark block used to be laid out that
         way while the fields below put their labels on top — two organisations in one
         panel. A rule closes the watermark block; it is read-only and written by whoever
         produced the file, everything under the rule belongs to whoever holds it. See
         ADR-0007. -->
    <div class="fields">
        {#if issuance?.watermark}
            <span class="label">{$t('issuance.originLabel')}</span>
            <span class="value strong">{issuance.watermark.origin}</span>

            {#if issuance.watermark.note}
                <span class="label">{$t('issuance.note')}</span>
                <span class="value note">{issuance.watermark.note}</span>
            {/if}

            <span class="label">{$t('issuance.signature')}</span>
            <span class="value">
                {issuance.watermark.issuerName}
                <span class="sep">·</span>
                <code>{issuance.watermark.issuerFingerprint}</code>
                <span class="mark" class:invalid={!issuance.watermark.signatureValid}>
                    {verdictOf(issuance.watermark)}
                </span>
            </span>

            <span class="label">{$t('issuance.markedOn')}</span>
            <span class="value">{shortDate(issuance.watermark.issuedAt)}</span>

            <hr />
        {/if}

        <label class="label" for="meta-user">{$t('metadata.user')}</label>
        <input id="meta-user" type="text" bind:value={user} onblur={saveMetadata} />

        <label class="label" for="meta-date">{$t('metadata.created')}</label>
        <input id="meta-date" type="date" bind:value={dateOfCreation} onchange={saveMetadata} />

        <label class="label" for="meta-version">{$t('metadata.version')}</label>
        <input id="meta-version" type="text" bind:value={databaseVersion} readonly />

        <label class="label desc-label" for="meta-description">{$t('metadata.description')}</label>
        <textarea id="meta-description" bind:value={description} onblur={saveMetadata} rows="2"></textarea>
    </div>
</div>

<style>
    /* One type scale for the whole panel. Form controls do not inherit the page font on
       their own — left alone they render in the browser's own control font, larger and in a
       different family than everything around them. */
    .metadata-panel {
        font-size: var(--font-size-base);
        padding: 6px 10px;
        height: 100%;
        overflow-y: auto;
        background: white;
        box-sizing: border-box;
    }

    .fields {
        display: grid;
        grid-template-columns: max-content minmax(0, 1fr);
        gap: 3px 10px;
        align-items: baseline;
    }

    .label {
        font-size: inherit;
        font-weight: 600;
        color: var(--color-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
    }

    /* A textarea is taller than its label's line, so the label sits at the top of the row
       rather than on the baseline of an empty box. */
    .desc-label {
        align-self: start;
        padding-top: 3px;
    }

    input,
    textarea {
        min-width: 0;
    }

    textarea {
        resize: vertical;
    }

    input:read-only {
        background: #f5f5f5;
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
        border-top: 1px solid #e0e0e0;
    }

    .value {
        color: #3c4043;
        overflow-wrap: anywhere;
    }

    .value.strong {
        font-weight: 600;
        color: #202124;
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
        color: #5f6368;
    }

    .sep {
        color: #bdc1c6;
    }

    /* The only colour in the panel: a watermark that does not verify must catch the eye. */
    .mark {
        font-weight: 600;
        color: #1a7f37;
    }

    .mark.invalid {
        color: #b3261e;
    }
</style>
