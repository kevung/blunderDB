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
    RencontreTableGrid,
    EnterResult,
    EnterForfeit,
    MoveMatchToTable,
    CancelMatch,
    LastDecision,
    CorrectResult,
    Participants,
    EntrySuggestions,
    AddParticipant,
    AddPair,
    UpdatePair,
    UpdateParticipant,
    WithdrawParticipant,
    ReinstateParticipant,
    MakeParticipantAbsent,
    MakeParticipantAvailable,
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
    WriteDirectionPage,
    WriteDirectionPairingSheet,
    WriteDirectionUpcomingSheet,
    DirectionRounds,
    Directory,
    DirectorySources,
    DirectoryEntrants,
    DirectoryCSV,
    ParseDirectoryCSV,
    DirectionFreeSlots,
    AddParticipantAtSlot,
    GetRencontre
} from '../../wailsjs/go/database/Database.js';
import { OpenDirectionOutputDialog, SaveCSV } from '../../wailsjs/go/gui/App.js';
import { language, messageBlock, tMsg } from '../i18n';
import { statusBarTextStore, activeTabStore } from './uiStore.js';
import { logger } from '../utils/logger.js';

/*
 * La Direction ouverte (ADR-0047) : une seule à la fois, rien à sauver (chaque décision est
 * déjà écrite), rien de dérivé gardé ici — tout revient rejoué à chaque rafraîchissement.
 * Aucun texte affichable ne transite : le moteur rend des codes, l'interface les traduit.
 */

/** @typedef {import('../../wailsjs/go/models').service.DirectionView} DirectionView */
/** @typedef {import('../../wailsjs/go/models').service.DirectionSummary} DirectionSummary */
/** @typedef {import('../../wailsjs/go/models').tournoi.PhaseConfig} PhaseConfig */
/** @typedef {import('../../wailsjs/go/models').tournoi.TableRule} TableRule */
/** @typedef {import('../../wailsjs/go/models').tournoi.Retention} Retention */
/** @typedef {import('../../wailsjs/go/models').tournoi.PrizeScale} PrizeScale */

/**
 * Une configuration de Direction côté front : même forme que `tournoi.Config`, en JSON simple.
 *
 * @typedef {Object} DirectionConfig
 * @property {string} name
 * @property {PhaseConfig[]} phases
 * @property {number} [min_per_point]
 * @property {{ count?: number, unavailable?: number[], reserved?: TableRule[] }} tables
 * @property {{ entry_fee?: number, retention?: Retention, sections?: Record<string, PrizeScale> }} [prizes]
 * @property {{ start: string, end: string }[]} [breaks]
 */

/** @typedef {{ name: string, club?: string, rating?: number }} EntrantInput Un joueur à inscrire en lot. */

/**
 * Un libellé structuré du moteur (`tournoi.Label`), en objet simple.
 *
 * @typedef {Object} DirectionLabel
 * @property {string} [kind]
 * @property {number} [n]
 * @property {number} [losses]
 * @property {number} [match]
 * @property {string} [section]
 * @property {string} [text]
 * @property {number} [players]
 * @property {number} [spots]
 * @property {DirectionLabel} [sub]
 */

/**
 * Une proposition du moteur (`tournoi.Action`) : c'est elle qui repart quand le directeur la confirme.
 *
 * @typedef {Object} ProposalAction
 * @property {string} kind
 * @property {number} phase
 * @property {string} [section]
 * @property {DirectionLabel} [label]
 * @property {number} [round]
 * @property {string} [key]
 * @property {string} [match]
 * @property {string} [a]
 * @property {string} [b]
 * @property {number} [length]
 * @property {number} [table]
 * @property {{ slots?: string[], groups?: string[][], lives?: Record<string, number> }} [draw]
 * @property {string} [reason]
 * @property {any} [until]
 * @property {string} [warn]
 */

/** La vue rejouée de la Direction ouverte, ou null si aucune ne l'est. */
/** @type {import('svelte/store').Writable<DirectionView | null>} */
export const directionStore = writable(null);

/** L'identifiant du tournoi dirigé qui est ouvert, ou null. */
/** @type {import('svelte/store').Writable<number | null>} */
export const openDirectionIdStore = writable(null);

