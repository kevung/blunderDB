import { writable } from 'svelte/store';

// Le dossier surveillé (#258, fiche I.2).
//
// `watchStatusStore` est ce que le Go dit d'une surveillance en cours — jamais
// ce que la configuration demande : une surveillance qui n'a pas pu démarrer
// (dossier disparu, partage démonté) ne doit pas s'afficher comme active.
/** @type {import('svelte/store').Writable<{running: boolean, folder: string, intervalSeconds: number}>} */
export const watchStatusStore = writable({ running: false, folder: '', intervalSeconds: 0 });

// `watchImportNoticeStore` porte la notification NON BLOQUANTE d'un import
// venu du dossier surveillé : un bandeau dans la barre de statut, pas une
// fenêtre modale. L'utilisateur était en train d'étudier une position quand
// ses matchs sont arrivés ; lui reprendre l'écran serait le pire moment.
//
// null = rien à signaler.
/** @type {import('svelte/store').Writable<{succeeded: number, skipped: number, failed: number} | null>} */
export const watchImportNoticeStore = writable(null);
