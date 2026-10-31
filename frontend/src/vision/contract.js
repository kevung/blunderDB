/**
 * Contrat de Z-vision : ce que le panneau reçoit et ce qu'il peut demander.
 *
 * Z-vision lit des pixels et ne connaît pas Svelte ; le panneau (Z-panneau) affiche et pose
 * des Repères et ne connaît pas les pixels. Entre les deux passent seulement un flux
 * d'événements et une poignée de commandes, définis ici. Ce contrat est figé : y ajouter un
 * type, un état ou une commande demande un ADR (ADR-0109).
 *
 * ## Événement
 *
 * `{ type, mediaTime, state, payload }`, objet gelé, construit par `makeEvent` :
 *
 * - `type` : une valeur de `EVENT_TYPES` ;
 * - `mediaTime` : l'instant du média en secondes (même échelle que `HTMLMediaElement.currentTime`),
 *   celui de la **première image de l'épisode** (la première image calme d'un repos, pas celle
 *   qui le confirme). L'horloge est `metadata.mediaTime` de `requestVideoFrameCallback` si un
 *   premier rappel est reçu, sinon `currentTime` lu dans `requestAnimationFrame` (voir
 *   `clock.js`) : à 1× l'écart est d'au plus une image, en lecture accélérée ±1 image ;
 * - `state` : une valeur de `STATES` — sûr, à vérifier, incertain (ADR-0108 règle 5) ; jamais
 *   un pourcentage ;
 * - `payload` : objet gelé propre au type, décrit à côté de chaque type ci-dessous ; `{}` par
 *   défaut. Le panneau ne lit que les champs documentés.
 *
 * ## Commandes
 *
 * Les méthodes de l'objet rendu par `createVision` (`vision.js`), nommées par `COMMANDS` :
 *
 * - `calibrate(calibration)` : `{ width, height, rois: [{ name, x, y, w, h }], witness: { x, y, w, h } }`,
 *   rectangles en pixels de l'image réduite (320×180 par défaut) ; la ROI témoin est hors
 *   plateau. Remplace la calibration courante et réinitialise les fonds ;
 * - `start({ session })` : commence à échantillonner la vidéo ; `session` nomme le journal
 *   (par exemple le `draftId` et le nom de la calibration). Sans calibration, aucune image
 *   n'est analysée ;
 * - `stop()` : arrête l'échantillonnage et vide le journal en attente vers IndexedDB ;
 * - `clearJournal(session?)` : efface le journal d'une session, ou tous sans argument ; rend
 *   une promesse résolue une fois l'effacement fait.
 *
 * `subscribe(listener)` reçoit chaque événement et rend sa fonction de désabonnement ;
 * `dispose()` arrête tout et termine le Worker.
 */

/** Les trois états d'un événement (ADR-0108 règle 5). */
export const STATES = Object.freeze({
    SURE: 'sure',
    CHECK: 'check',
    UNCERTAIN: 'uncertain'
});

/**
 * Les types d'événement. Charge (`payload`) de chacun :
 *
 * - `dice.thrown` : les dés quittent la main ; `{ side? }` (`'left' | 'right'`, moitié du
 *   plateau, absent si inconnue) ;
 * - `dice.rest` : les dés se posent et restent immobiles ; `{ side?, opening? }` (`opening`
 *   vrai pour le jet d'ouverture, un dé par moitié) ;
 * - `dice.pickup` : les dés sont ramassés ; `{ side? }` ;
 * - `hands.in` / `hands.out` : une main entre sur le plateau / en sort ; `{}` ;
 * - `game.start` / `game.end` : le plateau prend / quitte la position d'une partie ; `{}` ;
 * - `view.lost` / `view.found` : le plateau calibré sort de l'image / y revient (hystérésis
 *   du score de présence, ADR-0109 règle 3) ; `{ score }` ; entre les deux l'analyse se tait ;
 * - `frame.uncertain` : la ROI témoin a changé au-delà du seuil, l'image est incertaine et
 *   les fonds sont réinitialisés ; `{ witness }`, la fraction des pixels du témoin dont la signature census a changé ;
 * - `overlay.changed` : l'overlay du stream a changé sans action attendue (ADR-0109 règle 7) ;
 *   `{ distance }`, distance de Hamming du dHash ;
 * - `vision.clock` : l'horloge utilisée change ; `{ source }`, `'rvfc'` ou `'raf'` ;
 * - `vision.error` : l'analyse s'interrompt ; `{ reason }`, dont `'tainted'` (le canvas est
 *   devenu illisible, `SecurityError` — recharger la source) et `'worker'`.
 */
