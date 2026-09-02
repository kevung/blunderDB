/**
 * EntityAutocomplete.test.js — the shared text-field-with-dropdown behind
 * MatchPanel's tournament cell and TournamentPanel's "add a match" field:
 * filtering, keyboard selection, and where the list opens.
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick, createRawSnippet } from 'svelte';
import EntityAutocomplete from '../components/EntityAutocomplete.svelte';

const ITEMS = [{ name: 'Nordic Open' }, { name: 'Marseille' }, { name: 'Nice Open' }];

afterEach(() => {
    cleanup();
    vi.useRealTimers();
});

function layout(input, { top, bottom, innerHeight = 600 }) {
    input.getBoundingClientRect = () => ({ top, bottom, left: 10, right: 210, width: 200, height: bottom - top });
    window.innerHeight = innerHeight;
}

async function mount(props = {}) {
    const result = render(EntityAutocomplete, { props: { items: ITEMS, ...props } });
    const input = result.container.querySelector('input');
    layout(input, { top: 100, bottom: 120 });
    await tick();
    return { ...result, input };
}

const options = (container) => Array.from(container.querySelectorAll('.option')).map((el) => el.textContent.trim());

describe('EntityAutocomplete', () => {
    test('opens on focus with every item, then narrows the list as the user types', async () => {
        const { container, input } = await mount();
        expect(container.querySelector('.dropdown')).toBeNull();

        await fireEvent.focus(input);
        expect(options(container)).toEqual(['Nordic Open', 'Marseille', 'Nice Open']);

        await fireEvent.input(input, { target: { value: 'open' } });
        expect(options(container)).toEqual(['Nordic Open', 'Nice Open']);

        await fireEvent.input(input, { target: { value: 'zzz' } });
        expect(container.querySelector('.dropdown')).toBeNull();
    });

    test('a custom filter and label drive both the list and the picked value', async () => {
        const items = [
            { id: 1, player1_name: 'Ann', player2_name: 'Bob', match_length: 7 },
            { id: 2, player1_name: 'Cid', player2_name: 'Dee', match_length: 5 }
        ];
        const onSelect = vi.fn();
        const { container, input } = await mount({
            items,
            key: (m) => m.id,
            label: (m) => `${m.player1_name} vs ${m.player2_name}`,
            filter: (m, q) => m.player1_name.toLowerCase().includes(q.toLowerCase()) || m.player2_name.toLowerCase().includes(q.toLowerCase()),
            onSelect
        });
        await fireEvent.focus(input);
        await fireEvent.input(input, { target: { value: 'dee' } });
        expect(options(container)).toEqual(['Cid vs Dee']);

        await fireEvent.mouseDown(container.querySelector('.option'));
        expect(onSelect).toHaveBeenCalledWith(items[1]);
        expect(input.value).toBe('Cid vs Dee');
    });

    test('ArrowDown/ArrowUp move the highlight and Enter picks it, filling the field and closing the list', async () => {
        const onSelect = vi.fn();
        const onSubmit = vi.fn();
        const { container, input } = await mount({ onSelect, onSubmit });
        await fireEvent.focus(input);

        await fireEvent.keyDown(input, { key: 'ArrowDown' });
        await tick();
        await fireEvent.keyDown(input, { key: 'ArrowDown' });
        await tick();
        expect(container.querySelector('.option.active').textContent.trim()).toBe('Marseille');

        await fireEvent.keyDown(input, { key: 'ArrowUp' });
        await tick();
        expect(container.querySelector('.option.active').textContent.trim()).toBe('Nordic Open');

        await fireEvent.keyDown(input, { key: 'Enter' });
        expect(onSelect).toHaveBeenCalledWith(ITEMS[0]);
        expect(onSubmit).not.toHaveBeenCalled();
        expect(input.value).toBe('Nordic Open');
        expect(container.querySelector('.dropdown')).toBeNull();
    });

    test('Enter with nothing highlighted submits the typed text; Escape blurs and cancels', async () => {
        const onSubmit = vi.fn();
        const onCancel = vi.fn();
        const { container, input } = await mount({ onSubmit, onCancel });
        input.focus();
        await fireEvent.focus(input);
        await fireEvent.input(input, { target: { value: 'Brand new' } });

        await fireEvent.keyDown(input, { key: 'Enter' });
        expect(onSubmit).toHaveBeenCalledWith('Brand new');

        await fireEvent.keyDown(input, { key: 'Escape' });
        expect(onCancel).toHaveBeenCalledTimes(1);
        expect(document.activeElement).not.toBe(input);
        expect(container.querySelector('.dropdown')).toBeNull();
    });

    test('Enter, Escape and the arrows never reach the document (the panels own those keys)', async () => {
        const spy = vi.fn();
        document.addEventListener('keydown', spy);
        try {
            const { input } = await mount();
            await fireEvent.focus(input);
            for (const key of ['ArrowDown', 'ArrowUp', 'Enter', 'Escape']) {
                await fireEvent.keyDown(input, { key });
            }
            expect(spy).not.toHaveBeenCalled();
            await fireEvent.keyDown(input, { key: 'j' });
            expect(spy).toHaveBeenCalledTimes(1);
        } finally {
            document.removeEventListener('keydown', spy);
        }
    });

    test('fillOnSelect={false}: picking keeps the field and the list, for adding several in a row', async () => {
        const onSelect = vi.fn();
        const { container, input } = await mount({ onSelect, fillOnSelect: false });
        await fireEvent.focus(input);
        await fireEvent.input(input, { target: { value: 'n' } });
        await fireEvent.mouseDown(container.querySelectorAll('.option')[1]);
        expect(onSelect).toHaveBeenCalledWith(ITEMS[2]);
        expect(input.value).toBe('n');
        expect(container.querySelector('.dropdown')).not.toBeNull();
    });

    test('blur dismisses after the delay, so a mousedown on an option lands first', async () => {
        vi.useFakeTimers();
        const onDismiss = vi.fn();
        const { container, input } = await mount({ onDismiss, blurDelay: 200 });
        await fireEvent.focus(input);
        await fireEvent.blur(input);
        expect(onDismiss).not.toHaveBeenCalled();
        expect(container.querySelector('.dropdown')).not.toBeNull();
        vi.advanceTimersByTime(200);
        await tick();
        expect(onDismiss).toHaveBeenCalledTimes(1);
        expect(container.querySelector('.dropdown')).toBeNull();
    });

    test('the list opens below the input when there is room, above when there is not', async () => {
        const { container, input } = await mount();
        layout(input, { top: 100, bottom: 120, innerHeight: 600 });
        await fireEvent.focus(input);
        let style = container.querySelector('.dropdown').style;
        expect(style.position).toBe('fixed');
        expect(style.top).toBe('120px');
        expect(style.bottom).toBe('');
        expect(style.maxHeight).toBe('120px');

        // 40px left under the input, 560px above: flip.
        layout(input, { top: 540, bottom: 560, innerHeight: 600 });
        await fireEvent.input(input, { target: { value: '' } });
        style = container.querySelector('.dropdown').style;
        expect(style.bottom).toBe('60px');
        expect(style.top).toBe('');
        expect(style.maxHeight).toBe('120px');
    });

    test('placement="above" always anchors the list to the top edge of the input', async () => {
        const { container, input } = await mount({ placement: 'above', maxHeight: 90 });
        layout(input, { top: 100, bottom: 120, innerHeight: 600 });
        await fireEvent.focus(input);
        const style = container.querySelector('.dropdown').style;
        expect(style.bottom).toBe('500px');
        expect(style.maxHeight).toBe('90px');
    });

    test('a custom item snippet renders each option', async () => {
        const item = createRawSnippet((getEntry) => ({ render: () => `<b>${getEntry().name} · ${getEntry().pts}pt</b>` }));
        const { container, input } = await mount({ items: [{ name: 'X', pts: 7 }], item });
        await fireEvent.focus(input);
        expect(options(container)).toEqual(['X · 7pt']);
    });
});
