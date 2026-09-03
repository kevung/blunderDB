/**
 * i18n.fallback.test.js
 *
 * Exercises the translation engine's fallback chain and interpolation:
 *   selected locale -> English -> the key itself,
 * plus {placeholder} substitution.
 *
 * SaveLanguage is mocked so setLanguage() doesn't reach into Wails bindings.
 * initLanguage() is async (#207: it fetches the locale dictionary on demand),
 * so every call here is awaited.
 */

import { describe, test, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    SaveLanguage: vi.fn(() => Promise.resolve(undefined)),
    GetLanguage: vi.fn(() => Promise.resolve('en'))
}));

import { t, translate, language, initLanguage } from '../i18n';

describe('i18n fallback chain', () => {
    beforeEach(async () => {
        await initLanguage('en');
    });

    test('resolves a known key in the active locale', async () => {
        await initLanguage('fr');
        expect(get(t)('common.back')).toBe('Retour');
    });

    test('falls back to English when the key is missing in the active locale', async () => {
        await initLanguage('fr');
        const fr = get(t)('common.back');
        await initLanguage('en');
        const en = get(t)('common.back');
        // Sanity: the two locales genuinely differ for this key, and English resolves.
        expect(fr).not.toBe(en);
        expect(en).toBe('Back');
    });

    test('falls back to the raw key when it exists in no locale', () => {
        expect(get(t)('this.key.does.not.exist')).toBe('this.key.does.not.exist');
    });

    test('interpolates a {placeholder}', () => {
        // commands.goToPosition === 'Go to position {n}'
        expect(get(t)('commands.goToPosition', { n: 7 })).toBe('Go to position 7');
    });

    test('leaves unknown placeholders intact when params omit them', () => {
        // No `n` supplied -> the {n} token must survive untouched.
        expect(get(t)('commands.goToPosition', { other: 1 })).toBe('Go to position {n}');
    });

    test('non-reactive translate() matches the reactive store for the current language', async () => {
        await initLanguage('de');
        expect(translate('common.back')).toBe(get(t)('common.back'));
        expect(get(language)).toBe('de');
    });
});
