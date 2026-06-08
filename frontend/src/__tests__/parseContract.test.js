import { describe, test, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

// parsePosition lives in importService.js, which imports Wails bindings at load
// time. parsePosition is pure, so stub the bindings to import it under jsdom.
vi.mock('../../wailsjs/go/gui/App.js', () => ({}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));

const { normalize } = await import('./_parseGolden.gen.js');

// Regression lock: the committed corpus is the spec. The SAME testdata/
// parse_corpus.json is asserted by the Go parser (pkg/blunderdb/parser/
// parse_contract_test.go). If the GUI parser drifts from the corpus, this fails;
// if the Go parser drifts, the Go test fails. See the corpus _comment.
const __dirname = dirname(fileURLToPath(import.meta.url));
const corpusPath = resolve(__dirname, '../../../testdata/parse_corpus.json');
const corpus = JSON.parse(readFileSync(corpusPath, 'utf8'));

describe('parsePosition contract (GUI parser locked to shared corpus)', () => {
    test('corpus is non-empty', () => {
        expect(corpus.cases.length).toBeGreaterThan(0);
    });

    for (const c of corpus.cases) {
        test(c.name, () => {
            expect(normalize(c.input)).toEqual(c.expected);
        });
    }
});
