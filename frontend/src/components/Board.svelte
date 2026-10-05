<script>
    import { t, tMsg, translate } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { positionStore, matchContextStore } from '../stores/positionStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { isResponseCubeAction } from '../utils/cubeAction.js';
    import { parseMoveNotation, mirrorPosition, boardMetrics } from '../utils/boardGeometry.js';
    import { layerOf, drawStaticScene, drawDynamicScene, drawFrame } from '../utils/boardScene.js';
    import { defaultBoardConfig, applyPalette } from '../utils/boardConfig.js';
    import { applyStartingCheckers, attachBoardInteractions } from '../utils/boardInteractions.js';
    import { rankNeighboursOfCurrentPosition } from '../services/rankService.js';
    import { onMount, onDestroy } from 'svelte';
    import Two from 'two.js';
    import { get } from 'svelte/store';
    import { statusBarModeStore, isAnyModalOpen, activeModal, MODAL, pipcountVisibleStore, activeTabStore } from '../stores/uiStore';
    import { subscribeBoardRedrawTriggers as subscribeSharedRedrawTriggers } from '../services/boardRedraw.js';
    import { boardIsMirrored, labelsFlipped, screenOfModelPoint, screenOfNotationPoint } from '../services/boardOrientation.js';
    import { searchStructureModeStore, searchOfferedCubeStore } from '../stores/searchExcludePositionStore';
    import { boardColorsStore } from '../stores/boardColorsStore';
    import { sendPositionToEval } from '../services/positionService.js';
    import { copyBoardWithAnalysisImage, exportBoardImage } from '../services/clipboardService.js';
    import { setStatusBarMessage } from '../services/databaseService.js';
    import { viewStore } from '../stores/viewStore.js';
    import * as anki from '../services/ankiService.js';
    import { ankiDecksStore } from '../stores/ankiStore.js';
    import { quizPlayStore, quizPlayTargetsStore } from '../stores/quizPlayStore.js';
    import { transcriptionCubeRequestStore, transcriptionBoardSwapStore } from '../stores/transcriptionStore.js';
    import { resetBoardPlay } from '../services/transcriptionPlay.js';
    import ContextMenu from './ContextMenu.svelte';
    import { onPileStore, refreshPileState, togglePile } from '../services/pileService.js';
    import { duelHoldsBoardStore, duelBoardStore, duelStore } from '../stores/duelStore.js';
    import { duelBoardPress, duelBoardDrop, duelBoardContextMenu, duelBoardContext, suspendDuel, confirmStopDuel, resignDuel } from '../services/duelService.js';
    import { orderedDice, usedDice, isMine } from '../services/duelBoard.js';
    import DuelBoardPrompt from './DuelBoardPrompt.svelte';
    import { registerKeys } from '../services/keyDispatch.js';

    // Read-only mirrors of stores — always current when read inside drawing/handler functions
    let mode = $derived($statusBarModeStore);
    let showTakePoint2Modal = $derived($activeModal === MODAL.TAKE_POINT_2);
    let showTakePoint4Modal = $derived($activeModal === MODAL.TAKE_POINT_4);
    // No PANEL entry for the comment tab: CommentPanel is mounted by TabbedPanel per active tab.
    let showComment = $derived($activeTabStore === 'comments');
    // Préférence, surchargée pendant une question de Pions (uiStore.pipcountVisibleStore) ;
    // le redessin vient de boardRedraw.js.
    let showPipcount = $derived($pipcountVisibleStore);

    let canvasCfg = {
        aspectFactor: 0.72
    };

    // Partagée avec les diagrammes du rapport (utils/boardConfig.js) : une seule palette.
    let boardCfg = defaultBoardConfig();

    /** @typedef {import('../utils/boardGeometry.js').BoardPosition} BoardPosition */
    /** @typedef {{ label: string, onClick: () => void }} MenuItem */
    /** @typedef {import('../utils/boardGeometry.js').BoardMetrics} BoardMetrics */
    /** @typedef {import('two.js').default['scene']} Group */

    /** @type {Two | null} */
    let two = null;
    /** @type {HTMLElement | null} */
    let canvas = null;
    let width = 0;
    let height = 0;
    // Le milieu de la moitié droite du damier, depuis le centre du conteneur : la confirmation du
    // videau d'un Duel s'y pose, là où se lancent les dés.
    let promptAnchor = $state({ dx: 0, dy: 0 });
    /** @type {(() => void) | null} */
    let unsubscribeBoardRedrawTriggers = null;
    /** @type {(() => void) | null} */
    let detachInteractions = null;
    let cubePosition = { x: 0, y: 0, size: 0 }; // where the cube was last drawn (hit-testing)
    let previousDice = get(positionStore).dice; // Save previous dice values

    // enterEvalMode() starts with dice [0, 0]; reset so the first die click
    // does not restore a stale previousDice.
    $effect(() => {
        if (mode === 'EVAL') {
            previousDice = [0, 0];
        }
    });

    // Read by drawBoard(); redraw is triggered by scheduleRedraw().
    let selectedMove = $derived($selectedMoveStore);

    // Redraw coalescing: several stores can dirty the board in one tick.
    // scheduleRedraw() sets a flag and requests one animation frame; by then
    // every store has settled, and drawBoard() (which re-reads them all)
    // paints once per frame.
    let redrawScheduled = false;
    /** @type {number | null} */
    let redrawFrameId = null;
    function scheduleRedraw() {
        if (redrawScheduled) return;
        redrawScheduled = true;
        redrawFrameId = requestAnimationFrame(() => {
            redrawScheduled = false;
            redrawFrameId = null;
            if (two && canvas) drawBoard();
        });
    }

    // Svelte 5 store-rule exception (CLAUDE.md): drawBoard() re-reads every
    // store whichever changed, so one grouped subscription beats an $effect
    // per store. positionStore's callback also resets the selected move on a
    // real navigation (id change) BEFORE the redraw; synchronous .subscribe()
    // makes that ordering unambiguous, an $effect's would not be.
    function subscribeBoardRedrawTriggers() {
        /** @type {number | null} */
        let previousPositionId = null;
        const unsubPosition = positionStore.subscribe(() => {
            const position = get(positionStore);
            // Only on a real navigation (id change), not on edits or analysis refresh.
            if (position.id !== previousPositionId) {
                selectedMoveStore.set(null);
                previousPositionId = position.id;
            }
            scheduleRedraw();
        });

        // Les autres déclencheurs : liste nommée et testée dans services/boardRedraw.js.
        const unsubShared = subscribeSharedRedrawTriggers(scheduleRedraw);

        return () => {
            unsubPosition();
            unsubShared();
        };
    }

    // boardCfg is a plain object read imperatively: mutate in place, then redraw.
    $effect(() => {
        applyPalette(boardCfg, $boardColorsStore);
        invalidateStaticLayer(); // triangles, bar and frame carry the palette
        if (two && canvas) scheduleRedraw();
    });

    // Les libellés du plateau sont peints, pas rendus par Svelte : un changement de langue les repeint.
    $effect(() => {
        void $t;
        if (two && canvas) scheduleRedraw();
    });

    let boardDescription = $derived.by(() => {
        const pos = $positionStore;
        if (!pos || !pos.board || !pos.board.points) return $t('board.label');
        let pip1 = 0;
        let pip2 = 0;
        pos.board.points.forEach((point, index) => {
            if (point.color === 0) pip1 += point.checkers * index;
            else if (point.color === 1) pip2 += point.checkers * (25 - index);
        });
        const roller = pos.player_on_roll === 0 ? $t('board.player1') : $t('board.player2');
        return $t('board.description', { pip1, pip2, roller });
    });
    function resizeBoard() {
        if (!canvas || !two) return;
        const container = /** @type {HTMLElement} */ (canvas.parentElement);
        const containerWidth = container.clientWidth;
        const containerHeight = container.clientHeight;
        const heightFromWidth = containerWidth * canvasCfg.aspectFactor;
        if (heightFromWidth <= containerHeight) {
            width = containerWidth;
            height = heightFromWidth;
        } else {
            height = containerHeight;
            width = containerHeight / canvasCfg.aspectFactor;
        }
        two.width = width;
        two.height = height;
        two.renderer.setSize(width, height);
        invalidateStaticLayer(); // every coordinate depends on the size
        // Measure synchronously, repaint coalesced (a resize burst paints once per frame).
        scheduleRedraw();
    }

    function resetBoard() {
        positionStore.update((pos) => {
            pos.board.points.forEach((point) => (point.checkers = 0));
            pos.board.bearoff = [15, 15]; // Reset bearoff
            pos.cube.value = 0; // Set cube in the middle
            pos.cube.owner = -1; // Reset cube owner
            pos.score = [7, 7]; // Reset score to 7 away for both players
            pos.dice = [3, 1]; // Set dice to 3 and 1
            pos.decision_type = 0; // Checker decision
            pos.player_on_roll = 0; // Player on roll is below
            return pos;
        });
    }

    // Eval's "clear": blank board with enterEvalMode()'s defaults (money, no
    // dice), not resetBoard()'s 7-away score, which the race table would read.
    function resetEvalBoard() {
        positionStore.update((pos) => {
            pos.board.points.forEach((point) => (point.checkers = 0));
            pos.board.bearoff = [15, 15];
            pos.cube.value = 0;
            pos.cube.owner = -1;
            pos.score = [-1, -1]; // money
            pos.dice = [0, 0]; // no move in progress
            pos.decision_type = 0;
            pos.player_on_roll = 0;
            return pos;
        });
    }

    // The clear of the current mode (Backspace's), then the checkers of a new game.
    function startingBoard() {
        if (mode === 'EVAL') resetEvalBoard();
        else resetBoard();
        positionStore.update((pos) => applyStartingCheckers(pos));
    }

    function logCanvasSize() {
        if (!canvas || !two) return;
        const actualWidth = canvas.clientWidth;
        const actualHeight = canvas.clientHeight;
        logger.log('Actual canvas width: ', actualWidth, 'Actual canvas height: ', actualHeight);
        logger.log('Two.js width: ', two.width, 'Two.js height: ', two.height);
    }

    /** @param {'left' | 'right'} orientation */
    function setBoardOrientation(orientation) {
        boardCfg.orientation = orientation;
        invalidateStaticLayer(); // labels and bearoff side move
        scheduleRedraw();
    }

    /** @type {(() => void)[]} */
    let unregisterKeys = [];

    /** @param {KeyboardEvent} event */
    function handleOrientationChange(event) {
        const isAnyModalOpenVal = get(isAnyModalOpen);
        if (isAnyModalOpenVal || showComment) return; // Disable orientation change when any modal or comment panel is open
        if (event.ctrlKey && event.key === 'ArrowLeft') {
            setBoardOrientation('left');
        } else if (event.ctrlKey && event.key === 'ArrowRight') {
            setBoardOrientation('right');
        }
    }

    /** @param {KeyboardEvent} event */
    function handleKeyDown(event) {
        if ((mode !== 'EDIT' && mode !== 'EVAL') || showTakePoint2Modal || showTakePoint4Modal) return; // Disable shortcuts when TakePoint2Modal or TakePoint4Modal is open

        if (event.key === 'Backspace' && document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
            event.preventDefault();
            if (mode === 'EVAL') resetEvalBoard();
            else resetBoard();
        }
    }

    onMount(() => {
        const element = /** @type {HTMLElement} */ (document.getElementById('backgammon-board'));
        canvas = element;
        const container = /** @type {HTMLElement} */ (element.parentElement);
        const params = { width: window.innerWidth, height: window.innerHeight };
        const surface = new Two(params).appendTo(element);
        two = surface;

        const containerWidth = container.clientWidth;
        const containerHeight = container.clientHeight;
        const heightFromWidth = containerWidth * canvasCfg.aspectFactor;
        if (heightFromWidth <= containerHeight) {
            width = containerWidth;
            height = heightFromWidth;
        } else {
            height = containerHeight;
            width = containerHeight / canvasCfg.aspectFactor;
        }
        surface.width = width;
        surface.height = height;
        surface.renderer.setSize(width, height);

        // boardInteractions.js reads live state through getters: no re-attach on redraw.
        detachInteractions = attachBoardInteractions(element, {
            getMode: () => mode,
            getSize: () => ({ width, height }),
            cfg: boardCfg,
            getCubeBox: () => cubePosition,
            stores: {
                position: positionStore,
                structureMode: searchStructureModeStore,
                activeTab: activeTabStore,
                offeredCube: searchOfferedCubeStore,
                anyModalOpen: isAnyModalOpen,
                quizPlay: quizPlayStore,
                // Transcription : le plateau pose la demande, le panneau décide.
                transcriptionCube: transcriptionCubeRequestStore
            },
            // Plateau retourné : point cliqué → point du modèle, 25 - p.
            quizDisplayMirrored: () => displayMirrored(),
            // Côté de sortie du camp au trait : 0 en bas, 1 en haut (en
            // transcription le joueur 2 sort par le haut).
            quizBearoffSide: () => getDisplayPosition().player_on_roll,
            getPreviousDice: () => previousDice,
            setPreviousDice: (/** @type {number[]} */ dice) => (previousDice = dice),
            openContextMenu,
            displayRoller: () => getDisplayPosition().player_on_roll,
            togglePile,
            container,
            // Un Duel tient le plateau : chaque clic est le sien (services/duelBoard.js).
            duel: {
                holds: () => get(duelHoldsBoardStore),
                press: duelBoardPress,
                drop: duelBoardDrop,
                context: duelBoardContextMenu
            },
            logger
        });
        // No direct drawBoard(): the subscriptions fire synchronously and
        // schedule the first paint; drawing here too would build it twice.
        window.addEventListener('resize', resizeBoard);
        unregisterKeys = [registerKeys('boardOrientation', handleOrientationChange), registerKeys('boardEdit', handleKeyDown)];

        unsubscribeBoardRedrawTriggers = subscribeBoardRedrawTriggers();

        logCanvasSize();
        window.addEventListener('resize', logCanvasSize);
    });

    onDestroy(() => {
        if (detachInteractions) detachInteractions();
        window.removeEventListener('resize', resizeBoard);
        window.removeEventListener('resize', logCanvasSize);
        for (const unregister of unregisterKeys) unregister();
        if (unsubscribeBoardRedrawTriggers) unsubscribeBoardRedrawTriggers();
        // Cancel a pending rAF so drawBoard() never hits a detached two.js.
        if (redrawFrameId !== null) cancelAnimationFrame(redrawFrameId);
    });

    // Board context menu; gating (EDIT/EVAL only outside the frame, never over a modal) is
    // boardInteractions.js's.
    /** @type {{ x: number, y: number, items: MenuItem[] } | null} */
    let boardMenu = $state(null);

    /** @param {{ x: number, y: number }} at client coordinates */
    function openContextMenu({ x, y }) {
        if (get(duelHoldsBoardStore)) {
            boardMenu = { x, y, items: duelMenuItems() };
            return;
        }
        // EDIT and EVAL: the menu holds the two resets; the right button edits the board elsewhere.
        if (mode === 'EDIT' || mode === 'EVAL') {
            boardMenu = {
                x,
                y,
                items: [
                    { label: $t('board.menu.clearPosition'), onClick: () => (mode === 'EVAL' ? resetEvalBoard() : resetBoard()) },
                    { label: $t('board.menu.startingPosition'), onClick: () => startingBoard() }
                ]
            };
            return;
        }
        /** @type {MenuItem[]} */
        const items = [];
        // A play in progress (quiz, Transcription): its reset clears the play, not the position.
        if (get(quizPlayStore)) {
            items.push({
                label: $t('training.resetPlay'),
                onClick: () => quizPlayStore.update((s) => (s ? resetBoardPlay(s, get(positionStore)) : s))
            });
        }
        items.push(
            {
                label: $t('board.menu.evaluate'),
                // The position as displayed (possibly mirrored), not the stored record.
                onClick: () => sendPositionToEval(getDisplayPosition())
            },
            {
                label: $t('board.menu.evaluateMirror'),
                onClick: () => sendPositionToEval(mirrorPosition(getDisplayPosition()))
            },
            {
                label: $t('board.menu.copyImageWithAnalysis'),
                onClick: () => copyBoardWithAnalysisImage()
            },
            // Un fichier (souvent vectoriel) plutôt qu'une copie.
            {
                label: $t('board.menu.saveImageSVG'),
                onClick: () => exportBoardImage('svg')
            },
            {
                label: $t('board.menu.saveImagePNG'),
                onClick: () => exportBoardImage('png')
            },
            {
                label: $t('board.menu.newView'),
                onClick: () => viewStore.addView()
            }
        );

        // Anki cards are keyed by position id: nothing to add for a scratch board (id 0).
        const position = get(positionStore);
        if (position?.id) {
            items.push(...ankiDeckMenuItems(position.id));
            // « Positions voisines » (ADR-0043) ; pas pour un plateau brouillon (id 0).
            items.push({
                label: $t('board.menu.neighbours'),
                onClick: () => rankNeighboursOfCurrentPosition()
            });
        }

        boardMenu = { x, y, items };

        // Decks load only once the Anki tab is visited: fetch here, non-blocking,
        // and append the deck items when it resolves.
        if (position?.id && get(ankiDecksStore).length === 0) {
            anki.loadDecks()
                .then(() => {
                    if (boardMenu?.x !== x || boardMenu?.y !== y) return; // menu closed/reopened meanwhile
                    const extra = ankiDeckMenuItems(position.id);
                    if (extra.length > 0) boardMenu = { ...boardMenu, items: [...boardMenu.items, ...extra] };
                })
                .catch((err) => logger.error('Failed to load Anki decks for the board context menu:', err));
        }
    }

    // Le menu d'un Duel : rien qui évalue, analyse ou édite la position — le moteur se tait et le
    // Duel tient le plateau (ADR-0072 règle 9). Céder est une Action du jeu, à son tour seulement.
    /** @returns {MenuItem[]} */
    function duelMenuItems() {
        const mine = isMine(duelBoardContext());
        const cube = get(duelStore)?.state?.awaiting?.position?.cube?.value || 1;
        /** @type {MenuItem[]} */
        const resign = [1, 2, 3].map((level) => ({
            label: $t(level === 1 ? 'duel.menu.resignSingle' : level === 2 ? 'duel.menu.resignGammon' : 'duel.menu.resignBackgammon', { n: level * cube }),
            disabled: !mine,
            onClick: () => resignDuel(/** @type {1|2|3} */ (level))
        }));
        return [
            { label: $t(get(onPileStore) ? 'duel.menu.pileOff' : 'duel.menu.pileOn'), onClick: () => togglePile() },
            ...resign,
            { label: $t('duel.suspend'), onClick: () => suspendDuel() },
            { label: $t('duel.stopKeep'), onClick: () => confirmStopDuel(true) },
            { label: $t('duel.stopDiscard'), onClick: () => confirmStopDuel(false) }
        ];
    }

    /** @param {number} positionId */
    function ankiDeckMenuItems(positionId) {
        return get(ankiDecksStore).map((deck) => ({
            label: $t('board.menu.addToAnkiDeck', { deck: deck.name }),
            onClick: () => addPositionToAnkiDeck(deck, positionId)
        }));
    }

    /**
     * @param {{ id: number, name: string }} deck
     * @param {number} positionId
     */
    async function addPositionToAnkiDeck(deck, positionId) {
        try {
            await anki.addPositionToDeck(deck.id, positionId);
            setStatusBarMessage(tMsg('status.positionAddedToDeck', { deck: deck.name }));
        } catch (err) {
            logger.error('Failed to add the position to the Anki deck:', err);
            setStatusBarMessage(tMsg('status.failedAddToDeck', { err }));
        }
    }

    // Le sens du plateau est décidé une fois (services/boardOrientation.js) pour
    // les pions, les clics et les flèches.
    /** @param {BoardPosition} position */
    function displayIsMirrored(position) {
        return boardIsMirrored({
            mode,
            position,
            matchContext: get(matchContextStore),
            transcriptionSwap: get(transcriptionBoardSwapStore)
        });
    }

    // La position telle qu'elle est DESSINÉE.
    function getDisplayPosition() {
        const stored = get(positionStore);
        // Quiz : seul le damier suit le coup en cours ; le reste vient de la position.
        const play = get(quizPlayStore);
        let position = play ? { ...stored, board: play.board } : stored;
        // Duel : les dés dans l'ordre où le joueur les a rangés (le premier est celui qu'un clic joue).
        if (get(duelHoldsBoardStore) && get(duelBoardStore).swapped) position = { ...position, dice: orderedDice(position.dice, true) };
        return displayIsMirrored(position) ? mirrorPosition(position) : position;
    }

    // Le miroir en vigueur ; le coup en cours n'y entre pas.
    function displayMirrored() {
        return displayIsMirrored(get(positionStore));
    }

    // Points numérotés depuis le camp du joueur 2 ? Distinct du miroir : en
    // transcription le joueur 1 reste en bas, on renumérote sans retourner.
    /** @param {BoardPosition} displayPosition */
    function isPlayer2Perspective(displayPosition) {
        return labelsFlipped(displayPosition);
    }

    // Take/pass decision: the offered cube is drawn mid-board. Signalled by the
    // played cube action (match move, else analysis); in EDIT only for an
    // explicit take/pass search, never from stale analysis.
    /** @param {BoardPosition} position */
    function isOfferedCube(position) {
        if (position.decision_type !== 1) return false;
        if (mode === 'EDIT') return get(searchOfferedCubeStore) === true;
        const matchCtx = get(matchContextStore);
        if (matchCtx && matchCtx.isMatchMode && matchCtx.movePositions.length > 0) {
            const mp = /** @type {{ cube_action?: string } | undefined} */ (matchCtx.movePositions[matchCtx.currentIndex]);
            return !!mp && isResponseCubeAction(mp.cube_action);
        }
        const ana = get(analysisStore);
        /** @type {string[]} */
        const acts = (ana && ana.playedCubeActions) || [];
        return acts.some(isResponseCubeAction);
    }

    // Points offerts par le coup en cours, en numéros affichés : les pas de
    // `LegalMoves` sont absolus, convertis par le même `mirrored` que le clic.
    /** @param {boolean} mirrored */
    function playHighlights(mirrored) {
        const play = get(quizPlayStore);
        if (!play) return {};
        const shown = (/** @type {number} */ point) => screenOfModelPoint(point, mirrored);
        return {
            targets: [...$quizPlayTargetsStore].map(shown),
            selected: play.selected === null || play.selected === undefined ? null : shown(play.selected)
        };
    }

    // Flèches : la notation est dans la numérotation du camp au trait, donc
    // `flip` (joueur dessiné en haut), pas `mirrored` — ils divergent en transcription.
    /** @param {boolean} flipped */
    function selectedMoveArrows(flipped) {
        const moves = parseMoveNotation(selectedMove);
        if (moves.length === 0 || !flipped) return moves;
        return moves.map((m) => ({ ...m, from: screenOfNotationPoint(m.from, true), to: screenOfNotationPoint(m.to, true) }));
    }

    // Static layer (triangles, labels, bar, outline) is rebuilt only on size,
    // orientation, palette or numbering change; scheduleRedraw() refills only
    // the dynamic group. Board.redraw.test.js counts two.clear() as a static
    // rebuild and two.update() as a paint.
    /** @type {Group | null} */
    let staticLayer = null; // triangles, labels, bar — null = must be rebuilt
    /** @type {Group | null} */
    let dynamicLayer = null; // emptied and refilled on every redraw
    /** @type {boolean | null} */
    let staticFlip = null; // the label side staticLayer was built for

    function invalidateStaticLayer() {
        staticLayer = null;
    }

    /**
     * @param {Two} surface
     * @param {BoardMetrics} geom
     * @param {boolean} flip
     * @returns {Group}
     */
    function rebuildStaticLayers(surface, geom, flip) {
        surface.clear();
        const fixed = surface.makeGroup();
        const dynamic = surface.makeGroup();
        const frameLayer = surface.makeGroup(); // above the checkers so the outline keeps its linewidth
        drawStaticScene(layerOf(surface, fixed), geom, boardCfg, flip);
        drawFrame(layerOf(surface, frameLayer), geom, boardCfg);
        staticLayer = fixed;
        dynamicLayer = dynamic;
        staticFlip = flip;
        return dynamic;
    }

    /** @type {import('../utils/boardScene.js').SceneText} */
    const sceneText = (key, params) => translate(`board.scene.${key}`, params);

    export function drawBoard() {
        if (!two) return; // Safety check

        const geom = boardMetrics(width, height, boardCfg.widthFactor);
        const position = getDisplayPosition();
        const dx = geom.originX + geom.boardWidth / 4 - width / 2;
        const dy = geom.originY - height / 2;
        if (dx !== promptAnchor.dx || dy !== promptAnchor.dy) promptAnchor = { dx, dy };
        // `mirrored` convertit un point absolu, `flip` un point de notation (boardOrientation.js).
        const flip = isPlayer2Perspective(position);
        const mirrored = displayMirrored();
        logger.log('drawBoard', width, height, 'decision_type:', position.decision_type);

        const dynamic = !staticLayer || !dynamicLayer || staticFlip !== flip ? rebuildStaticLayers(two, geom, flip) : dynamicLayer;
        dynamic.remove(dynamic.children);
        cubePosition = drawDynamicScene(layerOf(two, dynamic), geom, boardCfg, position, {
            text: sceneText,
            offeredCube: isOfferedCube(position),
            showPipcount,
            play: playHighlights(mirrored),
            moves: selectedMoveArrows(flip),
            // Duel : un dé joué est grisé.
            diceUsed: get(duelHoldsBoardStore) ? usedDice(get(quizPlayStore), position.dice) : null
        });

        two.update();
    }

    // The Pile marker follows the position on the board, edits of a draft included.
    $effect(() => {
        void $positionStore;
        refreshPileState();
    });
</script>

<div class="canvas-container">
    <div id="backgammon-board" class="full-size-board" role="img" aria-label={boardDescription}></div>
    {#if $onPileStore}
        <span class="pile-badge" title={$t('board.onPile')} aria-label={$t('board.onPile')}>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" width="20" height="20" aria-hidden="true">
                <path d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z" />
            </svg>
        </span>
    {/if}
    {#if $duelHoldsBoardStore}
        <DuelBoardPrompt anchor={promptAnchor} />
    {/if}
    {#if boardMenu}
        <ContextMenu x={boardMenu.x} y={boardMenu.y} items={boardMenu.items} onClose={() => (boardMenu = null)} />
    {/if}
</div>

<style>
    .pile-badge {
        position: absolute;
        top: 6px;
        right: 6px;
        color: var(--color-primary);
        pointer-events: none;
    }

    .canvas-container {
        position: relative;
        width: 100%;
        height: 100%;
        display: flex;
        justify-content: center;
        align-items: center;
        margin: 0;
        padding: 0;
        overflow: hidden;
    }

    #backgammon-board {
        max-width: 100%;
        max-height: 100%;
        box-sizing: border-box;
        padding: 0;
        margin: 0;
        user-select: none;
    }
</style>
