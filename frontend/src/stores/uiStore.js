import { writable } from 'svelte/store';

export const statusBarTextStore = writable('');
export const statusBarModeStore = writable('NORMAL');

export const commandTextStore = writable('');

export const commentTextStore = writable('');

export const analysisDataStore = writable('This is where your analysis data will be displayed.');

export const currentPositionIndexStore = writable(0); // Add current position index store

export const showSearchModalStore = writable(false); // Add store for search modal visibility

export const showMetModalStore = writable(false); // Add store for MET modal visibility

export const showTakePoint2LastModalStore = writable(false); // Add store for TakePoint2Last modal visibility
