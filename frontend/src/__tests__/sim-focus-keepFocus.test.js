import { describe, test, expect, afterEach } from 'vitest';
import { keepFocus } from '../utils/keepFocus.js';

afterEach(() => (document.body.innerHTML = ''));
const flush = () => new Promise((r) => setTimeout(r, 0));

describe('keepFocus', () => {
    test('le focus retombé sur body après la disparition de son porteur va au repli', async () => {
        document.body.innerHTML = '<div id="n"><button id="a">a</button><button id="b">b</button></div>';
        const node = document.getElementById('n');
        keepFocus(node, () => document.getElementById('b'));
        document.getElementById('a').focus();
        document.getElementById('a').remove();
        expect(document.activeElement).toBe(document.body);
        await flush();
        expect(document.activeElement?.id).toBe('b');
    });
    test('un focus qui n’était pas dedans n’est pas volé', async () => {
        document.body.innerHTML = '<input id="o"><div id="n"><button id="b">b</button></div>';
        keepFocus(document.getElementById('n'), () => document.getElementById('b'));
        document.getElementById('o').focus();
        document.getElementById('o').blur();
        document.getElementById('n').appendChild(document.createElement('i'));
        await flush();
        expect(document.activeElement).toBe(document.body);
    });
});
