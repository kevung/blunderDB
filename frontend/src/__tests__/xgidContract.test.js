import { describe, test, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

// parsePosition lives in importService.js, which imports many Wails bindings at
// module load time. parsePosition itself is pure, so we stub the bindings to
// let the module import under jsdom.
vi.mock('../../wailsjs/go/gui/App.js', () => ({}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));

const { parsePosition } = await import('../services/importService.js');

// Same corpus the Go decoder asserts (pkg/blunderdb/domain/xgid_contract_test.go).
// This is the cross-implementation contract that keeps the GUI clipboard parser
// and the backend DecodeXGID from drifting (issue #13). See the corpus
// _comment for conventions and intentionally excluded fields / edge cases.
const __dirname = dirname(fileURLToPath(import.meta.url));
const corpusPath = resolve(__dirname, '../../../testdata/xgid_corpus.json');
const corpus = JSON.parse(readFileSync(corpusPath, 'utf8'));

describe('XGID decode contract (GUI parsePosition vs Go DecodeXGID)', () => {
    test('corpus is non-empty', () => {
        expect(corpus.cases.length).toBeGreaterThan(0);
    });

    for (const c of corpus.cases) {
        test(c.name, () => {
            const { positionData } = parsePosition(`XGID=${c.xgid}`);
            expect(positionData.cube.owner).toBe(c.cubeOwner);
            expect(positionData.cube.value).toBe(c.cubeValueExp);
            expect(positionData.dice).toEqual(c.dice);
            expect(positionData.player_on_roll).toBe(c.playerOnRoll);
            expect(positionData.score).toEqual(c.score);
            expect(positionData.has_jacoby).toBe(c.hasJacoby);
            expect(positionData.has_beaver).toBe(c.hasBeaver);
        });
    }
});
