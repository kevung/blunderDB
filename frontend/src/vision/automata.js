/**
 * Les automates mains / dés / partie et la cadence d'analyse (ADR-0109 règles 1 et 4).
 *
 * Une machine pure : elle ne voit que des relevés de `descriptors.js`, jamais de pixels, et
 * rend des événements du contrat. Rejouer un journal dans une machine neuve redonne donc les
 * mêmes événements, et c'est sur des journaux que se règlent les seuils de `THRESHOLDS`.
 *
 * ## Deux cadences
 *
 * Hors fenêtre, l'analyse tourne à `stateRate` et compare des **états** : la distance census
 * d'une ROI à l'image analysée précédente, une demi-seconde plus tôt. Des dés déplacés ou un
 * plateau changé la font franchir `E_hi` même si le geste est tombé entre deux images ; elle
 * ouvre alors une fenêtre où l'analyse passe à `denseRate` jusqu'à t + `windowAfter`, et que
 * chaque image active prolonge. Les automates ne comptent leurs durées que sur les images
 * denses. La borne passée de la fenêtre de l'ADR-0109 (−1 s) n'a de sens que pour un parcours
 * par seeks, qui peut revenir en arrière ; en lecture (le chemin opportuniste, à 1× ou plus),
 * cette part est déjà jouée : l'épisode commence au relevé grossier qui l'a vu, et son instant
 * n'est pas sûr.
 *
 * Qu'une image soit dense se lit sur le relevé lui-même : l'écart à la précédente est au plus
 * `denseMaxDt`. La machine ne dépend ainsi que du journal, pas de la cadence qu'elle a demandée.
 *
 * ## Mouvement, mains, dés
 *
 * Chaque ROI a son plancher de bruit : la médiane de sa distance census à l'image précédente sur
 * ses `noiseWindow` dernières images calmes. Une distance census ne tombe jamais à zéro (un
 * aplat retourne des bits sous le grain, et un dé posé est un aplat de plus) : on mesure donc
 * l'**excès** au-dessus du plancher, et l'hystérésis de l'ADR-0109 s'applique à la dispersion
 * σ du bruit (1,4826 × MAD) : `E_hi = max(hiFloor, hiFactor × σ)`, `E_lo = max(loFloor,
 * loFactor × σ)`. `e` est, par image, l'excès maximal sur les ROI.
 *
 * Une main couvre une grande part d'une moitié : au-dessus de `handsIn` pendant `handsInFrames`
 * images elle entre ; sous `handsOut` pendant `handsOutFrames` images elle sort. Entre `E_hi` et
 * `handsIn`, c'est un mouvement de dés : un lancer (`throwFrames` images au moins), puis un repos
 * de `restFrames` images sous `E_lo`, daté de sa première image calme. Un lancer ne commence
 * qu'une main bien partie (`handsOutFrames` images sous `handsIn`), et des dés qui bougent au plus
 * `throwGap` s après la sortie d'une main en sont partis : le lancer prend l'instant de cette
 * sortie. Après un repos, un lancer n'est admis qu'après une main (les dés ont dû être
 * ramassés) ; le ramassage est la dernière entrée de main avant le lancer, ou l'avant-dernière
 * si la dernière est la main qui lance. Le jet d'ouverture est un lancer qui a remué les deux
 * moitiés.
 *
 * ## Partie
 *
 * La distance au fond (`dBg`) est la distance au gabarit de départ quand l'analyse commence
 * sur la position initiale : une partie s'arme quand le plateau s'en éloigne au-delà de
 * `razHi` et recommence quand, calme, il y revient sous `razLo`. Un fond réinitialisé (image
 * incertaine) n'est plus ce gabarit : l'automate de partie se tait jusqu'au prochain `start`.
 *
 * ## États
 *
 * Un événement est **sûr** quand l'image qui l'ouvre est dense : son instant est connu à une
 * image près. Il est **à vérifier** quand il commence sur un relevé grossier, ou quand il est
 * déduit après coup (le ramassage, la fin de partie).
 */

import { EVENT_TYPES, makeEvent, STATES } from './contract.js';

