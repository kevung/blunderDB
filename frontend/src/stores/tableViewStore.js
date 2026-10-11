// What a table with a view id remembers (ADR-0090). Its sort lives in Config.yaml
// (`table_sorts`), like every layout preference, since the webview's storage is not
// guaranteed to survive a restart. Its column filters live for the session only: a
// filter kept across launches would hide rows with nothing on screen saying why.

import { GetTableSorts, SaveTableSort } from '../../wailsjs/go/main/Config.js';
import { logger } from '../utils/logger.js';

/** @typedef {{ column: string | null, direction: 'asc' | 'desc' }} TableSort */

/** @type {Record<string, TableSort>} */
let sorts = {};
/** @type {Promise<Record<string, TableSort>> | null} */
let loading = null;
/** @type {Map<string, Record<string, string>>} */
const filters = new Map();

/**
 * The remembered sorts, read from Config once.
 * @returns {Promise<Record<string, TableSort>>}
 */
export function loadTableSorts() {
    loading ??= Promise.resolve()
        .then(() => GetTableSorts())
        .then((/** @type {any} */ stored) => {
            for (const [id, s] of Object.entries(stored ?? {})) {
                if (s?.column) sorts[id] = { column: s.column, direction: s.direction === 'desc' ? 'desc' : 'asc' };
            }
            return sorts;
        })
        .catch((err) => {
            logger.error('Failed to load table sorts:', err);
            return sorts;
        });
    return loading;
}

/**
 * Remembers the sort of one table; a null column forgets it.
 * @param {string} viewId
 * @param {TableSort} sort
 */
export function rememberTableSort(viewId, sort) {
    if (!viewId) return;
    if (sort?.column) sorts[viewId] = { column: sort.column, direction: sort.direction };
    else delete sorts[viewId];
    Promise.resolve()
        .then(() => SaveTableSort(viewId, sort?.column ?? '', sort?.direction ?? 'asc'))
        .catch((err) => logger.error('Failed to save table sort:', err));
}

/**
 * The column filters a table had earlier in this session.
 * @param {string} viewId
 * @returns {Record<string, string>}
 */
export function sessionFilters(viewId) {
    return { ...(filters.get(viewId) ?? {}) };
}

/**
 * @param {string} viewId
 * @param {Record<string, string>} value
 */
export function rememberSessionFilters(viewId, value) {
    const kept = Object.fromEntries(Object.entries(value).filter(([, v]) => v && v.trim()));
    if (Object.keys(kept).length) filters.set(viewId, kept);
    else filters.delete(viewId);
}

/** Forgets everything; for tests. */
export function resetTableViews() {
    sorts = {};
    loading = null;
    filters.clear();
}
