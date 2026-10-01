<script>
    /*
     * La vue tournoi, à la place du plateau quand une Direction est ouverte (ADR-0047,
     * tasks/nicomaque/ux.md §2) ; tout autre onglet ramène le plateau sans rien fermer.
     */
    import { tick } from 'svelte';
    import { t } from '../../i18n';
    import { statusBarTextStore, activeTabStore } from '../../stores/uiStore';
    import { tMsg } from '../../i18n';
    import { logger } from '../../utils/logger.js';
    import { focusPanelUnlessTyping } from '../../utils/panelFocus.js';
    import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';
    import DirectionSettings from './DirectionSettings.svelte';
    import RencontrePanel from './RencontrePanel.svelte';
    import DirectoryPanel from './DirectoryPanel.svelte';
    import ProposalList from './ProposalList.svelte';
    import TableGrid from './TableGrid.svelte';
    import HallView from './HallView.svelte';
    import LastDecision from './LastDecision.svelte';
    import PlayersView from './PlayersView.svelte';
    import BracketsView from './BracketsView.svelte';
    import StandingsView from './StandingsView.svelte';
    import HistoryView from './HistoryView.svelte';
    import ClockBar from './ClockBar.svelte';
    import CreditModal from './CreditModal.svelte';
    import SlotsView from './SlotsView.svelte';
    import ContextMenu from '../ContextMenu.svelte';
    import { playerMenu } from '../../services/directionMenus.js';
    import { menuRequest } from '../../services/contextMenuTrigger.js';
    import { setTableOutOfService } from '../../stores/rencontreStore.js';
    import { renderWarning, csvFilename, seatLabel } from './labels.js';
    import {
        directionStore,
        openDirectionIdStore,
        epreuveTabsStore,
        switchEpreuve,
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
        addPair,
        updatePair,
        withdrawParticipant,
        reinstateParticipant,
        makeParticipantAbsent,
        makeParticipantAvailable,
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
        addParticipantAtSlot,
        hallGrid
    } from '../../stores/directionStore';

    /** @typedef {import('../../stores/directionStore.js').DirectionConfig} DirectionConfig */
    /** @typedef {import('../../stores/directionStore.js').ProposalAction} ProposalAction */
    /** @typedef {import('../../stores/directionStore.js').EntrantInput} EntrantInput */
    /** @typedef {import('../../../wailsjs/go/models').tournoi.Player} Player */
    /** @typedef {import('../../../wailsjs/go/models').service.BracketPhase} BracketPhase */
    /** @typedef {import('../../../wailsjs/go/models').service.ClockView} ClockView */
    /** @typedef {import('../../../wailsjs/go/models').service.ConfigPreview} ConfigPreview */
    /** @typedef {import('../../../wailsjs/go/models').service.DirectoryEntry} DirectoryEntry */
    /** @typedef {import('../../../wailsjs/go/models').service.DirectorySource} DirectorySource */
    /** @typedef {import('../../../wailsjs/go/models').service.EntrySuggestion} EntrySuggestion */
    /** @typedef {import('../../../wailsjs/go/models').service.FreeSlot} FreeSlot */
    /** @typedef {import('../../../wailsjs/go/models').service.HistoryEntry} HistoryEntry */
    /** @typedef {import('../../../wailsjs/go/models').service.LastDecision} LastDecision */
    /** @typedef {import('../../../wailsjs/go/models').service.ParticipantRow} ParticipantRow */
    /** @typedef {import('../../../wailsjs/go/models').service.SlotRow} SlotRow */
    /** @typedef {import('../../../wailsjs/go/models').service.SlotSuggestion} SlotSuggestion */
    /** @typedef {import('../../../wailsjs/go/models').service.StandingsView} StandingsView */
    /** @typedef {import('../../../wailsjs/go/models').service.TableCell} TableCell */

    const view = $derived($directionStore);
    // Pas `state` : chaque rune `$state` se lirait comme un abonnement au store `state`.
    const directionState = $derived(view?.state || 'draft');
    // Un retour "à la ronde N" n'a de sens que dans un suisse par rondes (absence.go) : sinon
    // le moteur le refuse, et la case ne doit même pas être proposée.
    const roundsMode = $derived.by(() => {
        const ph = view?.config?.phases?.[view?.phase ?? -1];
        return !!ph && ph.kind === 'swiss_lives' && ph.mode === 'rounds';
    });

    /* Onglet d'ouverture : Réglages en préparation, Direction en cours. Seulement à
       l'ouverture : les Réglages restent accessibles en cours de tournoi. */
    /* Prend le clavier à l'ouverture (services/directionKeys.js), sinon J / K iraient au
       panneau Tournois ; jamais au détriment d'un champ. */
    let root = $state(/** @type {HTMLElement | null} */ (null));
    $effect(() => {
        if (root) focusPanelUnlessTyping(root);
    });

    let tab = $state('settings');
    const paneKey = $derived(`${view?.tournamentId ?? 0}:${tab}`);

    /* Un onglet se monte à sa première visite puis reste monté, masqué : ses filtres, ses sections
       repliées et sa saisie survivent au changement d'onglet. */
    let visited = $state(/** @type {Record<string, boolean>} */ ({}));
    $effect(() => {
        visited[tab] = true;
    });

    /** Position de défilement par (épreuve, onglet), gardée tant que la vue vit.
     * @type {Map<string, number>} */
    const scrollPositions = new Map(); // eslint-disable-line svelte/prefer-svelte-reactivity -- non réactif : jamais lu par le rendu
    /** @type {Record<string, HTMLElement>} */
    const paneEls = {};
    let restoring = false;
    /** @type {(() => void) | null} */
    let stopRestore = null;

    /**
     * @param {HTMLElement} node
     * @param {string} id
     */
    function registerPane(node, id) {
        paneEls[id] = node;
        return { destroy: () => delete paneEls[id] };
    }

    /** @param {string} id */
    function onPaneScroll(id) {
        const el = paneEls[id];
        if (restoring || id !== tab || !el || el.clientHeight === 0) return;
        scrollPositions.set(paneKey, el.scrollTop);
    }

    /* Rend la position mémorisée une fois le contenu assez haut pour l'atteindre (les données
       arrivent après le montage) ; l'enregistrement est suspendu pendant ce temps, sans quoi le
       défilement clampé à 0 écraserait la mémoire. Un geste de l'utilisateur y met fin. */
    /** @param {HTMLElement} el @param {string} key */
    function restoreScroll(el, key) {
        stopRestore?.();
        const target = scrollPositions.get(key) ?? 0;
        el.scrollTop = target;
        if (target === 0 || el.scrollTop >= target - 1) {
            restoring = false;
            return;
        }
        restoring = true;
        const ro = new ResizeObserver(() => {
            el.scrollTop = target;
            if (el.scrollTop >= target - 1) stopRestore?.();
        });
        for (const c of Array.from(el.children)) ro.observe(c);
        const timer = setTimeout(() => stopRestore?.(), 1500);
        const cancel = () => stopRestore?.();
        el.addEventListener('wheel', cancel, { once: true, passive: true });
        el.addEventListener('pointerdown', cancel, { once: true, passive: true });
        el.addEventListener('keydown', cancel, { once: true });
        stopRestore = () => {
            ro.disconnect();
            clearTimeout(timer);
            el.removeEventListener('wheel', cancel);
            el.removeEventListener('pointerdown', cancel);
            el.removeEventListener('keydown', cancel);
            restoring = false;
            stopRestore = null;
        };
    }

    /* À chaque changement d'épreuve ou d'onglet : la position revient, et le volet actif prend le
       clavier (PageUp / PageDown / flèches le font défiler), sans voler un champ en cours de saisie. */
    $effect(() => {
        const key = paneKey;
        const id = tab;
        void visited[id];
        tick().then(() => {
            const el = paneEls[id];
            if (!el || id !== tab) return;
            restoreScroll(el, key);
            focusPanelUnlessTyping(el);
        });
    });
    let tabChosen = false;
    $effect(() => {
        // Tant que la Direction n'est pas chargée, son état n'est pas « brouillon » : il est
        // inconnu, et choisir sur lui ouvrirait les Réglages d'un tournoi en cours.
        if (tabChosen || !view) return;
        if (directionState !== 'draft') tab = 'direction';
        tabChosen = true;
    });

    /* Les menus contextuels mènent d'un écran à l'autre : la table d'un joueur, l'historique
       d'un nom. Chaque demande porte un numéro, pour que la même demande répétée se rejoue. */
    let reveal = $state(/** @type {{ table: number, open?: boolean, seq: number } | null} */ (null));
    let historyFilter = $state(/** @type {{ text: string, seq: number } | null} */ (null));
    let requestSeq = 0;
    /** Une demande à la file des propositions : appariement à la main, ou lancer ici. */
    let queueRequest = $state(/** @type {{ kind: 'manual' | 'launchHere', a?: string, table?: number, focus?: 'a' | 'b', seq: number } | null} */ (null));
    let waitingMenu = $state(/** @type {import('../../services/contextMenuTrigger.js').MenuRequest | null} */ (null));

    /** @param {number} table @param {boolean} open */
    function goToTable(table, open) {
        tab = 'direction';
        reveal = { table, open, seq: ++requestSeq };
    }

    /** @param {string} id */
    function pairManually(id) {
        tab = 'direction';
        queueRequest = { kind: 'manual', a: id, focus: 'b', seq: ++requestSeq };
    }

    /** Une table libre prend la proposition sélectionnée : offert tant que la file en a une à lancer. @type {((table: number) => void) | undefined} */
    const onLaunchHere = $derived(
        (view?.proposals || []).some((a) => a.kind === 'start_match')
            ? (/** @type {number} */ table) => {
                  queueRequest = { kind: 'launchHere', table, seq: ++requestSeq };
              }
            : undefined
    );

    /** « Joue aussi à <épreuve> » : l'onglet de cette épreuve de la Rencontre. @param {string} event */
    function goEpreuve(event) {
        const ep = $epreuveTabsStore.find((x) => x.name === event);
        if (ep) switchEpreuve(ep.tournamentId);
    }

    /** @param {string} name */
    function showHistory(name) {
        tab = 'history';
        historyFilter = { text: name, seq: ++requestSeq };
    }

    /** Hors service : la table d'une Rencontre, seule à porter cet état. @type {((table: number, out: boolean) => void) | undefined} */
    const onOutOfService = $derived(
        view?.rencontreId ? (/** @type {number} */ table, /** @type {boolean} */ out) => void act(() => setTableOutOfService(view?.rencontreId || 0, table, out), 'direction.result.error') : undefined
    );

    /** @param {MouseEvent | KeyboardEvent} ev @param {{ id: string, name: string }} p */
    function onWaitingMenu(ev, p) {
        const req = menuRequest(ev, () =>
            playerMenu(
                (k, m) => $t(k, m),
                { id: p.id, name: p.name, state: 'free', elsewhere: view?.elsewhere?.[p.id] },
                { busy, onHistory: showHistory, onWithdraw, onManual: pairManually, onGoElsewhere: goEpreuve }
            )
        );
        if (req) waitingMenu = req;
    }

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

    /*
     * La Salle (ADR-0056 §5) : un onglet à gauche des épreuves quand une Rencontre est ouverte.
     * Elle remplace les volets de l'épreuve sans les démonter ; un clic sur une épreuve ou sur un
     * onglet de vue la quitte. Rechargée quand l'épreuve ouverte ou une sœur change.
     */
    let hallOpen = $state(false);
    let hall = $state(/** @type {import('../../../wailsjs/go/models').service.HallView | null} */ (null));
    const hasHall = $derived($epreuveTabsStore.length > 1);
    $effect(() => {
        if (!hasHall) hallOpen = false;
    });
    $effect(() => {
        void view;
        void $epreuveTabsStore;
        if (hallOpen) loadHall();
    });

    let hallError = $state('');
    /* Une réponse dépassée rend `undefined` : on garde ce qu'on a. Une erreur se montre au lieu
       d'un chargement sans fin. */
    function loadHall() {
        hallGrid().then(
            (h) => {
                if (h === undefined) return;
                hall = h;
                hallError = '';
            },
            (e) => {
                hallError = String(e?.message ?? e);
            }
        );
    }

    /** @param {number} tournamentId */
    function pickEpreuve(tournamentId) {
        hallOpen = false;
        switchEpreuve(tournamentId);
    }

    /** L'historique d'un joueur de la Salle est celui de l'épreuve de sa case. @param {string} name @param {number} tid */
    async function hallHistory(name, tid) {
        hallOpen = false;
        if (tid && tid !== view?.tournamentId) await switchEpreuve(tid);
        showHistory(name);
    }

    /* Rafraîchi à la minute : le temps écoulé avance sans événement. */
    $effect(() => {
        const timer = setInterval(() => {
            if (hallOpen) loadHall();
            tableGrid().then((c) => (cells = c));
            clock().then((c) => (clockView = c));
        }, 60000);
        return () => clearInterval(timer);
    });

    /**
     * Rend `false` sur un échec, pour qu'une fiche ouverte le dise au lieu de se fermer.
     *
     * @param {() => Promise<unknown>} fn
     * @param {string | ((e: any) => import('../../i18n').StatusMessage)} key
     * @returns {Promise<boolean>}
     */
    async function act(fn, key) {
        busy = true;
        try {
            await fn();
            return true;
        } catch (e) {
            logger.error('direction: ' + (typeof key === 'string' ? key : 'failed'), e);
            statusBarTextStore.set(typeof key === 'string' ? tMsg(key) : key(e));
            return false;
        } finally {
            busy = false;
        }
    }

    /** @type {(m: string, w: string, a: number, b: number, note: string) => Promise<boolean>} */
    const onResult = (m, w, a, b, note) => act(() => enterResult(m, w, a, b, note), 'direction.result.error');
    /** @type {(m: string, w: string, note: string) => Promise<boolean>} */
    const onForfeit = (m, w, note) => act(() => enterForfeit(m, w, note), 'direction.result.error');
    /** @type {(m: string, table: number) => Promise<boolean>} */
    const onMove = (m, table) =>
        act(
            () => moveMatchToTable(m, table),
            // Le service dit pourquoi (table hors service…) : le refus se lit tel quel.
            (e) => tMsg('direction.table.moveRefused', { reason: String(e?.message ?? e).replace(/^direction:\s*/, '') })
        );
    /** @type {(m: string) => Promise<boolean>} */
    const onCancel = (m) => act(() => cancelMatch(m), 'direction.result.error');
    /** @type {(m: string, w: string, a: number, b: number) => Promise<boolean>} */
    const onCorrect = (m, w, a, b) => act(() => correctResult(m, w, a, b, ''), 'direction.result.error');
    /** @type {(n: string, c: string, r: number) => Promise<boolean>} */
    const onAdd = (n, c, r) => act(() => addParticipant(n, c, r), 'direction.players.error');
    /** @type {(n: string, c: string, r: number, section: string, key: string) => Promise<boolean>} */
    const onAddAtSlot = (n, c, r, section, key) => act(() => addParticipantAtSlot(n, c, r, section, key), 'direction.players.error');
    /** @type {(i: string, n: string, c: string, r: number) => Promise<boolean>} */
    const onUpdate = (i, n, c, r) => act(() => updateParticipant(i, n, c, r), 'direction.players.error');
    /** @type {(m: {name: string, club: string, rating: number}[], r: number) => Promise<boolean>} */
    const onAddPair = (m, r) => act(() => addPair(m, r), 'direction.players.error');
    /** @type {(i: string, m: {name: string, club: string, rating: number}[], r: number) => Promise<boolean>} */
    const onUpdatePair = (i, m, r) => act(() => updatePair(i, m, r), 'direction.players.error');
    /** @type {(i: string, after: boolean) => Promise<boolean>} */
    const onWithdraw = (i, after) => act(() => withdrawParticipant(i, after), 'direction.players.error');
    /** @type {(i: string) => Promise<boolean>} */
    const onReinstate = (i) => act(() => reinstateParticipant(i), 'direction.players.error');
    /** @type {(i: string, until: string, round: number) => Promise<boolean>} */
    const onAbsent = (i, until, round) => act(() => makeParticipantAbsent(i, until, round), 'direction.players.error');
    /** @type {(i: string) => Promise<boolean>} */
    const onReturn = (i) => act(() => makeParticipantAvailable(i), 'direction.players.error');

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
    /** @type {(slot: string, matchId: number) => Promise<boolean>} */
    const onAttach = (slot, matchId) =>
        act(async () => {
            await attachMatchToSlot(slot, matchId);
            slotRows = await slots();
            unattached = await unattachedMatches();
        }, 'direction.slots.error');
    /** @type {(slot: string) => Promise<boolean>} */
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

    /* L'arbre a une place ouverte sur une fiche dès qu'une phase a un graphe en cours : la
       pastille de l'onglet, pour le voir sans aller le chercher. */
    const bracketLive = $derived(phases.some((p) => p.current && p.sections.some((s) => s.kind !== 'barrage' && s.matches.length > 0)));

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

    /* Page murale depuis l'en-tête : sans dossier de sortie, on le demande d'abord. */
    async function openPageFromHeader() {
        if (!view?.outputDir) {
            const dir = await chooseDirectionOutputDir().catch(() => null);
            if (!dir) return;
        }
        await onOpenPage();
    }

    async function remove() {
        if (!window.confirm($t('direction.settings.deleteConfirm'))) return;
        try {
            const tid = view?.tournamentId;
            for (const k of Array.from(scrollPositions.keys())) if (k.startsWith(`${tid}:`)) scrollPositions.delete(k);
            await deleteDirection(/** @type {number} */ ($openDirectionIdStore));
            statusBarTextStore.set(tMsg('direction.settings.deleted'));
        } catch (e) {
            logger.error('direction: delete failed', e);
        }
    }
