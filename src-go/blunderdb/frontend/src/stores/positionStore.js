import { writable } from 'svelte/store';

export const importPositionPathStore = writable('');
export const pastePositionTextStore = writable('');
export const currentPositionStore = writable('10');
export const listPositionStore = writable('5432');
