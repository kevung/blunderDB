import { writable } from 'svelte/store';

export const statusBarTextStore = writable('');
export const statusBarModeStore = writable('NORMAL');

export const commandTextStore = writable('');

export const commentTextStore = writable('');

export const analysisDataStore = writable('This is where your analysis data will be displayed.');

export const currentPositionIndexStore = writable(0); // Add current position index store

export const showSearchModalStore = writable(false); // Add store for search modal visibility

export const showMetModalStore = writable(false); // Add store for MET modal visibility

export const showTakePoint2LastModalStore = writable(false); // Add store for TakePoint2Last modal visibility

export const showTakePoint2LiveModalStore = writable(false); // Add store for TakePoint2Live modal visibility

export const showTakePoint4LastModalStore = writable(false); // Add store for TakePoint4Last modal visibility

export const showTakePoint4LiveModalStore = writable(false); // Add store for TakePoint4Live modal visibility

export const showGammonValue1ModalStore = writable(false); // Add store for GammonValue1 modal visibility

export const showGammonValue2ModalStore = writable(false); // Add store for GammonValue2 modal visibility

export const showGammonValue4ModalStore = writable(false); // Add store for GammonValue4 modal visibility

export const showWarningModalStore = writable(false); // Add store for warning modal visibility

export const showMetadataModalStore = writable(false); // Add store for metadata modal visibility

export const showCommandStore = writable(false);
export const showAnalysisStore = writable(false);
export const showHelpStore = writable(false);
export const showCommentStore = writable(false);
export const showGoToPositionModalStore = writable(false);

export const showTakePoint2ModalStore = writable(false); // Add store for TakePoint2 modal visibility
export const showTakePoint4ModalStore = writable(false); // Add store for TakePoint4 modal visibility
