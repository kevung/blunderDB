<!--
  DuelPanel — l'onglet Duel (ADR-0072, ADR-0073), sur le patron de l'Entraînement : sans Duel
  ouvert, le formulaire et les Duels en suspens ; avec un Duel, le score, les horloges, la
  feuille de match (la vue de la Transcription) et les gestes. Le moteur se tait : aucune
  évaluation n'est montrée tant que le Duel court (règle 9).
-->
<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { duelStore, duelListStore, duelAnimatingStore } from '../stores/duelStore.js';
    import { quizPlayCompleteStore } from '../stores/quizPlayStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { BOT_LEVELS, MAX_MATCH_LENGTH, START, humanSide, normalizeForm } from '../services/duel.js';
    import { loadDuelForm, duelOffer, refreshDuels, startDuel, resumeDuel, suspendDuel, stopDuel, decide, validateMove, resetMove } from '../services/duelService.js';
    import { confirmAction } from '../services/confirmService.js';
    import TranscriptView from './TranscriptView.svelte';
    import DuelClocks from './DuelClocks.svelte';

    let form = $state(normalizeForm(null));
    let cadences = $state(/** @type {any[]} */ ([]));
    let resignLevel = $state(1);

    let open = $derived($duelStore);
    let duel = $derived(open?.state ?? null);
    let human = $derived(humanSide(duel));
    let awaiting = $derived(duel?.awaiting ?? null);
    let mine = $derived(!!awaiting && awaiting.side === human && !$duelAnimatingStore);
    let players = $derived(duel ? [duel.header?.player1 || $t('duel.player1'), duel.header?.player2 || $t('duel.player2')] : null);
    let lengthChoices = Array.from({ length: MAX_MATCH_LENGTH }, (_, i) => i + 1);

    onMount(async () => {
        refreshDuels();
        form = await loadDuelForm();
        cadences = (await duelOffer())?.cadences ?? [];
    });

    function play() {
        form = normalizeForm(form);
        startDuel(form, cadences);
    }

    /** @param {boolean} keep */
    async function stop(keep) {
        const go = await confirmAction(/** @type {string} */ ($t(keep ? 'duel.stopKeepConfirm' : 'duel.stopDiscardConfirm')), {
            confirmLabel: /** @type {string} */ ($t(keep ? 'duel.stopKeep' : 'duel.stopDiscard')),
            tone: keep ? 'primary' : 'danger'
        });
        if (go) stopDuel(keep);
    }

    async function resign() {
        const go = await confirmAction(/** @type {string} */ ($t('duel.resignConfirm', { n: resignLevel * (duel?.awaiting?.position?.cube?.value || 1) })), {
            confirmLabel: /** @type {string} */ ($t('duel.resign'))
        });
        if (go) decide('resign', resignLevel);
    }
</script>