/** Les tournois de la base qui portent une Direction, pour la liste du panneau. */
/** @type {import('svelte/store').Writable<DirectionSummary[]>} */
export const directionSummariesStore = writable([]);

/**
 * Le nombre de propositions à confirmer d'une vue rejouée : « attendre » n'en est pas une, c'est
 * le moteur qui dit qu'il n'y a rien à faire — la compter ferait clignoter un badge en permanence.
 * @param {DirectionView | null | undefined} view
 */
function pendingCountOf(view) {
    if (!view || !view.proposals) return 0;
    return view.proposals.filter((a) => a.kind !== 'wait').length;
}

/** Le nombre de propositions en attente : ce que le badge de l'onglet montre. */
export const pendingProposalsStore = derived(directionStore, ($d) => pendingCountOf($d));

/*
 * Les épreuves d'une Rencontre (ADR-0056 §5) : ouvrir l'une ouvre les autres, rejouées à côté et
 * gardées ici tant que la Rencontre est affichée, pour que chaque onglet porte son résumé sans
 * attendre qu'on le choisisse. `directionStore` reste la seule vue "active" (ADR-0047) ; ce
 * store est un aparté qui ne la remplace pas.
 */
/** @type {import('svelte/store').Writable<Record<number, DirectionView>>} */
const rencontreEpreuveViewsStore = writable({});

/** @type {import('svelte/store').Writable<{ tournamentId: number, name: string }[]>} */
const rencontreEpreuveOrderStore = writable([]);

/**
 * Les onglets d'épreuve à montrer dans l'en-tête, dans l'ordre de la Rencontre ; un tableau vide
 * quand le tournoi ouvert n'en a pas — aucun changement visible pour lui.
 */
export const epreuveTabsStore = derived([rencontreEpreuveOrderStore, rencontreEpreuveViewsStore, openDirectionIdStore], ([$order, $views, $activeId]) =>
    $order.map((m) => {
        const v = $views[m.tournamentId];
        return {
            tournamentId: m.tournamentId,
            name: m.name,
            active: m.tournamentId === $activeId,
            pending: pendingCountOf(v),
            running: v?.running?.length || 0,
            warning: (v?.warnings?.length || 0) > 0
        };
    })
);

/**
 * Recharge les épreuves sœurs de la Rencontre du tournoi ouvert (aucune si `rencontreId` est 0),
 * en gardant la vue déjà connue de l'épreuve active plutôt que de la redemander.
 * @param {number} rencontreId
 * @param {DirectionView} activeView
 */
async function refreshRencontreEpreuves(rencontreId, activeView) {
    if (!rencontreId) {
        rencontreEpreuveOrderStore.set([]);
        rencontreEpreuveViewsStore.set({});
        return;
    }
    try {
        const room = await GetRencontre(rencontreId);
        const members = room?.members || [];
        rencontreEpreuveOrderStore.set(members.map((m) => ({ tournamentId: m.tournamentId, name: m.name })));
        /** @type {Record<number, DirectionView>} */
        const views = {};
        for (const m of members) {
            if (m.tournamentId === activeView.tournamentId) {
                views[m.tournamentId] = activeView;
                continue;
            }
            try {
                views[m.tournamentId] = await GetDirection(m.tournamentId);
            } catch (e) {
                logger.error('direction: refreshing a sister épreuve failed', e);
            }
        }
        rencontreEpreuveViewsStore.set(views);
    } catch (e) {
        logger.error('direction: refreshing the Rencontre failed', e);
    }
}

/** Les avertissements courants, comptés dans la bande d'horloge. */
export const directionWarningsStore = derived(directionStore, ($d) => ($d && $d.warnings) || []);

/**
 * Une demande de la recherche rapide : mener à une table (sa fiche si elle est occupée), à un
 * joueur (sa liste) ou à une épreuve. La page la lit quand elle est montée et que l'épreuve
 * visée est la sienne, puis la rend ; `seq` fait rejouer deux fois la même demande.
 * @typedef {{ kind: 'table' | 'player' | 'epreuve', tournamentId: number, table?: number, open?: boolean, name?: string, seq: number }} DirectionJump
 */
