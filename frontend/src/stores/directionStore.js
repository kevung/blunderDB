import { writable, derived, get } from 'svelte/store';
import {
    ListDirections,
    GetDirection,
    HasDirection,
    CreateDirection,
    SetDirectionConfig,
    PreviewDirectionConfig,
    DeleteDirection,
    ConfirmProposal,
    ConfirmAllProposals,
    StartMatchManually,
    FreeParticipants,
    TableGrid,
    EnterResult,
    EnterForfeit,
    MoveMatchToTable,
    CancelMatch,
    LastDecision,
    CorrectResult,
    Participants,
    EntrySuggestions,
    AddParticipant,
    UpdateParticipant,
    WithdrawParticipant,
    EnterParticipants,
    Brackets,
    Standings,
    StandingsCSV,
    CloseDirection as CloseDirectionBinding,
    ReopenDirection as ReopenDirectionBinding,
    History,
    AddDirectionNote,
    Clock,
    Slots,
    UnattachedMatches,
    AttachMatchToSlot,
    DetachMatchFromSlot,
    StartTranscriptionFromSlot,
    SetDirectionOutputDir,
    SetDirectionStrings,
    WriteDirectionPage
} from '../../wailsjs/go/database/Database.js';
import { OpenDirectionOutputDialog } from '../../wailsjs/go/gui/App.js';
import { language, messageBlock } from '../i18n';
import { logger } from '../utils/logger.js';

/*
 * La Direction ouverte (ADR-0047).
 *
 * Une seule à la fois : en ouvrir une autre ferme la première, et il n'y a rien à sauver
 * puisque chaque décision est déjà écrite. Rien de dérivé n'est gardé ici — classement, arbres,
 * propositions et avertissements arrivent rejoués à chaque rafraîchissement, exactement comme
 * ils arrivent à l'ouverture de la base.
 *
 * Aucun texte affichable ne transite : les libellés, les notes de classement, les
 * avertissements et les raisons d'attente sont les codes du moteur, et c'est l'interface qui
 * les traduit dans la langue de l'utilisateur.
 */

/** La vue rejouée de la Direction ouverte, ou null si aucune ne l'est. */
export const directionStore = writable(null);

/** L'identifiant du tournoi dirigé qui est ouvert, ou null. */
export const openDirectionIdStore = writable(null);

/** Les tournois de la base qui portent une Direction, pour la liste du panneau. */
export const directionSummariesStore = writable([]);

/** Le nombre de propositions en attente : ce que le badge de l'onglet montre. */
export const pendingProposalsStore = derived(directionStore, ($d) => {
    if (!$d || !$d.proposals) return 0;
    // « Attendre » n'est pas une proposition à confirmer : c'est le moteur qui dit qu'il n'y a
    // rien à faire. La compter ferait clignoter un badge en permanence.
    return $d.proposals.filter((a) => a.kind !== 'wait').length;
});

/** Les avertissements courants, comptés dans la bande d'horloge. */
export const directionWarningsStore = derived(directionStore, ($d) => ($d && $d.warnings) || []);

/** Vrai quand la vue tournoi doit occuper la zone principale à la place du plateau. */
export const directionOpenStore = derived(openDirectionIdStore, ($id) => $id !== null);

/**
 * Les configurations nommées proposées à la création. Ce sont des points de départ, pas des
 * formats figés : chaque champ reste modifiable tant que la Direction est un brouillon.
 *
 * La première est celle que l'étude des formats recommande, et c'est le défaut : un directeur
 * qui ne connaît pas les formats ne doit pas avoir à choisir pour commencer.
 */
export const namedConfigs = [
    {
        id: 'suisse_tableau',
        recommended: true,
        build: (name) => ({
            name,
            tables: { count: 8 },
            phases: [
                { kind: 'swiss_lives', length: 7, lives: 2, mode: 'continuous', target: 16 },
                { kind: 'lives_bracket', length: 9, final_length: 11 }
            ]
        })
    },
    {
        id: 'elimination',
        build: (name) => ({
            name,
            tables: { count: 8 },
            phases: [{ kind: 'bracket', length: 11, final_length: 13 }]
        })
    },
    {
        id: 'consolante',
        build: (name) => ({
            name,
            tables: { count: 8 },
            phases: [{ kind: 'bracket', length: 9, consolation: true }]
        })
    },
    {
        id: 'double_elimination',
        build: (name) => ({
            name,
            tables: { count: 8 },
            phases: [
                {
                    kind: 'bracket',
                    length: 7,
                    consolation: true,
                    reconciliation: true,
                    recharge: true
                }
            ]
        })
    },
    {
        id: 'gsl',
        build: (name) => ({
            name,
            tables: { count: 8 },
            phases: [
                { kind: 'gsl', length: 7, target: 16 },
                { kind: 'lives_bracket', length: 9 }
            ]
        })
    },
    {
        id: 'poules',
        build: (name) => ({
            name,
            tables: { count: 8 },
            phases: [
                { kind: 'round_robin', length: 5, group_size: 4, qualifiers: 2 },
                { kind: 'bracket', length: 9 }
            ]
        })
    }
];

