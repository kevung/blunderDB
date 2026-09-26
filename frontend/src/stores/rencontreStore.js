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
    SetRencontreTableOutOfService
} from '../../wailsjs/go/database/Database.js';
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

/** Ce que le rattachement changera dans la configuration du tournoi, sans rien écrire. */
export async function previewAttach(tournamentId, rencontreId) {
    return PreviewAttachToRencontre(tournamentId, rencontreId);
}

export async function attachToRencontre(tournamentId, rencontreId) {
    const r = await AttachToRencontre(tournamentId, rencontreId);
    await refreshDirection();
    return r;
}

export async function detachFromRencontre(tournamentId) {
    await DetachFromRencontre(tournamentId);
    await refreshDirection();
}

/** Supprime la Rencontre par la corbeille : ses épreuves sont détachées, aucune supprimée. */
export async function trashRencontre(rencontreId) {
    await TrashRencontre(rencontreId);
    await refreshDirection();
}

/** Une table hors service (ou rendue), déclarée une fois pour toute la salle. */
export async function setTableOutOfService(rencontreId, table, out) {
    const r = await SetRencontreTableOutOfService(rencontreId, table, out);
    await refreshDirection();
    return r;
}
