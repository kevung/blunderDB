/**
 * i18n.reactivity.test.js
 *
 * Confirms the engine is reactive and persists:
 *   - `t` is a derived store: switching `language` re-renders $t(...) callers,
 *   - setLanguage() updates the store AND persists via SaveLanguage,
 *   - the reactive help store swaps content with the language,
 *   - an unsupported code falls back to English.
 */

import { describe, test, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

const SaveLanguage = vi.fn(() => Promise.resolve(undefined));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    SaveLanguage: (...args) => SaveLanguage(...args),
    GetLanguage: vi.fn(() => Promise.resolve('en'))
}));

import { t, language, setLanguage, initLanguage, FALLBACK_LOCALE, tMsg, resolveStatusMessage } from '../i18n';
import { help } from '../i18n/help/index.js';

describe('i18n reactivity', () => {
    beforeEach(() => {
        SaveLanguage.mockClear();
        initLanguage('en');
    });

    test('derived t re-emits when language changes', () => {
        const seen = [];
        const unsub = t.subscribe((fn) => seen.push(fn('common.back')));
        initLanguage('fr');
        initLanguage('de');
        unsub();
        // First emission is English, then French, then German.
        expect(seen).toContain('Back');
        expect(seen).toContain('Retour');
        expect(seen).toContain('Zurück');
    });

    test('setLanguage updates the store and persists', async () => {
        await setLanguage('it');
        expect(get(language)).toBe('it');
        expect(SaveLanguage).toHaveBeenCalledWith('it');
        expect(get(t)('common.back')).toBe(get(t)('common.back')); // resolves without throwing
    });

    test('setLanguage rejects an unsupported code and falls back', async () => {
        await setLanguage('zz');
        expect(get(language)).toBe(FALLBACK_LOCALE);
        expect(SaveLanguage).toHaveBeenCalledWith(FALLBACK_LOCALE);
    });

    test('setLanguage swallows a SaveLanguage failure but still switches', async () => {
        SaveLanguage.mockRejectedValueOnce(new Error('boom'));
        await setLanguage('es');
        expect(get(language)).toBe('es');
    });

    test('help store swaps content with the language', () => {
        initLanguage('en');
        const enManual = get(help).manual;
        initLanguage('fr');
        const frManual = get(help).manual;
        expect(typeof enManual).toBe('string');
        expect(typeof frManual).toBe('string');
        expect(frManual.length).toBeGreaterThan(0);
        // A translated manual should differ from the English one.
        expect(frManual).not.toBe(enManual);
    });

    // Status-bar messages are stored as tMsg() descriptors and resolved through
    // $t, so an already-displayed message re-translates when the language changes.
    test('tMsg descriptor re-resolves through $t after a language switch', () => {
        const descriptor = tMsg('common.back');
        expect(descriptor).toEqual({ i18nKey: 'common.back', i18nParams: null });

        initLanguage('en');
        expect(resolveStatusMessage(descriptor, get(t))).toBe('Back');
        initLanguage('fr');
        expect(resolveStatusMessage(descriptor, get(t))).toBe('Retour');
        initLanguage('de');
        expect(resolveStatusMessage(descriptor, get(t))).toBe('Zurück');
    });

    test('tMsg interpolates params on resolution', () => {
        const descriptor = tMsg('commands.goToPosition', { n: 5 });
        initLanguage('en');
        expect(resolveStatusMessage(descriptor, get(t))).toBe('Go to position 5');
    });

    test('resolveStatusMessage passes plain strings through unchanged', () => {
        // Player names / technical tokens are stored as raw strings, not descriptors.
        expect(resolveStatusMessage('Alice vs Bob', get(t))).toBe('Alice vs Bob');
        expect(resolveStatusMessage('', get(t))).toBe('');
    });
});
