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
