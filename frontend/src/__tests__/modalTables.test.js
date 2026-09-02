/**
 * MODAL_TABLES maps a modal id to the reference tables a DataTableModal shows.
 * These checks keep the table geometry honest: every key is a real MODAL id,
 * every row has `colCount` cells, and multi-table modals title their sections
 * (DataTableModal keys its sections by title).
 */
import { describe, test, expect } from 'vitest';
import { MODAL_TABLES } from '../components/modalTables.js';
import { MODAL } from '../stores/uiStore.js';

describe('MODAL_TABLES', () => {
    const ids = Object.values(MODAL);

    test('keys are MODAL ids', () => {
        for (const key of Object.keys(MODAL_TABLES)) expect(ids).toContain(key);
    });

    test('covers the nine reference-table modals', () => {
        expect(Object.keys(MODAL_TABLES).sort()).toEqual(
            [
                MODAL.TAKE_POINT_2_LAST,
                MODAL.TAKE_POINT_2_LIVE,
                MODAL.TAKE_POINT_4_LAST,
                MODAL.TAKE_POINT_4_LIVE,
                MODAL.GAMMON_VALUE_1,
                MODAL.GAMMON_VALUE_2,
                MODAL.GAMMON_VALUE_4,
                MODAL.TAKE_POINT_2,
                MODAL.TAKE_POINT_4
            ].sort()
        );
    });

    test.each(Object.entries(MODAL_TABLES))('%s: rows match colCount', (_id, tables) => {
        for (const t of tables) {
            expect(Array.isArray(t.data)).toBe(true);
            expect(t.data.length).toBeGreaterThan(0);
            for (const row of t.data) expect(row.length).toBe(t.colCount);
            expect(t.precision).toBeGreaterThanOrEqual(0);
            expect(t.colOffset).toBeGreaterThanOrEqual(2);
            expect(t.rowOffset).toBeGreaterThanOrEqual(2);
        }
        if (tables.length > 1) {
            const titles = tables.map((t) => t.title);
            expect(new Set(titles).size).toBe(titles.length);
            titles.forEach((title) => expect(title).toBeTruthy());
        }
    });
});
