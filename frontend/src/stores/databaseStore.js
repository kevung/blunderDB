import { writable, derived } from 'svelte/store';

export const databasePathStore = writable('');

export const databaseLoadedStore = derived(databasePathStore, ($databasePathStore) => $databasePathStore !== '');

/** Whether the open library was opened read-only (another instance holds its write lock). */
export const databaseReadOnlyStore = writable(false);

/**
 * Bumped whenever the active library becomes another one: a library group brought forward or
 * opened. databaseLoadedStore stays true from one library to the next, so what a panel reads
 * from the library is read again on this counter as well.
 */
export const libraryChangedStore = writable(0);

export function libraryChanged() {
    libraryChangedStore.update((n) => n + 1);
}

/**
 * The window title for a library: its file name, and the read-only suffix when it applies.
 * @param {string} path
 * @param {boolean} readOnly
 * @param {string} readOnlySuffix
 */
export function libraryTitle(path, readOnly, readOnlySuffix) {
    const file = (path || '').split(/[\\/]/).pop();
    if (!file) return 'blunderDB';
    return readOnly ? `blunderDB - ${file} ${readOnlySuffix}` : `blunderDB - ${file}`;
}

// ...existing code...
