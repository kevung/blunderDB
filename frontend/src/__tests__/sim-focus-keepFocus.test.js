import { must } from './helpers/must.js';
import { describe, test, expect, afterEach } from 'vitest';
import { keepFocus } from '../utils/keepFocus.js';

afterEach(() => (document.body.innerHTML = ''));
const flush = () => new Promise((r) => setTimeout(r, 0));

describe('keepFocus', () => {
    test('le focus retombé sur body après la disparition de son porteur va au repli', async () => {
        document.body.innerHTML = '<div id="n"><button id="a">a</button><button id="b">b</button></div>';
        const node = document.getElementById('n');
        keepFocus(must(node), () => document.getElementById('b'));
        must(document.getElementById('a')).focus();
        must(document.getElementById('a')).remove();
        expect(document.activeElement).toBe(document.body);
        await flush();
        expect(document.activeElement?.id).toBe('b');
    });
    test('un focus qui n’était pas dedans n’est pas volé', async () => {
        document.body.innerHTML = '<input id="o"><div id="n"><button id="b">b</button></div>';
        keepFocus(must(document.getElementById('n')), () => document.getElementById('b'));
        must(document.getElementById('o')).focus();
        must(document.getElementById('o')).blur();
        must(document.getElementById('n')).appendChild(document.createElement('i'));
        await flush();
        expect(document.activeElement).toBe(document.body);
    });
});
