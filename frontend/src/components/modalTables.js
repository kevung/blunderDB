/**
 * The static reference tables shown in a DataTableModal, keyed by the modal
 * id that opens them (uiStore MODAL). Each value is the `tables` prop of one
 * <DataTableModal>: one section per entry, `title` only when several share a
 * modal. ModalHost renders one modal per key, so adding a table here is the
 * whole job — no template to touch.
 *
 * `precision` is the decimal count of the cells; `colOffset` / `rowOffset`
 * are the labels of the first column / row (the tables start at 2-away or
 * 3-away, never at 1).
 */
import { MODAL } from '../stores/uiStore.js';
import { takePoint2LastTable } from '../stores/takePoint2LastTable';
import { takePoint2LiveTable } from '../stores/takePoint2LiveTable';
import { takePoint4LastTable } from '../stores/takePoint4LastTable';
import { takePoint4LiveTable } from '../stores/takePoint4LiveTable';
import { gammonValue1Table } from '../stores/gammonValue1Table';
import { gammonValue2Table } from '../stores/gammonValue2Table';
import { gammonValue4Table } from '../stores/gammonValue4Table';

const takePoint2Live = { data: takePoint2LiveTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 };
const takePoint2Last = { data: takePoint2LastTable, precision: 1, colCount: 8, colOffset: 2, rowOffset: 2 };
const takePoint4Live = { data: takePoint4LiveTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 };
const takePoint4Last = { data: takePoint4LastTable, precision: 0, colCount: 7, colOffset: 3, rowOffset: 3 };

export const MODAL_TABLES = Object.freeze({
    [MODAL.TAKE_POINT_2_LAST]: [takePoint2Last],
    [MODAL.TAKE_POINT_2_LIVE]: [takePoint2Live],
    [MODAL.TAKE_POINT_4_LAST]: [takePoint4Last],
    [MODAL.TAKE_POINT_4_LIVE]: [takePoint4Live],
    [MODAL.GAMMON_VALUE_1]: [{ data: gammonValue1Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 2 }],
    [MODAL.GAMMON_VALUE_2]: [{ data: gammonValue2Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 3 }],
    [MODAL.GAMMON_VALUE_4]: [{ data: gammonValue4Table, precision: 2, colCount: 8, colOffset: 2, rowOffset: 5 }],
    [MODAL.TAKE_POINT_2]: [
        { title: 'Long Races', ...takePoint2Live },
        { title: 'Last Roll', ...takePoint2Last }
    ],
    [MODAL.TAKE_POINT_4]: [
        { title: 'Long Races', ...takePoint4Live },
        { title: 'Last Roll', ...takePoint4Last }
    ]
});
