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

    {#if issuance?.watermark}
        <!-- A notice, not a form: everything above is editable and belongs to whoever holds
             the file, this single line states where the file came from and cannot be
             touched. Facts are separated by middots rather than laid out in rows — in a
             panel this short, a four-row table costs more space than the four facts are
             worth. The verdict is reduced to one mark whose tooltip carries the sentence.
             Nothing here is derived from this machine: see ADR-0007. -->
        <p class="origin">
            <span class="kicker">{$t('issuance.origin')}</span>
            <span class="value">{issuance.watermark.origin}</span>
            <span class="sep">·</span>
            <span>{issuance.watermark.issuerName}</span>
            <span class="sep">·</span>
            <span>{shortDate(issuance.watermark.issuedAt)}</span>
            <span class="sep">·</span>
            <code>{issuance.watermark.issuerFingerprint}</code>
            <span class="mark" class:invalid={!issuance.watermark.signatureValid} title={verdictOf(issuance.watermark)} aria-label={verdictOf(issuance.watermark)}>
                {issuance.watermark.signatureValid ? '✓' : '⚠'}
            </span>
            {#if issuance.watermark.note}
                <span class="sep">·</span>
                <span class="note">{issuance.watermark.note}</span>
            {/if}
        </p>
    {/if}
</div>

<style>
    .metadata-panel {
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
        font-size: 10px;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
        font-size: 12px;
        font-family: inherit;
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

    /* One wrapping line, a hairline to set it apart from the fields above, and no ground
       of its own: the point is that it takes almost no room. Only the verdict mark carries
       colour, so a tampered watermark still catches the eye. */
    .origin {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 0 5px;
        margin: 2px 0 0;
        padding-top: 4px;
        border-top: 1px solid #e0e0e0;
        font-size: 11px;
        color: #5f6368;
        line-height: 1.5;
    }

    .kicker {
        font-size: 10px;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
    }

    .origin .value {
        font-weight: 600;
        color: #202124;
        overflow-wrap: anywhere;
    }

    .origin code {
        font-family: monospace;
        font-size: 10px;
    }

    .sep {
        color: #bdc1c6;
    }

    .mark {
        font-weight: 700;
        color: #1a7f37;
        cursor: help;
    }

    .mark.invalid {
        color: #b3261e;
    }

    .note {
        font-style: italic;
        overflow-wrap: anywhere;
    }
</style>
