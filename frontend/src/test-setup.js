import '@testing-library/jest-dom/vitest';

// Node's own (file-less) `localStorage` global shadows jsdom's and has no methods: give the tests
// a plain in-memory Storage so code that remembers things in the browser can be exercised.
if (typeof globalThis.localStorage?.clear !== 'function') {
    const data = new Map();
    Object.defineProperty(globalThis, 'localStorage', {
        configurable: true,
        value: {
            getItem: (/** @type {string} */ k) => (data.has(k) ? data.get(k) : null),
            setItem: (/** @type {string} */ k, /** @type {unknown} */ v) => void data.set(k, String(v)),
            removeItem: (/** @type {string} */ k) => void data.delete(k),
            clear: () => data.clear()
        }
    });
}

// English is loaded on demand like every locale; component tests render without calling
// initLanguage, so load it before each file's tests (after its vi.mock calls took effect).
import { beforeAll } from 'vitest';
beforeAll(async () => {
    try {
        const mod = await import('./i18n/index.js');
        await mod.initLanguage('en');
    } catch {
        // A file may mock the whole module: then it owns its translations.
    }
});

// No backend under the tests: the hold of the library a write names (groupActivityStore's
// writingOnLibrary) is granted. A test of a refusal makes HoldLibrary reject.
import { vi } from 'vitest';
vi.mock('./stores/libraryHold.js', () => ({
    HoldLibrary: vi.fn().mockResolvedValue(1),
    ReleaseLibrary: vi.fn().mockResolvedValue(undefined)
}));