/** La configuration nommée par défaut, celle que l'étude recommande. */
export function defaultConfig(name) {
    return namedConfigs[0].build(name);
}

/** Recharge la liste des tournois dirigés. */
export async function refreshDirectionSummaries() {
    try {
        directionSummariesStore.set((await ListDirections()) || []);
    } catch (e) {
        logger.error('direction: listing failed', e);
        directionSummariesStore.set([]);
    }
}

/** Dit si un tournoi est dirigé, sans le rejouer. */
export async function tournamentIsDirected(tournamentId) {
    try {
        return await HasDirection(tournamentId);
    } catch (e) {
        logger.error('direction: HasDirection failed', e);
        return false;
    }
}

/** Rejoue la Direction ouverte et met la vue à jour. */
export async function refreshDirection() {
    const id = get(openDirectionIdStore);
    if (id === null) {
        directionStore.set(null);
        return null;
    }
    try {
        const view = await GetDirection(id);
        directionStore.set(view);
        return view;
    } catch (e) {
        logger.error('direction: refresh failed', e);
        directionStore.set(null);
        return null;
    }
}

/**
 * Ouvre la Direction d'un tournoi. Une seule est ouverte à la fois ; ouvrir la suivante ferme
 * la précédente, sans confirmation — rien n'attend d'être écrit.
 */
export async function openDirection(tournamentId) {
    openDirectionIdStore.set(tournamentId);
    await publishDirectionStrings();
    return refreshDirection();
}

/**
 * Donne au backend le bloc `direction` du catalogue courant (issue #386).
 *
 * La page d'affichage de la salle est écrite en Go, et Nicomaque n'émet que des codes : sans ce
 * geste elle sortirait avec des codes bruts. Le front passe donc ses propres mots, et les deux
 * côtés rendent les mêmes codes avec le même vocabulaire.
 */
export async function publishDirectionStrings() {
    try {
        await SetDirectionStrings(get(language), JSON.stringify(messageBlock('direction')));
    } catch (e) {
        logger.error('direction: publishing the catalogue failed', e);
    }
}

/* Changer de langue en cours de tournoi doit changer la page affichée dans la salle : le
   catalogue est republié dès que la langue bouge. Un `subscribe` est ici à sa place — la règle
   des stores Svelte 5 vise les composants, pas les modules. */
language.subscribe(() => {
    if (get(openDirectionIdStore) !== null) publishDirectionStrings();
});

/**
 * Réécrit la page d'affichage de la salle. Sans dossier choisi elle ne fait rien et ne dit
 * rien : un directeur qui n'a jamais demandé d'affichage n'a pas à en entendre parler à chaque
 * résultat. Un échec est signalé et n'interrompt JAMAIS la direction du tournoi.
 */
export async function writeDirectionPage() {
    const id = get(openDirectionIdStore);
    if (id === null) return '';
    try {
        return (await WriteDirectionPage(id)) || '';
    } catch (e) {
        logger.error('direction: writing the display page failed', e);
        return null;
    }
}

/**
 * Choisit le dossier d'affichage, une fois par Direction. Ensuite, plus aucun geste : la page
 * est réécrite à chaque événement.
 */
export async function chooseDirectionOutputDir() {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const dir = await OpenDirectionOutputDialog();
    if (!dir) return null;
    await SetDirectionOutputDir(id, dir);
    await refreshDirection();
    await writeDirectionPage();
    return dir;
}

/** Oublie le dossier d'affichage : la page cesse d'être réécrite, celle qui existe reste. */
export async function forgetDirectionOutputDir() {
    const id = get(openDirectionIdStore);
    if (id === null) return;
    await SetDirectionOutputDir(id, '');
    return refreshDirection();
}

/** Referme la Direction ouverte. Le plateau revient. */
export function closeDirection() {
    openDirectionIdStore.set(null);
    directionStore.set(null);
}

/** Commence à diriger un tournoi qui ne l'était pas, et l'ouvre. */
export async function createDirection(tournamentId, config, seed = 0) {
    await CreateDirection(tournamentId, JSON.stringify(config), seed);
    await refreshDirectionSummaries();
    return openDirection(tournamentId);
}

