<!--
  DuelPanel — l'onglet Duel (ADR-0072, ADR-0073), sur le patron de l'Entraînement : sans Duel
  ouvert, le formulaire et les Duels en suspens ; avec un Duel, les horloges d'une Cadence, la
  feuille de match (la vue de la Transcription), la pause et l'arrêt. Les gestes du jeu sont au
  plateau, et le score y est déjà ; l'empreinte du germe se lit avec l'origine du Match. Le moteur se tait : aucune
  évaluation n'est montrée tant que le Duel court (règle 9).
-->
<script>
    import { onMount } from 'svelte';
    import { t } from '../i18n';
    import { duelStore, duelListStore, duelAnimatingStore } from '../stores/duelStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { BOT_LEVELS, MAX_MATCH_LENGTH, START, humanSide, normalizeForm, levelLabelParts } from '../services/duel.js';
    import { loadDuelForm, duelOffer, refreshDuels, startDuel, resumeDuel, suspendDuel, confirmForfeitDuel, confirmCancelDuel } from '../services/duelService.js';
    import TranscriptView from './TranscriptView.svelte';
    import DuelClocks from './DuelClocks.svelte';

    let form = $state(normalizeForm(null));
    let cadences = $state(/** @type {any[]} */ ([]));

    let open = $derived($duelStore);
    let duel = $derived(open?.state ?? null);
    let human = $derived(humanSide(duel));
    let awaiting = $derived(duel?.awaiting ?? null);
    let mine = $derived(!!awaiting && awaiting.side === human && !$duelAnimatingStore);
    let players = $derived(duel ? [duel.header?.player1 || $t('duel.player1'), duel.header?.player2 || $t('duel.player2')] : null);
    /** @type {import('../../wailsjs/go/models').duel.LevelInfo[]} */
    let levels = $state([]);
    let lengthChoices = Array.from({ length: MAX_MATCH_LENGTH }, (_, i) => i + 1);

    onMount(async () => {
        refreshDuels();
        form = await loadDuelForm();
        const offer = await duelOffer();
        cadences = offer?.cadences ?? [];
        levels = offer?.levels ?? [];
    });

    function play() {
        form = normalizeForm(form);
        startDuel(form, cadences);
    }
</script>

<div class="duel-panel" data-testid="duel-panel">
    {#if duel}
        {#if duel.clock}
            <div class="head">
                <DuelClocks {duel} clocksOnly />
            </div>
        {/if}

        <!-- Les gestes du jeu sont au plateau ; ici, ce qui est attendu, en une ligne. -->
        <!-- Published before the first roll (ADR-0072 rule 8): the seed revealed with the Match must hash back to it; shown as a tooltip only. -->
        <p class="prompt" data-testid="duel-hint" title={duel.fingerprint ? $t('duel.fingerprintTitle', { fingerprint: duel.fingerprint }) : undefined}>
            {#if $duelAnimatingStore || (awaiting && !mine)}
                {$t('duel.botPlaying')}
            {:else if awaiting?.kind === 'cube'}
                {$t('duel.hint.cube')}
            {:else if awaiting?.kind === 'answer'}
                {$t('duel.hint.answer')}
            {:else if awaiting?.kind === 'move'}
                {$t('duel.hint.move')}
            {/if}
        </p>

        <TranscriptView annotated={open.sheet} {players} />

        <div class="gestures secondary">
            <span class="spacer"></span>
            <button type="button" class="danger" onclick={confirmForfeitDuel}>{$t('duel.forfeit')}</button>
            <button type="button" onclick={suspendDuel}>{$t('duel.pause')}</button>
            <button type="button" class="danger" onclick={confirmCancelDuel}>{$t('duel.cancel')}</button>
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
                <select bind:value={form.level} aria-label={$t('duel.level')} title={$t('duel.levelTitle')}>
                    {#each BOT_LEVELS as level (level)}
                        {@const label = levelLabelParts(level, levels)}
                        <option value={level}>{label.key ? $t(label.key, label.params) : level}</option>
                    {/each}
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
        /* Interface chrome is not text to copy; fields below opt back in. */
        user-select: none;
        -webkit-user-select: none;
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

    p.prompt {
        margin: 0;
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

    .duel-panel input {
        user-select: text;
        -webkit-user-select: text;
    }
</style>
