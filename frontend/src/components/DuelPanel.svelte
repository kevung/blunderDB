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
    import PanelHeader from './panels/PanelHeader.svelte';
    import FormGrid from './panels/FormGrid.svelte';
    import FormRow from './panels/FormRow.svelte';

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
        <!-- Les gestes du jeu sont au plateau ; la bande dit ce qui est attendu et porte l'arrêt. -->
        <PanelHeader title={players ? `${players[0]} – ${players[1]}` : $t('tabbedPanel.duel')}>
            <!-- Published before the first roll (ADR-0072 rule 8): the seed revealed with the Match must hash back to it; shown as a tooltip only. -->
            <span class="prompt" data-testid="duel-hint" title={duel.fingerprint ? $t('duel.fingerprintTitle', { fingerprint: duel.fingerprint }) : undefined}>
                {#if $duelAnimatingStore || (awaiting && !mine)}
                    {$t('duel.botPlaying')}
                {:else if awaiting?.kind === 'cube'}
                    {$t('duel.hint.cube')}
                {:else if awaiting?.kind === 'answer'}
                    {$t('duel.hint.answer')}
                {:else if awaiting?.kind === 'move'}
                    {$t('duel.hint.move')}
                {/if}
            </span>
            {#snippet actions()}
                <span class="gestures">
                    <button type="button" class="danger" onclick={confirmForfeitDuel}>{$t('duel.forfeit')}</button>
                    <button type="button" onclick={suspendDuel}>{$t('duel.pause')}</button>
                    <button type="button" class="danger" onclick={confirmCancelDuel}>{$t('duel.cancel')}</button>
                </span>
            {/snippet}
        </PanelHeader>

        <div class="body">
            {#if duel.clock}
                <div class="head">
                    <DuelClocks {duel} clocksOnly />
                </div>
            {/if}
            <TranscriptView annotated={open.sheet} {players} />
        </div>
    {:else}
        <PanelHeader title={$t('tabbedPanel.duel')}>
            {#snippet actions()}
                <button type="submit" form="duel-form" class="launch" data-testid="duel-play" disabled={!$databasePathStore}><span aria-hidden="true">▶</span> <span>{$t('duel.play')}</span></button>
            {/snippet}
        </PanelHeader>

        <div class="body launcher">
            <form
                id="duel-form"
                onsubmit={(e) => {
                    e.preventDefault();
                    play();
                }}
            >
                <FormGrid>
                    <FormRow label={$t('duel.type')}>
                        <label class="field"><input type="radio" bind:group={form.money} value={false} /> {$t('duel.match')}</label>
                        <select bind:value={form.matchLength} disabled={form.money} aria-label={$t('duel.matchLength')}>
                            {#each lengthChoices as n (n)}<option value={n}>{$t('duel.lengthPoints', { n })}</option>{/each}
                        </select>
                        <label class="field"><input type="radio" bind:group={form.money} value={true} /> {$t('duel.moneySession')}</label>
                        <label class="field"><input type="checkbox" bind:checked={form.jacoby} disabled={!form.money} /> {$t('duel.jacoby')}</label>
                    </FormRow>

                    <FormRow label={$t('duel.start')} for="duel-start">
                        <select id="duel-start" bind:value={form.start}>
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
                    </FormRow>

                    <FormRow label={$t('duel.side')}>
                        <label class="field"><input type="radio" bind:group={form.side} value={0} /> {$t('duel.player1')}</label>
                        <label class="field"><input type="radio" bind:group={form.side} value={1} /> {$t('duel.player2')}</label>
                    </FormRow>

                    <FormRow label={$t('duel.level')} for="duel-level">
                        <select id="duel-level" bind:value={form.level} title={$t('duel.levelTitle')}>
                            {#each BOT_LEVELS as level (level)}
                                {@const label = levelLabelParts(level, levels)}
                                <option value={level}>{label.key ? $t(label.key, label.params) : level}</option>
                            {/each}
                        </select>
                    </FormRow>

                    <FormRow label={$t('duel.cadence')} for="duel-cadence">
                        <select id="duel-cadence" bind:value={form.cadence}>
                            <option value="">{$t('duel.noCadence')}</option>
                            {#each cadences as c (c.name)}<option value={c.name}>{c.name}</option>{/each}
                        </select>
                    </FormRow>

                    {#if form.cadence}
                        <FormRow label={$t('duel.timeOut')} for="duel-timeout">
                            <select id="duel-timeout" bind:value={form.timeOut}>
                                <option value="continue">{$t('duel.timeContinue')}</option>
                                <option value="lose_match">{$t('duel.timeLose')}</option>
                            </select>
                        </FormRow>
                    {/if}

                    <FormRow label={$t('duel.playerName')} for="duel-player">
                        <input id="duel-player" type="text" bind:value={form.player} />
                    </FormRow>

                    <FormRow label="">
                        <label class="field"><input type="checkbox" bind:checked={form.record} /> {$t('duel.record')}</label>
                    </FormRow>
                </FormGrid>
            </form>

            <!-- Nothing suspended, no section: an empty list says nothing the form does not. -->
            {#if $duelListStore.length > 0}
                <section class="suspended" data-testid="duel-suspended">
                    <h3>{$t('duel.suspended')}</h3>
                    <ul>
                        {#each $duelListStore as item (item.id)}
                            <li>
                                <span>{item.label}</span>
                                <span class="hint">{item.updatedAt}</span>
                                <span class="spacer"></span>
                                <button type="button" onclick={() => resumeDuel(item.id)}>{$t('duel.resume')}</button>
                            </li>
                        {/each}
                    </ul>
                </section>
            {/if}
        </div>
    {/if}
</div>

<style>
    .duel-panel {
        /* Interface chrome is not text to copy; fields below opt back in. */
        user-select: none;
        -webkit-user-select: none;
        display: flex;
        flex-direction: column;
        height: 100%;
        box-sizing: border-box;
        text-align: start;
    }

    .body {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        padding: 0 var(--space-2) var(--space-2);
        flex: 1 1 auto;
        min-height: 0;
        overflow-y: auto;
    }

    /* The dock is short: the suspended Duels sit beside the form, not under its fold. */
    .body.launcher {
        flex-direction: row;
        flex-wrap: wrap;
        align-items: flex-start;
        column-gap: var(--space-3);
        overflow-y: auto;
    }

    .suspended {
        padding-top: var(--space-2);
        min-width: 18em;
    }

    .head {
        padding-top: var(--space-2);
    }

    form :global(.form-grid) {
        padding: var(--space-2) 0 0;
    }

    .gestures,
    .suspended li {
        display: flex;
        align-items: center;
        gap: var(--space-1);
    }

    .field {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
    }

    .hint,
    .prompt {
        color: var(--color-text-muted);
    }

    .prompt {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        min-width: 0;
    }

    input[type='number'] {
        width: 4em;
    }

    button {
        cursor: pointer;
        padding: 2px var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
    }

    /* The button that starts a game: filled, at the end of the strip (ADR-0085, G3). */
    button.launch {
        font-size: var(--font-size-base);
        font-weight: 600;
        padding: 2px var(--space-3, 12px);
        background: var(--color-primary);
        border-color: var(--color-primary);
        color: white;
    }

    button.danger {
        color: var(--color-danger);
    }

    button:disabled {
        cursor: default;
        opacity: 0.5;
    }

    .spacer {
        flex: 1;
    }

    h3 {
        margin: 0;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
    }

    .suspended ul {
        list-style: none;
        margin: var(--space-1) 0 0;
        padding: 0;
        max-width: 40em;
    }

    .duel-panel input {
        user-select: text;
        -webkit-user-select: text;
    }
</style>
