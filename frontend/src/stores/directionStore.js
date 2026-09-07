import { writable, derived, get } from 'svelte/store';
import {
    ListDirections,
    GetDirection,
    HasDirection,
    CreateDirection,
    SetDirectionConfig,
    StartDirection,
    DeleteDirection,
    ConfirmProposal,
    ConfirmAllProposals,
    StartMatchManually,
    FreeParticipants,
    TableGrid,
    EnterResult,
    EnterForfeit,
    MoveMatchToTable,
    CancelMatch
} from '../../wailsjs/go/database/Database.js';
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
    return refreshDirection();
}

/** Referme la Direction ouverte. Le plateau revient. */
export function closeDirection() {
    openDirectionIdStore.set(null);
    directionStore.set(null);
}

/** Commence à diriger un tournoi qui ne l'était pas, et l'ouvre. */
export async function createDirection(tournamentId, config) {
    await CreateDirection(tournamentId, JSON.stringify(config));
    await refreshDirectionSummaries();
    return openDirection(tournamentId);
}

/** Réécrit la configuration d'un brouillon. Refusée une fois le tournoi commencé. */
export async function saveDirectionConfig(tournamentId, config) {
    await SetDirectionConfig(tournamentId, JSON.stringify(config));
    return refreshDirection();
}

/**
 * Lance le tournoi : la configuration se fige dans le journal, les inscrits y sont écrits, et
 * la Direction passe en cours.
 */
export async function startDirection(tournamentId, players, seed = 0) {
    await StartDirection(tournamentId, seed, JSON.stringify(players || []));
    await refreshDirectionSummaries();
    return refreshDirection();
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
