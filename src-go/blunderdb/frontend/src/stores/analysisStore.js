import { writable } from 'svelte/store';

export const analysisStore = writable({
    positionId: null,
    xgid: '',
    player1: '',
    player2: '',
    extremeGammonVersion: '',
    analysisType: '',
    doublingCubeAnalysis: null,
    checkerAnalysis: [],
});

