<script>
    // Lancer un rollout sur la position courante ou sur la recherche courante, le suivre, l'annuler,
    // lire ceux qui sont stockés (ADR-0060). L'état vit dans rolloutStore : il survit au panneau.
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { positionStore } from '../stores/positionStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { isMoneyPosition } from '../utils/cubeDecision.js';
    import { rolloutStore, rolloutChoiceStore } from '../stores/rolloutStore.js';
    import { ensureRolloutEvents, syncRolloutStatus, loadRolloutPresets, chosenSettings, startRolloutOfCurrent, startRolloutOfSearch, cancelRollout, boardKey } from '../services/rolloutService.js';
    import { logger } from '../utils/logger.js';
    import RolloutResults from './RolloutResults.svelte';

    let presets = $state(/** @type {any} */ (null));
    let stored = $state(/** @type {any[]} */ ([]));

    let rollout = $derived($rolloutStore);
    let choice = $derived($rolloutChoiceStore);
    let positionId = $derived($positionStore?.id ?? 0);
    let settings = $derived(chosenSettings(choice, presets));
    let custom = $derived(choice.preset === 'custom');

    // Mounted mid-way, the job is still there: ask, then listen.
    onMount(() => {
        ensureRolloutEvents();
        syncRolloutStatus();
        loadRolloutPresets().then((p) => (presets = p));
    });

    // What is stored for the position on the board, again whenever a rollout ended.
    $effect(() => {
        const id = positionId;
        void rollout.revision;
        let current = true;
        if (!id) {
            stored = [];
            return;
        }
        import('../../wailsjs/go/database/Database.js')
            .then(({ LoadRollouts }) => LoadRollouts(id))
            .then((list) => {
                if (current) stored = list ?? [];
            })
            .catch((err) => {
                logger.error('Error loading rollouts:', err);
                if (current) stored = [];
            });
        return () => {
            current = false;
        };
    });

    /** @param {string} preset */
    function choose(preset) {
        if (preset === 'custom') {
            rolloutChoiceStore.set({ preset, custom: { ...(settings ?? presets?.standard) } });
        } else {
            rolloutChoiceStore.set({ preset, custom: choice.custom });
        }
    }

    /** @param {string} key @param {Event} event */
    function edit(key, event) {
        const value = /** @type {HTMLInputElement} */ (event.currentTarget).value;
        rolloutChoiceStore.set({ preset: 'custom', custom: { ...choice.custom, [key]: value === '' ? '' : Number(value) } });
    }

    async function start() {
        if (settings) await startRolloutOfCurrent(settings);
    }
    async function startBatch() {
        if (settings) await startRolloutOfSearch(settings);
    }

    // Escape drops what was typed in a settings field, leaves it and keeps the panel open (the panel
    // closes on the next one). The value is put back first: leaving a field would commit it.
    /** @param {string} key @param {KeyboardEvent} event */
    function leaveField(key, event) {
        if (event.key === 'Escape') {
            event.preventDefault();
            const input = /** @type {HTMLInputElement} */ (event.currentTarget);
            input.value = String(settings?.[key] ?? '');
            input.blur();
        }
    }

    const FIELDS = [
        ['truncation', 'rollout.truncation', 'rollout.truncationHint', 1],
        ['min_games', 'rollout.minGames', 'rollout.minGamesHint', 36],
        ['max_games', 'rollout.maxGames', 'rollout.maxGamesHint', 36],
        ['jsd_limit', 'rollout.jsdLimit', 'rollout.jsdLimitHint', 0.1],
        ['ply', 'rollout.ply', 'rollout.plyHint', 1],
        ['candidates', 'rollout.candidates', 'rollout.candidatesHint', 1],
        ['seed', 'rollout.seed', 'rollout.seedHint', 1],
        ['workers', 'rollout.workers', 'rollout.workersHint', 1]
    ];
    const PRESETS = [
        ['fast', 'rollout.fast'],
        ['standard', 'rollout.standard'],
        ['custom', 'rollout.custom']
    ];

    let busyOther = $derived(rollout.running && (rollout.kind === 'batch' || rollout.positionId !== positionId));
    let live = $derived(rollout.running && rollout.kind === 'position' && rollout.positionId === positionId ? rollout.candidates : []);
    // A rollout of a board that is not saved describes that board, in that database: it goes with them.
    let unsaved = $derived(!positionId && !rollout.running && rollout.result && rollout.resultKey === boardKey($positionStore) ? rollout.result : null);
    let isMoney = $derived(isMoneyPosition($positionStore));
    let out = $derived(rollout.outcome);
    let percent = $derived(rollout.maxGames > 0 ? Math.min(100, Math.round((100 * rollout.games) / rollout.maxGames)) : 0);
