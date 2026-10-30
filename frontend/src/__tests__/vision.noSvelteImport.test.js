import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

// Z-vision reads pixels and knows nothing of Svelte; the panel knows nothing of pixels. The two
// meet only through the event contract of vision/contract.js (ADR-0108).
const ROOT = join(__dirname, '..', 'vision');

/** @param {string} dir @returns {string[]} */
function files(dir) {
    return readdirSync(dir).flatMap((name) => {
        const p = join(dir, name);
        return statSync(p).isDirectory() ? files(p) : [p];
    });
}

describe('frontend/src/vision imports no Svelte', () => {
    it('has no svelte import nor .svelte file', () => {
        const all = files(ROOT);
        expect(all.length).toBeGreaterThan(0);
        const offenders = all.filter((p) => {
            if (p.endsWith('.svelte')) return true;
            const src = readFileSync(p, 'utf8');
            return /(?:from\s+|import\s*\(\s*|import\s+)['"](?:svelte(?:\/[^'"]*)?|[^'"]*\.svelte)['"]/.test(src);
        });
        expect(offenders.map((p) => relative(ROOT, p))).toEqual([]);
    });
});
