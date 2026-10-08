/**
 * The clock a test hands the Transcription panel in place of a player: jsdom plays no
 * media, so the instant is set by hand and the seeks and toggles are recorded.
 */
export const fakeVideo = {
    /** @type {number | null} */
    now: null,
    /** @type {number[]} */
    seeks: [],
    toggles: 0,
    /** @type {number[]} */
    rateSteps: [],
    reset() {
        this.now = null;
        this.seeks = [];
        this.toggles = 0;
        this.rateSteps = [];
    }
};