const TUNED = {
    /** Cadence de la comparaison d'états, en images du média par seconde. */
    stateRate: 2,
    /** Cadence des automates dans une fenêtre. */
    denseRate: 10,
    /** Écart maximal (s) entre deux relevés pour que le second compte comme dense : 4 img/s. */
    denseMaxDt: 0.25,
    /** Écart (s) au-delà duquel le fil est rompu (seek, décodeur à l'arrêt) : les automates repartent. */
    maxGap: 2,
    /** Durée (s) de la fenêtre dense après un changement d'état. */
    windowAfter: 2,
    /** Nombre d'images calmes sur lequel se prend la médiane du bruit. */
    noiseWindow: 20,
    /** Images calmes avant que le plancher de bruit soit pris pour connu. */
    noiseMin: 3,
    /** `E_hi` et `E_lo` en multiples de la médiane calme. */
    hiFactor: 3,
    loFactor: 1.5,
    /** Planchers de `E_hi` et `E_lo` quand le bruit est quasi nul (fraction des bits census). */
    hiFloor: 0.003,
    loFloor: 0.002,
    /** Une main entre au-dessus de `handsIn` et sort sous `handsOut`. */
    handsIn: 0.03,
    handsOut: 0.02,
    /** Durées minimales, en images à `denseRate` (ADR-0109 règle 4). */
    throwFrames: 1,
    restFrames: 5,
    handsInFrames: 2,
    handsOutFrames: 3,
    /** Des dés qui bougent au plus `throwGap` s après la sortie d'une main en sont partis. */
    throwGap: 1,
    /** Un lancer qui ne se pose pas en `maxRoll` secondes n'en était pas un. */
    maxRoll: 4,
    /** Distance au gabarit de départ : partie en cours au-delà de `razHi`, plateau remis sous `razLo`. */
    razHi: 0.012,
    razLo: 0.004
};

/** Les seuils, en un seul endroit ; chacun est tenu par `vision.automata.replay.test.js`. */
export const THRESHOLDS = Object.freeze(TUNED);

/** @typedef {typeof THRESHOLDS} Thresholds */
/** @typedef {import('./descriptors.js').DescriptorRecord} DescriptorRecord */
/** @typedef {import('./contract.js').VisionEvent} VisionEvent */

const SIDES = new Set(['left', 'right']);

/**
 * Une durée de `n` images à `denseRate`, ramenée à la cadence mesurée : à 5 img/s on divise
 * par deux, sans descendre sous 2 images (une seule image ne distingue pas un palier d'un pic),
 * sauf pour une durée d'une image.
 * @param {number} n
 * @param {number} dt écart au relevé précédent, en secondes
 * @param {Thresholds} th
 */
export function framesAt(n, dt, th = THRESHOLDS) {
    if (n <= 1) return n;
    const fps = Math.min(th.denseRate, 1 / dt);
    return Math.min(n, Math.max(2, Math.round((n * fps) / th.denseRate)));
}

/** @param {number[]} values */
function median(values) {
    const s = [...values].sort((a, b) => a - b);
    const m = s.length >> 1;
    return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2;
}

/**
 * @param {Partial<Thresholds>} [overrides] pour rejouer un journal sous d'autres seuils
 */
