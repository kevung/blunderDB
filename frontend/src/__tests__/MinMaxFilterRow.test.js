/**
 * MinMaxFilterRow.test.js
 *
 * Component test for the min/max/range filter row extracted from SearchPanel.
 * Pins the reactive part of the extraction: which of the four number inputs is
 * enabled follows the selected radio (`option`), and the HTML min/max bounds are
 * applied. Two-way binding itself is Svelte framework behaviour.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import MinMaxFilterRow from '../components/MinMaxFilterRow.svelte';

function nums(container) {
    return [...container.querySelectorAll('input[type="number"]')];
}
function radios(container) {
    return [...container.querySelectorAll('input[type="radio"]')];
}

describe('MinMaxFilterRow', () => {
    afterEach(cleanup);

    test('renders 3 radios and 4 number inputs with the passed values', () => {
        const { container } = render(MinMaxFilterRow, {
            props: { option: 'min', minVal: 5, maxVal: 10, rangeMin: 1, rangeMax: 9, min: 0, max: 100 }
        });
        expect(radios(container)).toHaveLength(3);
        const n = nums(container);
        expect(n).toHaveLength(4);
        expect(n[0].value).toBe('5');
        expect(n[3].value).toBe('9');
    });

    test('option="min" enables only the min input', () => {
        const { container } = render(MinMaxFilterRow, {
            props: { option: 'min', minVal: 0, maxVal: 0, rangeMin: 0, rangeMax: 0 }
        });
        const [minIn, maxIn, rMin, rMax] = nums(container);
        expect(minIn.disabled).toBe(false);
        expect(maxIn.disabled).toBe(true);
        expect(rMin.disabled).toBe(true);
        expect(rMax.disabled).toBe(true);
    });

    test('option="range" enables only the two range inputs', () => {
        const { container } = render(MinMaxFilterRow, {
            props: { option: 'range', minVal: 0, maxVal: 0, rangeMin: 0, rangeMax: 0 }
        });
        const [minIn, maxIn, rMin, rMax] = nums(container);
        expect(minIn.disabled).toBe(true);
        expect(maxIn.disabled).toBe(true);
        expect(rMin.disabled).toBe(false);
        expect(rMax.disabled).toBe(false);
    });

    test('applies the HTML min/max bounds, and omits them when not given', () => {
        const { container: withBounds } = render(MinMaxFilterRow, {
            props: { option: 'min', minVal: 0, maxVal: 0, rangeMin: 0, rangeMax: 0, min: 0, max: 375 }
        });
        const n1 = nums(withBounds);
        expect(n1[0].getAttribute('min')).toBe('0');
        expect(n1[0].getAttribute('max')).toBe('375');
        cleanup();

        const { container: noBounds } = render(MinMaxFilterRow, {
            props: { option: 'min', minVal: 0, maxVal: 0, rangeMin: 0, rangeMax: 0 }
        });
        const n2 = nums(noBounds);
        expect(n2[0].hasAttribute('min')).toBe(false);
        expect(n2[0].hasAttribute('max')).toBe(false);
    });

    test('selecting the range radio re-enables the range inputs (internal reactivity)', async () => {
        const { container } = render(MinMaxFilterRow, {
            props: { option: 'min', minVal: 0, maxVal: 0, rangeMin: 0, rangeMax: 0 }
        });
        const [minIn, , rMin, rMax] = nums(container);
        expect(minIn.disabled).toBe(false);
        expect(rMin.disabled).toBe(true);

        await fireEvent.click(radios(container)[2]); // the "range" radio
        await tick();

        expect(rMin.disabled).toBe(false);
        expect(rMax.disabled).toBe(false);
        expect(minIn.disabled).toBe(true);
    });
});
