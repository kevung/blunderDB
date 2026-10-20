/**
 * The active group's header writes the library's name in white on the group's hue, in either
 * theme: each hue must keep 4.5:1 (WCAG AA for small text).
 */

import { test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';

const tokens = fs.readFileSync(path.join(__dirname, '../tokens.css'), 'utf8');

/** @param {string} hex */
function luminance(hex) {
    const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255).map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

test('white on every group hue keeps 4.5:1', () => {
    const hues = [...tokens.matchAll(/--group-color-(\d): (#[0-9a-f]{6});/gi)].map((m) => ({ n: m[1], hex: m[2] }));
    expect(hues.length).toBeGreaterThanOrEqual(6);
    for (const { n, hex } of hues) expect(1.05 / (luminance(hex) + 0.05), `--group-color-${n} ${hex}`).toBeGreaterThanOrEqual(4.5);
});
