import { describe, test, expect, vi } from 'vitest';

// Wails bindings are only invoked at call-time, but mock them so importing the
// module graph does not touch window['go'].
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveComment: vi.fn(),
    Migrate_1_0_0_to_1_1_0: vi.fn(),
    ClearCommandHistory: vi.fn(),
    SaveSearchHistory: vi.fn(),
    Migrate_1_1_0_to_1_2_0: vi.fn(),
    Migrate_1_2_0_to_1_3_0: vi.fn()
}));

import { parseFilters } from '../commandProcessor.js';

describe('parseFilters — exclusion structure token "x"', () => {
    test('recognizes the x token', () => {
        const parsed = parseFilters(['x'], 's x');
        expect(parsed.excludeStructure).toBe(true);
    });

    test('absent when no x token', () => {
        const parsed = parseFilters(['p>5'], 's p>5');
        expect(parsed.excludeStructure).toBe(false);
    });

    test('coexists with other filters', () => {
        const parsed = parseFilters(['p>5', 'w>0.55', 'x'], 's p>5 w>0.55 x');
        expect(parsed.excludeStructure).toBe(true);
        expect(parsed.pipCountFilter).toBe('p>5');
        expect(parsed.winRateFilter).toBe('w>0.55');
    });
});
