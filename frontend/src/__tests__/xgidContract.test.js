/**
 * xgidContract.test.js
 *
 * GUI half of the XGID contract corpus (testdata/xgid_corpus.json), the other
 * half being pkg/blunderdb/domain/xgid_contract_test.go.
 *
 * The GUI no longer parses XGIDs itself (commit cd33de85: it parses via the
 * backend), so this side does not decode anything. It starts from `position`,
 * the exact document the Go decoder is asserted to produce for `xgid`, and
 * checks the two GUI functions that read such a position:
 *   - generateXGID(position) must equal `xgidCanonical` — the re-encoding
 *     under the GUI's own conventions (see the corpus _comment): Jacoby/Beaver
 *     round-trip losslessly since D.11/#211, match length is reconstructed as
 *     the larger away score (harmless — it still decodes to the same away
 *     scores), Crawford is inferred from an away score of 1, max cube is
 *     emitted as 0 (blunderDB has no capped-cube concept), and an offered cube
 *     re-encodes as dice 00;
 *   - computePipCount(position) must equal `pips`, the Go ComputePipCounts.
 * The flat fields (cubeOwner, dice, score, ...) restate `position` for the
 * reader; they are cross-checked so a corpus edit cannot leave them stale.
 */

import { describe, test, expect, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

// generateXGID is pure, but positionService.js pulls the Wails bindings and
// databaseService in at module load; stub them so the module imports under jsdom.
vi.mock('../../wailsjs/go/database/Database.js', () => ({}));
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));

const { generateXGID } = await import('../services/positionService.js');
const { computePipCount } = await import('../utils/boardGeometry.js');

const __dirname = dirname(fileURLToPath(import.meta.url));
const corpus = JSON.parse(readFileSync(resolve(__dirname, '../../../testdata/xgid_corpus.json'), 'utf8'));

describe('XGID contract corpus (GUI generateXGID / computePipCount vs Go DecodeXGID)', () => {
    test('corpus is non-empty', () => {
        expect(corpus.cases.length).toBeGreaterThan(0);
    });

    for (const c of corpus.cases) {
        describe(c.name, () => {
            test('the flat fields restate position', () => {
                expect(c.position.cube.owner).toBe(c.cubeOwner);
                expect(c.position.cube.value).toBe(c.cubeValueExp);
                expect(c.position.dice).toEqual(c.dice);
                expect(c.position.player_on_roll).toBe(c.playerOnRoll);
                expect(c.position.score).toEqual(c.score);
                expect(c.position.has_jacoby).toBe(c.hasJacoby);
                expect(c.position.has_beaver).toBe(c.hasBeaver);
            });

            test('generateXGID re-encodes the decoded position to xgidCanonical', () => {
                expect(generateXGID(c.position)).toBe(c.xgidCanonical);
            });

            test('computePipCount agrees with Go ComputePipCounts', () => {
                const { pipCount1, pipCount2 } = computePipCount(c.position);
                expect([pipCount1, pipCount2]).toEqual(c.pips);
            });
        });
    }
});
