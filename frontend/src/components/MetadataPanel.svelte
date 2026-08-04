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
        <!-- Sealed: written by the producer, read-only for everyone else. The fields above
             belong to whoever holds the file; this states where the file came from. It is
             deliberately shaped as a card rather than as more form rows — nothing in it is
             editable, and it should not read like the inputs it sits under. Nothing here is
             derived from this machine: see ADR-0007. -->
        <section class="origin" class:invalid={!issuance.watermark.signatureValid}>
            <header>
                <span class="origin-kicker">{$t('issuance.origin')}</span>
                <span class="badge" class:invalid={!issuance.watermark.signatureValid}>
                    {verdictOf(issuance.watermark)}
                </span>
            </header>
            <p class="origin-value">{issuance.watermark.origin}</p>
            <p class="origin-meta">
                <span>{issuance.watermark.issuerName}</span>
                <span class="dot">·</span>
                <span>{shortDate(issuance.watermark.issuedAt)}</span>
                <span class="dot">·</span>
                <code>{issuance.watermark.issuerFingerprint}</code>
            </p>
            {#if issuance.watermark.note}
                <p class="origin-note">{issuance.watermark.note}</p>
            {/if}
        </section>
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

    /* A card, not more form rows: the accent border and tinted ground say "read-only, and
       not written by you" without needing a lock icon or a sentence to explain it. The
       accent turns red when the signature does not verify, so a tampered mark is visible
       before any text is read. */
    .origin {
        margin-top: 2px;
        padding: 6px 10px;
        border-left: 3px solid #1a73e8;
        border-radius: 0 4px 4px 0;
        background: #f5f8fe;
        font-size: 12px;
        line-height: 1.4;
    }

    .origin.invalid {
        border-left-color: #b3261e;
        background: #fdf3f2;
    }

    .origin header {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        gap: 10px;
    }

    .origin-kicker {
        font-size: 10px;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
    }

    .badge {
        flex: none;
        font-size: 11px;
        font-weight: 600;
        color: #1a7f37;
        white-space: nowrap;
    }

    .badge.invalid {
        color: #b3261e;
    }

    /* The origin is the statement the whole card exists for, so it carries the weight. */
    .origin-value {
        margin: 1px 0 0;
        font-size: 13px;
        font-weight: 600;
        color: #202124;
        overflow-wrap: anywhere;
    }

    /* Producer, date and fingerprint on one wrapping line: three short facts, none of which
       deserves a row of its own in a panel this short. */
    .origin-meta {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 0 6px;
        margin: 1px 0 0;
        color: #5f6368;
    }

    .origin-meta code {
        font-family: monospace;
        font-size: 11px;
        color: #5f6368;
    }

    .dot {
        color: #bdc1c6;
    }

    .origin-note {
        margin: 3px 0 0;
        font-style: italic;
        color: #3c4043;
        overflow-wrap: anywhere;
    }
</style>
