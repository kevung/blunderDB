/**
 * CubeVerdictTable.challengeMask.test.js
 *
 * Défi must leak nothing from the cube block (ADR-0020 rule 7). This panel has
 * leaked twice already — the gammonNet cube table stayed in clear while the
 * other zones were masked (ADR-0017's opening context), and CandidateMovesTable
 * was mounted with no mask at all for the whole life of the feature (ADR-0018
 * rule 6). Twice is a pattern, so the property is a test rather than an
 * intention.
 *
 * The subtle half is the emphasis. Once the best option is marked by weight and
 * colour instead of by position (rule 2), that marking becomes a second carrier
 * of the verdict: masking every figure while leaving the bold row would let the
 * exercise be solved by looking for it.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import CubeVerdictTable from '../components/CubeVerdictTable.svelte';
import { cubeDecision } from '../utils/cubeDecision.js';

const money = { cubeless: 0.4, no_double: 0.55, double_take: 0.71, double_pass: 1.0, verdict: 'double_take' };

function masked() {
    return render(CubeVerdictTable, {
        props: {
            decision: cubeDecision({ isRace: true, race: { money } }),
            showInfo: false,
            masked: true
        }
    });
}

describe('CubeVerdictTable under Défi', () => {
    afterEach(cleanup);

    test('no equity and no error survives the mask', () => {
        const { container } = masked();
        const text = container.textContent;
        for (const figure of ['0.55', '0.71', '1.000', '0.550', '0.710', '0.16', '0.29']) {
            expect(text).not.toContain(figure);
        }
        // Every value cell is the placeholder, not a formatted number.
        const values = [...container.querySelectorAll('td.equity, td.error')];
        expect(values).toHaveLength(6);
        expect(values.every((td) => td.textContent.trim() === '···')).toBe(true);
    });

    test('the verdict is masked too, not merely the figures', () => {
        const { container } = masked();
        expect(container.querySelector('td.verdict').textContent.trim()).toBe('···');
    });

    test('the best-option emphasis is suppressed — it would give the answer away', () => {
        const { container } = masked();
        expect(container.querySelectorAll('tr.best')).toHaveLength(0);
    });

    test('the structure itself never moves: three option rows, masked or not', () => {
        const { container } = masked();
        expect(container.querySelectorAll('tbody tr')).toHaveLength(4); // 3 options + the verdict row
        cleanup();

        const shown = render(CubeVerdictTable, {
            props: { decision: cubeDecision({ isRace: true, race: { money } }), showInfo: false }
        });
        expect(shown.container.querySelectorAll('tbody tr')).toHaveLength(4);
        expect(shown.container.querySelectorAll('tr.best')).toHaveLength(1);
        expect(shown.container.textContent).toContain('+0.710');
    });
});
