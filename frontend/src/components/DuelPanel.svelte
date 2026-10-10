<!--
  DuelPanel — l'onglet Duel (ADR-0072, ADR-0073), sur le patron de l'Entraînement : sans Duel
  ouvert, le formulaire et les Duels en suspens ; avec un Duel, les horloges d'une Cadence, la
  feuille de match (la vue de la Transcription), la pause et l'arrêt. Les gestes du jeu sont au
  plateau, et le score y est déjà ; l'empreinte du germe se lit avec l'origine du Match. Le moteur se tait : aucune
  évaluation n'est montrée tant que le Duel court (règle 9).
-->
<script>
    import { onMount, untrack } from 'svelte';
    import { get } from 'svelte/store';
    import { t } from '../i18n';
    import { duelStore, duelListStore, duelAnimatingStore, duelHoldsBoardStore } from '../stores/duelStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { statusBarModeStore } from '../stores/uiStore.js';
    import { positionStore } from '../stores/positionStore.js';
    import {
        BOT_LEVELS,
        MAX_MATCH_LENGTH,
        START,
        CRAWFORD,
        POST_CRAWFORD,
        humanSide,
        normalizeForm,
        levelLabelParts,
        cadenceChoices,
        chooseCadence,
        saveCadence,
        deleteCadence,
        awayChoices,
        formFromBoard,
        boardFromForm,
        boardHasDice,
        boardCubeDecision
    } from '../services/duel.js';
    import { loadDuelForm, saveDuelForm, duelOffer, refreshDuels, startDuel, resumeDuel, suspendDuel, confirmForfeitDuel, confirmCancelDuel } from '../services/duelService.js';
    import { enterEvalOnDisplayed, exitEvalMode } from '../services/modeMachine.js';
    import TranscriptView from './TranscriptView.svelte';
    import DuelClocks from './DuelClocks.svelte';
    import PanelHeader from './panels/PanelHeader.svelte';
    import FormGrid from './panels/FormGrid.svelte';
    import FormRow from './panels/FormRow.svelte';

    let form = $state(normalizeForm(null));
    let loaded = $state(false);
    let cadences = $state(/** @type {any[]} */ ([]));
    let cadenceName = $state('');

    let open = $derived($duelStore);
    let duel = $derived(open?.state ?? null);
    let human = $derived(humanSide(duel));
    let awaiting = $derived(duel?.awaiting ?? null);
    let mine = $derived(!!awaiting && awaiting.side === human && !$duelAnimatingStore);
    let players = $derived(duel ? [duel.header?.player1 || $t('duel.player1'), duel.header?.player2 || $t('duel.player2')] : null);
    /** @type {import('../../wailsjs/go/models').duel.LevelInfo[]} */
    let levels = $state([]);
    let hint = $derived.by(() => {
        if ($duelAnimatingStore || (awaiting && !mine)) return $t('duel.botPlaying');
        if (awaiting?.kind === 'cube') return $t('duel.hint.cube');
        if (awaiting?.kind === 'answer') return $t('duel.hint.answer');
        if (awaiting?.kind === 'move') return $t('duel.hint.move');
        return '';
    });
    let hintTitle = $derived([hint, duel?.fingerprint ? $t('duel.fingerprintTitle', { fingerprint: duel.fingerprint }) : ''].filter(Boolean).join('\n') || undefined);
    let lengthChoices = Array.from({ length: MAX_MATCH_LENGTH }, (_, i) => i + 1);

    // The board, while the launcher starts from it: Eval's scratch board.
    let board = $derived(!duel && form.start === START.BOARD && $statusBarModeStore === 'EVAL' ? $positionStore : null);
    let showDice = $derived(!!board && boardHasDice(board));
    let showCube = $derived(!!board && boardCubeDecision(board, form));
    let choices = $derived(cadenceChoices(form, cadences));
    let chosen = $derived(choices.find((c) => c.name === form.cadence) ?? null);
    let awayOptions = $derived(awayChoices(form.matchLength));

    onMount(async () => {
        refreshDuels();
        form = await loadDuelForm();
        loaded = true;
        const offer = await duelOffer();
        cadences = offer?.cadences ?? [];
        levels = offer?.levels ?? [];
    });

    // Starting from the board, the board is edited as in Eval; starting from the opening, it is not.
    $effect(() => {
        if (!loaded || duel || $duelHoldsBoardStore) return;
        const mode = $statusBarModeStore;
        const fromBoard = form.start === START.BOARD;
        untrack(() => {
            if (fromBoard && mode !== 'EVAL' && mode !== 'DUEL') enterEvalOnDisplayed();
            else if (!fromBoard && mode === 'EVAL') exitEvalMode();
        });
    });

    // The board's score, edited on the board, fills the fields.
    $effect(() => {
        if (!board) return;
        const next = formFromBoard(
            untrack(() => form),
            board
        );
        untrack(() => {
            if (next !== form) form = next;
        });
    });

    /** The fields' score, edited here, goes onto the board. */
    function scoreEdited() {
        if (!board) return;
        const next = boardFromForm(get(positionStore), form);
        if (next !== get(positionStore)) positionStore.set(next);
    }

    /** @param {number} previous the length before the change */
    function lengthEdited(previous) {
        const L = form.matchLength;
        // The start of the match follows the length; a score keeps within it.
        if (form.away[0] === previous && form.away[1] === previous) form.away = [L, L];
        else form.away = form.away.map((/** @type {number} */ a) => Math.min(a, L));
        if (L === 1) form.away = [CRAWFORD, CRAWFORD];
        scoreEdited();
    }

    /** @param {number} a */
    function awayLabel(a) {
        if (a === CRAWFORD && form.matchLength > 1) return $t('duel.awayCrawford');
        if (a === POST_CRAWFORD) return $t('duel.awayPostCrawford');
        return String(a);
    }

    /** @type {Record<string, string>} */
    const PRESET_LABELS = { standard: 'duel.cadencePreset.standard', speed: 'duel.cadencePreset.speed' };

    /** @param {any} c */
    function cadenceLabel(c) {
        const name = c.preset ? $t(PRESET_LABELS[c.name] ?? c.name) : c.name;
        return $t('duel.cadenceLabel', { name, minutes: c.minutesPerPoint, delay: c.delay });
    }

    function storeCadence() {
        form = saveCadence(form, cadenceName);
        cadenceName = '';
        saveDuelForm(form);
    }

    function dropCadence() {
        if (!chosen || chosen.preset) return;
        form = deleteCadence(form, chosen.name);
        saveDuelForm(form);
    }

    function play() {
        startDuel(form, cadences);
    }
