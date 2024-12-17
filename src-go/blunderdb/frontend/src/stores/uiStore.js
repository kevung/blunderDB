import { writable } from 'svelte/store';

export const statusBarTextStore = writable('');
export const statusBarModeStore = writable('NORMAL');

export const commandTextStore = writable('');

export const commentTextStore = writable('');

export const analysisDataStore = writable('This is where your analysis data will be displayed.');

export const currentPositionIndexStore = writable(0); // Add current position index store
