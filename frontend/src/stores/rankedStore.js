import { writable } from 'svelte/store';

// La distance de chaque voisine d'un classement `like` (ADR-0043).
//
// Elle vit dans un store à part et non sur la position, parce qu'elle n'est pas
// une propriété de la position : c'est le résultat d'une COMPARAISON avec une
// autre, et la même position a une distance différente selon ce qu'on lui
// compare. Une liste ordinaire laisse ce store vide.
//
// Clé : l'identifiant de la position. Valeur : la distance en pions-pas.
/** @type {import('svelte/store').Writable<Map<number, number>>} */
export const rankedDistancesStore = writable(new Map());

// La cible du classement en cours, pour que la voisine puisse dire de QUOI
// elle est proche. Zéro quand la liste parcourue n'est pas un classement.
export const rankedTargetStore = writable(0);
