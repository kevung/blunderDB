// The match list shared by the Matches and Tournaments panels: one page of
// rows at a time, filtered and ordered by SQL (ListMatches), the total beside
// it (CountMatches), and a row refreshed in place after an edit rather than
// the whole list reloaded.

import { writable, get } from 'svelte/store';
import { ListMatches, CountMatches, GetMatchByID } from '../../wailsjs/go/database/Database.js';
import { sortMatches } from '../utils/matchTable.js';
import { logger } from '../utils/logger.js';

export const MATCH_PAGE_SIZE = 500;

// Columns the database can order by, with the ascending / descending key of
// each (storage.MatchListOpts.Sort). `date` descends by default, as the list
// always has.
const SQL_SORT = {
    date: ['date_asc', 'date'],
    player1: ['player1', 'player1_desc'],
    player2: ['player2', 'player2_desc'],
    length: ['length_asc', 'length_desc'],
    tournament: ['tournament', 'tournament_desc']
};

/**
 * The SQL sort key of a column and direction; '' (the default order) when
 * nothing is sorted, `null` for a column the database cannot order by (PR, MWC
 * are computed badges).
 * @param {string | null | undefined} column
 * @param {string} direction
 * @returns {string | null}
 */
export function sqlSortKey(column, direction) {
    if (!column) return '';
    const keys = /** @type {Record<string, string[]>} */ (SQL_SORT)[column];
    if (!keys) return null;
    return direction === 'desc' ? keys[1] : keys[0];
}

// Fields an edit can change, copied from a fresh read of the row. PR and MWC
// stay: they come from the page badges, which GetMatchByID does not compute.
const ROW_FIELDS = ['player1_name', 'player2_name', 'match_date', 'tournament_id', 'tournament_name', 'event', 'comment'];

/**
 * @param {object} [deps]
 * @param {(opts: any) => Promise<any[]>} [deps.list]
 * @param {(opts: any) => Promise<number>} [deps.count]
 * @param {(id: number) => Promise<any>} [deps.getByID]
 * @param {number} [deps.pageSize]
 */
export function createMatchList({ list = ListMatches, count = CountMatches, getByID = GetMatchByID, pageSize = MATCH_PAGE_SIZE } = {}) {
    const store = writable({
        rows: /** @type {any[]} */ ([]),
        total: 0,
        text: '',
        column: /** @type {string | null} */ (null),
        direction: 'asc',
        loading: false,
        loaded: false
    });
    const { subscribe, update } = store;
    let generation = 0;

    function opts(/** @type {any} */ state, /** @type {number} */ offset, /** @type {number} */ limit) {
        return { Text: state.text, Sort: sqlSortKey(state.column, state.direction) ?? '', Limit: limit, Offset: offset };
    }

    /** Load the first page again, under the current filter and order. */
    async function reload() {
        const gen = ++generation;
        const state = get(store);
        // A column the database cannot order by is sorted here, over the whole
        // filtered list, since a page of it would order only itself.
        const clientSort = sqlSortKey(state.column, state.direction) === null;
        update((s) => ({ ...s, loading: true }));
        try {
            const [rows, total] = await Promise.all([list(opts(state, 0, clientSort ? 0 : pageSize)), count(opts(state, 0, 0))]);
            if (gen !== generation) return;
            const page = rows || [];
            update((s) => ({ ...s, rows: clientSort ? sortMatches(page, state.column, state.direction) : page, total: total ?? page.length, loading: false, loaded: true }));
        } catch (error) {
            logger.error('Error loading matches:', error);
            if (gen === generation) update((s) => ({ ...s, rows: [], total: 0, loading: false, loaded: true }));
        }
    }

    /** Append the next page, when there is one. */
    async function loadMore() {
        const state = get(store);
        if (state.loading || state.rows.length >= state.total) return;
        const gen = generation;
        update((s) => ({ ...s, loading: true }));
        try {
            const page = (await list(opts(state, state.rows.length, pageSize))) || [];
            if (gen !== generation) return;
            update((s) => ({ ...s, rows: [...s.rows, ...page], total: page.length === 0 ? s.rows.length : s.total, loading: false }));
        } catch (error) {
            logger.error('Error loading more matches:', error);
            if (gen === generation) update((s) => ({ ...s, loading: false }));
        }
    }

    function setText(/** @type {string} */ text) {
        if (get(store).text === text) return Promise.resolve();
        update((s) => ({ ...s, text }));
        return reload();
    }

    function setSort(/** @type {string|null} */ column, /** @type {string} */ direction) {
        const s = get(store);
        if (s.column === column && s.direction === direction) return Promise.resolve();
        update((st) => ({ ...st, column, direction }));
        return reload();
    }

    /** Re-read one row and replace it in place; the others are untouched. */
    async function refreshRow(/** @type {number} */ id) {
        try {
            const fresh = await getByID(id);
            if (!fresh) return;
            const patch = Object.fromEntries(ROW_FIELDS.filter((f) => f in fresh).map((f) => [f, fresh[f]]));
            patchRow(id, patch);
        } catch (error) {
            logger.error('Error refreshing match row:', error);
        }
    }

    function patchRow(/** @type {number} */ id, /** @type {Record<string, any>} */ patch) {
        update((s) => ({ ...s, rows: s.rows.map((r) => (r.id === id ? { ...r, ...patch } : r)) }));
    }

    return { subscribe, reload, loadMore, setText, setSort, refreshRow, patchRow };
}

export const matchListStore = createMatchList();
