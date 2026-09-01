import { writable } from 'svelte/store';

/**
 * A fresh, empty analysis record — the shape every consumer of analysisStore
 * expects (AnalysisPanel, the Eval panel, session save). Returns a new object
 * on each call so callers can mutate their copy without touching the store's
 * previous value.
 */
export function emptyAnalysis() {
    return {
        positionId: null,
        xgid: '',
        player1: '',
        player2: '',
        analysisType: '',
        analysisEngineVersion: '',
        checkerAnalysis: {
            moves: []
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
        allCubeAnalyses: [],
        playedMove: '', // Deprecated: for backward compatibility
        playedCubeAction: '', // Deprecated: for backward compatibility
        playedMoves: [], // All moves played in this position across different matches
        playedCubeActions: [], // All cube actions taken in this position across different matches
        creationDate: '',
        lastModifiedDate: ''
    };
}

export const analysisStore = writable(emptyAnalysis());

// Store for tracking the selected move in the analysis panel
export const selectedMoveStore = writable(null);
