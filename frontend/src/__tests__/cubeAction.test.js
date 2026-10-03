import { describe, test, expect } from 'vitest';
import { normalizeCubeAction, isResponseCubeAction } from '../utils/cubeAction.js';

describe('isResponseCubeAction', () => {
    test('pure take/pass responses are responses', () => {
        for (const a of ['Take', 'Pass', 'take', 'Drop', 'dt', 'dp']) {
            expect(isResponseCubeAction(a)).toBe(true);
        }
    });

    test('doubling decisions (incl. combined) and no-double are not responses', () => {
        for (const a of ['Double', 'Double/Take', 'Double/Pass', 'No Double', 'NoDouble', 'Redouble', '', undefined]) {
            expect(isResponseCubeAction(a)).toBe(false);
        }
    });
});

describe('normalizeCubeAction', () => {
    test('maps combined and standalone actions to canonical row parts', () => {
        expect(normalizeCubeAction('Double/Take')).toEqual(['double', 'take']);
        expect(normalizeCubeAction('Double/Pass')).toEqual(['double', 'pass']);
        expect(normalizeCubeAction('Take')).toEqual(['double', 'take']);
        expect(normalizeCubeAction('Pass')).toEqual(['double', 'pass']);
        expect(normalizeCubeAction('No Double')).toEqual(['nodouble']);
        expect(normalizeCubeAction('Double')).toEqual(['double']);
    });
});

// The same labels feed engine.CanonicalCubeAction (analysiscodec_test.go): the two tables agree
// because both read testdata/cube_action_labels.json, labels the importers really write.
import labels from '../../../testdata/cube_action_labels.json';
import { test as t2, expect as e2 } from 'vitest';
import { normalizeCubeAction as norm2 } from '../utils/cubeAction.js';

t2.each(labels)('normalizeCubeAction reads "$label" as $parts', ({ label, parts }) => {
    e2(norm2(label)).toEqual(parts);
});
