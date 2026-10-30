import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

// Vision runs in plain JS in the webview: no inference runtime enters the bundle (ADR-0108).
describe('frontend/package.json has no inference dependency', () => {
    it('names none of onnxruntime-web, @tensorflow/*, opencv*', () => {
        const pkg = JSON.parse(readFileSync(join(__dirname, '..', '..', 'package.json'), 'utf8'));
        const names = Object.keys({ ...pkg.dependencies, ...pkg.devDependencies, ...pkg.optionalDependencies, ...pkg.peerDependencies });
        expect(names.filter((n) => n === 'onnxruntime-web' || n.startsWith('@tensorflow/') || /^(@[^/]+\/)?opencv/i.test(n))).toEqual([]);
    });
});
