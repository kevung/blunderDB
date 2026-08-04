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
    {#if issuance?.watermark}
        <!-- First in the panel, and in the same type as the fields below: this is what the
             user came to check, so it should not read as a footnote. It stays read-only —
             everything below belongs to whoever holds the file, this states where the file
             came from and cannot be touched. Nothing here is derived from this machine: see
             ADR-0007. -->
        <dl class="origin">
            <dt>{$t('issuance.originLabel')}</dt>
            <dd class="value">{issuance.watermark.origin}</dd>

            {#if issuance.watermark.note}
                <dt>{$t('issuance.note')}</dt>
                <dd class="note">{issuance.watermark.note}</dd>
            {/if}

            <dt>{$t('issuance.signature')}</dt>
            <dd>
                {issuance.watermark.issuerName}
                <span class="sep">·</span>
                <code>{issuance.watermark.issuerFingerprint}</code>
                <span class="mark" class:invalid={!issuance.watermark.signatureValid}>
                    {verdictOf(issuance.watermark)}
                </span>
            </dd>

            <dt>{$t('issuance.markedOn')}</dt>
            <dd>{shortDate(issuance.watermark.issuedAt)}</dd>
        </dl>
    {/if}

    <div class="meta-row">
        <div class="form-group">
            <label for="meta-user">{$t('metadata.user')}</label>
            <input id="meta-user" type="text" bind:value={user} onblur={saveMetadata} />
        </div>
        <div class="form-group">
            <label for="meta-date">{$t('metadata.created')}</label>
            <input id="meta-date" type="date" bind:value={dateOfCreation} onchange={saveMetadata} />
        </div>
        <div class="form-group">
            <label for="meta-version">{$t('metadata.version')}</label>
            <input id="meta-version" type="text" bind:value={databaseVersion} readonly />
        </div>
    </div>
    <div class="form-group desc-group">
        <label for="meta-description">{$t('metadata.description')}</label>
        <textarea id="meta-description" bind:value={description} onblur={saveMetadata} rows="2"></textarea>
    </div>
</div>

<style>
    /* One type scale for the whole panel. Form controls do not inherit the page font on
       their own — left alone they render in the browser's own control font, larger and in a
       different family than everything around them, which is what made the read-only block
       and the editable fields look like two different panels. */
    .metadata-panel {
        font-size: 12px;
        padding: 6px 10px;
        display: flex;
        flex-direction: column;
        gap: 4px;
        height: 100%;
        overflow-y: auto;
        background: white;
        box-sizing: border-box;
    }

    .meta-row {
        display: flex;
        gap: 12px;
        align-items: flex-end;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 1px;
    }

    .desc-group {
        flex: 1;
        min-height: 0;
    }

    label {
        font-size: inherit;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
    }

    input,
    textarea {
        font: inherit;
    }

    textarea {
        flex: 1;
        min-height: 30px;
        resize: vertical;
    }

    input:read-only {
        background: #f5f5f5;
        color: #888;
    }

    input:focus,
    textarea:focus {
        outline: none;
        border-color: #1a73e8;
    }

    /* Same type as the fields below — label in the panel's own idiom, value at the panel's
       own size — so the block reads as part of the panel rather than as small print. It
       sits first and is separated by a hairline; that is all the prominence it needs. */
    .origin {
        display: grid;
        grid-template-columns: max-content 1fr;
        gap: 1px 10px;
        margin: 0 0 4px;
        padding-bottom: 5px;
        border-bottom: 1px solid #e0e0e0;
    }

    .origin dt {
        font-size: inherit;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
    }

    .origin dd {
        margin: 0;
        color: #3c4043;
        overflow-wrap: anywhere;
    }

    .origin .value {
        font-weight: 600;
        color: #202124;
    }

    .origin .note {
        font-style: italic;
    }

    /* Monospace looks a size larger than a proportional face at the same nominal size, so
       it is nudged down to sit level with the text beside it. */
    .origin code {
        font-family: monospace;
        font-size: 0.92em;
        color: #5f6368;
    }

    .sep {
        color: #bdc1c6;
    }

    /* The only colour in the block: a watermark that does not verify must catch the eye
       without the rest of the panel shouting. */
    .mark {
        font-weight: 600;
        color: #1a7f37;
    }

    .mark.invalid {
        color: #b3261e;
    }
</style>
