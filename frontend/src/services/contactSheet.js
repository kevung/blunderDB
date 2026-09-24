// La planche-contact de la liste parcourue (#287, fiche I.31).
//
// Une recherche rend une LISTE, et le plateau n'en montre qu'une position à
// la fois : pour voir ce qu'une recherche a ramené, il fallait la parcourir
// d'un bout à l'autre. La planche-contact montre la liste en grille de
// mini-plateaux, et une vignette choisie ouvre sa position sur le plateau.
//
// Le dessin est celui de diagramService.renderPositionSVG (#278/#279) : les
// mêmes fonctions de scène que le plateau, donc la même image, palette de
// l'utilisateur comprise. Aucun second dessinateur.
//
// Ce module ne tient que ce qui se teste sans composant : le découpage en
// pages, le déplacement au clavier dans la grille, et la garde d'ouverture.
// Le rendu, qui coûte, est dans ContactSheetModal.svelte.

import { get } from 'svelte/store';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { statusBarModeStore, openModal, MODAL } from '../stores/uiStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { tMsg } from '../i18n';

/**
 * Les vignettes d'une page. Vingt-quatre : un nombre qui se range en 2, 3, 4,
 * 6 ou 8 colonnes sans ligne boiteuse, et qu'on dessine sans attente
 * perceptible. Une liste de cinquante mille positions ne dessine jamais que
 * sa page — c'est toute la raison de la pagination.
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
 * Le nombre de colonnes qui tiennent dans une largeur. Une largeur inconnue
 * (zéro : pas encore mesurée, ou un test sans mise en page) donne quatre.
 * @param {number} width
 * @param {number} tileMin largeur minimale d'une vignette
 * @param {number} gap espace entre deux vignettes
 */
export function columnsFor(width, tileMin, gap) {
    if (!(width > 0)) return 4;
    return Math.max(1, Math.floor((width + gap) / (tileMin + gap)));
}

/**
 * La vignette que vise une touche, à partir de `index`, ou null quand la
 * touche n'est pas une touche de la grille. Les flèches se déplacent dans la
 * grille ; J et K sont « suivante » et « précédente », comme sur le plateau ;
 * Début et Fin vont aux bouts de la page ; Page préc./suiv. changent de page.
 * Le résultat est toujours dans la liste : une flèche au bord ne fait rien.
 *
 * Les lettres passent par event.key (la convention du clavier : J est la
 * touche marquée J, en AZERTY comme en QWERTY).
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
 * Ouvre la planche-contact, ou dit pourquoi elle ne s'ouvre pas. Elle montre
 * la liste parcourue — résultats d'une recherche, bibliothèque, collection —
 * et ouvrir une vignette déplace le curseur de cette liste : rien de cela n'a
 * de sens en mode édition (on quitterait la position en cours d'édition) ni
 * dans un match, qui se parcourt par ses coups.
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
