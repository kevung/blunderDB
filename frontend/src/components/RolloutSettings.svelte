<script>
    // Le réglage des rollouts (Réglages > gammonNet) : Rapide, Standard, ou Libre et ses champs.
    // Le menu des coups et la touche R le lisent ; il est gardé dans la configuration.
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { rolloutChoiceStore } from '../stores/rolloutStore.js';
    import { loadRolloutPresets, chosenSettings, setRolloutChoice, settingsProblem } from '../services/rolloutService.js';

    let presets = $state(/** @type {any} */ (null));
    let choice = $derived($rolloutChoiceStore);
    let settings = $derived(chosenSettings(choice, presets));
    let problem = $derived(choice.preset === 'custom' && settings ? settingsProblem(settings) : '');

    onMount(() => {
        loadRolloutPresets().then((p) => (presets = p));
    });

    /** @param {string} preset */
    function choose(preset) {
        if (preset === 'custom') setRolloutChoice({ preset, custom: { ...(choice.custom ?? settings ?? presets?.standard) } });
        else setRolloutChoice({ preset, custom: choice.custom });
    }

    /** @param {string} key @param {Event} event */
    function edit(key, event) {
        const value = /** @type {HTMLInputElement} */ (event.currentTarget).value;
        setRolloutChoice({
            preset: 'custom',
            custom: /** @type {import('../../wailsjs/go/models.js').rollout.Settings} */ (/** @type {unknown} */ ({ ...choice.custom, [key]: value === '' ? '' : Number(value) }))
        });
    }

    /** @param {string} key */
    function fieldValue(key) {
        return /** @type {Record<string, number> | null} */ (settings)?.[key];
    }

    /** @type {[string, string, string, number][]} */
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
</script>

<div class="rollout-settings" data-testid="rollout-settings">
    <div class="row">
        <span class="label">{$t('rollout.setting')}</span>
        <div class="presets" role="group" aria-label={$t('rollout.setting')}>
            {#each PRESETS as [name, key] (name)}
                <button type="button" class="preset" aria-pressed={choice.preset === name} onclick={() => choose(name)} data-testid={'rollout-preset-' + name}>{$t(key)}</button>
            {/each}
        </div>
    </div>
    {#if settings}
        {#if choice.preset === 'custom'}
            <div class="fields">
                {#each FIELDS as [key, label, hint, step] (key)}
                    <label title={$t(hint)}>
                        <span>{$t(label)}</span>
                        <input type="number" min="0" {step} value={fieldValue(key)} onchange={(e) => edit(key, e)} data-testid={'rollout-' + key} />
                    </label>
                {/each}
            </div>
            {#if problem}<p class="note error">{problem}</p>{/if}
        {:else}
            <p class="note">{$t('rollout.summary', { games: settings.max_games, min: settings.min_games, truncation: settings.truncation, jsd: settings.jsd_limit, ply: settings.ply })}</p>
        {/if}
    {/if}
    <p class="note">{$t('rollout.settingNote')}</p>
</div>

<style>
    .rollout-settings {
        margin-top: var(--space-3);
    }
    .row {
        display: flex;
        align-items: center;
        gap: var(--space-3);
        flex-wrap: wrap;
    }
    .presets {
        display: flex;
        gap: var(--space-2);
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
    .fields {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(11em, 1fr));
        gap: var(--space-2);
        margin-top: var(--space-2);
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
    .note {
        margin: var(--space-2) 0 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }
    .note.error {
        color: var(--color-danger);
    }
</style>
