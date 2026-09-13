import { describe, test, expect, afterEach } from 'vitest';
import { focusPanelUnlessTyping, isTypingTarget } from '../utils/panelFocus.js';

// See utils/panelFocus.js: a panel focusing itself after a delay used to take
// the caret from whatever field the user had reached in the meantime.

/** @param {string} html */
function mount(html) {
    document.body.innerHTML = html;
}

/** @param {string} selector */
const el = (selector) => /** @type {HTMLElement} */ (document.querySelector(selector));

describe('focusPanelUnlessTyping', () => {
    afterEach(() => {
        document.body.innerHTML = '';
    });

    test('takes the focus when nothing is focused', () => {
        mount('<section id="panel" tabindex="-1"></section>');
        const panel = el('#panel');

        expect(focusPanelUnlessTyping(panel)).toBe(true);
        expect(document.activeElement).toBe(panel);
    });

    test('takes the focus from a button (a tab just clicked)', () => {
        mount('<button id="tab">Tournaments</button><section id="panel" tabindex="-1"></section>');
        el('#tab').focus();
        const panel = el('#panel');

        expect(focusPanelUnlessTyping(panel)).toBe(true);
        expect(document.activeElement).toBe(panel);
    });

    test('leaves the caret in a field inside the panel', () => {
        mount('<section id="panel" tabindex="-1"><input id="name" /></section>');
        const input = el('#name');
        input.focus();

        expect(focusPanelUnlessTyping(el('#panel'))).toBe(false);
        expect(document.activeElement).toBe(input);
    });

    test('leaves the caret in a field outside the panel (the command line)', () => {
        mount('<input class="command-input" /><section id="panel" tabindex="-1"></section>');
        const commandLine = el('.command-input');
        commandLine.focus();

        expect(focusPanelUnlessTyping(el('#panel'))).toBe(false);
        expect(document.activeElement).toBe(commandLine);
    });

    test('a missing panel is a no-op', () => {
        expect(focusPanelUnlessTyping(null)).toBe(false);
    });
});

describe('isTypingTarget', () => {
    test.each([
        ['<input />', true],
        ['<textarea></textarea>', true],
        ['<select></select>', true],
        ['<div contenteditable="true"></div>', true],
        ['<button></button>', false],
        ['<section tabindex="-1"></section>', false]
    ])('%s → %s', (html, expected) => {
        document.body.innerHTML = html;
        expect(isTypingTarget(document.body.firstElementChild)).toBe(expected);
    });

    test('no element is not a typing target', () => {
        expect(isTypingTarget(null)).toBe(false);
    });
});