export function createAutomata(overrides = {}) {
    /** @type {Thresholds} */
    const th = Object.freeze({ ...THRESHOLDS, ...overrides });

    /** Recent calm distances of each ROI to its previous frame: its noise floor. @type {Map<string, number[]>} */
    const noise = new Map();
    /** The levels of the current record, as excess over each ROI's noise floor. */
    const lv = { hi: 0, lo: 0, handsIn: th.handsIn, handsOut: th.handsOut, razHi: th.razHi, razLo: th.razLo };
    let lastT = NaN;
    let windowTo = -Infinity;
    let rate = th.stateRate;
    let gameEnabled = true;
    /** A coarse record saw a change and no dense calm frame has followed: starts are not sure. */
    let coarseDirty = false;
    /** Consecutive dense frames under the hands level: a roll needs a hand to be well gone. */
    let sinceHand = 0;

    // Hands.
    let handsIn = false;
    let handsCount = 0;
    /** @type {{ t: number, sure: boolean } | null} */
    let handsStart = null;

    // Dice: 'idle' (history unknown), 'rolling', 'rested'.
    /** @type {'idle' | 'rolling' | 'rested'} */
    let dice = 'idle';
    /** @type {'idle' | 'rested'} */
    let diceBefore = 'idle';
    let handsSinceRest = false;
    /**
     * Hands episodes since the last rest, oldest first: the first places a game's end, the
     * pickup is among the last. `end` is NaN while the hand is in.
     * @type {{ start: number, end: number, coarse: boolean }[]}
     */
    let episodes = [];
    /** @type {{ t: number, sure: boolean, motion: Map<string, number>, moves: number, calm: number, calmStart: { t: number, sure: boolean } | null, steady: boolean } | null} */
    let roll = null;

    // Game.
    let armed = false;
    let razCount = 0;
    /** @type {number | null} */
    let razStart = null;

    /**
     * @param {string} k
     * @param {number} v
     */
    function learn(k, v) {
        let n = noise.get(k);
        if (!n) noise.set(k, (n = []));
        n.push(v);
        if (n.length > th.noiseWindow) n.shift();
    }

    function resetAutomata() {
        coarseDirty = false;
        sinceHand = 0;
        handsIn = false;
        handsCount = 0;
        handsStart = null;
        dice = 'idle';
        diceBefore = 'idle';
        handsSinceRest = false;
        episodes = [];
        roll = null;
        razCount = 0;
        razStart = null;
    }

    /** @param {number} t */
    function extendWindow(t) {
        windowTo = Math.max(windowTo, t + th.windowAfter);
    }

    /** @param {Map<string, number>} motion */
    function sidePayload(motion) {
        const named = [...motion.keys()].filter((k) => SIDES.has(k));
        if (named.length >= 2) return { opening: true };
        let best = null;
        let max = -1;
        for (const [k, v] of motion)
            if (v > max) {
                max = v;
                best = k;
            }
        return best !== null && SIDES.has(best) ? { side: best } : {};
    }

    /**
     * @param {DescriptorRecord} record
     * @returns {VisionEvent[]}
     */
    function push(record) {
        /** @type {VisionEvent[]} */
        const events = [];
        const t = record.t;
        const dt = t - lastT;
        const broken = !(dt > 0) || dt > th.maxGap;
        lastT = t;

        if (record.u === 1) {
            // The backgrounds were reset on this frame: the start template is gone with them.
            resetAutomata();
            gameEnabled = false;
            // A cut or an exposure jump changes the noise too.
            noise.clear();
            // Dense frames relearn the floor in a few tenths of a second instead of seconds.
            extendWindow(t);
            return finish(events, t);
        }
        if (broken) resetAutomata();

        /** @type {Map<string, number>} raw distance of each ROI to its previous frame */
        const raw = new Map();
        for (const [name, v] of Object.entries(record.r)) if (v[1] !== null) raw.set(name, v[1]);
        if (raw.size === 0 || broken) return finish(events, t);

        if ([...raw.keys()].some((k) => (noise.get(k)?.length ?? 0) < th.noiseMin)) {
            // No noise floor yet: nothing can be told from motion.
            for (const [k, v] of raw) learn(k, v);
            return finish(events, t);
        }
        // Each ROI is measured above its own floor: texture and light differ between halves.
        let spread = 0;
        /** @type {Map<string, number>} */
        const floor = new Map();
        for (const k of raw.keys()) {
            const n = /** @type {number[]} */ (noise.get(k));
            const m = median(n);
            floor.set(k, m);
            spread = Math.max(spread, 1.4826 * median(n.map((v) => Math.abs(v - m))));
        }
        lv.hi = Math.max(th.hiFloor, th.hiFactor * spread);
        lv.lo = Math.max(th.loFloor, th.loFactor * spread);
        const { hi, lo } = lv;
        /** @type {Map<string, number>} */
        const e = new Map();
        let eMax = -Infinity;
        let bgMax = -Infinity;
        for (const [k, v] of raw) {
            const x = v - /** @type {number} */ (floor.get(k));
            e.set(k, x);
            eMax = Math.max(eMax, x);
            const bg = record.r[k][2];
            if (bg !== null) bgMax = Math.max(bgMax, bg - /** @type {number} */ (floor.get(k)));
        }
        const dense = dt <= th.denseMaxDt;
        if (!dense && eMax >= hi) coarseDirty = true;
        const sureStart = dense && !coarseDirty;
        if (dense && eMax < hi) coarseDirty = false;
        if (eMax >= lv.handsIn) sinceHand = 0;
        else if (dense) sinceHand++;

        if (eMax >= hi) extendWindow(t);

        if (!dense) {
            coarse(events, t, e, eMax, hi);
        } else {
            denseHands(events, t, dt, eMax, sureStart);
            denseDice(events, t, dt, e, eMax, hi, lo, sureStart);
            if (gameEnabled && bgMax > -Infinity) denseGame(events, t, dt, eMax, bgMax);
        }
        if (dice === 'rolling' || handsIn || handsCount > 0 || razCount > 0) extendWindow(t);
        if (!handsIn && dice !== 'rolling' && eMax < hi) for (const [k, v] of raw) learn(k, v);
        return finish(events, t);
    }

    /**
     * @param {VisionEvent[]} events
     * @param {number} t
     */
    function finish(events, t) {
        rate = t < windowTo ? th.denseRate : th.stateRate;
        return events;
    }

    /**
     * A coarse record compares states half a second apart: a change opens the window, and a
     * dice-sized change with no hand in sight starts a roll the dense frames will settle.
     * @param {VisionEvent[]} events
     * @param {number} t
     * @param {Map<string, number>} e
     * @param {number} eMax
     * @param {number} hi
     */
    function coarse(events, t, e, eMax, hi) {
        handsCount = 0;
        handsStart = null;
        if (roll && t - roll.t > th.maxRoll) cancelRoll();
        if (roll) {
            roll.steady = false;
            roll.calm = 0;
            roll.calmStart = null;
            return;
        }
        if (eMax < hi) return;
        if (eMax >= lv.handsIn) {
            // A hand came and went between two samples: it counts for the pickup, not as an event.
            noteHands(t, true);
            episodes[episodes.length - 1].end = t;
            return;
        }
        if (!handsIn && (dice === 'idle' || handsSinceRest)) startRoll(t, false, e, hi);
    }

    /**
     * Records a hands entry; rends l'instant retenu pour son début.
     * @param {number} t
     * @param {boolean} [coarse] seen only by a coarse record
     */
    function noteHands(t, coarse = false) {
        handsSinceRest = true;
        const last = episodes[episodes.length - 1];
        // A hand first seen by a coarse record, then confirmed densely, is one episode.
        if (!coarse && last?.coarse && t - last.end <= 1 / th.stateRate) {
            last.coarse = false;
            last.end = NaN;
            return last.start;
        }
        episodes.push({ start: t, end: NaN, coarse });
        return t;
    }

    /**
     * @param {number} t
     * @param {boolean} sure
     * @param {Map<string, number>} e
     * @param {number} hi
     */
    function startRoll(t, sure, e, hi) {
        // Dice that start moving as a hand leaves left that hand: the throw is the hand's exit,
        // which the hands automaton saw densely.
        const last = episodes[episodes.length - 1];
        if (last && !last.coarse && t - last.end <= th.throwGap) {
            t = last.end;
            sure = true;
        }
        coarseDirty = false;
        diceBefore = dice === 'rested' ? 'rested' : 'idle';
        dice = 'rolling';
        /** @type {Map<string, number>} */
        const motion = new Map();
        for (const [k, v] of e) if (v >= hi) motion.set(k, v);
        roll = { t, sure, motion, moves: 1, calm: 0, calmStart: null, steady: sure };
    }

    /**
     * The pickup is the latest hands episode before the throw, unless that episode is the
     * throwing hand itself (it ended within `throwGap` of the throw): then the one before it.
     * @param {number} throwAt
     */
    function pickupOf(throwAt) {
        const n = episodes.length;
        if (n === 0) return null;
        if (n >= 2 && throwAt - episodes[n - 1].end <= th.throwGap) return episodes[n - 2].start;
        return episodes[n - 1].start;
    }

    function cancelRoll() {
        dice = diceBefore;
        roll = null;
    }

    /**
     * @param {VisionEvent[]} events
     * @param {number} t
     * @param {number} dt
     * @param {number} eMax
     * @param {boolean} sureStart
     */
    function denseHands(events, t, dt, eMax, sureStart) {
        if (!handsIn) {
            if (eMax >= lv.handsIn) {
                handsStart ??= { t, sure: sureStart };
                if (++handsCount >= framesAt(th.handsInFrames, dt, th)) {
                    handsIn = true;
                    const start = noteHands(handsStart.t);
                    events.push(makeEvent(EVENT_TYPES.HANDS_IN, start, handsStart.sure && start === handsStart.t ? STATES.SURE : STATES.CHECK));
                    coarseDirty = false;
                    // What looked like dice was the hand coming in.
                    if (roll) cancelRoll();
                    handsCount = 0;
                    handsStart = null;
                }
            } else if (eMax < lv.handsOut) {
                // Hysteresis: a frame between the two levels neither confirms nor clears an entry.
                handsCount = 0;
                handsStart = null;
            }
            return;
        }
        if (eMax < lv.handsOut) {
            handsStart ??= { t, sure: true };
            if (++handsCount >= framesAt(th.handsOutFrames, dt, th)) {
                handsIn = false;
                events.push(makeEvent(EVENT_TYPES.HANDS_OUT, handsStart.t, STATES.SURE));
                if (episodes.length) episodes[episodes.length - 1].end = handsStart.t;
                handsCount = 0;
                handsStart = null;
            }
        } else {
            handsCount = 0;
            handsStart = null;
            // The hand is still there: dice motion seen while it was leaving was the hand's.
            if (roll) cancelRoll();
        }
    }

    /**
     * @param {VisionEvent[]} events
     * @param {number} t
     * @param {number} dt
     * @param {Map<string, number>} e
     * @param {number} eMax
     * @param {number} hi
     * @param {number} lo
     * @param {boolean} sureStart
     */
    function denseDice(events, t, dt, e, eMax, hi, lo, sureStart) {
        if (dice !== 'rolling') {
            // Not while a hand is leaving: its tail looks like dice.
            const handGone = sinceHand >= framesAt(th.handsOutFrames, dt, th);
            if (eMax >= hi && handGone && !handsIn && (dice === 'idle' || handsSinceRest)) startRoll(t, sureStart, e, hi);
            return;
        }
        const r = /** @type {NonNullable<typeof roll>} */ (roll);
        // Dice never move like a hand: a hand-sized frame ends the roll before the hand is confirmed.
        if (eMax >= lv.handsIn) {
            cancelRoll();
            return;
        }
        // Still moving past maxRoll: not dice settling but something else on the board.
        if (eMax >= lo && t - r.t > th.maxRoll) {
            cancelRoll();
            return;
        }
        for (const [k, v] of e) if (v >= hi) r.motion.set(k, (r.motion.get(k) ?? 0) + v);
        if (eMax < lo && !handsIn) {
            r.calmStart ??= { t, sure: r.steady };
            r.calm++;
        } else {
            r.calm = 0;
            r.calmStart = null;
            r.steady = true;
            r.moves++;
        }
        if (r.calm < framesAt(th.restFrames, dt, th)) return;
        if (r.moves < framesAt(th.throwFrames, dt, th)) {
            // Too brief to be a throw: a blip.
            cancelRoll();
            return;
        }
        const payload = sidePayload(r.motion);
        const pickup = pickupOf(r.t);
        if (diceBefore === 'rested' && pickup !== null) events.push(makeEvent(EVENT_TYPES.DICE_PICKUP, pickup, STATES.CHECK));
        events.push(makeEvent(EVENT_TYPES.DICE_THROWN, r.t, r.sure ? STATES.SURE : STATES.CHECK, 'side' in payload ? { side: payload.side } : {}));
        const calmStart = /** @type {{ t: number, sure: boolean }} */ (r.calmStart);
        events.push(makeEvent(EVENT_TYPES.DICE_REST, calmStart.t, calmStart.sure ? STATES.SURE : STATES.CHECK, payload));
        dice = 'rested';
        roll = null;
        handsSinceRest = false;
        episodes = [];
    }

    /**
     * @param {VisionEvent[]} events
     * @param {number} t
     * @param {number} dt
     * @param {number} eMax
     * @param {number} bg
     */
    function denseGame(events, t, dt, eMax, bg) {
        if (bg >= lv.razHi) armed = true;
        // No motion is enough here: after a reset the noise floor is still the old board's, and the
        // distance to the template is what says the board is back.
        if (armed && bg < lv.razLo && eMax < lv.hi && (!handsIn || handsCount > 0) && dice !== 'rolling') {
            razStart ??= t;
            if (++razCount < framesAt(th.restFrames, dt, th)) return;
            if (episodes.length) events.push(makeEvent(EVENT_TYPES.GAME_END, episodes[0].start, STATES.CHECK));
            events.push(makeEvent(EVENT_TYPES.GAME_START, razStart, STATES.SURE));
            armed = false;
            // A new game opens on the opening roll, which needs no pickup before it.
            dice = 'idle';
            handsSinceRest = false;
            episodes = [];
        }
        razCount = 0;
        razStart = null;
    }

    return {
        push,
        /** Cadence voulue après le dernier relevé, en images du média par seconde. */
        get rate() {
            return rate;
        },
        /** Les niveaux du dernier relevé (plancher de bruit compris), pour régler les seuils sur un journal. */
        get levels() {
            /** @type {Record<string, number>} */
            const floor = {};
            for (const [k, n] of noise) floor[k] = median(n);
            return { ...lv, floor, dice, handsAreIn: handsIn };
        },
        thresholds: th
    };
}
