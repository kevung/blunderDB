<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { skipDuplicatesStore } from '../stores/corpusStore.js';
    import { SetSkipDuplicates, FindDuplicateMatches, ListAliases, SetAlias, RemoveAlias, SuggestAliases } from '../../wailsjs/go/database/Database.js';

    // The aliases of a kind ("player" or "event") are edited here; the merge of
    // players in the match panel records the same aliases.
    let kind = $state('player');
    let aliases = $state([]);
    let suggestions = $state(null);
    let aliasInput = $state('');
    let canonicalInput = $state('');
    let suspects = $state(null);
    let busy = $state(false);
    let error = $state('');

    async function loadAliases() {
        try {
            aliases = (await ListAliases(kind)) || [];
        } catch (e) {
            logger.error('CorpusSettings: list aliases failed', e);
            error = String(e);
        }
    }

    onMount(loadAliases);

    async function onSkipToggle(event) {
        const skip = event.currentTarget.checked;
        try {
            await SetSkipDuplicates(skip);
            skipDuplicatesStore.set(skip);
        } catch (e) {
            logger.error('CorpusSettings: set skip duplicates failed', e);
            error = String(e);
        }
    }

    async function selectKind(next) {
        kind = next;
        suggestions = null;
        error = '';
        await loadAliases();
    }

    async function addAlias(alias, canonical) {
        error = '';
        try {
            await SetAlias(kind, alias, canonical);
            aliasInput = '';
            canonicalInput = '';
            await loadAliases();
            if (suggestions) await suggest();
        } catch (e) {
            error = String(e);
        }
    }

    async function removeAlias(alias) {
        error = '';
        try {
            await RemoveAlias(kind, alias);
            await loadAliases();
        } catch (e) {
            error = String(e);
        }
    }

    async function suggest() {
        try {
            suggestions = (await SuggestAliases(kind)) || [];
        } catch (e) {
            error = String(e);
        }
    }

    async function findDuplicates() {
        busy = true;
        error = '';
        try {
            suspects = (await FindDuplicateMatches()) || [];
        } catch (e) {
            logger.error('CorpusSettings: find duplicates failed', e);
            error = String(e);
        } finally {
            busy = false;
        }
    }
</script>

<div class="corpus-settings">
    <p class="setting-note">{$t('corpus.intro')}</p>
    <div class="setting-row">
        <label for="config-skip-duplicates">{$t('corpus.skipDuplicates')}</label>
        <input id="config-skip-duplicates" type="checkbox" checked={$skipDuplicatesStore} onchange={onSkipToggle} />
    </div>
    <p class="setting-note">{$t('corpus.skipDuplicatesNote')}</p>

    <h4>{$t('corpus.aliasesTitle')}</h4>
    <div class="kind-switch" role="group">
        <button type="button" class="secondary-button" class:active={kind === 'player'} aria-pressed={kind === 'player'} onclick={() => selectKind('player')}>{$t('corpus.players')}</button>
        <button type="button" class="secondary-button" class:active={kind === 'event'} aria-pressed={kind === 'event'} onclick={() => selectKind('event')}>{$t('corpus.events')}</button>
    </div>
    {#if aliases.length === 0}
        <p class="setting-note" data-testid="no-aliases">{$t('corpus.noAliases')}</p>
    {:else}
        <table class="alias-table">
            <thead><tr><th>{$t('corpus.aliasLabel')}</th><th>{$t('corpus.canonicalLabel')}</th><th></th></tr></thead>
            <tbody>
                {#each aliases as a (a.alias)}
                    <tr>
                        <td>{a.alias}</td>
                        <td>{a.canonical}</td>
                        <td><button type="button" class="secondary-button" onclick={() => removeAlias(a.alias)}>{$t('corpus.remove')}</button></td>
                    </tr>
                {/each}
            </tbody>
        </table>
    {/if}
    <div class="setting-row alias-form">
        <input class="setting-input" aria-label={$t('corpus.aliasLabel')} placeholder={$t('corpus.aliasLabel')} bind:value={aliasInput} />
        <input class="setting-input" aria-label={$t('corpus.canonicalLabel')} placeholder={$t('corpus.canonicalLabel')} bind:value={canonicalInput} />
        <button type="button" class="secondary-button" disabled={!aliasInput.trim() || !canonicalInput.trim()} onclick={() => addAlias(aliasInput, canonicalInput)}>{$t('corpus.add')}</button>
    </div>
    <div class="setting-row">
        <span class="setting-label">{$t('corpus.suggestNote')}</span>
        <button type="button" class="secondary-button" data-testid="suggest-aliases" onclick={suggest}>{$t('corpus.suggest')}</button>
    </div>
    {#if suggestions}
        {#if suggestions.length === 0}
            <p class="setting-note">{$t('corpus.noSuggestion')}</p>
        {:else}
            <ul class="suggestions">
                {#each suggestions as s (s.canonical)}
                    {#each s.aliases as a (a)}
                        <li>
                            <span>{$t('corpus.suggestion', { alias: a, canonical: s.canonical })}</span>
                            <button type="button" class="secondary-button" onclick={() => addAlias(a, s.canonical)}>{$t('corpus.apply')}</button>
                        </li>
                    {/each}
                {/each}
            </ul>
        {/if}
    {/if}

    <h4>{$t('corpus.duplicatesTitle')}</h4>
    <p class="setting-note">{$t('corpus.duplicatesNote')}</p>
    <div class="setting-row">
        <button type="button" class="secondary-button" data-testid="find-duplicates" disabled={busy} onclick={findDuplicates}>{$t('corpus.findDuplicates')}</button>
    </div>
    {#if suspects}
        {#if suspects.length === 0}
            <p class="setting-note" data-testid="no-duplicates">{$t('corpus.noDuplicates')}</p>
        {:else}
            <ul class="suspects" data-testid="duplicate-suspects">
                {#each suspects as d (d.matchId + '-' + d.otherId)}
                    <li>
                        {d.kind === 'longer'
                            ? $t('corpus.suspectLonger', { match: d.matchId, other: d.otherId, players: d.players })
                            : $t('corpus.suspectSameDice', { match: d.matchId, other: d.otherId, players: d.players, otherPlayers: d.otherPlayers })}
                    </li>
                {/each}
            </ul>
        {/if}
    {/if}
    {#if error}
        <p class="setting-note warn">{error}</p>
    {/if}
</div>

<style>
    .corpus-settings h4 {
        margin: 12px 0 4px;
    }
    .kind-switch {
        display: flex;
        gap: 6px;
        margin-bottom: 6px;
    }
    .kind-switch .active {
        font-weight: bold;
    }
    .alias-table {
        border-collapse: collapse;
        margin-bottom: 6px;
    }
    .alias-table th,
    .alias-table td {
        padding: 2px 8px;
        text-align: left;
    }
    .alias-form {
        gap: 6px;
    }
    .suggestions,
    .suspects {
        margin: 4px 0;
        padding-left: 18px;
    }
    .suggestions li {
        display: flex;
        gap: 8px;
        align-items: center;
    }
</style>
