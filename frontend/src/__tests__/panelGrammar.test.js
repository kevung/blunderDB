/**
 * panelGrammar.test.js — the shared pieces of a panel (ADR-0085): the header strip, the empty
 * state's own action, the count that leads to its positions, the label │ control grid.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick, createRawSnippet } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({ LoadAnalysis: vi.fn(() => Promise.resolve(null)) }));
vi.mock('../services/positionService.js', async (importOriginal) => ({ ...(await importOriginal()), sendPositionToEval: vi.fn() }));

import PanelHeader from '../components/panels/PanelHeader.svelte';
import EmptyState from '../components/panels/EmptyState.svelte';
import CountLink from '../components/panels/CountLink.svelte';
import FormGrid from '../components/panels/FormGrid.svelte';
import FormRow from '../components/panels/FormRow.svelte';
import AnalysisPanel from '../components/AnalysisPanel.svelte';
import { databasePathStore } from '../stores/databaseStore';
import { positionStore } from '../stores/positionStore.js';
import { sendPositionToEval } from '../services/positionService.js';
import { get } from 'svelte/store';

const initialPosition = get(positionStore);

afterEach(() => {
    cleanup();
    databasePathStore.set('');
    positionStore.set(initialPosition);
});

const button = (label) => createRawSnippet(() => ({ render: () => `<button type="button" class="primary">${label}</button>` }));

describe('PanelHeader', () => {
    test('title, count, then the actions after a spacer, the primary last', () => {
        const { container } = render(PanelHeader, { props: { title: 'Collections', count: '3', actions: button('New') } });
        const strip = container.querySelector('[data-testid="panel-header"]');
        const parts = [...strip.children].map((el) => el.className.split(' ').find((c) => !c.startsWith('svelte-')));
        expect(parts).toEqual(['panel-title', 'panel-count', 'spacer', 'panel-actions']);
        expect(strip.querySelector('.panel-actions').textContent).toBe('New');
    });

    test('a back arrow and a count that leads somewhere, only when asked for', async () => {
        const onBack = vi.fn();
        const onCount = vi.fn();
        const { getByTestId } = render(PanelHeader, { props: { title: 'Blitz', onBack, count: '12 positions', onCount } });
        await fireEvent.click(getByTestId('panel-back'));
        await fireEvent.click(getByTestId('count-link'));
        expect(onBack).toHaveBeenCalledOnce();
        expect(onCount).toHaveBeenCalledOnce();
    });

    test('no title, no count: the strip still stands, for a panel whose filter is its header', () => {
        const { container } = render(PanelHeader);
        expect(container.querySelector('.panel-title')).toBeNull();
        expect(container.querySelector('[data-testid="count-link"]')).toBeNull();
        expect(container.querySelector('[data-testid="panel-header"]')).not.toBeNull();
    });
});

describe('EmptyState', () => {
    test("the panel's own action replaces the import", async () => {
        databasePathStore.set('/tmp/x.db');
        const onClick = vi.fn();
        const { getByTestId, queryByText } = render(EmptyState, { props: { text: 'No collection', action: { label: 'New collection', onClick } } });
        await fireEvent.click(getByTestId('empty-action'));
        expect(onClick).toHaveBeenCalledOnce();
        expect(queryByText(/import/i)).toBeNull();
    });
});

describe('CountLink', () => {
    test('a button, so a count of positions opens them', async () => {
        const onclick = vi.fn();
        const { getByTestId } = render(CountLink, { props: { label: '5 positions', onclick } });
        expect(getByTestId('count-link').tagName).toBe('BUTTON');
        await fireEvent.click(getByTestId('count-link'));
        expect(onclick).toHaveBeenCalledOnce();
    });
});

describe('FormGrid / FormRow', () => {
    test('mount: one label and one control cell per row', () => {
        const control = createRawSnippet(() => ({ render: () => '<input id="n" />' }));
        const { container } = render(FormGrid, { props: { children: createRawSnippet(() => ({ render: () => '<span></span>' })) } });
        expect(container.querySelector('[data-testid="form-grid"]')).not.toBeNull();
        const row = render(FormRow, { props: { label: 'Name', for: 'n', children: control } });
        expect(row.container.querySelector('label.form-label').getAttribute('for')).toBe('n');
        expect(row.container.querySelector('.form-control input')).not.toBeNull();
    });
});

describe('AnalysisPanel — nothing analysed', () => {
    test('says so and offers to evaluate the position, instead of a blank panel', async () => {
        const position = { ...initialPosition, id: 7 };
        positionStore.set(position);
        const { getByTestId } = render(AnalysisPanel);
        await tick();
        await fireEvent.click(getByTestId('empty-action'));
        expect(sendPositionToEval).toHaveBeenCalledWith(position);
    });
});