/** @type {import('svelte/store').Writable<DirectionJump | null>} */
export const directionJumpStore = writable(null);
let jumpSeq = 0;

/** @param {Omit<DirectionJump, 'seq'>} jump */
export function requestDirectionJump(jump) {
    directionJumpStore.set({ ...jump, seq: ++jumpSeq });
}

/** Vrai quand la vue tournoi doit occuper la zone principale à la place du plateau. */
export const directionOpenStore = derived(openDirectionIdStore, ($id) => $id !== null);

/**
 * La page Direction remplace le plateau : onglet Tournois actif ET Direction ouverte (ADR-0047).
 * App.svelte et le clavier (services/directionKeys.js) lisent ce seul signal.
 */
export const directionPageShownStore = derived([activeTabStore, directionOpenStore], ([$tab, $open]) => $tab === 'tournaments' && $open);

/**
 * Les configurations nommées proposées à la création, modifiables tant que la Direction est un
 * brouillon. La première, recommandée, est le défaut : un directeur novice n'a pas à choisir.
 */
/** @type {{ id: string, recommended?: boolean, build: (name: string) => DirectionConfig }[]} */
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

/** @param {string} name */
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

/** Dit si un tournoi est dirigé, sans le rejouer. @param {number} tournamentId */
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
        clearBatchTimer();
        refreshRencontreEpreuves(0, null);
        return null;
    }
    try {
        const before = get(pendingProposalsStore);
        const view = await GetDirection(id);
        directionStore.set(view);
        announceNewProposals(before, get(pendingProposalsStore));
        scheduleBatchRefresh(view);
        await refreshRencontreEpreuves(view.rencontreId || 0, view);
        return view;
    } catch (e) {
        logger.error('direction: refresh failed', e);
        directionStore.set(null);
        clearBatchTimer();
        refreshRencontreEpreuves(0, null);
        return null;
    }
}

/*
 * L'échéance d'une micro-ronde tombe sans qu'aucun événement ne soit écrit. Le rafraîchissement
 * est donc programmé dans le store, pas dans la vue (non montée quand le directeur est ailleurs,
 * justement quand le badge doit apparaître). Le minuteur redemande l'état, il ne lance rien.
 */
/** @type {ReturnType<typeof setTimeout> | null} */
let batchTimer = null;

function clearBatchTimer() {
    if (batchTimer !== null) {
        clearTimeout(batchTimer);
        batchTimer = null;
    }
}

/**
 * La prochaine échéance portée par la file, en millisecondes d'ici là, ou null.
 *
 * @param {{ proposals?: ({ kind?: string, reason?: string, until?: any } | null | undefined)[] } | null | undefined} view
 * @param {number} [now]
 * @returns {number | null}
 */
export function nextBatchDelay(view, now = Date.now()) {
    let soonest = null;
    for (const a of view?.proposals || []) {
        if (!a || !a.until) continue;
        const at = new Date(a.until).getTime();
        // Une date zéro côté Go arrive en l'an 1 : ce n'est pas une échéance.
        if (!Number.isFinite(at) || at < 946684800000) continue;
        if (soonest === null || at < soonest) soonest = at;
    }
    if (soonest === null) return null;
    return Math.max(0, soonest - now);
}

/** @param {DirectionView} view */
function scheduleBatchRefresh(view) {
    clearBatchTimer();
    const delay = nextBatchDelay(view);
    if (delay === null) return;
    // Une seconde de marge : l'échéance est comparée à l'horloge du backend, et deux horloges
    // qui se croisent à la milliseconde près feraient un aller-retour pour rien.
    batchTimer = setTimeout(
        () => {
            batchTimer = null;
            refreshDirection();
        },
        Math.min(delay + 1000, 3600000)
    );
}

/**
 * Annonce une fois, dans la barre d'état, des propositions apparues pendant que le directeur est
 * ailleurs. Ne lance rien.
 *
 * @param {number} before
 * @param {number} after
 */
function announceNewProposals(before, after) {
    if (after <= before || after === 0) return;
    if (get(activeTabStore) === 'tournaments') return;
    statusBarTextStore.set(tMsg('direction.proposals.appeared', { n: after }));
}

