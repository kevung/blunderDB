/**
 * ContextMenu.submenu.test.js — a menu item that holds a submenu: → or a click opens it on its
 * first item, ← and Escape close it back onto the item that opened it, the arrows move inside
 * one level only, and a click on a leaf runs it and closes the whole menu.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import ContextMenu from '../components/ContextMenu.svelte';

/** @param {string} key */
async function press(key) {
    const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
    (document.activeElement ?? document.body).dispatchEvent(event);
    await tick();
    await tick();
}

function mount() {
    const copy = vi.fn();
    const evaluate = vi.fn();
    const onClose = vi.fn();
    render(ContextMenu, {
        props: {
            x: 0,
            y: 0,
            onClose,
            items: [
                { label: 'Evaluate', onClick: evaluate },
                {
                    label: 'Copy',
                    submenu: [
                        { label: 'XGID', onClick: copy },
                        { label: 'Text', onClick: vi.fn() }
                    ]
                },
                { label: 'Manage', submenu: [{ label: 'Trash', onClick: vi.fn() }] }
            ]
        }
    });
    return { copy, evaluate, onClose };
}

const focused = () => document.activeElement?.textContent?.trim();

describe('ContextMenu submenus', () => {
    afterEach(cleanup);

    test('a submenu item announces its popup and stays closed until asked', async () => {
        mount();
        await tick();
        const copy = screen.getByRole('menuitem', { name: /^Copy/ });
        expect(copy.getAttribute('aria-haspopup')).toBe('menu');
        expect(copy.getAttribute('aria-expanded')).toBe('false');
        expect(screen.queryByRole('menuitem', { name: 'XGID' })).toBeNull();
    });

    test('→ opens the submenu on its first item; ↓ stays inside it; ← comes back', async () => {
        mount();
        await tick();
        expect(focused()).toBe('Evaluate');
        await press('ArrowDown');
        expect(focused()).toMatch(/^Copy/);
        await press('ArrowRight');
        expect(screen.getByRole('menuitem', { name: /^Copy/ }).getAttribute('aria-expanded')).toBe('true');
        expect(focused()).toBe('XGID');
        await press('ArrowDown');
        expect(focused()).toBe('Text');
        await press('ArrowDown');
        expect(focused(), 'the arrows wrap within the submenu').toBe('XGID');
        await press('ArrowLeft');
        expect(screen.queryByRole('menuitem', { name: 'XGID' })).toBeNull();
        expect(focused()).toMatch(/^Copy/);
    });

    test('Escape closes the submenu first, then the menu', async () => {
        const { onClose } = mount();
        await tick();
        await press('ArrowDown');
        await press('ArrowRight');
        await press('Escape');
        expect(screen.queryByRole('menuitem', { name: 'XGID' })).toBeNull();
        expect(focused()).toMatch(/^Copy/);
        expect(onClose).not.toHaveBeenCalled();
        await press('Escape');
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('hover opens one submenu at a time; a click on a leaf runs it and closes the menu', async () => {
        const { copy, onClose } = mount();
        await tick();
        screen.getByRole('menuitem', { name: /^Manage/ }).dispatchEvent(new MouseEvent('mouseenter'));
        await tick();
        expect(screen.getByRole('menuitem', { name: 'Trash' })).toBeTruthy();
        screen.getByRole('menuitem', { name: /^Copy/ }).dispatchEvent(new MouseEvent('mouseenter'));
        await tick();
        expect(screen.queryByRole('menuitem', { name: 'Trash' })).toBeNull();
        screen.getByRole('menuitem', { name: 'XGID' }).click();
        expect(copy).toHaveBeenCalledTimes(1);
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    test('a click on a submenu item opens it without closing the menu', async () => {
        const { onClose } = mount();
        await tick();
        screen.getByRole('menuitem', { name: /^Copy/ }).click();
        await tick();
        await tick();
        expect(focused()).toBe('XGID');
        expect(onClose).not.toHaveBeenCalled();
    });
});
