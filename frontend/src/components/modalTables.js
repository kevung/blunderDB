/**
 * The static reference tables shown in a DataTableModal, keyed by the modal
 * id that opens them (uiStore MODAL). Each value is the `tables` prop of one
 * <DataTableModal>: one section per entry, `title` only when several share a
 * modal. ModalHost renders one modal per key, so adding a table here is the
 * whole job — no template to touch.
 *
 * The tables themselves — their values, their `precision`, and the away their
 * first row and column stand for — are declared once in
 * services/referenceTables.js, which the Scores exercise reads too (ADR-0040
 * règle 4). This file only says which of them a modal shows, and under what
 * title.
 */
import { MODAL } from '../stores/uiStore.js';
import { REFERENCE_TABLES } from '../services/referenceTables.js';

const takePoint2Live = REFERENCE_TABLES['tp2.live'];
const takePoint2Last = REFERENCE_TABLES['tp2.last'];
const takePoint4Live = REFERENCE_TABLES['tp4.live'];
const takePoint4Last = REFERENCE_TABLES['tp4.last'];

export const MODAL_TABLES = Object.freeze({
    [MODAL.TAKE_POINT_2_LAST]: [takePoint2Last],
    [MODAL.TAKE_POINT_2_LIVE]: [takePoint2Live],
    [MODAL.TAKE_POINT_4_LAST]: [takePoint4Last],
    [MODAL.TAKE_POINT_4_LIVE]: [takePoint4Live],
    [MODAL.GAMMON_VALUE_1]: [REFERENCE_TABLES.gv1],
    [MODAL.GAMMON_VALUE_2]: [REFERENCE_TABLES.gv2],
    [MODAL.GAMMON_VALUE_4]: [REFERENCE_TABLES.gv4],
    [MODAL.TAKE_POINT_2]: [
        { title: 'Long Races', ...takePoint2Live },
        { title: 'Last Roll', ...takePoint2Last }
    ],
    [MODAL.TAKE_POINT_4]: [
        { title: 'Long Races', ...takePoint4Live },
        { title: 'Last Roll', ...takePoint4Last }
    ]
});