/**
 * Ouvre la Direction d'un tournoi ; ferme la précédente sans confirmation (rien n'attend d'être écrit).
 * @param {number} tournamentId
 */
export async function openDirection(tournamentId) {
    openDirectionIdStore.set(tournamentId);
    await publishDirectionStrings();
    return refreshDirection();
}

/**
 * Change d'épreuve dans la Rencontre ouverte (ADR-0056 §5) : un clic, sans confirmation, sans
 * rien fermer ni rejouer depuis le début — la vue déjà connue de l'épreuve visée (si la Rencontre
 * l'a chargée) s'affiche tout de suite, `refreshDirection` la rafraîchit ensuite. Mêmes gestes que
 * `openDirection`, l'épreuve quittée restant présente dans `epreuveTabsStore`.
 * @param {number} tournamentId
 */
export async function switchEpreuve(tournamentId) {
    const cached = get(rencontreEpreuveViewsStore)[tournamentId];
    if (cached) directionStore.set(cached);
    openDirectionIdStore.set(tournamentId);
    await publishDirectionStrings();
    return refreshDirection();
}

/**
 * Donne au backend le bloc `direction` du catalogue courant : la page de salle, écrite en Go,
 * rend les codes du moteur avec le même vocabulaire que le front.
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
 * Réécrit la page d'affichage de la salle ; sans dossier choisi, ne fait ni ne dit rien. Un
 * échec est signalé et n'interrompt jamais la direction du tournoi.
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

/** Écrit la feuille d'appariements d'une ronde (0 = la plus récente) et rend le fichier à ouvrir. */
export async function writePairingSheet(round = 0) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    try {
        return await WriteDirectionPairingSheet(id, round);
    } catch (e) {
        logger.error('direction: writing the pairing sheet failed', e);
        return null;
    }
}

/**
 * Écrit la feuille de la ronde proposée, datée, AVANT de la lancer, et rend le fichier. Ne lance
 * rien, n'écrit rien au journal.
 * @param {string} announced  la date et l'heure annoncées, telles que tapées
 */
export async function writeUpcomingSheet(announced) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    try {
        return await WriteDirectionUpcomingSheet(id, announced || '');
    } catch (e) {
        logger.error('direction: writing the announced sheet failed', e);
        return null;
    }
}

/** Le nombre de rondes imprimables de la phase courante. */
export async function directionRounds() {
    const id = get(openDirectionIdStore);
    if (id === null) return 0;
    try {
        return (await DirectionRounds(id)) || 0;
    } catch (e) {
        logger.error('direction: counting rounds failed', e);
        return 0;
    }
}

/*
 * Retardataires : le moteur ne refait jamais un tirage — le retardataire prend une place
 * d'exemption libre ou entre plus tard, et l'interface dit laquelle avant la validation.
 */

/** Les places d'exemption encore libres, dans l'ordre du tableau. */
export async function freeSlots() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    try {
        return (await DirectionFreeSlots(id)) || [];
    } catch (e) {
        logger.error('direction: free slots failed', e);
        return [];
    }
}

/**
 * Inscrit un retardataire sur une place d'exemption nommée.
 * @param {string} name
 * @param {string} club
 * @param {number | string} rating
 * @param {string} section
 * @param {string} key
 */
export async function addParticipantAtSlot(name, club, rating, section, key) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await AddParticipantAtSlot(id, name, club, Number(rating) || 0, section, key);
    directionStore.set(view);
    return view;
}

/*
 * L'annuaire : une vue dérivée de toutes les Directions de la base, jamais une table (blunderDB
 * n'a aucune notion de personne ; supprimer une Direction en retire ses Participants).
 */

/** Tous les Participants de toutes les Directions, dédoublonnés par nom. */
export async function directory() {
    try {
        return (await Directory()) || [];
    } catch (e) {
        logger.error('direction: directory failed', e);
        return [];
    }
}

/** Les tournois dirigés dont on peut reprendre les inscrits d'un coup. */
export async function directorySources() {
    try {
        return (await DirectorySources()) || [];
    } catch (e) {
        logger.error('direction: directory sources failed', e);
        return [];
    }
}

