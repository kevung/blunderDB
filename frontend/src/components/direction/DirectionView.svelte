<script>
    /*
     * La vue tournoi : ce que la zone principale affiche à la place du plateau quand une
     * Direction est ouverte (ADR-0047, tasks/nicomaque/ux.md §2).
     *
     * C'est la première fois de l'application que la zone principale montre autre chose que le
     * plateau, et cela n'arrive QUE là : tout autre onglet ramène le plateau sans rien fermer.
     *
     * Ce lot pose le squelette et l'écran des réglages. La page « Direction » elle-même — file
     * des propositions, grille des tables, fiche de résultat — arrive avec les issues qui la
     * mesurent (#370, #371), et les autres vues avec les leurs. Les onglets absents ne sont pas
     * un oubli : ils sont vides tant que leur issue n'a pas livré, et le disent.
     */
    import { t } from '../../i18n';
    import { statusBarTextStore, activeTabStore } from '../../stores/uiStore';
    import { tMsg } from '../../i18n';
    import { logger } from '../../utils/logger.js';
    import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
    import DirectionSettings from './DirectionSettings.svelte';
    import ProposalList from './ProposalList.svelte';
    import TableGrid from './TableGrid.svelte';
    import LastDecision from './LastDecision.svelte';
    import PlayersView from './PlayersView.svelte';
    import BracketsView from './BracketsView.svelte';
    import StandingsView from './StandingsView.svelte';
    import HistoryView from './HistoryView.svelte';
    import ClockBar from './ClockBar.svelte';
    import CreditModal from './CreditModal.svelte';
    import SlotsView from './SlotsView.svelte';
    import { renderWarning } from './labels.js';
    import {
        directionStore,
        openDirectionIdStore,
        saveDirectionConfig,
        deleteDirection,
        closeDirection,
        confirmProposal,
        confirmAllProposals,
        startMatchManually,
        freeParticipants,
        tableGrid,
        enterResult,
        enterForfeit,
        moveMatchToTable,
        cancelMatch,
        lastDecision,
        correctResult,
        participants,
        entrySuggestions,
        addParticipant,
        updateParticipant,
        withdrawParticipant,
        brackets,
        standings,
        standingsCSV,
        finishTournament,
        reopenTournament,
        history,
        addNote,
        clock,
        slots,
        unattachedMatches,
        attachMatchToSlot,
        detachMatchFromSlot,
        transcribeFromSlot,
        previewDirectionConfig,
        writeDirectionPage,
        chooseDirectionOutputDir,
        forgetDirectionOutputDir,
        writePairingSheet,
        directionRounds
    } from '../../stores/directionStore';

    const view = $derived($directionStore);
    const state = $derived(view?.state || 'draft');

    /* En préparation la vue s'ouvre sur les Réglages, puisque c'est le seul geste possible ;
       en cours elle s'ouvrira sur la page Direction, où le directeur passe 95 % de son temps.
       Le choix ne vaut QUE pour l'ouverture : les Réglages restent accessibles en cours de
       tournoi — c'est là qu'on baisse la bascule à 22 h (#385) — et un onglet qui se dérobe
       sous le curseur serait pire que pas d'onglet du tout. */
    let tab = $state('settings');
    let tabChosen = false;
    $effect(() => {
        if (tabChosen) return;
        if (state !== 'draft') tab = 'direction';
        tabChosen = true;
    });

    let config = $state(null);
    $effect(() => {
        // La configuration éditée est une copie : tant qu'elle n'est pas enregistrée, elle ne
        // doit pas se confondre avec ce que la base contient.
        const c = view?.config;
        config = c ? JSON.parse(JSON.stringify(c)) : null;
    });

    const tabs = [
        { id: 'direction', labelKey: 'direction.tabs.direction' },
        { id: 'players', labelKey: 'direction.tabs.players' },
        { id: 'brackets', labelKey: 'direction.tabs.brackets' },
        { id: 'slots', labelKey: 'direction.tabs.slots' },
        { id: 'standings', labelKey: 'direction.tabs.standings' },
        { id: 'history', labelKey: 'direction.tabs.history' },
        { id: 'settings', labelKey: 'direction.tabs.settings' }
    ];

    async function apply(next) {
        try {
            await saveDirectionConfig($openDirectionIdStore, next);
            statusBarTextStore.set(tMsg('direction.settings.saved'));
        } catch (e) {
            logger.error('direction: saving configuration failed', e);
            statusBarTextStore.set(tMsg('direction.settings.errorSaving'));
        }
    }

    let busy = $state(false);
    let free = $state([]);
    let cells = $state([]);
    let last = $state(null);
    let rows = $state([]);
    let suggestions = $state([]);
    let phases = $state([]);
    let ranking = $state(null);
    let entries = $state([]);
    let clockView = $state(null);
    let creditOpen = $state(false);
    let slotRows = $state([]);
    let unattached = $state([]);
    let configPreview = $state(null);
    let rounds = $state(0);
    let sheetRound = $state(0);

    /* La file d'attente est DÉRIVÉE : elle se recalcule à chaque changement de la vue, jamais
       stockée. C'est la même règle que pour le classement et les arbres. */
    $effect(() => {
        // La dépendance explicite : la file se recalcule dès qu'un événement est écrit.
        void view?.eventCount;
        freeParticipants().then((p) => (free = p));
        tableGrid().then((c) => (cells = c));
        lastDecision().then((l) => (last = l));
        participants().then((r) => (rows = r));
        brackets().then((p) => (phases = p));
        standings().then((r) => (ranking = r));
        history().then((h) => (entries = h));
        clock().then((c) => (clockView = c));
        slots().then((r) => (slotRows = r));
        unattachedMatches().then((u) => (unattached = u));
        // Les verrous de configuration : quels formats de phase ne changent plus, et pourquoi.
        // Ils se lisent sur la configuration EN VIGUEUR, pas sur la copie en cours d'édition.
        const saved = view?.config;
        if (saved) previewDirectionConfig(saved).then((p) => (configPreview = p));
        // L'affichage de la salle est réécrit à chaque événement, sans aucun geste (#386).
        // L'appel ne fait rien tant qu'aucun dossier n'a été choisi, et un échec d'écriture
        // n'interrompt jamais la direction du tournoi.
        writeDirectionPage().then((path) => {
            if (path === null) statusBarTextStore.set(tMsg('direction.display.error'));
        });
        directionRounds().then((n) => (rounds = n));
    });

    /* Les Players de la base ne changent pas pendant un tournoi : une seule lecture suffit. */
    $effect(() => {
        entrySuggestions().then((s) => (suggestions = s));
    });

    /* Le temps écoulé d'un match avance sans qu'aucun événement ne soit écrit : la grille et
       l'horloge se rafraîchissent donc à la minute, sans quoi une table qui traîne resterait
       invisible. */
    $effect(() => {
        const timer = setInterval(() => {
            tableGrid().then((c) => (cells = c));
            clock().then((c) => (clockView = c));
        }, 60000);
        return () => clearInterval(timer);
    });

    async function act(fn, key) {
        busy = true;
        try {
            await fn();
        } catch (e) {
            logger.error('direction: ' + key, e);
            statusBarTextStore.set(tMsg(key));
        } finally {
            busy = false;
        }
    }

    const onResult = (m, w, a, b, note) => act(() => enterResult(m, w, a, b, note), 'direction.result.error');
    const onForfeit = (m, w, note) => act(() => enterForfeit(m, w, note), 'direction.result.error');
    const onMove = (m, table) => act(() => moveMatchToTable(m, table), 'direction.result.error');
    const onCancel = (m) => act(() => cancelMatch(m), 'direction.result.error');
    const onCorrect = (m, w, a, b) => act(() => correctResult(m, w, a, b, ''), 'direction.result.error');
    const onAdd = (n, c, r) => act(() => addParticipant(n, c, r), 'direction.players.error');
    const onUpdate = (i, n, c, r) => act(() => updateParticipant(i, n, c, r), 'direction.players.error');
    const onWithdraw = (i, after) => act(() => withdrawParticipant(i, after), 'direction.players.error');

    const onClose = () => act(() => finishTournament(), 'direction.standings.error');
    /* Rouvrir est un geste explicite : le classement final cesse d'être final, et le journal
       en gardera la trace. Une confirmation, pas davantage. */
    const onReopen = () => {
        if (!window.confirm($t('direction.standings.reopenConfirm'))) return;
        return act(() => reopenTournament(), 'direction.standings.error');
    };
    const onNote = (text) => act(() => addNote(text), 'direction.history.error');

    /* L'affichage de la salle : un dossier choisi une fois, puis plus rien à faire. La page
       s'ouvre dans le navigateur du poste, hors ligne — c'est la seule chose sur laquelle on
       puisse compter un dimanche matin. */
    const onChooseOutput = () => act(() => chooseDirectionOutputDir(), 'direction.display.error');
    const onForgetOutput = () => act(() => forgetDirectionOutputDir(), 'direction.display.error');
    /* La feuille d'appariements : un clic, et le dialogue d'impression du système s'ouvre. Ce
       que le directeur voulait, c'est le papier, pas un onglet. */
    async function onPrintSheet() {
        const path = await writePairingSheet(sheetRound);
        if (!path) {
            statusBarTextStore.set(tMsg('direction.sheet.error'));
            return;
        }
        BrowserOpenURL('file://' + path);
    }

    async function onOpenPage() {
        const path = await writeDirectionPage();
        if (!path) {
            statusBarTextStore.set(tMsg('direction.display.error'));
            return;
        }
        BrowserOpenURL('file://' + path);
    }
    const onAttach = (slot, matchId) =>
        act(async () => {
            await attachMatchToSlot(slot, matchId);
            slotRows = await slots();
            unattached = await unattachedMatches();
        }, 'direction.slots.error');
    const onDetach = (slot) =>
        act(async () => {
            await detachMatchFromSlot(slot);
            slotRows = await slots();
            unattached = await unattachedMatches();
        }, 'direction.slots.error');

    /* Transcrire depuis un emplacement ouvre l'onglet Transcription : le brouillon se tape
       devant le plateau, pas dans la vue tournoi. */
    async function onTranscribe(slotId) {
        busy = true;
        try {
            await transcribeFromSlot(slotId);
            slotRows = await slots();
            activeTabStore.set('transcription');
        } catch (e) {
            logger.error('direction: transcribe from slot failed', e);
            statusBarTextStore.set(tMsg('direction.slots.error'));
        } finally {
            busy = false;
        }
    }

    /* Ouvrir le Match d'un emplacement ramène au plateau, dans l'onglet Matchs. */
    function onOpenMatch() {
        activeTabStore.set('matches');
    }

    /* Le CSV part dans le presse-papier : il n'y a pas de dialogue d'enregistrement ici, et
       coller dans un tableur est le geste qu'un directeur fait de toute façon. */
    async function onCSV() {
        try {
            const csv = await standingsCSV();
            await navigator.clipboard.writeText(csv);
            statusBarTextStore.set(tMsg('direction.standings.copied'));
        } catch (e) {
            logger.error('direction: standings csv failed', e);
            statusBarTextStore.set(tMsg('direction.standings.error'));
        }
    }

    /* Corriger depuis l'historique : la fiche vit sur la grille des tables, donc on y ramène. */
    function correctFromHistory() {
        tab = 'direction';
    }

    /* Cliquer une place de l'arbre ramène à la page Direction, où le match se saisit : la
       fiche de résultat vit sur la grille des tables, et il n'y en a qu'une. */
    function openBracketMatch(m) {
        if (!m.matchId) return;
        tab = 'direction';
    }

    function playerName(id) {
        const p = (view?.players || []).find((x) => x.id === id);
        return p ? p.name : id;
    }

    async function confirm(action) {
        busy = true;
        try {
            await confirmProposal(action);
        } catch (e) {
            logger.error('direction: confirm failed', e);
            statusBarTextStore.set(tMsg('direction.proposals.errorConfirm'));
        } finally {
            busy = false;
        }
    }

    async function confirmAll() {
        busy = true;
        try {
            await confirmAllProposals();
        } catch (e) {
            logger.error('direction: confirm all failed', e);
            statusBarTextStore.set(tMsg('direction.proposals.errorConfirm'));
        } finally {
            busy = false;
        }
    }

    async function manual(a, b, length, table) {
        busy = true;
        try {
            await startMatchManually(a, b, length, table);
        } catch (e) {
            logger.error('direction: manual pairing failed', e);
            statusBarTextStore.set(tMsg('direction.proposals.errorManual'));
        } finally {
            busy = false;
        }
    }

    async function remove() {
        if (!window.confirm($t('direction.settings.deleteConfirm'))) return;
        try {
            await deleteDirection($openDirectionIdStore);
            statusBarTextStore.set(tMsg('direction.settings.deleted'));
        } catch (e) {
            logger.error('direction: delete failed', e);
        }
    }
