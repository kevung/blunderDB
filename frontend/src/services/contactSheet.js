// La planche-contact : la liste parcourue en grille de mini-plateaux ; une
// vignette ouvre sa position sur le plateau. Dessin par
// diagramService.renderPositionSVG, les mêmes fonctions de scène que le
// plateau. Ici, ce qui se teste sans composant (pages, clavier, garde
// d'ouverture) ; le rendu est dans ContactSheetModal.svelte.

import { get } from 'svelte/store';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { statusBarModeStore, openModal, MODAL } from '../stores/uiStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { tMsg } from '../i18n';

/**
 * Vignettes par page : se range en 2, 3, 4, 6 ou 8 colonnes sans ligne
 * boiteuse, et borne le dessin quelle que soit la taille de la liste.
 */
export const PAGE_SIZE = 24;

/** @param {number} index */
export function pageOf(index) {
    return index > 0 ? Math.floor(index / PAGE_SIZE) : 0;
}

/** @param {number} total */
export function pageCount(total) {
    return total > 0 ? Math.ceil(total / PAGE_SIZE) : 0;
}

/**
 * Les indices [from, to) de la page `page` d'une liste de `total` positions.
 * @param {number} page
 * @param {number} total
 */
export function pageBounds(page, total) {
    const from = Math.min(Math.max(0, page) * PAGE_SIZE, Math.max(0, total));
    return { from, to: Math.min(from + PAGE_SIZE, total) };
}

/**
 * Les colonnes qui tiennent dans une largeur ; quatre si elle est inconnue (0).
 * @param {number} width
 * @param {number} tileMin largeur minimale d'une vignette
 * @param {number} gap espace entre deux vignettes
 */
export function columnsFor(width, tileMin, gap) {
    if (!(width > 0)) return 4;
    return Math.max(1, Math.floor((width + gap) / (tileMin + gap)));
}

/**
 * La vignette visée par une touche depuis `index`, ou null si ce n'est pas une
 * touche de la grille : flèches dans la grille, J / K suivante / précédente
 * (event.key), Début / Fin aux bouts de la page, Page préc. / suiv. Toujours
 * dans la liste : une flèche au bord ne fait rien.
 *
 * @param {number} index
 * @param {{key: string, ctrlKey?: boolean, metaKey?: boolean, altKey?: boolean, shiftKey?: boolean}} event
 * @param {{columns: number, total: number}} grid
 * @returns {number | null}
 */
export function targetIndex(index, event, { columns, total }) {
    if (total <= 0) return null;
    if (event.ctrlKey || event.metaKey || event.altKey) return null;
    const clamp = (/** @type {number} */ i) => Math.min(Math.max(0, i), total - 1);
    const { from, to } = pageBounds(pageOf(index), total);
    const key = event.key.length === 1 && !event.shiftKey ? event.key.toLowerCase() : event.key;
    switch (key) {
        case 'ArrowRight':
        case 'j':
            return clamp(index + 1);
        case 'ArrowLeft':
        case 'k':
            return clamp(index - 1);
        case 'ArrowDown':
            return index + columns < total ? index + columns : index;
        case 'ArrowUp':
            return index - columns >= 0 ? index - columns : index;
        case 'Home':
            return from;
        case 'End':
            return to - 1;
        case 'PageDown':
            return clamp(index + PAGE_SIZE);
        case 'PageUp':
            return clamp(index - PAGE_SIZE);
        default:
            return null;
    }
}

/**
 * Ouvre la planche-contact, ou dit pourquoi pas. Ouvrir une vignette déplace le
 * curseur de la liste : sans objet en mode édition ni dans un match, qui se
 * parcourt par ses coups.
 * @returns {boolean} vrai quand la planche s'est ouverte.
 */
export function openContactSheet() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return false;
    }
    const mode = get(statusBarModeStore);
    if (mode === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return false;
    }
    if (mode === 'MATCH') {
        setStatusBarMessage(tMsg('contactSheet.notInMatch'));
        return false;
    }
    if (get(positionsStore).length === 0) {
        setStatusBarMessage(tMsg('commands.noPositionsFound'));
        return false;
    }
    openModal(MODAL.CONTACT_SHEET);
    return true;
}