/** Reprend les inscrits d'un tournoi précédent. @param {number} sourceTournamentId */
export async function takeEntrantsFrom(sourceTournamentId) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const entrants = (await DirectoryEntrants(sourceTournamentId)) || [];
    return enterParticipants(entrants.map((e) => ({ name: e.name, club: e.club || '', rating: e.rating || 0 })));
}

/** L'annuaire en CSV, tel qu'un directeur le garde d'une saison sur l'autre. */
export async function directoryCSV() {
    return (await DirectoryCSV()) || '';
}

/**
 * Enregistre un CSV dans un fichier choisi et rend le chemin, ou '' si annulé. L'appelant passe
 * le même texte qu'à la copie : fichier et presse-papier ne peuvent pas différer.
 *
 * @param {string} defaultName
 * @param {string} body
 */
export async function saveCSV(defaultName, body) {
    return (await SaveCSV(defaultName, body)) || '';
}

/**
 * Lit un CSV collé SANS RIEN ÉCRIRE : le directeur voit les lignes fautives avant l'entrée.
 *
 * @param {string} body
 */
export async function parseDirectoryCSV(body) {
    // La Direction ouverte : un nom qu'elle a déjà inscrit revient en avertissement.
    return await ParseDirectoryCSV(get(openDirectionIdStore) ?? 0, body || '');
}

/** Choisit le dossier d'affichage, une fois par Direction ; la page est ensuite réécrite à chaque événement. */
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
    rencontreEpreuveOrderStore.set([]);
    rencontreEpreuveViewsStore.set({});
}

/**
 * Commence à diriger un tournoi et l'ouvre.
 * @param {number} tournamentId
 * @param {DirectionConfig} config
 * @param {number} [seed]
 */
export async function createDirection(tournamentId, config, seed = 0) {
    await CreateDirection(tournamentId, JSON.stringify(config), seed);
    await refreshDirectionSummaries();
    return openDirection(tournamentId);
}

/**
 * Installe une configuration. Le moteur refuse deux choses — retirer une phase ouverte, changer
 * le format d'une phase commencée — et previewDirectionConfig le dit avant le clic.
 *
 * @param {number} tournamentId
 * @param {DirectionConfig} config
 */
export async function saveDirectionConfig(tournamentId, config) {
    await SetDirectionConfig(tournamentId, JSON.stringify(config));
    return refreshDirection();
}

/**
 * Compare une configuration candidate à celle en vigueur, sans rien écrire. Inchangée, elle
 * remplit quand même les verrous : la vue Réglages sait ainsi quels formats sont figés.
 * @param {DirectionConfig | import('../../wailsjs/go/models').tournoi.Config} config
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

/**
 * Inscrit plusieurs joueurs d'un coup (import, reprise d'annuaire).
 * @param {EntrantInput[]} players
 */
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

/**
 * Inscrit un joueur.
 * @param {string} name
 * @param {string} club
 * @param {number} rating
 */
export async function addParticipant(name, club, rating) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await AddParticipant(id, name, club, rating);
    directionStore.set(view);
    return view;
}

/**
 * Inscrit une paire : un Participant « A / B » pour le moteur, deux personnes pour l'annuaire.
 * Une cote à 0 vaut « la moyenne des deux ».
 * @param {{ name: string, club: string, rating: number }[]} members
 * @param {number} rating
 */
export async function addPair(members, rating) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await AddPair(id, JSON.stringify(members), rating);
    directionStore.set(view);
    return view;
}

/**
 * Corrige une paire (ses personnes, sa cote) sans changer son identifiant.
 * @param {string} participantId
 * @param {{ name: string, club: string, rating: number }[]} members
 * @param {number} rating
 */
export async function updatePair(participantId, members, rating) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await UpdatePair(id, participantId, JSON.stringify(members), rating);
    directionStore.set(view);
    return view;
}

/**
 * Corrige une inscription SANS changer son identifiant, qui désigne un emplacement : un Match
 * déjà rattaché reste rattaché.
 *
 * @param {string} participantId
 * @param {string} name
 * @param {string} club
 * @param {number} rating
 */
