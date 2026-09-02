/**
 * positionService.showDatesAndMetadata.test.js
 *
 * Ctrl-G : la ligne de statut lit chaque table statique à (score − décalage) ;
 * un score hors table donne « N/A ». Les valeurs attendues sont lues dans les
 * tables elles-mêmes pour que le test suive leurs données.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

import { statusBarTextStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { tableData as metTable } from '../stores/metTable';
import { takePoint2LiveTable } from '../stores/takePoint2LiveTable';
import { gammonValue2Table } from '../stores/gammonValue2Table';
import { takePoint4LastTable } from '../stores/takePoint4LastTable';
import { gammonValue4Table } from '../stores/gammonValue4Table';
import { showDatesAndMetadata } from '../services/positionService.js';
import { showDatesAndMetadata as direct } from '../services/metadataStatus.js';

function setPosition(score, cubeValue) {
    positionsStore.set([{ id: 1, score, cube: { owner: -1, value: cubeValue } }]);
    currentPositionIndexStore.set(0);
}

beforeEach(() => {
    statusBarTextStore.set('');
    analysisStore.set({ creationDate: '2026-09-01T10:00:00Z', lastModifiedDate: '2026-09-02T11:30:00Z' });
});

describe('showDatesAndMetadata', () => {
    test('est ré-exportée par positionService', () => {
        expect(showDatesAndMetadata).toBe(direct);
    });

    test('videau centré : MET, take points à 2 et gammon values 1/2', () => {
        setPosition([5, 5], 0);
        showDatesAndMetadata();
        const text = get(statusBarTextStore);
        expect(text).toContain(`met: ${metTable[4][4].toFixed(1)}`);
        expect(text).toContain(`tp2_live: ${takePoint2LiveTable[3][3].toFixed(1)}`);
        expect(text).toContain(`gv2: ${gammonValue2Table[2][3].toFixed(2)}`);
        expect(text).not.toContain('tp4_');
    });

    test('videau à 2 : take points à 4 et gammon values 2/4', () => {
        setPosition([7, 6], 1);
        showDatesAndMetadata();
        const text = get(statusBarTextStore);
        expect(text).toContain(`tp4_last: ${takePoint4LastTable[4][3].toFixed(0)}`);
        expect(text).toContain(`gv4: ${gammonValue4Table[2][4].toFixed(2)}`);
        expect(text).not.toContain('tp2_');
    });

    test('score hors table : N/A, sans lever', () => {
        setPosition([1, 1], 0);
        showDatesAndMetadata();
        const text = get(statusBarTextStore);
        expect(text).toContain(`met: ${metTable[0][0].toFixed(1)}`);
        expect(text).toContain('tp2_live: N/A');
        expect(text).toContain('gv1: N/A');
    });

    test('sans position courante : mention explicite après les dates', () => {
        positionsStore.set([]);
        currentPositionIndexStore.set(-1);
        showDatesAndMetadata();
        expect(get(statusBarTextStore)).toMatch(/2026\/09\/01 .* \| /);
    });

    test('sans analyse datée : message « pas de base »', () => {
        analysisStore.set({});
        setPosition([5, 5], 0);
        showDatesAndMetadata();
        expect(get(statusBarTextStore)).not.toContain('met:');
    });
});