<div class="duel-panel" data-testid="duel-panel">
    {#if duel}
        <div class="head">
            <DuelClocks {duel} />
        </div>

        <div class="gestures" role="group" aria-label={$t('duel.gestures')}>
            {#if $duelAnimatingStore || (awaiting && !mine)}
                <span class="prompt">{$t('duel.botPlaying')}</span>
            {:else if awaiting?.kind === 'cube'}
                <span class="prompt">{$t('duel.awaitCube')}</span>
                <button type="button" class="primary" onclick={() => decide('roll')}>{$t('duel.roll')}</button>
                <button type="button" onclick={() => decide('double')}>{$t('duel.double')}</button>
            {:else if awaiting?.kind === 'answer'}
                <span class="prompt">{$t('duel.awaitAnswer')}</span>
                <button type="button" class="primary" onclick={() => decide('take')}>{$t('duel.take')}</button>
                <button type="button" onclick={() => decide('pass')}>{$t('duel.pass')}</button>
            {:else if awaiting?.kind === 'move'}
                <span class="prompt">{$t('duel.awaitMove')}</span>
                <button type="button" class="primary" disabled={!$quizPlayCompleteStore} onclick={validateMove}>{$t('duel.validate')}</button>
                <button type="button" onclick={resetMove}>{$t('duel.reset')}</button>
            {/if}
        </div>

        <TranscriptView annotated={open.sheet} {players} />

        <div class="gestures secondary">
            <label class="field">
                <select bind:value={resignLevel} disabled={!mine} aria-label={$t('duel.resignLevel')}>
                    <option value={1}>{$t('duel.resignSingle')}</option>
                    <option value={2}>{$t('duel.resignGammon')}</option>
                    <option value={3}>{$t('duel.resignBackgammon')}</option>
                </select>
            </label>
            <button type="button" disabled={!mine} onclick={resign}>{$t('duel.resign')}</button>
            <span class="spacer"></span>
            <button type="button" onclick={suspendDuel}>{$t('duel.suspend')}</button>
            <button type="button" onclick={() => stop(true)}>{$t('duel.stopKeep')}</button>
            <button type="button" class="danger" onclick={() => stop(false)}>{$t('duel.stopDiscard')}</button>
        </div>
    {:else}
        <form
            class="form"
            onsubmit={(e) => {
                e.preventDefault();
                play();
            }}
        >
            <div class="row">
                <label class="field"><input type="radio" bind:group={form.money} value={false} /> {$t('duel.match')}</label>
                <select bind:value={form.matchLength} disabled={form.money} aria-label={$t('duel.matchLength')}>
                    {#each lengthChoices as n (n)}<option value={n}>{$t('duel.lengthPoints', { n })}</option>{/each}
                </select>
                <label class="field"><input type="radio" bind:group={form.money} value={true} /> {$t('duel.moneySession')}</label>
                <label class="field"><input type="checkbox" bind:checked={form.jacoby} disabled={!form.money} /> {$t('duel.jacoby')}</label>
            </div>

            <div class="row">
                <span class="field-label">{$t('duel.start')}</span>
                <select bind:value={form.start} aria-label={$t('duel.start')}>
                    <option value={START.OPENING}>{$t('duel.startOpening')}</option>
                    <option value={START.BOARD}>{$t('duel.startBoard')}</option>
                    <option value={START.SCORE} disabled={form.money}>{$t('duel.startScore')}</option>
                </select>
                {#if form.start === START.SCORE && !form.money}
                    <label class="field"
                        >{$t('duel.awayPlayer1')}
                        <input type="number" min="1" max={form.matchLength} bind:value={form.away[0]} />
                    </label>
                    <label class="field"
                        >{$t('duel.awayPlayer2')}
                        <input type="number" min="1" max={form.matchLength} bind:value={form.away[1]} />
                    </label>
                {/if}
            </div>

            <div class="row">
                <span class="field-label">{$t('duel.side')}</span>
                <label class="field"><input type="radio" bind:group={form.side} value={0} /> {$t('duel.player1')}</label>
                <label class="field"><input type="radio" bind:group={form.side} value={1} /> {$t('duel.player2')}</label>
                <span class="field-label">{$t('duel.level')}</span>
                <select bind:value={form.level} aria-label={$t('duel.level')}>
                    {#each BOT_LEVELS as level (level)}<option value={level}>{level}</option>{/each}
                </select>
            </div>

            <div class="row">
                <span class="field-label">{$t('duel.cadence')}</span>
                <select bind:value={form.cadence} aria-label={$t('duel.cadence')}>
                    <option value="">{$t('duel.noCadence')}</option>
                    {#each cadences as c (c.name)}<option value={c.name}>{c.name}</option>{/each}
                </select>
                {#if form.cadence}
                    <span class="field-label">{$t('duel.timeOut')}</span>
                    <select bind:value={form.timeOut} aria-label={$t('duel.timeOut')}>
                        <option value="continue">{$t('duel.timeContinue')}</option>
                        <option value="lose_match">{$t('duel.timeLose')}</option>
                    </select>
                {/if}
            </div>

            <div class="row">
                <label class="field">{$t('duel.playerName')} <input type="text" bind:value={form.player} /></label>
                <label class="field"><input type="checkbox" bind:checked={form.record} /> {$t('duel.record')}</label>
                <button type="submit" class="primary" disabled={!$databasePathStore}>{$t('duel.play')}</button>
            </div>
        </form>

        <section class="suspended">
            <h3>{$t('duel.suspended')}</h3>
            {#if $duelListStore.length === 0}
                <p class="hint">{$t('duel.noneSuspended')}</p>
            {:else}
                <ul>
                    {#each $duelListStore as item (item.id)}
                        <li>
                            <span>{item.label}</span>
                            <span class="hint">{item.updatedAt}</span>
                            <button type="button" onclick={() => resumeDuel(item.id)}>{$t('duel.resume')}</button>
                        </li>
                    {/each}
                </ul>
            {/if}
        </section>
    {/if}
</div>

<style>
    .duel-panel {
        display: flex;
        flex-direction: column;
        gap: 0.6em;
        padding: 0.6em 0.8em;
    }

    .row,
    .gestures,
    .suspended li {
        display: flex;
        align-items: center;
        gap: 0.5em;
        flex-wrap: wrap;
    }

    .form {
        display: flex;
        flex-direction: column;
        gap: 0.5em;
    }

    .field {
        display: inline-flex;
        align-items: center;
        gap: 0.3em;
    }

    .field-label,
    .hint,
    .prompt {
        color: var(--color-text-muted);
    }

    input[type='number'] {
        width: 4em;
    }

    button {
        cursor: pointer;
        padding: 0.15em 0.6em;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        background: var(--color-surface);
        color: var(--color-text);
    }

    button.primary {
        border-color: var(--color-primary);
        color: var(--color-primary);
    }

    button.danger {
        color: var(--color-danger);
    }

    button:disabled {
        cursor: default;
        color: var(--color-text-muted);
        border-color: var(--color-border);
    }

    .spacer {
        flex: 1;
    }

    h3 {
        margin: 0;
        font-size: var(--font-size-base);
        font-weight: 600;
    }

    .suspended ul {
        list-style: none;
        margin: 0.3em 0 0;
        padding: 0;
    }
</style>