export async function updateParticipant(participantId, name, club, rating) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await UpdateParticipant(id, participantId, name, club, rating);
    directionStore.set(view);
    return view;
}

/**
 * Réinscrit un joueur retiré, avec ses résultats et ses vies.
 * @param {string} participantId
 */
export async function reinstateParticipant(participantId) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await ReinstateParticipant(id, participantId);
    directionStore.set(view);
    return view;
}

/**
 * Retire un joueur, tout de suite ou après le match qu'il joue.
 *
 * @param {string} participantId
 * @param {boolean} afterCurrent
 */
export async function withdrawParticipant(participantId, afterCurrent) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await WithdrawParticipant(id, participantId, afterCurrent);
    directionStore.set(view);
    return view;
}

/**
 * Absente un joueur (D7.1) : il n'est plus apparié jusqu'à l'échéance, mais garde son rang, ses
 * vies, sa place — un `until` (ISO) OU un `round`, jamais les deux (le moteur refuse sinon).
 *
 * @param {string} participantId
 * @param {string} until ISO 8601, vide si round est donné
 * @param {number} round 0 si until est donné
 */
export async function makeParticipantAbsent(participantId, until, round) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await MakeParticipantAbsent(id, participantId, until || '', round || 0);
    directionStore.set(view);
    return view;
}

/**
 * Lève l'absence d'un joueur, tout de suite : le retour en un clic.
 * @param {string} participantId
 */
export async function makeParticipantAvailable(participantId) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await MakeParticipantAvailable(id, participantId);
    directionStore.set(view);
    return view;
}

/**
 * Supprime la Direction ; le tournoi et ses matchs restent, sans leur emplacement.
 *
 * @param {number} tournamentId
 */
export async function deleteDirection(tournamentId) {
    await DeleteDirection(tournamentId);
    if (get(openDirectionIdStore) === tournamentId) closeDirection();
    return refreshDirectionSummaries();
}

/**
 * Confirme une proposition. L'événement est écrit avant d'être appliqué ; la vue rendue est déjà
 * rejouée, sans second aller-retour.
 *
 * @param {ProposalAction} action
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
 * Lance un match non proposé (le joueur qui a un train). Hors graphe il est accepté avec un
 * avertissement ; seul l'impossible est refusé.
 *
 * @param {string} a
 * @param {string} b
 * @param {number} [length]
 * @param {number} [table]
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

/** Le numéro de la dernière demande de Salle : une réponse plus ancienne arrivée après est ignorée. */
let hallSeq = 0;

/**
 * La Salle de la Rencontre ouverte (ADR-0056 §5) : une case par table, quelle que soit
 * l'épreuve, et les propositions de chacune — fusionnées en Go, rejouées à chaque appel. Null
 * hors Rencontre ; `undefined` quand une demande plus récente l'a dépassée (sa réponse, ou son
 * erreur, ne doit pas écraser celle d'après). Une erreur remonte à l'appelant, qui la montre.
 *
 * @returns {Promise<import('../../wailsjs/go/models').service.HallView | null | undefined>}
 */
export async function hallGrid() {
    const seq = ++hallSeq;
    const rid = get(directionStore)?.rencontreId || 0;
    if (!rid) return null;
    try {
        const h = await RencontreTableGrid(rid);
        return seq === hallSeq ? h : undefined;
    } catch (e) {
        if (seq !== hallSeq) return undefined;
        logger.error('direction: hall grid failed', e);
        throw e;
    }
}

/**
 * Un geste de la Salle vise l'épreuve de sa case, qui n'est pas forcément l'épreuve ouverte :
 * sa page d'affichage est réécrite (celle de l'épreuve ouverte l'est par la vue), puis
 * l'épreuve ouverte et ses sœurs sont rejouées, pour que chaque onglet dise vrai.
 *
 * @template T
 * @param {number} tid
 * @param {() => Promise<T>} fn
 * @returns {Promise<T>}
 */
async function hallGesture(tid, fn) {
    const v = await fn();
    try {
        await WriteDirectionPage(tid);
    } catch (e) {
        logger.error('direction: writing the page of a hall gesture failed', e);
    }
    await refreshDirection();
    return v;
}

