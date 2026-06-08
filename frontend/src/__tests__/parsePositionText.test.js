import { describe, test, expect, vi, beforeEach } from 'vitest';

// The Go parser is the source of truth (pkg/blunderdb/parser, contract-tested
// against testdata/parse_corpus.json). Here we only lock the thin GUI adapter
// `parsePositionText`, which reshapes the backend Result into the legacy
// { positionData, parsedAnalysis } the callers consume — in particular the
// defaulting that prevents the old `doublingCubeAnalysis is undefined` crash for
// checker positions.
const ParsePositionText = vi.fn();
vi.mock('../../wailsjs/go/gui/App.js', () => ({}));
vi.mock('../../wailsjs/go/database/Database.js', () => ({ ParsePositionText }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));

const { parsePositionText } = await import('../services/importService.js');

beforeEach(() => ParsePositionText.mockReset());

describe('parsePositionText adapter', () => {
    test('checker result → doublingCubeAnalysis defaults to {} and moves unwrap to a bare array', async () => {
        ParsePositionText.mockResolvedValue({
            position: { decision_type: 0 },
            analysis: {
                xgid: 'X',
                analysisType: 'CheckerMove',
                analysisEngineVersion: 'eXtreme Gammon Version: 2.19',
                checkerAnalysis: { moves: [{ index: 1, move: '24/23', equity: 0.1 }] }
                // doublingCubeAnalysis intentionally absent
            },
            comment: 'note'
        });

        const { positionData, parsedAnalysis } = await parsePositionText('XGID=X');

        expect(positionData).toEqual({ decision_type: 0 });
        expect(parsedAnalysis.analysisType).toBe('CheckerMove');
        expect(parsedAnalysis.doublingCubeAnalysis).toEqual({});
        expect(parsedAnalysis.checkerAnalysis).toEqual([{ index: 1, move: '24/23', equity: 0.1 }]);
        expect(parsedAnalysis.comment).toBe('note');
        expect(parsedAnalysis.xgid).toBe('X');
    });

    test('cube result → checkerAnalysis defaults to a bare empty array', async () => {
        ParsePositionText.mockResolvedValue({
            position: { decision_type: 1 },
            analysis: {
                xgid: 'Y',
                analysisType: 'DoublingCube',
                doublingCubeAnalysis: { bestCubeAction: 'Double / Take', cubefulDoubleTakeEquity: 0.3 }
                // checkerAnalysis intentionally absent
            },
            comment: ''
        });

        const { parsedAnalysis } = await parsePositionText('XGID=Y');

        expect(parsedAnalysis.checkerAnalysis).toEqual([]);
        expect(parsedAnalysis.doublingCubeAnalysis.bestCubeAction).toBe('Double / Take');
        expect(parsedAnalysis.comment).toBe('');
    });
});
