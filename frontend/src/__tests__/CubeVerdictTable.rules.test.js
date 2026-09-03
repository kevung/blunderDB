/**
 * CubeVerdictTable.rules.test.js
 *
 * #190/C.3 point 5: Jacoby and Beaver are stored on every position
 * (domain.Position.HasJacoby/HasBeaver) but were never shown next to the
 * verdict they change the value of. This pins the badge down: present only
 * when the flag is true, absent otherwise, and never a third phantom badge
 * for "cube max" — the plan's own premise for a third stored flag does not
 * hold (only these two exist in the schema), so there is nothing to badge
 * for it and nothing invented here.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import CubeVerdictTable from '../components/CubeVerdictTable.svelte';
import { cubeDecision } from '../utils/cubeDecision.js';

const money = { cubeless: 0.4, no_double: 0.55, double_take: 0.71, double_pass: 1.0, verdict: 'double_take' };

function renderWith(props) {
    return render(CubeVerdictTable, {
        props: { decision: cubeDecision({ isRace: true, race: { money } }), showInfo: false, ...props }
    });
}

describe('CubeVerdictTable — Jacoby/Beaver badges', () => {
    afterEach(cleanup);

    test('neither flag: no badge at all', () => {
        const { container } = renderWith({});
        expect(container.querySelector('.cube-rules')).toBeNull();
    });

    test('Jacoby only', () => {
        const { container } = renderWith({ jacoby: true });
        const badges = [...container.querySelectorAll('.rule-badge')].map((el) => el.textContent);
        expect(badges).toEqual(['Jacoby']);
    });

    test('Beaver only', () => {
        const { container } = renderWith({ beaver: true });
        const badges = [...container.querySelectorAll('.rule-badge')].map((el) => el.textContent);
        expect(badges).toEqual(['Beaver']);
    });

    test('both flags: both badges, Jacoby first', () => {
        const { container } = renderWith({ jacoby: true, beaver: true });
        const badges = [...container.querySelectorAll('.rule-badge')].map((el) => el.textContent);
        expect(badges).toEqual(['Jacoby', 'Beaver']);
    });
});