/**
 * Installe une configuration, en préparation comme en cours de tournoi (issue #385). Le moteur
 * refuse exactement deux choses : retirer une phase ouverte, et changer le format d'une phase
 * commencée — previewDirectionConfig le dit AVANT le clic.
 */
export async function saveDirectionConfig(tournamentId, config) {
    await SetDirectionConfig(tournamentId, JSON.stringify(config));
    return refreshDirection();
}

/**
 * Compare une configuration candidate à celle en vigueur, sans rien écrire.
 *
 * Appelée avec la configuration inchangée elle ne rapporte aucun changement et remplit quand
 * même les verrous : c'est ainsi que la vue Réglages sait, en s'ouvrant, quels formats sont
 * figés et pourquoi.
 */
export async function previewDirectionConfig(config) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    try {
        return await PreviewDirectionConfig(id, JSON.stringify(config));
    } catch (e) {
        logger.error('direction: previewing configuration failed', e);
        return null;
    }
}

/** Inscrit plusieurs joueurs d'un coup : ce que produit un import ou une reprise d'annuaire. */
export async function enterParticipants(players) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    await EnterParticipants(id, JSON.stringify(players || []));
    return refreshDirection();
}

/** Les inscrits, avec ce que l'état rejoué sait d'eux. */
export async function participants() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await Participants(id)) || [];
    } catch (e) {
        logger.error('direction: participants failed', e);
        return [];
    }
}

/** Les Players de la base, proposés à l'inscription. */
export async function entrySuggestions() {
    try {
        return (await EntrySuggestions()) || [];
    } catch (e) {
        logger.error('direction: entry suggestions failed', e);
        return [];
    }
}

/** Inscrit un joueur. */
export async function addParticipant(name, club, rating) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await AddParticipant(id, name, club, rating);
    directionStore.set(view);
    return view;
}

/**
 * Corrige une inscription SANS changer son identifiant : c'est lui que désigne un emplacement,
 * donc corriger un nom ne doit pas défaire un Match déjà rattaché.
 */
export async function updateParticipant(participantId, name, club, rating) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await UpdateParticipant(id, participantId, name, club, rating);
    directionStore.set(view);
    return view;
}

/** Retire un joueur, tout de suite ou après le match qu'il joue. */
export async function withdrawParticipant(participantId, afterCurrent) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await WithdrawParticipant(id, participantId, afterCurrent);
    directionStore.set(view);
    return view;
}

/**
 * Supprime la Direction. Le tournoi reste, ses matchs aussi : ils perdent seulement
 * l'emplacement qu'ils remplissaient.
 */
export async function deleteDirection(tournamentId) {
    await DeleteDirection(tournamentId);
    if (get(openDirectionIdStore) === tournamentId) closeDirection();
    return refreshDirectionSummaries();
}

/**
 * Confirme une proposition. L'événement est écrit AVANT d'être appliqué, côté Go ; la vue qui
 * revient est déjà rejouée, si bien que la file se met à jour sans second aller-retour.
 */
export async function confirmProposal(action) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await ConfirmProposal(id, JSON.stringify(action));
    directionStore.set(view);
    return view;
}

/** Confirme toute la file. Deux clics pour le directeur, quel que soit le nombre. */
export async function confirmAllProposals() {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await ConfirmAllProposals(id);
    directionStore.set(view);
    return view;
}

/**
 * Lance un match que le moteur n'a pas proposé. C'est l'échappatoire qui rend le panneau
 * utilisable par un vrai directeur — celui qui sait qu'un joueur a un train. Un appariement
 * hors graphe est accepté et laisse un avertissement ; seul l'impossible est refusé.
 */
export async function startMatchManually(a, b, length = 0, table = 0) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await StartMatchManually(id, a, b, length, table);
    directionStore.set(view);
    return view;
}

/** Les Participants de la phase courante qui ne jouent pas : la file d'attente. */
export async function freeParticipants() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await FreeParticipants(id)) || [];
    } catch (e) {
        logger.error('direction: free participants failed', e);
        return [];
    }
}

/** La grille des tables : une case par table de la salle, dérivée à chaque appel. */
export async function tableGrid() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await TableGrid(id)) || [];
    } catch (e) {
        logger.error('direction: table grid failed', e);
        return [];
    }
}

/**
 * Enregistre un résultat. Le VAINQUEUR est la seule chose exigée : les scores peuvent être nuls
 * tous les deux, ce que produit un directeur qui a seulement écrit « Alice gagne ».
 */
export async function enterResult(matchId, winner, scoreA = 0, scoreB = 0, note = '') {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await EnterResult(id, matchId, winner, scoreA, scoreB, note);
    directionStore.set(view);
    return view;
}