</script>

<div class="duel-panel" data-testid="duel-panel">
    {#if duel}
        <!-- Les gestes du jeu sont au plateau ; la bande dit ce qui est attendu et porte l'arrêt. -->
        <PanelHeader title={players ? `${players[0]} – ${players[1]}` : $t('tabbedPanel.duel')}>
            <!-- Published before the first roll (ADR-0072 rule 8): the seed revealed with the Match must hash back to it; shown as a tooltip only. -->
            <!-- The strip may cut the hint short: its tooltip gives it whole, with the fingerprint. -->
            <span class="prompt" data-testid="duel-hint" title={hintTitle}>{hint}</span>
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
            <span class="band" data-testid="duel-band">
                <select bind:value={form.start} aria-label={$t('duel.start')} title={$t('duel.start')} data-testid="duel-start">
                    <option value={START.OPENING}>{$t('duel.startOpening')}</option>
                    <option value={START.BOARD}>{$t('duel.startBoard')}</option>
                </select>
                {#if showDice}
                    <select bind:value={form.reroll} aria-label={$t('duel.dice')} title={$t('duel.dice')} data-testid="duel-dice">
                        <option value={false}>{$t('duel.diceBoard')}</option>
                        <option value={true}>{$t('duel.diceReroll')}</option>
                    </select>
                {/if}
                {#if showCube}
                    <select bind:value={form.afterCube} aria-label={$t('duel.cubeTiming')} title={$t('duel.cubeTiming')} data-testid="duel-cube">
                        <option value={false}>{$t('duel.cubeBefore')}</option>
                        <option value={true}>{$t('duel.cubeAfter')}</option>
                    </select>
                {/if}
                <select
                    value={form.cadence}
                    onchange={(e) => (form = chooseCadence(form, /** @type {HTMLSelectElement} */ (e.currentTarget).value, cadences))}
                    disabled={form.money}
                    aria-label={$t('duel.cadence')}
                    title={$t('duel.cadence')}
                    data-testid="duel-cadence"
                >
                    <option value="">{$t('duel.noCadence')}</option>
                    {#each choices as c (c.name)}<option value={c.name}>{cadenceLabel(c)}</option>{/each}
                </select>
            </span>
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
                        <label class="field"><input type="radio" bind:group={form.money} value={false} onchange={scoreEdited} /> {$t('duel.match')}</label>
                        <select
                            value={form.matchLength}
                            onchange={(e) => {
                                const previous = form.matchLength;
                                form.matchLength = Number(/** @type {HTMLSelectElement} */ (e.currentTarget).value);
                                lengthEdited(previous);
                            }}
                            disabled={form.money}
                            aria-label={$t('duel.matchLength')}
                            data-testid="duel-length"
                        >
                            {#each lengthChoices as n (n)}<option value={n}>{$t('duel.lengthPoints', { n })}</option>{/each}
                        </select>
                        <label class="field"><input type="radio" bind:group={form.money} value={true} onchange={scoreEdited} /> {$t('duel.moneySession')}</label>
                        <label class="field"><input type="checkbox" bind:checked={form.jacoby} disabled={!form.money} /> {$t('duel.jacoby')}</label>
                    </FormRow>

                    <FormRow label={$t('duel.score')}>
                        {#each [0, 1] as p (p)}
                            <label class="field"
                                >{$t(p === 0 ? 'duel.awayPlayer1' : 'duel.awayPlayer2')}
                                <select bind:value={form.away[p]} onchange={scoreEdited} disabled={form.money} data-testid="duel-away-{p + 1}">
                                    {#each awayOptions as a (a)}<option value={a}>{awayLabel(a)}</option>{/each}
                                </select>
                            </label>
                        {/each}
                        <label class="field"><input type="checkbox" bind:checked={form.singleGame} data-testid="duel-single-game" /> {$t('duel.singleGame')}</label>
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

                    {#if form.cadence && !form.money}
                        <FormRow label={$t('duel.cadence')}>
                            <label class="field"
                                >{$t('duel.minutesPerPoint')}
                                <input type="number" min="0.1" max="60" step="0.1" bind:value={form.minutesPerPoint} data-testid="duel-minutes" />
                            </label>
                            <label class="field"
                                >{$t('duel.delaySeconds')}
                                <input type="number" min="0" max="600" step="1" bind:value={form.delay} data-testid="duel-delay" />
                            </label>
                            <select bind:value={form.timeOut} aria-label={$t('duel.timeOut')} title={$t('duel.timeOut')}>
                                <option value="continue">{$t('duel.timeContinue')}</option>
                                <option value="lose_match">{$t('duel.timeLose')}</option>
                            </select>
                            <input type="text" class="name" bind:value={cadenceName} placeholder={$t('duel.cadenceName')} aria-label={$t('duel.cadenceName')} data-testid="duel-cadence-name" />
                            <button type="button" onclick={storeCadence} disabled={!cadenceName.trim()} data-testid="duel-cadence-save">{$t('duel.cadenceSave')}</button>
                            {#if chosen && !chosen.preset}
                                <button type="button" class="danger" onclick={dropCadence} data-testid="duel-cadence-delete">{$t('duel.cadenceDelete')}</button>
                            {/if}
                        </FormRow>
                    {/if}

                    <FormRow label={$t('duel.playerName')} for="duel-player">
                        <input id="duel-player" type="text" bind:value={form.player} />
                    </FormRow>

                    <FormRow label="">
                        <label class="field"><input type="checkbox" bind:checked={form.record} /> {$t('duel.record')}</label>
                        <label class="field"><input type="checkbox" bind:checked={form.pipcount} data-testid="duel-pipcount" /> {$t('duel.pipcount')}</label>
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
                                <button type="button" onclick={() => resumeDuel(item.id, form)}>{$t('duel.resume')}</button>
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
        width: 4.5em;
    }

    input.name {
        width: 8em;
    }

    .band {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
        min-width: 0;
        flex-wrap: wrap;
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
