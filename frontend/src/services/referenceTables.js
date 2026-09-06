/**
 * Les tables de référence du score — points de prise et valeurs du gammon —
 * décrites UNE fois, avec leur géométrie.
 *
 * Elles étaient jusqu'ici de simples tableaux que `modalTables.js` habillait
 * de ses décalages au moment de les afficher. L'exercice Scores les lit aussi
 * (ADR-0040 règle 4), et la carte Anki les lira demain (ADR-0042) : une copie,
 * plusieurs lecteurs. Le décalage n'est donc plus un détail d'affichage, c'est
 * un fait de la table — tp4 commence à 3 away, gv2 à 3, gv4 à 5 — et c'est ce
 * fait qui décide quelles cases une fiche de score porte.
 *
 * Convention de lecture, commune aux sept tables : la LIGNE est l'away du
 * joueur dont on parle (celui qui prend, celui qui gagne le gammon), la
 * COLONNE l'away de son adversaire.
 */
import { takePoint2LastTable } from '../stores/takePoint2LastTable';
import { takePoint2LiveTable } from '../stores/takePoint2LiveTable';
import { takePoint4LastTable } from '../stores/takePoint4LastTable';
import { takePoint4LiveTable } from '../stores/takePoint4LiveTable';
import { gammonValue1Table } from '../stores/gammonValue1Table';
import { gammonValue2Table } from '../stores/gammonValue2Table';
import { gammonValue4Table } from '../stores/gammonValue4Table';

/**
 * @typedef {object} ReferenceTable
 * @property {number[][]} data les valeurs, ligne par ligne
 * @property {number} precision le nombre de décimales à l'affichage
 * @property {number} colCount le nombre de colonnes
 * @property {number} colOffset l'away de la première colonne
 * @property {number} rowOffset l'away de la première ligne
 */

/** @type {Readonly<Record<string, ReferenceTable>>} */
export const REFERENCE_TABLES = Object.freeze({
    'tp2.live': { data: takePoint2LiveTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 },
    'tp2.last': { data: takePoint2LastTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 },
    'tp4.live': { data: takePoint4LiveTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 },
    'tp4.last': { data: takePoint4LastTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 },
    gv1: { data: gammonValue1Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 2 },
    gv2: { data: gammonValue2Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 3 },
    gv4: { data: gammonValue4Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 5 }
});

/**
 * La valeur d'une table pour un couple d'away, ou `null` quand la table ne la
 * définit pas. `null` et non 0 : « gv4 n'a pas d'objet à 3 away » n'est pas
 * « gv4 vaut zéro à 3 away », et c'est cette distinction qui fait qu'une fiche
 * de score n'a pas toujours ses quatorze cases.
 *
 * @param {string} id une clé de REFERENCE_TABLES
 * @param {number} rowAway l'away du joueur dont on parle
 * @param {number} colAway l'away de son adversaire
 * @returns {number|null}
 */
export function referenceValue(id, rowAway, colAway) {
    const table = REFERENCE_TABLES[id];
    if (!table) return null;
    const row = rowAway - table.rowOffset;
    const col = colAway - table.colOffset;
    if (row < 0 || col < 0 || row >= table.data.length || col >= table.data[row].length) return null;
    const value = table.data[row][col];
    return Number.isFinite(value) ? value : null;
}
