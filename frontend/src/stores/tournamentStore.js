import { writable } from 'svelte/store';

/** @typedef {import('../../wailsjs/go/models').domain.Tournament} Tournament */

export const tournamentsStore = writable(/** @type {Tournament[]} */ ([]));
export const selectedTournamentStore = writable(/** @type {Tournament | null} */ (null));
export const tournamentMatchesStore = writable(/** @type {import('../../wailsjs/go/models').domain.Match[]} */ ([]));
