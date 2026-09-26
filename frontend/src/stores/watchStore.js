import { writable } from 'svelte/store';

// `watchStatusStore` : ce que le Go dit de la surveillance en cours, jamais ce que la
// configuration demande — une surveillance qui n'a pas pu démarrer ne s'affiche pas active.
/** @type {import('svelte/store').Writable<{running: boolean, folder: string, intervalSeconds: number}>} */
export const watchStatusStore = writable({ running: false, folder: '', intervalSeconds: 0 });

// `watchImportNoticeStore` : la notification NON BLOQUANTE (barre de statut, pas de modale) d'un
// import venu du dossier surveillé, qui arrive pendant que l'utilisateur étudie ; null = rien.
/** @type {import('svelte/store').Writable<{succeeded: number, skipped: number, failed: number} | null>} */
export const watchImportNoticeStore = writable(null);
