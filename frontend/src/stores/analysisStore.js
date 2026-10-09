import { writable } from 'svelte/store';

/** A fresh, empty analysis record (the shape every consumer expects), new on each call. */
export function emptyAnalysis() {
    return {
        positionId: /** @type {number | null} */ (null),
        xgid: '',
        player1: '',
        player2: '',
        analysisType: '',
        analysisEngineVersion: '',
        checkerAnalysis: {
            moves: /** @type {import('../../wailsjs/go/models').domain.CheckerMove[]} */ ([])
        },
        doublingCubeAnalysis: {
            analysisDepth: '',
            playerWinChances: 0,
            playerGammonChances: 0,
            playerBackgammonChances: 0,
            opponentWinChances: 0,
            opponentGammonChances: 0,
            opponentBackgammonChances: 0,
            cubelessNoDoubleEquity: 0,
            cubelessDoubleEquity: 0,
            cubefulNoDoubleEquity: 0,
            cubefulNoDoubleError: 0,
            cubefulDoubleTakeEquity: 0,
            cubefulDoubleTakeError: 0,
            cubefulDoublePassEquity: 0,
            cubefulDoublePassError: 0,
            bestCubeAction: '',
            wrongPassPercentage: 0,
            wrongTakePercentage: 0
        },
        allCubeAnalyses: /** @type {import('../../wailsjs/go/models').domain.DoublingCubeAnalysis[]} */ ([]),
        playedMove: '', // Deprecated: for backward compatibility
        playedCubeAction: '', // Deprecated: for backward compatibility
        playedMoves: /** @type {string[]} */ ([]), // All moves played in this position across different matches
        playedCubeActions: /** @type {string[]} */ ([]), // All cube actions taken in this position across different matches
        creationDate: '',
        lastModifiedDate: ''
    };
}

export const analysisStore = writable(emptyAnalysis());

// Store for tracking the selected move in the analysis panel
/** @type {import('svelte/store').Writable<string | null>} */
export const selectedMoveStore = writable(null);

// The Eval panel's live gammonNet result, in analysisStore's shape plus the panel's
// own cube `decision`, for the image copy: in EVAL mode analysisStore describes
// another position. Null when the panel shows nothing to copy.
export const evalAnalysisStore = writable(null);
