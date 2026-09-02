/**
 * positionService.duplicate.test.js
 *
 * Éditer une position jusqu'à la rendre identique à une autre est refusé par le
 * backend (storage.DuplicatePositionError, « this position already exists (id N) »).
 * Le service lit le numéro dans ce message pour afficher un texte traduit qui
 * nomme la position existante ; tout autre message reste une erreur générique.
 */

import { describe, test, expect, vi } from 'vitest';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadAllPositions: vi.fn(() => Promise.resolve([])),
    DeletePosition: vi.fn(),
    DeleteAnalysis: vi.fn(),
    UpdatePosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadPositionsByFilters: vi.fn(() => Promise.resolve([])),
    ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
    SaveLastVisitedPosition: vi.fn(),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    SaveEditPosition: vi.fn(),
    SaveExcludePosition: vi.fn(),
    SaveFilter: vi.fn(),
    LoadComment: vi.fn(() => Promise.resolve(''))
}));

vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));

import { duplicatePositionId } from '../services/positionService.js';

describe('duplicatePositionId', () => {
    test('reads the existing id out of the backend refusal', () => {
        expect(duplicatePositionId(new Error('sqlite: update position: this position already exists (id 42)'))).toBe(42);
        expect(duplicatePositionId('this position already exists (id 7)')).toBe(7);
    });

    test('is null for any other error', () => {
        expect(duplicatePositionId(new Error('sqlite: update position: database is locked'))).toBeNull();
        expect(duplicatePositionId('UNIQUE constraint failed: position.zobrist_hash')).toBeNull();
        expect(duplicatePositionId(undefined)).toBeNull();
    });
});
