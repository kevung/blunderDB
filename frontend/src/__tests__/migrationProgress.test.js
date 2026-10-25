import { test, expect, vi } from 'vitest';

vi.mock('../../wailsjs/go/gui/App.js', () => ({ CancelLibraryOpen: vi.fn(async () => {}) }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ EventsOn: vi.fn(() => () => {}) }));

import { migrationPhaseLabel } from '../services/migrationProgress.js';

test('a migration phase is named in words, a table conversion by its table, an unknown one by its id', () => {
    expect(migrationPhaseLabel('rederive_columns')).toBe('derived columns');
    expect(migrationPhaseLabel('position_2_31')).toBe('positions');
    expect(migrationPhaseLabel('some_future_phase')).toBe('some_future_phase');
});