export const EVENT_TYPES = Object.freeze({
    DICE_THROWN: 'dice.thrown',
    DICE_REST: 'dice.rest',
    DICE_PICKUP: 'dice.pickup',
    HANDS_IN: 'hands.in',
    HANDS_OUT: 'hands.out',
    GAME_START: 'game.start',
    GAME_END: 'game.end',
    VIEW_LOST: 'view.lost',
    VIEW_FOUND: 'view.found',
    FRAME_UNCERTAIN: 'frame.uncertain',
    OVERLAY_CHANGED: 'overlay.changed',
    CLOCK: 'vision.clock',
    ERROR: 'vision.error'
});

/** Les commandes du panneau vers Z-vision : noms des méthodes de `createVision`. */
export const COMMANDS = Object.freeze({
    CALIBRATE: 'calibrate',
    START: 'start',
    STOP: 'stop',
    CLEAR_JOURNAL: 'clearJournal'
});

const TYPE_VALUES = new Set(Object.values(EVENT_TYPES));
const STATE_VALUES = new Set(Object.values(STATES));

/**
 * @typedef {{ readonly type: string, readonly mediaTime: number, readonly state: string, readonly payload: Readonly<Record<string, unknown>> }} VisionEvent
 */

/**
 * Construit un événement conforme au contrat ; lève sur un type, un instant ou un état hors
 * contrat, pour qu'une erreur de Z-vision ne traverse jamais vers le panneau.
 * @param {string} type
 * @param {number} mediaTime
 * @param {string} state
 * @param {Record<string, unknown>} [payload]
 * @returns {VisionEvent}
 */
export function makeEvent(type, mediaTime, state, payload = {}) {
    if (!TYPE_VALUES.has(type)) throw new TypeError(`vision: unknown event type ${type}`);
    if (!Number.isFinite(mediaTime) || mediaTime < 0) throw new RangeError(`vision: invalid mediaTime ${mediaTime}`);
    if (!STATE_VALUES.has(state)) throw new TypeError(`vision: unknown state ${state}`);
    if (payload === null || typeof payload !== 'object' || Array.isArray(payload)) throw new TypeError('vision: payload must be an object');
    return Object.freeze({ type, mediaTime, state, payload: Object.freeze({ ...payload }) });
}

/**
 * Vérifie une calibration et la rend normalisée (entiers, ROI dans l'image, au moins 3×3 pour
 * que le census ait un intérieur, noms uniques).
 * @param {{ width?: number, height?: number, overlay?: { x: number, y: number, w: number, h: number } | null, rois: Array<{ name: string, x: number, y: number, w: number, h: number }>, witness: { x: number, y: number, w: number, h: number } }} calibration
 */
export function normalizeCalibration(calibration) {
    if (!calibration || typeof calibration !== 'object') throw new TypeError('vision: calibration required');
    const width = calibration.width ?? 320;
    const height = calibration.height ?? 180;
    if (!Number.isInteger(width) || !Number.isInteger(height) || width < 3 || height < 3) throw new RangeError('vision: invalid frame size');
    /** @param {{ x: number, y: number, w: number, h: number }} r @param {string} name */
    const rect = (r, name) => {
        if (!r || ![r.x, r.y, r.w, r.h].every(Number.isInteger)) throw new TypeError(`vision: ROI ${name} needs integer x, y, w, h`);
        if (r.w < 3 || r.h < 3 || r.x < 0 || r.y < 0 || r.x + r.w > width || r.y + r.h > height) throw new RangeError(`vision: ROI ${name} outside the frame or under 3×3`);
        return Object.freeze({ name, x: r.x, y: r.y, w: r.w, h: r.h });
    };
    if (!Array.isArray(calibration.rois) || calibration.rois.length === 0) throw new TypeError('vision: at least one ROI');
    const names = new Set();
    const rois = calibration.rois.map((r) => {
        if (typeof r?.name !== 'string' || r.name === '' || r.name === 'witness' || names.has(r.name)) throw new TypeError(`vision: ROI name ${r?.name} invalid or repeated`);
        names.add(r.name);
        return rect(r, r.name);
    });
    const witness = rect(calibration.witness, 'witness');
    if (calibration.overlay === undefined || calibration.overlay === null) return Object.freeze({ width, height, rois: Object.freeze(rois), witness });
    const overlay = rect(calibration.overlay, 'overlay');
    if (overlay.w < 9 || overlay.h < 8) throw new RangeError('vision: overlay ROI under the 9×8 hash grid');
    return Object.freeze({ width, height, rois: Object.freeze(rois), witness, overlay });
}
