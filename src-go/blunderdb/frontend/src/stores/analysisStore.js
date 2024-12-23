import { writable } from 'svelte/store';

export const analysisStore = writable({
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
        wrongTakePercentage: 0,
    },
});