</script>

<div class="direction-view" tabindex="-1" bind:this={root}>
    {#if $epreuveTabsStore.length > 1}
        <!-- La Rencontre (ADR-0056 §5) : un onglet par épreuve, résumé visible sans y aller, un
             clic pour y passer — rien ne ferme ni ne rejoue l'épreuve quittée. -->
        <nav class="epreuve-tabs" data-testid="epreuve-tabs">
            <button type="button" data-testid="epreuve-tab-hall" class="hall-tab" class:active={hallOpen} aria-pressed={hallOpen} onclick={() => (hallOpen = true)}>
                <span class="epreuve-name">{$t('direction.hall.tab')}</span>
            </button>
            {#each $epreuveTabsStore as ep (ep.tournamentId)}
                <button type="button" data-testid="epreuve-tab-{ep.tournamentId}" class:active={ep.active && !hallOpen} onclick={() => pickEpreuve(ep.tournamentId)}>
                    <span class="epreuve-name">{ep.name}</span>
                    {#if ep.pending}<span class="badge pending" title={$t('direction.epreuves.pending', { n: ep.pending })}>{ep.pending}</span>{/if}
                    {#if ep.running}<span class="badge running" title={$t('direction.epreuves.running', { n: ep.running })}>{ep.running}</span>{/if}
                    {#if ep.warning}<span class="badge warning" title={$t('direction.epreuves.alert')}>!</span>{/if}
                </button>
            {/each}
        </nav>
    {/if}
    <header>
        <span class="name">{view?.config?.name || ''}</span>
        <span class="state">{$t(`direction.state.${directionState}`)}</span>
        <nav>
            {#each tabs as item (item.id)}
                <button
                    type="button"
                    data-testid="direction-tab-{item.id}"
                    class:active={!hallOpen && tab === item.id}
                    onclick={() => {
                        hallOpen = false;
                        tab = item.id;
                    }}
                    >{$t(item.labelKey)}{#if item.id === 'brackets' && bracketLive}<span class="tab-dot" data-testid="brackets-dot" title={$t('direction.bracket.live')}>●</span>{/if}</button
                >
            {/each}
        </nav>
        <span class="spacer"></span>
        <button type="button" class="page-btn" data-testid="direction-open-page" onclick={openPageFromHeader}>{$t('direction.display.open')}</button>
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
        {#if hallOpen || hall || hallError}
            <div class="pane" hidden={!hallOpen} tabindex="-1" data-testid="direction-pane-hall" use:registerPane={'hall'} onscroll={() => onPaneScroll('hall')}>
                <HallView {hall} error={hallError} {busy} {act} onHistory={hallHistory} {onOutOfService} />
            </div>
        {/if}
        {#if visited.settings}
            <div class="pane" hidden={hallOpen || tab !== 'settings'} tabindex="-1" data-testid="direction-pane-settings" use:registerPane={'settings'} onscroll={() => onPaneScroll('settings')}>
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
                {#if view}
                    <RencontrePanel tournamentId={view.tournamentId} rencontreId={view.rencontreId || 0} />
                {/if}
            </div>
        {/if}
        {#if visited.direction}
            <div class="pane" hidden={hallOpen || tab !== 'direction'} tabindex="-1" data-testid="direction-pane-direction" use:registerPane={'direction'} onscroll={() => onPaneScroll('direction')}>
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
                    <TableGrid {cells} {busy} {onResult} {onForfeit} {onMove} {onCancel} onHistory={showHistory} {onOutOfService} {onLaunchHere} {reveal}>
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
                    <ProposalList
                        proposals={view?.proposals || []}
                        players={free}
                        elsewhere={view?.elsewhere || {}}
                        {busy}
                        onConfirm={confirm}
                        onConfirmAll={confirmAll}
                        onManual={manual}
                        request={queueRequest}
                        onPrintSheet={rounds > 0 ? onPrintSheet : undefined}
                    />
                    <section class="waiting">
                        <h3>{$t('direction.waiting.title', { n: free.length })}</h3>
                        <p data-testid="direction-waiting">
                            {#each free as p (p.id)}
                                <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_static_element_interactions -->
                                <span class="who" tabindex="0" oncontextmenu={(ev) => onWaitingMenu(ev, p)} onkeydown={(ev) => onWaitingMenu(ev, p)}
                                    >{p.name}{#if view?.elsewhere?.[p.id]}
                                        <span class="elsewhere">({seatLabel($t, view.elsewhere[p.id])})</span>{/if}</span
                                >
                            {/each}
                        </p>
                    </section>
                </div>
            </div>
        {/if}
        {#if visited.slots}
            <div class="pane" hidden={hallOpen || tab !== 'slots'} tabindex="-1" data-testid="direction-pane-slots" use:registerPane={'slots'} onscroll={() => onPaneScroll('slots')}>
                <SlotsView slots={slotRows} {unattached} {busy} {onTranscribe} {onAttach} {onDetach} {onOpenMatch} />
            </div>
        {/if}
        {#if visited.standings}
            <div class="pane" hidden={hallOpen || tab !== 'standings'} tabindex="-1" data-testid="direction-pane-standings" use:registerPane={'standings'} onscroll={() => onPaneScroll('standings')}>
                <StandingsView view={ranking} {busy} running={view?.running?.length || 0} {onClose} {onReopen} {onCSV} onSave={onSaveStandings} />
            </div>
        {/if}
        {#if visited.history}
            <div class="pane" hidden={hallOpen || tab !== 'history'} tabindex="-1" data-testid="direction-pane-history" use:registerPane={'history'} onscroll={() => onPaneScroll('history')}>
                <HistoryView {entries} {busy} {onCorrect} {onCancel} {onNote} filterRequest={historyFilter} />
            </div>
        {/if}
        {#if visited.brackets}
            <div class="pane" hidden={hallOpen || tab !== 'brackets'} tabindex="-1" data-testid="direction-pane-brackets" use:registerPane={'brackets'} onscroll={() => onPaneScroll('brackets')}>
                <BracketsView {phases} {cells} {busy} {onResult} {onForfeit} {onMove} {onCancel} {onCorrect} onHistory={showHistory} />
            </div>
        {/if}
        {#if visited.players}
            <div class="pane" hidden={hallOpen || tab !== 'players'} tabindex="-1" data-testid="direction-pane-players" use:registerPane={'players'} onscroll={() => onPaneScroll('players')}>
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
                <PlayersView
                    {rows}
                    {suggestions}
                    {busy}
                    started={directionState !== 'draft'}
                    {onAdd}
                    {onUpdate}
                    {onAddPair}
                    {onUpdatePair}
                    pairs={view?.pairs || {}}
                    {onWithdraw}
                    {onReinstate}
                    {onAbsent}
                    {onReturn}
                    onGoTable={goToTable}
                    onHistory={showHistory}
                    onManual={pairManually}
                    onGoEpreuve={goEpreuve}
                    elsewhere={view?.elsewhere || {}}
                    {roundsMode}
                    slots={openSlots}
                    infos={view?.infos || []}
                    {onAddAtSlot}
                />
            </div>
        {/if}
    </div>
    {#if waitingMenu}
        <ContextMenu x={waitingMenu.x} y={waitingMenu.y} items={waitingMenu.items} onClose={() => (waitingMenu = null)} />
    {/if}
</div>

<style>
    .direction-view:focus {
        outline: none;
    }

    .direction-view {
        --font-size-base: var(--td-font);
        --font-size-small: var(--td-font);
        position: relative;
        width: 100%;
        flex: 1;
        min-width: 0;
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

    /* Cibles de la salle : au moins --td-target, pour les contrôles de la vue (en-tête, onglets,
       feuille, actions des panneaux), les joueurs en attente qui ouvrent un menu, et les entrées
       des menus contextuels de la vue, touchés au doigt comme le reste ; jamais dans une fiche
       ni une surcouche, où une petite commande reste petite. */
    header button:not(.credit-btn),
    .epreuve-tabs button,
    .sheet :global(button),
    .sheet :global(select),
    .sheet :global(input),
    .pane :global(.primary),
    .pane :global(.actions button),
    .pane :global(.td-target) {
        min-height: var(--td-target);
        min-width: var(--td-target);
    }

    .direction-view :global(.context-menu-item) {
        min-height: var(--td-target);
    }

    .waiting .who {
        display: inline-flex;
        align-items: center;
        min-height: var(--td-target);
    }

    .pane[hidden] {
        display: none;
    }

    nav button,
    .close,
    .page-btn {
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

    .epreuve-tabs {
        padding: var(--space-1) var(--space-2) 0;
        border-bottom: 1px solid var(--color-border);
    }

    .epreuve-tabs button {
        display: flex;
        align-items: center;
        gap: 0.3em;
    }

    .badge {
        font-size: var(--font-size-small);
        line-height: 1;
        padding: 0.1rem 0.35rem;
        border-radius: var(--radius);
        background: var(--color-border);
        color: var(--color-text);
    }

    .tab-dot {
        margin-left: 0.3em;
        font-size: var(--font-size-small);
        color: var(--color-primary);
    }

    .badge.warning {
        background: var(--color-danger, #b00020);
        color: #fff;
        font-weight: 600;
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
        display: flex;
        flex-direction: column;
    }

    /* Un seul défilement par onglet (et par épreuve) : la position est mémorisée par paneKey. Le
       conteneur sert aussi aux container queries des vues. */
    .pane {
        flex: 1;
        min-height: 0;
        overflow: auto;
        container-type: inline-size;
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

    .waiting .who + .who::before {
        content: '· ';
    }

    .waiting .elsewhere {
        color: var(--color-danger);
    }

    /* Le crédit s'ouvre sous l'en-tête, sans interrompre. */
    .credit-layer {
        position: absolute;
        top: 2.2rem;
        right: var(--space-2);
        z-index: 30;
    }

    .credit-btn {
        min-width: 0;
        border-color: transparent;
        color: var(--color-text-muted);
        background: transparent;
        cursor: pointer;
        font-size: var(--font-size-base);
        padding: 0 0.3rem;
    }
</style>
