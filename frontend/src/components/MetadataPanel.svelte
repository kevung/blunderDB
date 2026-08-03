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

    // Issuance stays discreet: the sections below appear only when this database really is
    // an issued copy, carries material from one, or has issued copies itself. An ordinary
    // database shows the panel exactly as it always was. See ADR-0007.
    let issuance = $state(null);
    let showIssued = $state(false);
    let showLineage = $state(false);

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
        <!-- Sealed: written by the issuer, read-only for the holder. The fields above belong
             to whoever holds the file; this block belongs to whoever issued it. -->
        <section class="issuance sealed">
            <h3>{$t('issuance.issuedCopy')}</h3>
            <dl>
                <dt>{$t('issuance.distribution')}</dt>
                <dd>{issuance.watermark.distribution}</dd>
                <dt>{$t('issuance.issuedBy')}</dt>
                <dd>
                    {issuance.watermark.issuerName}
                    <span class="fingerprint">{issuance.watermark.issuerFingerprint}</span>
                    <span class="verdict" class:invalid={!issuance.watermark.signatureValid}>
                        {verdictOf(issuance.watermark)}
                    </span>
                </dd>
                <dt>{$t('issuance.issuedTo')}</dt>
                <dd>
                    {#if issuance.watermark.nominative}
                        {issuance.watermark.recipient}
                        {#if issuance.watermark.total > 0}
                            <span class="muted"
                                >{$t('issuance.copyOf', {
                                    number: issuance.watermark.number,
                                    total: issuance.watermark.total
                                })}</span
                            >
                        {/if}
                    {:else}
                        <span class="muted">{$t('issuance.collective')}</span>
                    {/if}
                </dd>
                <dt>{$t('issuance.issuedOn')}</dt>
                <dd>{shortDate(issuance.watermark.issuedAt)}</dd>
                <dt>{$t('issuance.holders')}</dt>
                <dd>
                    {#if !issuance.holders?.length}
                        <span class="muted">{$t('issuance.holdersNone')}</span>
                    {:else}
                        {$t('issuance.holdersCount', { count: issuance.holders.length })}
                        <ul class="holders">
                            {#each issuance.holders as holder (holder.fingerprint)}
                                <li>
                                    <span class="fingerprint">{holder.fingerprint}</span>
                                    {shortDate(holder.firstSeen)} → {shortDate(holder.lastSeen)}
                                    <span class="muted">{$t('issuance.openings', { count: holder.openings })}</span>
                                </li>
                            {/each}
                        </ul>
                        {#if !issuance.chainIntact}
                            <p class="warn">{$t('issuance.chainBroken')}</p>
                        {/if}
                    {/if}
                </dd>
            </dl>
        </section>
    {/if}

    {#if issuance?.lineage?.length}
        <section class="issuance">
            <h3>
                <button type="button" class="disclosure" onclick={() => (showLineage = !showLineage)}>
                    {showLineage ? '▾' : '▸'}
                    {$t('issuance.lineage', { count: issuance.lineage.length })}
                </button>
            </h3>
            {#if showLineage}
                <ul class="records">
                    {#each issuance.lineage as entry, i (i)}
                        <li>
                            <strong>{entry.distribution}</strong>
                            — {$t('issuance.issuedBy')}
                            {entry.issuerName}
                            {#if entry.nominative}
                                — {$t('issuance.issuedTo')}
                                {entry.recipient}
                            {/if}
                            <span class="muted">{shortDate(entry.issuedAt)}</span>
                            <span class="verdict" class:invalid={!entry.signatureValid}>{verdictOf(entry)}</span>
                        </li>
                    {/each}
                </ul>
            {/if}
        </section>
    {/if}

    {#if issuance?.issued?.length}
        <!-- The issue register. It never travels inside an issued copy: it lists every
             recipient of a distribution and its password. -->
        <section class="issuance">
            <h3>
                <button type="button" class="disclosure" onclick={() => (showIssued = !showIssued)}>
                    {showIssued ? '▾' : '▸'}
                    {$t('issuance.issuedCopies', { count: issuance.issued.length })}
                </button>
            </h3>
            {#if showIssued}
                <ul class="records">
                    {#each issuance.issued as record (record.signature)}
                        <li>
                            <span class="muted">{record.number}/{record.total}</span>
                            <strong>{record.recipient || $t('issuance.collectiveShort')}</strong>
                            — {record.distribution}
                            <span class="muted">{shortDate(record.issuedAt)}</span>
                            {#if record.fileName}<span class="muted">{record.fileName}</span>{/if}
                            {#if record.password}
                                <span class="password">{$t('issuance.password')}: {record.password}</span>
                            {/if}
                        </li>
                    {/each}
                </ul>
                <p class="muted note">{$t('issuance.registerStaysHere')}</p>
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

    .issuance {
        border-top: 1px solid #e0e0e0;
        padding-top: 4px;
        font-size: 12px;
    }

    .issuance h3 {
        margin: 0 0 2px;
        font-size: 12px;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
    }

    .issuance.sealed h3::before {
        content: '🔒 ';
    }

    .disclosure {
        background: none;
        border: none;
        padding: 0;
        font: inherit;
        color: inherit;
        text-transform: inherit;
        letter-spacing: inherit;
        cursor: pointer;
    }

    .issuance dl {
        display: grid;
        grid-template-columns: max-content 1fr;
        gap: 1px 10px;
        margin: 0;
    }

    .issuance dt {
        color: #888;
    }

    .issuance dd {
        margin: 0;
    }

    .fingerprint {
        font-family: monospace;
        color: #555;
    }

    .verdict {
        color: #1a7f37;
    }

    .verdict.invalid {
        color: #b3261e;
        font-weight: 600;
    }

    .muted {
        color: #888;
    }

    .warn {
        color: #b3261e;
        margin: 2px 0 0;
    }

    .password {
        font-family: monospace;
    }

    .holders,
    .records {
        list-style: none;
        margin: 2px 0 0;
        padding: 0;
    }

    .holders li,
    .records li {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;
    }

    .note {
        margin: 2px 0 0;
    }
</style>
