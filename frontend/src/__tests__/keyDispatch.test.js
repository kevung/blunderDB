/**
 * keyDispatch.test.js — the one keydown dispatch: tier order, and the rules no handler has to
 * restate (a modal keeps its keys, a claimed key never reaches the global shortcuts, a stopped
 * key reaches nothing further).
 */
import { describe, test, expect, vi, afterEach } from 'vitest';
import { registerKeys, registeredScopes } from '../services/keyDispatch.js';
import { activeModal, MODAL } from '../stores/uiStore.js';
import { confirmModalStore } from '../services/confirmService.js';

/** @type {(() => void)[]} */
let cleanups = [];

/** @param {string} scope @param {(e: KeyboardEvent) => void} [fn] */
function reg(scope, fn = () => {}) {
    const spy = vi.fn(fn);
    cleanups.push(registerKeys(scope, spy));
    return spy;
}

/** @param {object} [init] @param {EventTarget} [target] */
function press(init = { key: 'j' }, target = document.body) {
    const event = new KeyboardEvent('keydown', { bubbles: true, cancelable: true, ...init });
    target.dispatchEvent(event);
    return event;
}

afterEach(() => {
    for (const c of cleanups) c();
    cleanups = [];
    activeModal.set(null);
    confirmModalStore.set(null);
});

describe('keyDispatch', () => {
    test('an undeclared scope is refused', () => {
        expect(() => registerKeys('nowhere', () => {})).toThrow(/undeclared/);
    });

    test('scopes run by tier, whatever the order they registered in', () => {
        /** @type {string[]} */
        const order = [];
        reg('global', () => order.push('global'));
        reg('boardEdit', () => order.push('board'));
        reg('search', () => order.push('search'));
        reg('directionQueue', () => order.push('queue'));
        press();
        expect(order).toEqual(['queue', 'search', 'board', 'global']);
        expect(registeredScopes()).toEqual(['directionQueue', 'search', 'boardEdit', 'global']);
    });

    test('the capture tiers run before the focused element, the others after it', () => {
        /** @type {string[]} */
        const order = [];
        reg('overlay', () => order.push('overlay'));
        reg('global', () => order.push('global'));
        const field = document.createElement('input');
        document.body.appendChild(field);
        field.addEventListener('keydown', () => order.push('field'));
        press({ key: 'x' }, field);
        field.remove();
        expect(order).toEqual(['overlay', 'field', 'global']);
    });

    test('a key claimed by an earlier scope never reaches the global shortcuts', () => {
        reg('boardEdit', (e) => e.preventDefault());
        const global = reg('global');
        press({ key: 'Backspace' });
        expect(global).not.toHaveBeenCalled();
    });

    test('a claimed Escape still reaches the global dispatcher, which reads the claim', () => {
        reg('search', (e) => e.preventDefault());
        const global = reg('global');
        press({ key: 'Escape' });
        expect(global).toHaveBeenCalledTimes(1);
        expect(global.mock.calls[0][0].defaultPrevented).toBe(true);
    });

    test('a stopped key reaches no later scope', () => {
        reg('search', (e) => e.stopPropagation());
        const board = reg('boardEdit');
        const global = reg('global');
        press();
        expect(board).not.toHaveBeenCalled();
        expect(global).not.toHaveBeenCalled();
    });

    test.each([
        ['an app modal', () => activeModal.set(MODAL.HELP)],
        ['the confirm dialog', () => confirmModalStore.set({ message: 'sure?' })]
    ])('while %s is open, no bubble-phase scope runs', (_, open) => {
        open();
        const search = reg('search');
        const global = reg('global');
        const overlay = reg('overlay');
        press({ key: 'Escape' });
        expect(search).not.toHaveBeenCalled();
        expect(global).not.toHaveBeenCalled();
        // The overlay tier checks the modals itself (escapeService.js).
        expect(overlay).toHaveBeenCalledTimes(1);
    });

    test('a key pressed during an input-method composition reaches no scope', () => {
        const capture = reg('overlay');
        const panel = reg('search');
        press({ key: 'Enter', isComposing: true });
        expect(capture).not.toHaveBeenCalled();
        expect(panel).not.toHaveBeenCalled();
        press({ key: 'Enter' });
        expect(panel).toHaveBeenCalledTimes(1);
    });

    test('an open context menu claims its arrows before a docked panel sees them', () => {
        reg('contextMenu', (e) => e.preventDefault());
        const panel = reg('matchPanel', (e) => {
            if (e.defaultPrevented) return;
            e.stopPropagation();
        });
        const event = press({ key: 'ArrowDown' });
        expect(event.defaultPrevented).toBe(true);
        expect(panel).toHaveBeenCalledTimes(1);
    });

    test('registeredScopes lists the scopes in dispatch order', () => {
        reg('global');
        reg('matchPanel');
        reg('overlay');
        const scopes = registeredScopes();
        expect(scopes.indexOf('overlay')).toBeLessThan(scopes.indexOf('matchPanel'));
        expect(scopes.indexOf('matchPanel')).toBeLessThan(scopes.indexOf('global'));
    });

    test('an unregistered scope no longer sees the keys', () => {
        const spy = vi.fn();
        const off = registerKeys('global', spy);
        off();
        press();
        expect(spy).not.toHaveBeenCalled();
    });
});
