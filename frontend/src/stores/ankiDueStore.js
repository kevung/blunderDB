import { writable } from 'svelte/store';

// Le badge de cartes dues sur l'onglet Anki (#287, fiche I.31).
//
// Une révision espacée ne sert que si l'on sait qu'il y a quelque chose à
// réviser. Le compte était derrière un onglet qu'il fallait ouvrir pour le
// voir — c'est-à-dire au mauvais endroit : ce chiffre est la RAISON d'ouvrir
// l'onglet, pas ce qu'on y trouve.
//
// Le compte porte sur TOUS les paquets. Un badge par paquet aurait demandé
// d'ouvrir l'onglet pour être lu, ce qui est le problème qu'on résout.

/** @type {import('svelte/store').Writable<number>} */
export const ankiDueStore = writable(0);

/**
 * Rafraîchit le compte des cartes dues, tous paquets confondus. Zéro à défaut :
 * un badge qui reste sur un ancien chiffre est pire que pas de badge.
 */
export async function refreshAnkiDue() {
    const { get } = await import('svelte/store');
    const { databasePathStore } = await import('./databaseStore.js');
    if (!get(databasePathStore)) {
        ankiDueStore.set(0);
        return;
    }
    try {
        const { GetAnkiForecast } = await import('../../wailsjs/go/database/Database.js');
        // Le jour 0 de la prévision absorbe le retard : c'est exactement
        // « ce qui est à réviser maintenant ».
        const forecast = (await GetAnkiForecast(0, 1)) || [];
        ankiDueStore.set(forecast.length > 0 ? forecast[0].due || 0 : 0);
    } catch {
        ankiDueStore.set(0);
    }
}