</script>

<section class="rollout-section" aria-label={$t('rollout.title')} data-testid="rollout-section">
    <h3 class="rollout-title">{$t('rollout.title')}</h3>

    <div class="presets" role="group" aria-label={$t('rollout.setting')}>
        {#each PRESETS as [name, key] (name)}
            <button type="button" class="preset" aria-pressed={choice.preset === name} disabled={rollout.running} onclick={() => choose(name)}>{$t(key)}</button>
        {/each}
    </div>

    {#if settings}
        {#if custom}
            <div class="fields">
                {#each FIELDS as [key, label, hint, step] (key)}
                    <label title={$t(hint)}>
                        <span>{$t(label)}</span>
                        <input
                            type="number"
                            min="0"
                            {step}
                            value={settings[key]}
                            disabled={rollout.running}
                            onchange={(e) => edit(key, e)}
                            onkeydown={(e) => leaveField(key, e)}
                            data-testid={'rollout-' + key}
                        />
                    </label>
                {/each}
            </div>
        {:else}
            <p class="summary">{$t('rollout.summary', { games: settings.max_games, min: settings.min_games, truncation: settings.truncation, jsd: settings.jsd_limit, ply: settings.ply })}</p>
        {/if}
    {/if}

    <div class="actions">
        {#if rollout.running}
            <button type="button" onclick={cancelRollout} data-testid="rollout-cancel">{$t('rollout.cancel')}</button>
        {:else}
            <button type="button" onclick={start} disabled={!settings} title={$t('rollout.shortcut')} data-testid="rollout-start">{$t('rollout.start')}</button>
            <button type="button" onclick={startBatch} disabled={!settings || !$databaseLoadedStore} data-testid="rollout-batch">{$t('rollout.batch')}</button>
        {/if}
    </div>

    {#if rollout.running}
        <div class="progress" role="status" data-testid="rollout-progress">
            {#if rollout.kind === 'batch'}
                <span>{$t('rollout.batchProgress', { done: rollout.done, total: rollout.total })}</span>
            {:else if busyOther}
                <span>{$t('rollout.elsewhere')}</span>
            {/if}
            <progress max="100" value={percent}></progress>
            <span class="muted">{rollout.games}/{rollout.maxGames}</span>
        </div>
    {/if}

    {#if rollout.error}
        <p class="error" role="alert">{$t('rollout.error', { message: rollout.error })}</p>
    {:else if out && !rollout.running}
        <p class="outcome" role="status" data-testid="rollout-outcome">
            {#if out.type === 'done'}
                {out.stored ? $t('rollout.doneStored') : $t('rollout.doneNotStored')}
            {:else if out.type === 'cancelled'}
                {$t('rollout.cancelled')}
            {:else if out.type === 'batch-done'}
                {$t('rollout.batchDone', { rolledOut: out.rolledOut, total: out.total, refused: out.refused, failed: out.failed })}
            {:else if out.type === 'batch-cancelled'}
                {$t('rollout.batchCancelled', { rolledOut: out.rolledOut, total: out.total })}
            {:else}
                {$t('rollout.error', { message: out.message ?? '' })}
            {/if}
        </p>
    {/if}

    <RolloutResults rollouts={stored} {live} liveGames={rollout.games} liveMaxGames={rollout.maxGames} {unsaved} {isMoney} />
</section>

<style>
    .rollout-section {
        margin-top: var(--space-4);
        padding-top: var(--space-3);
        border-top: 2px solid var(--color-border);
    }
    .rollout-title {
        margin: 0 0 var(--space-2);
        font-size: var(--font-size-base);
        font-weight: 600;
    }
    .presets,
    .actions {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-2);
        margin-bottom: var(--space-2);
    }
    button {
        padding: 2px var(--space-3);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }
    button[aria-pressed='true'] {
        border-color: var(--color-primary);
        color: var(--color-primary);
        font-weight: 600;
    }
    button:disabled {
        opacity: 0.5;
        cursor: default;
    }
    .fields {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(11em, 1fr));
        gap: var(--space-2);
        margin-bottom: var(--space-2);
    }
    .fields label {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    input {
        padding: 2px var(--space-1);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
    }
    .summary,
    .muted {
        color: var(--color-text-muted);
    }
    .summary {
        margin: 0 0 var(--space-2);
        font-size: var(--font-size-small);
    }
    .progress {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        margin-bottom: var(--space-2);
    }
    progress {
        flex: 1;
        min-width: 6em;
    }
    .error {
        color: var(--color-danger);
        margin: 0 0 var(--space-2);
    }
    .outcome {
        margin: 0 0 var(--space-2);
        color: var(--color-text-muted);
    }
</style>
