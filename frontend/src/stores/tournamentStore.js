import { writable } from 'svelte/store';

export const tournamentsStore = writable([]);
export const selectedTournamentStore = writable(null);
export const tournamentMatchesStore = writable([]);