/** Forfait pour CE match, sans retirer le joueur du tournoi. */
export async function enterForfeit(matchId, winner, note = '') {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await EnterForfeit(id, matchId, winner, note);
    directionStore.set(view);
    return view;
}

/** Déplace un match en cours vers une autre table. */
export async function moveMatchToTable(matchId, table) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await MoveMatchToTable(id, matchId, table);
    directionStore.set(view);
    return view;
}

/** Annule un match lancé par erreur. Rien n'est effacé du journal. */
export async function cancelMatch(matchId) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await CancelMatch(id, matchId);
    directionStore.set(view);
    return view;
}

/** Le dernier geste du directeur, avec de quoi le reprendre en deux clics. */
export async function lastDecision() {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    try {
        return await LastDecision(id);
    } catch (e) {
        logger.error('direction: last decision failed', e);
        return null;
    }
}

/**
 * Corrige le résultat d'un match déjà fini. Rien n'est effacé : la correction est un événement
 * de plus, et le premier résultat reste dans l'historique là où il a eu lieu.
 */
export async function correctResult(matchId, winner, scoreA = 0, scoreB = 0, note = '') {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await CorrectResult(id, matchId, winner, scoreA, scoreB, note);
    directionStore.set(view);
    return view;
}

/** Les arbres de toutes les phases : structure brute, codes non traduits. */
export async function brackets() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await Brackets(id)) || [];
    } catch (e) {
        logger.error('direction: brackets failed', e);
        return [];
    }
}
/** Le classement et les prix, rejoués. */
export async function standings() {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    try {
        return await Standings(id);
    } catch (e) {
        logger.error('direction: standings failed', e);
        return null;
    }
}

/** Le classement en CSV, tel que le moteur l'écrit. */
export async function standingsCSV() {
    const id = get(openDirectionIdStore);
    if (id === null) return '';
    return StandingsCSV(id);
}

/**
 * Clot le TOURNOI et fige le classement final — à ne pas confondre avec closeDirection, qui
 * ferme seulement la vue. Le mot « fermer » a deux sens ici, et les confondre coûterait cher.
 */
export async function finishTournament() {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await CloseDirectionBinding(id);
    directionStore.set(view);
    await refreshDirectionSummaries();
    return view;
}

/** Rouvre un tournoi clos, parce qu'un résultat était faux. Le journal garde tout. */
export async function reopenTournament() {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await ReopenDirectionBinding(id);
    directionStore.set(view);
    await refreshDirectionSummaries();
    return view;
}

/** L'historique des décisions, filtrable par joueur ou par match. */
export async function history(player = '', match = '') {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await History(id, player, match)) || [];
    } catch (e) {
        logger.error('direction: history failed', e);
        return [];
    }
}

/** Une annotation libre du directeur, horodatée. */
export async function addNote(text) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await AddDirectionNote(id, text);
    directionStore.set(view);
    return view;
}

/** L'horloge : le temps, les matchs, l'allure, les matchs lents, la prochaine pause. */
export async function clock() {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    try {
        return await Clock(id);
    } catch (e) {
        logger.error('direction: clock failed', e);
        return null;
    }
}

/** Les emplacements du tournoi, avec ce qui les remplit. */
export async function slots() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await Slots(id)) || [];
    } catch (e) {
        logger.error('direction: slots failed', e);
        return [];
    }
}

/** Les matchs du tournoi qui ne remplissent aucun emplacement, avec la suggestion. */
export async function unattachedMatches() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await UnattachedMatches(id)) || [];
    } catch (e) {
        logger.error('direction: unattached matches failed', e);
        return [];
    }
}

/**
 * Rattache un Match à un emplacement. C'est TOUJOURS un geste : une coïncidence de noms est une
 * suggestion, jamais une décision du logiciel.
 */
export async function attachMatchToSlot(slotId, matchId) {
    const id = get(openDirectionIdStore);
    if (id === null) return;
    await AttachMatchToSlot(id, slotId, matchId);
    await refreshDirection();
}

/** Vide un emplacement, sans toucher au Match ni au résultat enregistré. */
export async function detachMatchFromSlot(slotId) {
    const id = get(openDirectionIdStore);
    if (id === null) return;
    await DetachMatchFromSlot(id, slotId);
    await refreshDirection();
}

/**
 * Ouvre un brouillon depuis un emplacement : l'en-tête est déjà rempli, et l'emplacement est
 * réservé dès le brouillon — sans quoi deux personnes taperaient le même match.
 */
export async function transcribeFromSlot(slotId) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    return StartTranscriptionFromSlot(id, slotId);
}