</script>

<div class="direction-view">
    <header>
        <span class="name">{view?.config?.name || ''}</span>
        <span class="state">{$t(`direction.state.${state}`)}</span>
        <nav>
            {#each tabs as item (item.id)}
                <button type="button" class:active={tab === item.id} onclick={() => (tab = item.id)}>{$t(item.labelKey)}</button>
            {/each}
        </nav>
        <span class="spacer"></span>
        <button type="button" class="credit-btn" title={$t('direction.credit.open')} aria-label={$t('direction.credit.open')} onclick={() => (creditOpen = !creditOpen)}>ⓘ</button>
        <button type="button" class="close" onclick={closeDirection}>{$t('direction.close')}</button>
    </header>

    {#if creditOpen}
        <div class="credit-layer">
            <CreditModal engineVersion={view?.engineVersion || ''} onClose={() => (creditOpen = false)} />
        </div>
    {/if}

    <ClockBar clock={clockView} warnings={view?.warnings?.length || 0} onWarnings={() => (tab = 'direction')} />

    <div class="body">
        {#if tab === 'settings'}
            <DirectionSettings
                bind:config
                {state}
                tournamentName={view?.config?.name || ''}
                entrantCount={view?.players?.length || 0}
                onApply={apply}
                onDelete={remove}
                onPreview={previewDirectionConfig}
                locks={configPreview?.locks || []}
                opened={configPreview?.opened || 0}
                outputDir={view?.outputDir || ''}
                {onChooseOutput}
                {onForgetOutput}
                {onOpenPage}
            />
        {:else if tab === 'direction'}
            <div class="direction-page">
                {#if (view?.warnings || []).length}
                    <!-- Un avertissement est visible EN PERMANENCE et ne bloque rien : il
                         disparaît quand sa cause disparaît, jamais parce qu'on l'a lu. -->
                    <ul class="warnings">
                        {#each view.warnings as w, i (w.code + (w.match || '') + i)}
                            <li>{renderWarning($t, w, playerName)}</li>
                        {/each}
                    </ul>
                {/if}
                {#if rounds > 0}
                    <div class="sheet">
                        {#if rounds > 1}
                            <label title={$t('direction.sheet.roundHint')}>
                                {$t('direction.sheet.round')}
                                <select bind:value={sheetRound}>
                                    <option value={0}>{$t('direction.sheet.latest')}</option>
                                    {#each Array.from({ length: rounds }, (_, i) => i + 1) as n (n)}
                                        <option value={n}>{n}</option>
                                    {/each}
                                </select>
                            </label>
                        {/if}
                        <button type="button" title={$t('direction.sheet.hint')} onclick={onPrintSheet}>
                            {$t('direction.sheet.print')}
                        </button>
                    </div>
                {/if}
                <ProposalList proposals={view?.proposals || []} players={free} {busy} onConfirm={confirm} onConfirmAll={confirmAll} onManual={manual} />
                <TableGrid {cells} {busy} {onResult} {onForfeit} {onMove} {onCancel} />
                <LastDecision {last} {busy} {onCorrect} onCancelMatch={onCancel} />
                <section class="waiting">
                    <h3>{$t('direction.waiting.title', { n: free.length })}</h3>
                    <p>{free.map((p) => p.name).join(' · ')}</p>
                </section>
            </div>
        {:else if tab === 'slots'}
            <SlotsView slots={slotRows} {unattached} {busy} {onTranscribe} {onAttach} {onDetach} {onOpenMatch} />
        {:else if tab === 'standings'}
            <StandingsView view={ranking} {busy} {onClose} {onReopen} {onCSV} />
        {:else if tab === 'history'}
            <HistoryView {entries} {busy} onCorrect={correctFromHistory} {onCancel} {onNote} />
        {:else if tab === 'brackets'}
            <BracketsView {phases} onOpenMatch={openBracketMatch} />
        {:else if tab === 'players'}
            <PlayersView {rows} {suggestions} {busy} started={state !== 'draft'} {onAdd} {onUpdate} {onWithdraw} />
        {:else}
            <p class="placeholder">{$t('direction.tabs.notYet')}</p>
        {/if}
    </div>
</div>

<style>
    .direction-view {
        position: relative;
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
        text-align: left;
        color: var(--color-text);
        background: var(--color-surface);
    }

    header {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-1) var(--space-2);
        border-bottom: 1px solid var(--color-border);
        flex-wrap: wrap;
    }

    .name {
        font-weight: 600;
    }

    .state {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    nav {
        display: flex;
        gap: var(--space-1);
        flex-wrap: wrap;
    }

    nav button,
    .close {
        font-size: var(--font-size-small);
        padding: 0.15rem 0.55rem;
        border: 1px solid transparent;
        border-radius: var(--radius);
        background: transparent;
        color: var(--color-text-muted);
        cursor: pointer;
    }

    nav button.active {
        border-color: var(--color-primary);
        color: var(--color-text);
    }

    .close {
        border-color: var(--color-border);
        color: var(--color-text);
    }

    .spacer {
        flex: 1;
    }

    .body {
        flex: 1;
        min-height: 0;
        overflow: auto;
    }

    .direction-page {
        display: flex;
        flex-direction: column;
        min-height: 0;
    }

    /* La feuille d'appariements se demande d'une ligne discrète : c'est un geste de quelques
       fois par tournoi, pas de toutes les minutes. */
    .sheet {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-1) var(--space-2);
    }

    .sheet label {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .sheet button,
    .sheet select {
        padding: 0.2rem 0.6rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    /* Les avertissements sont une bande, pas une fenêtre : le directeur doit pouvoir continuer
       à travailler avec eux sous les yeux. */
    .warnings {
        list-style: none;
        margin: 0;
        padding: var(--space-1) var(--space-2);
        border-bottom: 1px solid var(--color-danger);
        color: var(--color-danger);
        font-size: var(--font-size-small);
    }

    .waiting {
        padding: var(--space-1) var(--space-2);
        border-top: 1px solid var(--color-border);
    }

    .waiting h3 {
        margin: 0 0 2px;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
    }

    .waiting p {
        margin: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* Le crédit s'ouvre sous l'en-tête, là où le bouton est : il n'interrompt pas le travail
       en cours et se referme d'un clic. */
    .credit-layer {
        position: absolute;
        top: 2.2rem;
        right: var(--space-2);
        z-index: 30;
    }

    .credit-btn {
        border-color: transparent;
        color: var(--color-text-muted);
        background: transparent;
        cursor: pointer;
        font-size: var(--font-size-base);
        padding: 0 0.3rem;
    }

    .placeholder {
        margin: var(--space-4);
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
