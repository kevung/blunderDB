<script>
    /*
     * La vue tournoi, à la place du plateau quand une Direction est ouverte (ADR-0047,
     * tasks/nicomaque/ux.md §2) ; tout autre onglet ramène le plateau sans rien fermer.
     */
    import { t } from '../../i18n';
    import { statusBarTextStore, activeTabStore } from '../../stores/uiStore';
    import { tMsg } from '../../i18n';
    import { logger } from '../../utils/logger.js';
    import { focusPanelUnlessTyping } from '../../utils/panelFocus.js';
    import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
    import DirectionSettings from './DirectionSettings.svelte';
    import DirectoryPanel from './DirectoryPanel.svelte';
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
    import { renderWarning, csvFilename } from './labels.js';
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
        reinstateParticipant,
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
        writeUpcomingSheet,
        directionRounds,
        directory,
        directorySources,
        takeEntrantsFrom,
        directoryCSV,
        saveCSV,
        parseDirectoryCSV,
        enterParticipants,
        freeSlots,
        addParticipantAtSlot
    } from '../../stores/directionStore';

    /** @typedef {import('../../stores/directionStore.js').DirectionConfig} DirectionConfig */
    /** @typedef {import('../../stores/directionStore.js').ProposalAction} ProposalAction */
    /** @typedef {import('../../stores/directionStore.js').EntrantInput} EntrantInput */
    /** @typedef {import('../../../wailsjs/go/models').tournoi.Player} Player */
    /** @typedef {import('../../../wailsjs/go/models').database.BracketPhase} BracketPhase */
    /** @typedef {import('../../../wailsjs/go/models').database.ClockView} ClockView */
    /** @typedef {import('../../../wailsjs/go/models').database.ConfigPreview} ConfigPreview */
    /** @typedef {import('../../../wailsjs/go/models').database.DirectoryEntry} DirectoryEntry */
    /** @typedef {import('../../../wailsjs/go/models').database.DirectorySource} DirectorySource */
    /** @typedef {import('../../../wailsjs/go/models').database.EntrySuggestion} EntrySuggestion */
    /** @typedef {import('../../../wailsjs/go/models').database.FreeSlot} FreeSlot */
    /** @typedef {import('../../../wailsjs/go/models').database.HistoryEntry} HistoryEntry */
    /** @typedef {import('../../../wailsjs/go/models').database.LastDecision} LastDecision */
    /** @typedef {import('../../../wailsjs/go/models').database.ParticipantRow} ParticipantRow */
    /** @typedef {import('../../../wailsjs/go/models').database.SlotRow} SlotRow */
    /** @typedef {import('../../../wailsjs/go/models').database.SlotSuggestion} SlotSuggestion */
    /** @typedef {import('../../../wailsjs/go/models').database.StandingsView} StandingsView */
    /** @typedef {import('../../../wailsjs/go/models').database.TableCell} TableCell */

    const view = $derived($directionStore);
    // Pas `state` : chaque rune `$state` se lirait comme un abonnement au store `state`.
    const directionState = $derived(view?.state || 'draft');

    /* Onglet d'ouverture : Réglages en préparation, Direction en cours. Seulement à
       l'ouverture : les Réglages restent accessibles en cours de tournoi. */
    /* Prend le clavier à l'ouverture (services/directionKeys.js), sinon J / K iraient au
       panneau Tournois ; jamais au détriment d'un champ. */
    let root = $state(/** @type {HTMLElement | null} */ (null));
    $effect(() => {
        if (root) focusPanelUnlessTyping(root);
    });

    let tab = $state('settings');
    let tabChosen = false;
    $effect(() => {
        if (tabChosen) return;
        if (directionState !== 'draft') tab = 'direction';
        tabChosen = true;
    });

    let config = $state(/** @type {DirectionConfig | null} */ (null));
    $effect(() => {
        // Copie éditée, distincte de la base tant qu'elle n'est pas enregistrée.
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

    /** @param {DirectionConfig} next */
    async function apply(next) {
        try {
            await saveDirectionConfig(/** @type {number} */ ($openDirectionIdStore), next);
            statusBarTextStore.set(tMsg('direction.settings.saved'));
        } catch (e) {
            logger.error('direction: saving configuration failed', e);
            statusBarTextStore.set(tMsg('direction.settings.errorSaving'));
        }
    }

    let busy = $state(false);
    let free = $state(/** @type {Player[]} */ ([]));
    let cells = $state(/** @type {TableCell[]} */ ([]));
    let last = $state(/** @type {LastDecision | null} */ (null));
    let rows = $state(/** @type {ParticipantRow[]} */ ([]));
    let suggestions = $state(/** @type {EntrySuggestion[]} */ ([]));
    let phases = $state(/** @type {BracketPhase[]} */ ([]));
    let ranking = $state(/** @type {StandingsView | null} */ (null));
    let entries = $state(/** @type {HistoryEntry[]} */ ([]));
    let clockView = $state(/** @type {ClockView | null} */ (null));
    let creditOpen = $state(false);
    let slotRows = $state(/** @type {SlotRow[]} */ ([]));
    let unattached = $state(/** @type {SlotSuggestion[]} */ ([]));
    let configPreview = $state(/** @type {ConfigPreview | null} */ (null));
    let rounds = $state(0);
    let dirEntries = $state(/** @type {DirectoryEntry[]} */ ([]));
    let dirSources = $state(/** @type {DirectorySource[]} */ ([]));
    let openSlots = $state(/** @type {FreeSlot[]} */ ([]));
    let sheetRound = $state(0);
    /* La feuille d'une ronde à venir : les appariements de la file, datés par le
       directeur, imprimés sans rien lancer. */
    let announcing = $state(false);
    let announced = $state('');
    const upcoming = $derived((view?.proposals || []).filter((a) => a.kind === 'start_match').length);

    /* La file est dérivée, jamais stockée (comme le classement et les arbres). */
    $effect(() => {
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
        // Verrous de format, lus sur la configuration en vigueur, pas sur la copie.
        const saved = view?.config;
        if (saved) previewDirectionConfig(saved).then((p) => (configPreview = p));
        // Affichage de salle réécrit à chaque événement ; sans dossier, rien ; un échec
        // n'interrompt jamais la direction.
        writeDirectionPage().then((path) => {
            if (path === null) statusBarTextStore.set(tMsg('direction.display.error'));
        });
        directionRounds().then((n) => (rounds = n));
        // L'annuaire dérive de toutes les Directions de la base, celle-ci comprise.
        directory().then((e) => (dirEntries = e));
        directorySources().then((s) => (dirSources = s));
        // Les places d'exemption encore libres : ce qu'on propose à un retardataire.
        freeSlots().then((s) => (openSlots = s));
    });

    /* Les Players de la base ne changent pas pendant un tournoi : une seule lecture suffit. */
    $effect(() => {
        entrySuggestions().then((s) => (suggestions = s));
    });

    /* Rafraîchi à la minute : le temps écoulé avance sans événement. */
    $effect(() => {
        const timer = setInterval(() => {
            tableGrid().then((c) => (cells = c));
            clock().then((c) => (clockView = c));
        }, 60000);
        return () => clearInterval(timer);
    });

    /**
     * @param {() => Promise<unknown>} fn
     * @param {string} key
     */
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

    /** @type {(m: string, w: string, a: number, b: number, note: string) => Promise<void>} */
    const onResult = (m, w, a, b, note) => act(() => enterResult(m, w, a, b, note), 'direction.result.error');
    /** @type {(m: string, w: string, note: string) => Promise<void>} */
    const onForfeit = (m, w, note) => act(() => enterForfeit(m, w, note), 'direction.result.error');
    /** @type {(m: string, table: number) => Promise<void>} */
    const onMove = (m, table) => act(() => moveMatchToTable(m, table), 'direction.result.error');
    /** @type {(m: string) => Promise<void>} */
    const onCancel = (m) => act(() => cancelMatch(m), 'direction.result.error');
    /** @type {(m: string, w: string, a: number, b: number) => Promise<void>} */
    const onCorrect = (m, w, a, b) => act(() => correctResult(m, w, a, b, ''), 'direction.result.error');
    /** @type {(n: string, c: string, r: number) => Promise<void>} */
    const onAdd = (n, c, r) => act(() => addParticipant(n, c, r), 'direction.players.error');
    /** @type {(n: string, c: string, r: number, section: string, key: string) => Promise<void>} */
    const onAddAtSlot = (n, c, r, section, key) => act(() => addParticipantAtSlot(n, c, r, section, key), 'direction.players.error');
    /** @type {(i: string, n: string, c: string, r: number) => Promise<void>} */
    const onUpdate = (i, n, c, r) => act(() => updateParticipant(i, n, c, r), 'direction.players.error');
    /** @type {(i: string, after: boolean) => Promise<void>} */
    const onWithdraw = (i, after) => act(() => withdrawParticipant(i, after), 'direction.players.error');
    /** @type {(i: string) => Promise<void>} */
    const onReinstate = (i) => act(() => reinstateParticipant(i), 'direction.players.error');

    const onClose = () => act(() => finishTournament(), 'direction.standings.error');
    /* Rouvrir est explicite (le classement cesse d'être final), confirmé dans le Classement. */
    const onReopen = () => act(() => reopenTournament(), 'direction.standings.error');
    const onNote = (/** @type {string} */ text) => act(() => addNote(text), 'direction.history.error');

    /* Affichage de salle : un dossier choisi une fois ; page hors ligne dans le navigateur. */
    const onChooseOutput = () => act(() => chooseDirectionOutputDir(), 'direction.display.error');
    const onForgetOutput = () => act(() => forgetDirectionOutputDir(), 'direction.display.error');
    /* Feuille d'appariements : un clic ouvre le dialogue d'impression. */
    async function onPrintSheet() {
        const path = await writePairingSheet(sheetRound);
        if (!path) {
            statusBarTextStore.set(tMsg('direction.sheet.error'));
            return;
        }
        BrowserOpenURL('file://' + path);
    }
    async function onPrintUpcoming() {
        const path = await writeUpcomingSheet(announced);
        if (!path) {
            statusBarTextStore.set(tMsg('direction.sheet.error'));
            return;
        }
        announcing = false;
        BrowserOpenURL('file://' + path);
    }

    /* L'annuaire : reprendre les inscrits d'un tournoi précédent en un clic. */
    const onTakeEntrants = (/** @type {number} */ sourceId) => act(() => takeEntrantsFrom(sourceId), 'direction.directory.failed');
    const onImportEntrants = (/** @type {EntrantInput[]} */ rows) => act(() => enterParticipants(rows), 'direction.directory.failed');
    async function onExportDirectory() {
        try {
            await navigator.clipboard.writeText(await directoryCSV());
            statusBarTextStore.set(tMsg('direction.directory.copied'));
        } catch (e) {
            logger.error('direction: directory export failed', e);
            statusBarTextStore.set(tMsg('direction.directory.failed'));
        }
    }

    /* CSV vers fichier : le texte de la copie, au dialogue natif ; annuler ne dit rien. */
    /**
     * @param {() => Promise<string>} body
     * @param {string} word
     * @param {string} savedKey
     * @param {string} failedKey
     */
    async function saveAsFile(body, word, savedKey, failedKey) {
        try {
            const path = await saveCSV(csvFilename(view?.config?.name || '', word), await body());
            if (path) statusBarTextStore.set(tMsg(savedKey, { path }));
        } catch (e) {
            logger.error('direction: saving a csv failed', e);
            statusBarTextStore.set(tMsg(failedKey));
        }
    }
    const onSaveDirectory = () => saveAsFile(directoryCSV, $t('direction.directory.fileWord'), 'direction.directory.saved', 'direction.directory.saveFailed');
    const onSaveStandings = () => saveAsFile(standingsCSV, $t('direction.standings.fileWord'), 'direction.standings.saved', 'direction.standings.saveFailed');

    async function onOpenPage() {
        const path = await writeDirectionPage();
        if (!path) {
            statusBarTextStore.set(tMsg('direction.display.error'));
            return;
        }
        BrowserOpenURL('file://' + path);
    }
    /** @type {(slot: string, matchId: number) => Promise<void>} */
    const onAttach = (slot, matchId) =>
        act(async () => {
            await attachMatchToSlot(slot, matchId);
            slotRows = await slots();
            unattached = await unattachedMatches();
        }, 'direction.slots.error');
    /** @type {(slot: string) => Promise<void>} */
    const onDetach = (slot) =>
        act(async () => {
            await detachMatchFromSlot(slot);
            slotRows = await slots();
            unattached = await unattachedMatches();
        }, 'direction.slots.error');

    /* Transcrire ouvre l'onglet Transcription, devant le plateau. */
    /** @param {string} slotId */
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

    /* CSV au presse-papier ; onSaveStandings en fait un fichier. */
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

    /* Une place de l'arbre ramène à la page Direction : la seule fiche de résultat. */
    /** @param {{ matchId?: string }} m */
    function openBracketMatch(m) {
        if (!m.matchId) return;
        tab = 'direction';
    }

    /** @param {string | undefined} id */
    function playerName(id) {
        const p = (view?.players || []).find((x) => x.id === id);
        return p ? p.name : id;
    }

    /** @param {ProposalAction} action */
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

    /**
     * @param {string} a
     * @param {string} b
     * @param {number} length
     * @param {number} table
     */
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
            await deleteDirection(/** @type {number} */ ($openDirectionIdStore));
            statusBarTextStore.set(tMsg('direction.settings.deleted'));
        } catch (e) {
            logger.error('direction: delete failed', e);
        }
    }
