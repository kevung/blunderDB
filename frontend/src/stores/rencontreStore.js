/*
 * La Rencontre : la salle que partagent plusieurs Directions (ADR-0056). Le cycle de vie —
 * créer, rattacher, détacher, supprimer — et les gestes de salle déclarés une fois. La règle
 * vit côté Go (Database) ; ce module appelle, puis rafraîchit la Direction ouverte, dont la
 * file et la grille dépendent des tables occupées à côté.
 */
import {
    ListRencontres,
    CreateRencontre,
    AttachToRencontre,
    PreviewAttachToRencontre,
    DetachFromRencontre,
    TrashRencontre,
    SetRencontreTableOutOfService,
    SetRencontreOutputDir,
    WriteRencontrePage
} from '../../wailsjs/go/database/Database.js';
import { OpenDirectionOutputDialog } from '../../wailsjs/go/gui/App.js';
import { refreshDirection } from './directionStore.js';
import { logger } from '../utils/logger.js';

/** @typedef {import('../../wailsjs/go/models').database.RencontreView} RencontreView */

/** Toutes les Rencontres de la base, la plus récente d'abord. */
export async function listRencontres() {
    try {
        return (await ListRencontres()) || [];
    } catch (e) {
        logger.error('rencontre: list failed', e);
        return [];
    }
}

/**
 * Ouvre une salle, puis y rattache le tournoi (son rattachement aligne ses tables).
 * @param {string} name @param {string} startsOn @param {string} endsOn @param {number} tables
 */
export async function createRencontre(name, startsOn, endsOn, tables) {
    return CreateRencontre(name, startsOn, endsOn, tables);
}

/**
 * Ce que le rattachement changera dans la configuration du tournoi, sans rien écrire.
 * @param {number} tournamentId @param {number} rencontreId
 */
export async function previewAttach(tournamentId, rencontreId) {
    return PreviewAttachToRencontre(tournamentId, rencontreId);
}

/** @param {number} tournamentId @param {number} rencontreId */
export async function attachToRencontre(tournamentId, rencontreId) {
    const r = await AttachToRencontre(tournamentId, rencontreId);
    await refreshDirection();
    return r;
}

/** @param {number} tournamentId */
export async function detachFromRencontre(tournamentId) {
    await DetachFromRencontre(tournamentId);
    await refreshDirection();
}

/**
 * Supprime la Rencontre par la corbeille : ses épreuves sont détachées, aucune supprimée.
 * @param {number} rencontreId
 */
export async function trashRencontre(rencontreId) {
    await TrashRencontre(rencontreId);
    await refreshDirection();
}

/**
 * Une table hors service (ou rendue), déclarée une fois pour toute la salle.
 * @param {number} rencontreId @param {number} table @param {boolean} out
 */
export async function setTableOutOfService(rencontreId, table, out) {
    const r = await SetRencontreTableOutOfService(rencontreId, table, out);
    await refreshDirection();
    return r;
}

/**
 * Choisit le dossier de sortie de la Rencontre — le geste unique de D6.5 : le dossier réglé, la
 * page murale est déjà écrite, et chaque épreuve rattachée y écrit désormais les siennes.
 * @param {number} rencontreId
 */
export async function chooseRencontreOutputDir(rencontreId) {
    const dir = await OpenDirectionOutputDialog();
    if (!dir) return null;
    return SetRencontreOutputDir(rencontreId, dir);
}

/**
 * Oublie le dossier de la Rencontre : la page murale cesse d'être réécrite, celle qui existe reste.
 * @param {number} rencontreId
 */
export async function forgetRencontreOutputDir(rencontreId) {
    return SetRencontreOutputDir(rencontreId, '');
}

/**
 * Réécrit la page murale et rend le fichier à ouvrir.
 * @param {number} rencontreId
 */
export async function writeRencontrePage(rencontreId) {
    return (await WriteRencontrePage(rencontreId)) || '';
}
