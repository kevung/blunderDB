/**
 * metadataStatus.js — Ctrl-G: the analysis dates plus the MET / take-point /
 * gammon-value figures for the position on screen, printed in the status bar.
 *
 * The only consumer of the static score tables (metTable, takePoint*,
 * gammonValue*) outside App.svelte's table modals; it lives here so that
 * positionService does not carry seven table imports for one shortcut.
 */

import { get } from 'svelte/store';
import { positionsStore } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { currentPositionIndexStore, statusBarTextStore } from '../stores/uiStore.js';
import { tMsg, t } from '../i18n';
import { tableData as metTable } from '../stores/metTable';
import { takePoint2LiveTable } from '../stores/takePoint2LiveTable';
import { takePoint2LastTable } from '../stores/takePoint2LastTable';
import { gammonValue1Table } from '../stores/gammonValue1Table';
import { gammonValue2Table } from '../stores/gammonValue2Table';
import { gammonValue4Table } from '../stores/gammonValue4Table';
import { takePoint4LiveTable } from '../stores/takePoint4LiveTable';
import { takePoint4LastTable } from '../stores/takePoint4LastTable';

// Each table is indexed by (away score − offset) on both axes; a score outside
// the table reads as 'N/A'.
function lookup(table, score, rowOffset, colOffset, decimals) {
    const row = score[0] - rowOffset;
    const col = score[1] - colOffset;
    if (row < 0 || row >= table.length || col < 0 || col >= table[0].length) return 'N/A';
    return table[row][col].toFixed(decimals);
}

function formatDate(date) {
    const [year, month, day] = date.toLocaleDateString('sv-SE').split('-');
    const time = date.toLocaleTimeString('sv-SE', { hour: '2-digit', minute: '2-digit' });
    return `${year}/${month}/${day} ${time}`;
}

export function showDatesAndMetadata() {
    const analysis = get(analysisStore);
    const positionCount = get(positionsStore).length;
    const currentIndex = get(currentPositionIndexStore);
    const tr = get(t);

    if (!analysis || !analysis.creationDate || !analysis.lastModifiedDate) {
        statusBarTextStore.set(tMsg('statusBar.noDatabaseOpened'));
        return;
    }

    const creationDate = formatDate(new Date(analysis.creationDate));
    const lastModifiedDate = formatDate(new Date(analysis.lastModifiedDate));
    let statusText = tr('statusBar.createdModified', { created: creationDate, modified: lastModifiedDate });

    // peek: the shown position is in the window cache (the index effect
    // loaded it); a miss only happens before that load has landed.
    const current = positionCount > 0 ? positionsStore.peek(currentIndex) : undefined;
    if (!current) {
        statusText += ` | ${tr('statusBar.noPositionData')}`;
    } else {
        const { score, cube } = current;
        let metadata = `met: ${lookup(metTable, score, 1, 1, 1)}`;
        if (cube.value === 0) {
            metadata += ` | tp2_live: ${lookup(takePoint2LiveTable, score, 2, 2, 1)}`;
            metadata += ` | tp2_last: ${lookup(takePoint2LastTable, score, 2, 2, 1)}`;
            metadata += ` | gv1: ${lookup(gammonValue1Table, score, 2, 2, 2)}`;
            metadata += ` | gv2: ${lookup(gammonValue2Table, score, 3, 2, 2)}`;
        } else if (cube.value === 1) {
            metadata += ` | tp4_live: ${lookup(takePoint4LiveTable, score, 3, 3, 0)}`;
            metadata += ` | tp4_last: ${lookup(takePoint4LastTable, score, 3, 3, 0)}`;
            metadata += ` | gv2: ${lookup(gammonValue2Table, score, 3, 2, 2)}`;
            metadata += ` | gv4: ${lookup(gammonValue4Table, score, 5, 2, 2)}`;
        } else if (cube.value === 2) {
            metadata += ` | gv4: ${lookup(gammonValue4Table, score, 5, 2, 2)}`;
        }
        statusText += ` | ${metadata}`;
    }

    statusBarTextStore.set(statusText);
}
