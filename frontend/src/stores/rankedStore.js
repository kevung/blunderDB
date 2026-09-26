import { writable } from 'svelte/store';

// La distance de chaque voisine d'un classement `like` (ADR-0043), en pions-pas, par id de
// position. À part car c'est le résultat d'une comparaison, pas une propriété de la position ;
// vide pour une liste ordinaire.
/** @type {import('svelte/store').Writable<Map<number, number>>} */
export const rankedDistancesStore = writable(new Map());

// La cible du classement en cours, pour que la voisine puisse dire de QUOI
// elle est proche. Zéro quand la liste parcourue n'est pas un classement.
export const rankedTargetStore = writable(0);