/** @param {number} tid @param {string} matchId @param {string} winner @param {number} [a] @param {number} [b] @param {string} [note] */
export const hallEnterResult = (tid, matchId, winner, a = 0, b = 0, note = '') => hallGesture(tid, () => EnterResult(tid, matchId, winner, a, b, note));

/** @param {number} tid @param {string} matchId @param {string} winner @param {string} [note] */
export const hallEnterForfeit = (tid, matchId, winner, note = '') => hallGesture(tid, () => EnterForfeit(tid, matchId, winner, note));

/**
 * Déplace un match de la Salle ; sur une table occupée par une épreuve sœur, le service échange
 * les deux matchs, un changement de table dans chaque journal.
 *
 * @param {number} tid @param {string} matchId @param {number} table
 */
export const hallMoveMatch = (tid, matchId, table) => hallGesture(tid, () => MoveMatchToTable(tid, matchId, table));

/** @param {number} tid @param {string} matchId */
export const hallCancelMatch = (tid, matchId) => hallGesture(tid, () => CancelMatch(tid, matchId));

/** @param {number} tid @param {ProposalAction} action */
export const hallConfirmProposal = (tid, action) => hallGesture(tid, () => ConfirmProposal(tid, JSON.stringify(action)));

/**
 * Enregistre un résultat. Seul le vainqueur est exigé : les scores peuvent être nuls tous deux.
 *
 * @param {string} matchId
 * @param {string} winner
 * @param {number} [scoreA]
 * @param {number} [scoreB]
 * @param {string} [note]
 */
export async function enterResult(matchId, winner, scoreA = 0, scoreB = 0, note = '') {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await EnterResult(id, matchId, winner, scoreA, scoreB, note);
    directionStore.set(view);
    return view;
}

/**
 * Forfait pour CE match, sans retirer le joueur du tournoi.
 *
 * @param {string} matchId
 * @param {string} winner
 * @param {string} [note]
 */
export async function enterForfeit(matchId, winner, note = '') {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await EnterForfeit(id, matchId, winner, note);
    directionStore.set(view);
    return view;
}

/**
 * Déplace un match en cours vers une autre table.
 *
 * @param {string} matchId
 * @param {number} table
 */
export async function moveMatchToTable(matchId, table) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    const view = await MoveMatchToTable(id, matchId, table);
    directionStore.set(view);
    return view;
}

/**
 * Annule un match lancé par erreur. Rien n'est effacé du journal.
 *
 * @param {string} matchId
 */
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
 * Corrige le résultat d'un match fini : un événement de plus, le premier résultat reste au journal.
 *
 * @param {string} matchId
 * @param {string} winner
 * @param {number} [scoreA]
 * @param {number} [scoreB]
 * @param {string} [note]
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
 * Clôt le TOURNOI et fige le classement final — à ne pas confondre avec closeDirection, qui
 * ferme seulement la vue.
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

/**
 * Une annotation libre du directeur, horodatée.
 * @param {string} text
 */
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
 * Rattache un Match à un emplacement — toujours un geste : une coïncidence de noms n'est qu'une suggestion.
 *
 * @param {string} slotId
 * @param {number} matchId
 */
export async function attachMatchToSlot(slotId, matchId) {
    const id = get(openDirectionIdStore);
    if (id === null) return;
    await AttachMatchToSlot(id, slotId, matchId);
    await refreshDirection();
}

/**
 * Vide un emplacement, sans toucher au Match ni au résultat enregistré.
 *
 * @param {string} slotId
 */
export async function detachMatchFromSlot(slotId) {
    const id = get(openDirectionIdStore);
    if (id === null) return;
    await DetachMatchFromSlot(id, slotId);
    await refreshDirection();
}

/**
 * Ouvre un brouillon depuis un emplacement, en-tête rempli ; l'emplacement est réservé dès le
 * brouillon, sinon deux personnes taperaient le même match.
 *
 * @param {string} slotId
 */
export async function transcribeFromSlot(slotId) {
    const id = get(openDirectionIdStore);
    if (id === null) return null;
    return StartTranscriptionFromSlot(id, slotId);
}