</script>

<div class="direction-view" tabindex="-1" bind:this={root}>
    <header>
        <span class="name">{view?.config?.name || ''}</span>
        <span class="state">{$t(`direction.state.${directionState}`)}</span>
        <nav>
            {#each tabs as item (item.id)}
                <button type="button" data-testid="direction-tab-{item.id}" class:active={tab === item.id} onclick={() => (tab = item.id)}>{$t(item.labelKey)}</button>
            {/each}
        </nav>
        <span class="spacer"></span>
        <button type="button" class="credit-btn" data-testid="direction-credit" title={$t('direction.credit.open')} aria-label={$t('direction.credit.open')} onclick={() => (creditOpen = !creditOpen)}
            >ⓘ</button
        >
        <button type="button" class="close" data-testid="direction-close" onclick={closeDirection}>{$t('direction.close')}</button>
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
                {directionState}
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
                    <!-- Visible tant que dure sa cause, jamais bloquant. -->
                    <ul class="warnings" data-testid="direction-warnings">
                        {#each view?.warnings || [] as w, i (w.code + (w.match || '') + i)}
                            <li>{renderWarning($t, w, playerName)}</li>
                        {/each}
                    </ul>
                {/if}
                <!-- La grille avant la file, qui grandit avec les inscrits : les tables
                     restent à l'écran. -->
                <TableGrid {cells} {busy} {onResult} {onForfeit} {onMove} {onCancel}>
                    {#snippet actions()}
                        {#if rounds > 0 || upcoming > 0}
                            <div class="sheet">
                                {#if announcing}
                                    <label title={$t('direction.sheet.announcedHint')}>
                                        {$t('direction.sheet.announcedFor')}
                                        <input type="text" data-testid="direction-sheet-announced" bind:value={announced} placeholder={$t('direction.sheet.announcedPlaceholder')} />
                                    </label>
                                    <button type="button" data-testid="direction-sheet-upcoming-print" onclick={onPrintUpcoming}>
                                        {$t('direction.sheet.print')}
                                    </button>
                                    <button type="button" data-testid="direction-sheet-upcoming-cancel" onclick={() => (announcing = false)}>
                                        {$t('common.cancel')}
                                    </button>
                                {:else}
                                    {#if upcoming > 0}
                                        <button type="button" data-testid="direction-sheet-upcoming" title={$t('direction.sheet.upcomingHint')} onclick={() => (announcing = true)}>
                                            {$t('direction.sheet.upcoming')}
                                        </button>
                                    {/if}
                                {/if}
                                {#if rounds > 1 && !announcing}
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
                                {#if rounds > 0 && !announcing}
                                    <button type="button" data-testid="direction-sheet-print" title={$t('direction.sheet.hint')} onclick={onPrintSheet}>
                                        {$t('direction.sheet.print')}
                                    </button>
                                {/if}
                            </div>
                        {/if}
                    {/snippet}
                </TableGrid>
                <LastDecision {last} {busy} {onCorrect} onCancelMatch={onCancel} />
                <ProposalList proposals={view?.proposals || []} players={free} {busy} onConfirm={confirm} onConfirmAll={confirmAll} onManual={manual} />
                <section class="waiting">
                    <h3>{$t('direction.waiting.title', { n: free.length })}</h3>
                    <p>{free.map((p) => p.name).join(' · ')}</p>
                </section>
            </div>
        {:else if tab === 'slots'}
            <SlotsView slots={slotRows} {unattached} {busy} {onTranscribe} {onAttach} {onDetach} {onOpenMatch} />
        {:else if tab === 'standings'}
            <StandingsView view={ranking} {busy} running={view?.running?.length || 0} {onClose} {onReopen} {onCSV} onSave={onSaveStandings} />
        {:else if tab === 'history'}
            <HistoryView {entries} {busy} {onCorrect} {onCancel} {onNote} />
        {:else if tab === 'brackets'}
            <BracketsView {phases} onOpenMatch={openBracketMatch} />
        {:else if tab === 'players'}
            <DirectoryPanel
                sources={dirSources}
                entries={dirEntries}
                {busy}
                onTake={onTakeEntrants}
                onExport={onExportDirectory}
                onSave={onSaveDirectory}
                onParse={parseDirectoryCSV}
                onImport={onImportEntrants}
            />
            <PlayersView {rows} {suggestions} {busy} started={directionState !== 'draft'} {onAdd} {onUpdate} {onWithdraw} {onReinstate} slots={openSlots} infos={view?.infos || []} {onAddAtSlot} />
        {/if}
    </div>
</div>

<style>
    .direction-view:focus {
        outline: none;
    }

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

    /* Dans l'en-tête de la grille, sans barre à elle : geste rare, hauteur comptée. */
    .sheet {
        display: flex;
        align-items: center;
        gap: var(--space-2);
    }

    .sheet label {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .sheet button,
    .sheet select,
    .sheet input {
        padding: 0.1rem 0.6rem;
        font-size: var(--font-size-small);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    /* Une bande, pas une fenêtre : on travaille avec. */
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

    /* Le crédit s'ouvre sous l'en-tête, sans interrompre. */
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
</style>
