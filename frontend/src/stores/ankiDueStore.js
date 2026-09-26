import { writable } from 'svelte/store';

// Le badge de cartes dues sur l'onglet Anki, tous paquets confondus : le chiffre est la raison
// d'ouvrir l'onglet, il doit donc se lire sans l'ouvrir.

/** @type {import('svelte/store').Writable<number>} */
export const ankiDueStore = writable(0);

/** Rafraîchit le compte ; zéro à défaut — un ancien chiffre est pire que pas de badge. */
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
